package ruby

import (
	"bytes"
	"errors"
	"fmt"
	"math"
	"math/big"
	"reflect"
	"strings"
	"testing"

	rb "github.com/go-embedded-ruby/ruby"
	nd "github.com/go-ndarray/ndarray"
)

// newInterp is an interpreter with every module registered.
func newInterp(t *testing.T) *rb.Interp {
	t.Helper()
	var out bytes.Buffer
	in := rb.New(rb.WithStdout(&out), rb.WithStderr(&out))
	if err := Register(in); err != nil {
		t.Fatal(err)
	}
	return in
}

// eval runs src and returns its value converted to Go.
func eval(t *testing.T, in *rb.Interp, src string) any {
	t.Helper()
	v, err := in.Eval(src, "test.rb", 1)
	if err != nil {
		t.Fatalf("%s: %v", src, err)
	}
	return v.Go()
}

// evalErr runs src, which must raise an exception of class whose message
// contains msg.
func evalErr(t *testing.T, in *rb.Interp, src, class, msg string) {
	t.Helper()
	_, err := in.Eval(src, "test.rb", 1)
	var e *rb.Error
	if !errors.As(err, &e) {
		t.Fatalf("%s: no exception (err = %v)", src, err)
	}
	if e.Class != class || !strings.Contains(e.Message, msg) {
		t.Fatalf("%s: raised %s: %s; want %s containing %q", src, e.Class, e.Message, class, msg)
	}
}

// same reports whether a value Ruby returned (converted to Go) is the Go
// result, bit for bit: floats compare by their bits (NaN equal to NaN), an
// NDArray equals a []float64 or an NDArray with the same elements and shape,
// an Array a slice element by element, and structs field by field.
func same(got, want any) bool {
	return sameValue(reflect.ValueOf(got), reflect.ValueOf(want))
}

func sameValue(g, w reflect.Value) bool {
	if !g.IsValid() || !w.IsValid() {
		return g.IsValid() == w.IsValid() || isNilish(g) && isNilish(w)
	}
	if g.Kind() == reflect.Interface {
		return sameValue(g.Elem(), w)
	}
	if w.Kind() == reflect.Interface {
		return sameValue(g, w.Elem())
	}
	if ga, ok := g.Interface().(*nd.Array); ok && ga != nil {
		return sameFloats(arrayFloats(ga), floatsOf(w), shapeOf(ga), shapeOfValue(w))
	}
	if wa, ok := w.Interface().(*nd.Array); ok && wa != nil {
		return sameFloats(floatsOf(g), arrayFloats(wa), shapeOfValue(g), shapeOf(wa))
	}
	switch w.Kind() {
	case reflect.Float64, reflect.Float32:
		gf, ok := asFloat(g)
		wf := w.Float()
		return ok && (math.Float64bits(gf) == math.Float64bits(wf) || math.IsNaN(gf) && math.IsNaN(wf))
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return g.CanInt() && g.Int() == w.Int()
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return g.CanInt() && uint64(g.Int()) == w.Uint()
	case reflect.Slice, reflect.Array:
		if w.Kind() == reflect.Slice && w.IsNil() {
			return isNilish(g)
		}
		if g.Kind() != reflect.Slice && g.Kind() != reflect.Array || g.Len() != w.Len() {
			return false
		}
		for i := range w.Len() {
			if !sameValue(g.Index(i), w.Index(i)) {
				return false
			}
		}
		return true
	case reflect.Pointer:
		if w.IsNil() {
			return isNilish(g)
		}
		if g.Kind() == reflect.Pointer {
			if b, ok := w.Interface().(*big.Int); ok {
				gb, ok := g.Interface().(*big.Int)
				return ok && gb.Cmp(b) == 0
			}
			return sameValue(g.Elem(), w.Elem())
		}
		return sameValue(g, w.Elem())
	case reflect.Struct:
		if g.Kind() == reflect.Pointer {
			g = g.Elem()
		}
		if g.Type() != w.Type() {
			return false
		}
		for i := range w.NumField() {
			if !w.Type().Field(i).IsExported() {
				continue
			}
			if !sameValue(g.Field(i), w.Field(i)) {
				return false
			}
		}
		return true
	case reflect.Map:
		if g.Kind() != reflect.Map || g.Len() != w.Len() {
			return false
		}
		for _, k := range w.MapKeys() {
			gv := g.MapIndex(k)
			if !gv.IsValid() {
				// Ruby keys convert: a Go string key may be a Symbol there.
				gv = g.MapIndex(reflect.ValueOf(rb.Symbol(fmt.Sprint(k.Interface()))))
			}
			if !gv.IsValid() || !sameValue(gv, w.MapIndex(k)) {
				return false
			}
		}
		return true
	}
	return reflect.DeepEqual(g.Interface(), w.Interface())
}

