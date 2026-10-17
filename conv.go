package ruby

import (
	"fmt"
	"math/big"
	"reflect"
	"sort"
	"strings"
	"unicode"

	rb "github.com/go-embedded-ruby/ruby"
	nd "github.com/go-ndarray/ndarray"
)

// This file converts between Ruby values, as rbgo hands them to Go, and the
// Go types of the bound APIs. One rule set serves positional arguments,
// keyword arguments (which fill the fields of a Go option struct) and results.

func argError(format string, args ...any) error {
	return rb.NewError("ArgumentError", fmt.Sprintf(format, args...))
}

func typeError(x any, into string) error {
	return rb.NewError("TypeError", fmt.Sprintf("no implicit conversion of %s into %s", describe(x), into))
}

// describe names a converted Ruby value (see rb.Value.Go) the way MRI's
// conversion errors do: nil, true and false by value, everything else by
// class.
func describe(x any) string {
	switch v := x.(type) {
	case nil:
		return "nil"
	case bool:
		return fmt.Sprint(v)
	case int64, *big.Int:
		return "Integer"
	case float64:
		return "Float"
	case complex128:
		return "Complex"
	case *big.Rat:
		return "Rational"
	case string:
		return "String"
	case rb.Symbol:
		return "Symbol"
	case []any:
		return "Array"
	case map[any]any:
		return "Hash"
	case *nd.Array:
		return "NDArray"
	case rb.Value:
		return v.ClassName()
	}
	return fmt.Sprintf("%T", x)
}

// goError is the Ruby exception for an error a Go library returned: an
// ArgumentError carrying its message, which is what SciPy raises (ValueError)
// for the same inputs.
func goError(err error) error { return rb.NewError("ArgumentError", err.Error()) }

// toFloat reads a real number: an Integer, a Float, a big Integer or a
// Rational.
func toFloat(v any) (float64, error) {
	switch x := v.(type) {
	case float64:
		return x, nil
	case int64:
		return float64(x), nil
	case *big.Int:
		f, _ := new(big.Float).SetInt(x).Float64()
		return f, nil
	case *big.Rat:
		f, _ := x.Float64()
		return f, nil
	}
	return 0, typeError(v, "Float")
}

// toInt reads an Integer that fits an int.
func toInt(v any) (int64, error) {
	if x, ok := v.(int64); ok {
		return x, nil
	}
	return 0, typeError(v, "Integer")
}

// toFloats reads a vector: an Array of real numbers, or an NDArray (any
// shape, read in row-major order).
func toFloats(v any) ([]float64, error) {
	switch x := v.(type) {
	case *nd.Array:
		return arrayFloats(x), nil
	case []any:
		out := make([]float64, len(x))
		for i, e := range x {
			f, err := toFloat(e)
			if err != nil {
				return nil, err
			}
			out[i] = f
		}
		return out, nil
	}
	return nil, typeError(v, "an Array of Floats")
}

// arrayFloats copies an array's elements, row-major, as float64.
func arrayFloats(a *nd.Array) []float64 {
	flat := a.Flatten()
	out := make([]float64, flat.Size())
	for i := range out {
		out[i] = flat.At(i)
	}
	return out
}

// data reads the observations of a statistic: an NDArray is passed to Go as
// it is, an Array becomes a []float64. It is the Go side's
// []float64 | *ndarray.Array.
func data(v any) (any, error) {
	if a, ok := v.(*nd.Array); ok {
		return a, nil
	}
	return toFloats(v)
}

// ndArray reads an NDArray, or an Array of numbers as a 1-d one.
func ndArray(v any) (*nd.Array, error) {
	d, err := data(v)
	if err != nil {
		return nil, err
	}
	if a, ok := d.(*nd.Array); ok {
		return a, nil
	}
	f := d.([]float64)
	a, _ := nd.FromData(f, len(f))
	return a, nil
}

// vec is a result vector as Ruby gets it: an NDArray.
func vec(x []float64) *nd.Array {
	if x == nil {
		return nil
	}
	a, _ := nd.FromData(append([]float64(nil), x...), len(x))
	return a
}

// converters convert to a Go type that the general rules do not reach, such
// as optimize.Bound; the files binding those types register them.
var converters = map[reflect.Type]func(any) (reflect.Value, error){}

