package ruby

import (
	"fmt"
	"reflect"
	"strings"
	"testing"

	"github.com/go-numerics/special"
)

// specialArgs are arguments in each function's domain: reals and orders.
func specialArgs(ft reflect.Type, real float64) ([]reflect.Value, []string) {
	vals := make([]reflect.Value, ft.NumIn())
	src := make([]string, ft.NumIn())
	for i := range vals {
		switch ft.In(i).Kind() {
		case reflect.Float64:
			x := real + 0.25*float64(i)
			vals[i], src[i] = reflect.ValueOf(x), fmt.Sprint(x)
		case reflect.Int:
			vals[i], src[i] = reflect.ValueOf(2+i), fmt.Sprint(2+i)
		case reflect.Int64:
			vals[i], src[i] = reflect.ValueOf(int64(5-i)), fmt.Sprint(5-i)
		case reflect.Slice:
			vals[i], src[i] = reflect.ValueOf([]float64{0.5, 1.5, -1}), "[0.5, 1.5, -1]"
		}
	}
	return vals, src
}

// Every function of Special is go-numerics/special's on the same inputs; a
// ufunc also elementwise over an Array and an NDArray, with broadcasting.
func TestSpecialMatchesGo(t *testing.T) {
	in := newInterp(t)
	for _, f := range specialFuncs {
		fn := reflect.ValueOf(f.fn)
		ft := fn.Type()
		name := "Special." + snakeName(f.name)
		switch {
		case ft == reflect.TypeOf(special.Softmax):
			check(t, in, name+"([0.5, 1.5, -1])", fn.Call([]reflect.Value{reflect.Zero(ft.In(0)), reflect.ValueOf([]float64{0.5, 1.5, -1})})[0].Interface())
			continue
		case !isUfunc(ft):
			vals, src := specialArgs(ft, 0.7)
			out := fn.Call(vals)
			var want any = out[0].Interface()
			if len(out) == 2 {
				want = []any{out[0].Interface(), out[1].Interface()}
			}
			check(t, in, name+"("+strings.Join(src, ", ")+")", want)
			continue
		}
		for _, x := range []float64{0.3, 0.7, 1.9, 3.5} {
			vals, src := specialArgs(ft, x)
			check(t, in, name+"("+strings.Join(src, ", ")+")", fn.Call(vals)[0].Float())
		}
		// Elementwise: the first real parameter an Array, then an NDArray of
		// shape (3, 1) against a second real of shape (2,) when there is one.
		first := -1
		reals := 0
		for i := range ft.NumIn() {
			if ft.In(i).Kind() == reflect.Float64 {
				if first < 0 {
					first = i
				}
				reals++
			}
		}
		xs := []float64{0.3, 0.7, 1.9}
		want := make([]float64, len(xs))
		srcList := make([]string, ft.NumIn())
		for k, x := range xs {
			vals, src := specialArgs(ft, 0.6)
			vals[first] = reflect.ValueOf(x)
			want[k] = fn.Call(vals)[0].Float()
			copy(srcList, src)
		}
		srcList[first] = "[0.3, 0.7, 1.9]"
		check(t, in, name+"("+strings.Join(srcList, ", ")+")", want)
		srcList[first] = "NDArray.from([0.3, 0.7, 1.9], [3, 1])"
		if reals == 1 {
			check(t, in, name+"("+strings.Join(srcList, ", ")+")", vecShape(want, 3, 1))
			continue
		}
		second := -1
		for i := first + 1; i < ft.NumIn(); i++ {
			if ft.In(i).Kind() == reflect.Float64 {
				second = i
				break
			}
		}
		ys := []float64{0.45, 1.2}
		grid := make([]float64, 0, 6)
		for _, x := range xs {
			for _, y := range ys {
				vals, _ := specialArgs(ft, 0.6)
				vals[first], vals[second] = reflect.ValueOf(x), reflect.ValueOf(y)
				grid = append(grid, fn.Call(vals)[0].Float())
			}
		}
		srcList[second] = "NDArray.from([0.45, 1.2], [2])"
		check(t, in, name+"("+strings.Join(srcList, ", ")+")", vecShape(grid, 3, 2))
	}
}

func TestSpecialErrors(t *testing.T) {
	in := newInterp(t)
	evalErr(t, in, "Special.gamma", "ArgumentError", "wrong number of arguments (given 0, expected 1)")
	evalErr(t, in, "Special.gamma(1, x: 2)", "ArgumentError", "unknown keyword: :x")
	evalErr(t, in, "Special.gamma('a')", "TypeError", "no implicit conversion of String into Float")
	evalErr(t, in, "Special.gamma(['a'])", "TypeError", "no implicit conversion of String into Float")
	evalErr(t, in, "Special.beta(NDArray.zeros(2), NDArray.zeros(3))", "ArgumentError", "broadcast")
	evalErr(t, in, "Special.softmax", "ArgumentError", "wrong number of arguments")
	evalErr(t, in, "Special.softmax('a')", "TypeError", "Array of Floats")
	evalErr(t, in, "Special.jn(1.5, 1.0)", "TypeError", "into Integer")
}