func isNilish(v reflect.Value) bool {
	if !v.IsValid() {
		return true
	}
	switch v.Kind() {
	case reflect.Pointer, reflect.Slice, reflect.Map, reflect.Interface:
		return v.IsNil()
	}
	return false
}

func asFloat(v reflect.Value) (float64, bool) {
	switch v.Kind() {
	case reflect.Float64, reflect.Float32:
		return v.Float(), true
	}
	return 0, false
}

func floatsOf(v reflect.Value) []float64 {
	if v.Kind() == reflect.Interface {
		v = v.Elem()
	}
	if a, ok := v.Interface().(*nd.Array); ok {
		return arrayFloats(a)
	}
	if v.Kind() == reflect.Float64 {
		return []float64{v.Float()}
	}
	if v.Kind() != reflect.Slice && v.Kind() != reflect.Array {
		return nil
	}
	out := make([]float64, v.Len())
	for i := range out {
		e := v.Index(i)
		if e.Kind() == reflect.Interface {
			e = e.Elem()
		}
		out[i], _ = asFloat(e)
	}
	return out
}

func shapeOf(a *nd.Array) []int { return a.Shape() }

func shapeOfValue(v reflect.Value) []int {
	if v.Kind() == reflect.Interface {
		v = v.Elem()
	}
	if a, ok := v.Interface().(*nd.Array); ok {
		return a.Shape()
	}
	if v.Kind() == reflect.Float64 {
		return []int{}
	}
	return []int{v.Len()}
}

func sameFloats(a, b []float64, sa, sb []int) bool {
	if len(a) != len(b) {
		return false
	}
	// A 1-d slice matches an array of any shape with its elements (the Go
	// side often returns []float64 where Ruby gets an NDArray).
	if len(sa) > 1 && len(sb) > 1 && !reflect.DeepEqual(sa, sb) {
		return false
	}
	for i := range a {
		if math.Float64bits(a[i]) != math.Float64bits(b[i]) && !(math.IsNaN(a[i]) && math.IsNaN(b[i])) {
			return false
		}
	}
	return true
}

// check fails unless the Ruby source evaluates to the Go value.
func check(t *testing.T, in *rb.Interp, src string, want any) {
	t.Helper()
	got := eval(t, in, src)
	if !same(got, want) {
		t.Errorf("%s\n  ruby: %#v\n  go:   %#v", src, got, want)
	}
}

// must is a Go call's value, failing the test on its error.
func must[T any](t *testing.T) func(v T, err error) T {
	return func(v T, err error) T {
		t.Helper()
		if err != nil {
			t.Fatal(err)
		}
		return v
	}
}

// same is the instrument of every comparison with Go, so it must see a
// difference of one ulp, of shape, of length and of a field.
func TestSameIsStrict(t *testing.T) {
	type pair struct{ A, B float64 }
	a2, _ := nd.FromData([]float64{1, 2, 3, 4}, 2, 2)
	a4, _ := nd.FromData([]float64{1, 2, 3, 4}, 4, 1)
	for i, c := range []struct {
		got, want any
		same      bool
	}{
		{1.0, 1.0, true},
		{1.0, math.Nextafter(1, 2), false},
		{math.NaN(), math.NaN(), true},
		{int64(3), 3, true},
		{int64(3), 4, false},
		{[]any{1.0, 2.0}, []float64{1, 2}, true},
		{[]any{1.0, 2.0}, []float64{1, 2, 3}, false},
		{a2, []float64{1, 2, 3, 4}, true},
		{a2, a4, false},
		{pair{1, 2}, pair{1, 2}, true},
		{pair{1, 2}, pair{1, 3}, false},
		{&pair{1, 2}, pair{1, 2}, true},
		{nil, (*pair)(nil), true},
		{"a", "a", true},
		{"a", "b", false},
		{map[any]any{rb.Symbol("k"): 1.0}, map[string]float64{"k": 1}, true},
		{map[any]any{rb.Symbol("k"): 2.0}, map[string]float64{"k": 1}, false},
	} {
		if got := same(c.got, c.want); got != c.same {
			t.Errorf("case %d: same(%#v, %#v) = %v", i, c.got, c.want, got)
		}
	}
}