// convert converts a Ruby value, as rb.Value.Go gives it, to the Go type t.
func convert(v any, t reflect.Type) (reflect.Value, error) {
	if c, ok := converters[t]; ok {
		return c(v)
	}
	if vals, ok := enumTable[t]; ok {
		return enumValueOf(v, t, vals)
	}
	switch t.Kind() {
	case reflect.Float64, reflect.Float32:
		f, err := toFloat(v)
		return reflect.ValueOf(f).Convert(t), err
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		i, err := toInt(v)
		return reflect.ValueOf(i).Convert(t), err
	case reflect.Bool:
		b, ok := v.(bool)
		if !ok {
			return reflect.Value{}, typeError(v, "true or false")
		}
		return reflect.ValueOf(b), nil
	case reflect.String:
		s, err := toString(v)
		return reflect.ValueOf(s).Convert(t), err
	case reflect.Slice:
		if t.Elem().Kind() == reflect.Float64 {
			f, err := toFloats(v)
			return reflect.ValueOf(f), err
		}
		return convertList(v, t)
	case reflect.Array:
		return convertList(v, t)
	case reflect.Map:
		return convertMap(v, t)
	case reflect.Pointer:
		if t == ndArrayType {
			a, err := ndArray(v)
			return reflect.ValueOf(a), err
		}
		if v == nil {
			return reflect.Zero(t), nil
		}
		if reflect.TypeOf(v) != t {
			// An optional value, such as Ddof *float64 or Range *[2]float64.
			ev, err := convert(v, t.Elem())
			if err != nil {
				return reflect.Value{}, err
			}
			p := reflect.New(t.Elem())
			p.Elem().Set(ev)
			return p, nil
		}
	case reflect.Struct:
		if h, ok := v.(map[any]any); ok {
			return convertStruct(h, t)
		}
	}
	if v != nil && reflect.TypeOf(v) == t {
		return reflect.ValueOf(v), nil // a Go value that went through Ruby
	}
	return reflect.Value{}, typeError(v, t.String())
}

var ndArrayType = reflect.TypeFor[*nd.Array]()

// convertStruct fills a struct from a Hash, as options fills an option
// struct from keywords: bandwidth: {rule: :silverman}.
func convertStruct(h map[any]any, t reflect.Type) (reflect.Value, error) {
	p := reflect.New(t)
	fields := fieldsByName(t)
	for k, e := range h {
		name, err := toString(k)
		if err != nil {
			return reflect.Value{}, err
		}
		f, ok := fields[normalize(name)]
		if !ok {
			return reflect.Value{}, argError("unknown key %s for %s (known: %s)", name, t.Name(), strings.Join(keywordNames(t), ", "))
		}
		ev, err := convert(e, f.Type)
		if err != nil {
			return reflect.Value{}, err
		}
		p.Elem().FieldByIndex(f.Index).Set(ev)
	}
	return p.Elem(), nil
}

// toString reads a String or a Symbol.
func toString(v any) (string, error) {
	switch s := v.(type) {
	case string:
		return s, nil
	case rb.Symbol:
		return string(s), nil
	}
	return "", typeError(v, "String")
}

// convertList converts an Array element by element into a slice or a
// fixed-size array.
func convertList(v any, t reflect.Type) (reflect.Value, error) {
	elems, ok := v.([]any)
	if !ok {
		return reflect.Value{}, typeError(v, "Array")
	}
	var out reflect.Value
	if t.Kind() == reflect.Array {
		if len(elems) != t.Len() {
			return reflect.Value{}, argError("expected an Array of %d elements, got %d", t.Len(), len(elems))
		}
		out = reflect.New(t).Elem()
	} else {
		out = reflect.MakeSlice(t, len(elems), len(elems))
	}
	for i, e := range elems {
		ev, err := convert(e, t.Elem())
		if err != nil {
			return reflect.Value{}, err
		}
		out.Index(i).Set(ev)
	}
	return out, nil
}

// convertMap converts a Hash whose keys are Strings or Symbols.
func convertMap(v any, t reflect.Type) (reflect.Value, error) {
	h, ok := v.(map[any]any)
	if !ok || t.Key().Kind() != reflect.String {
		return reflect.Value{}, typeError(v, "Hash")
	}
	out := reflect.MakeMapWithSize(t, len(h))
	for k, e := range h {
		ks, err := toString(k)
		if err != nil {
			return reflect.Value{}, err
		}
		ev, err := convert(e, t.Elem())
		if err != nil {
			return reflect.Value{}, err
		}
		out.SetMapIndex(reflect.ValueOf(ks).Convert(t.Key()), ev)
	}
	return out, nil
}

// enumValueOf reads a symbol (or a string) naming a value of an enumerated
// type: :two_sided, "two-sided" and :"L-BFGS-B" all match, case and
// punctuation aside, the symbol or the Go string value.
func enumValueOf(v any, t reflect.Type, vals []enumValue) (reflect.Value, error) {
	s, err := toString(v)
	if err != nil {
		return reflect.Value{}, typeError(v, "Symbol")
	}
	want := normalize(s)
	names := make([]string, len(vals))
	for i, e := range vals {
		rv := reflect.ValueOf(e.value)
		if normalize(e.name) == want || (rv.Kind() == reflect.String && normalize(rv.String()) == want) {
			return rv, nil
		}
		names[i] = ":" + e.name
	}
	return reflect.Value{}, argError("unknown %s %q (expected one of %s)", snakeName(t.Name()), s, strings.Join(names, ", "))
}

// normalize folds case and drops punctuation, so names compare by their
// letters: "two_sided", "TwoSided" and "two-sided" are one.
func normalize(s string) string {
	var b strings.Builder
	for _, r := range s {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			b.WriteRune(unicode.ToLower(r))
		}
	}
	return b.String()
}

// options fills the option struct dst points to from keyword arguments: the
// keyword omit_nan sets the field OmitNaN, and so on, each converted to the
// field's type. An unknown keyword is an ArgumentError naming the known ones.
func options(dst any, kw map[string]rb.Value) error {
	rv := reflect.ValueOf(dst).Elem()
	fields := fieldsByName(rv.Type())
	for _, k := range sortedKeys(kw) {
		f, ok := fields[normalize(k)]
		if !ok {
			return argError("unknown keyword: :%s (known: %s)", k, strings.Join(keywordNames(rv.Type()), ", "))
		}
		val, err := convert(kw[k].Go(), f.Type)
		if err != nil {
			return err
		}
		rv.FieldByIndex(f.Index).Set(val)
	}
	return nil
}

func sortedKeys(kw map[string]rb.Value) []string {
	keys := make([]string, 0, len(kw))
	for k := range kw {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

// fieldsByName indexes a struct's settable fields by normalized name.
func fieldsByName(t reflect.Type) map[string]reflect.StructField {
	out := map[string]reflect.StructField{}
	for _, f := range reflect.VisibleFields(t) {
		if f.IsExported() && !f.Anonymous && f.Type.Kind() != reflect.Func {
			out[normalize(f.Name)] = f
		}
	}
	return out
}

// keywordNames lists the keywords an option struct accepts.
func keywordNames(t reflect.Type) []string {
	var out []string
	for _, f := range reflect.VisibleFields(t) {
		if f.IsExported() && !f.Anonymous && f.Type.Kind() != reflect.Func {
			out = append(out, snakeName(f.Name))
		}
	}
	return out
}

// split takes the keywords a binding handles itself out of kw, leaving the
// rest for an option struct.
func split(kw map[string]rb.Value, names ...string) (mine, rest map[string]rb.Value) {
	mine, rest = map[string]rb.Value{}, map[string]rb.Value{}
	for k, v := range kw {
		if contains(names, k) {
			mine[k] = v
		} else {
			rest[k] = v
		}
	}
	return mine, rest
}

func contains(list []string, s string) bool {
	for _, e := range list {
		if e == s {
			return true
		}
	}
	return false
}

// snakeOverrides are names whose Ruby spelling is SciPy's rather than the
// mechanical one.
var snakeOverrides = map[string]string{
	"PValue": "pvalue", "ZStatistic": "zstatistic", "NObs": "nobs", "OmitNaN": "omit_nan",
	"LogPDF": "logpdf", "LogCDF": "logcdf", "LogSF": "logsf", "LogPMF": "logpmf",
	"DF": "df", "NaN": "nan", "IntegrateBox1D": "integrate_box_1d",
}

// snakeName is the Ruby name of a Go identifier: PValue is pvalue, NumIter
// num_iter, XArray x_array.
func snakeName(s string) string {
	if o, ok := snakeOverrides[s]; ok {
		return o
	}
	r := []rune(s)
	var b strings.Builder
	for i, c := range r {
		if i > 0 && unicode.IsUpper(c) && (unicode.IsLower(r[i-1]) || unicode.IsDigit(r[i-1]) ||
			(i+1 < len(r) && unicode.IsLower(r[i+1]) && unicode.IsUpper(r[i-1]))) {
			b.WriteByte('_')
		}
		b.WriteRune(unicode.ToLower(c))
	}
	return b.String()
}
