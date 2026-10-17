package ruby

import (
	"reflect"

	rb "github.com/go-embedded-ruby/ruby"
	nd "github.com/go-ndarray/ndarray"
)

// dataFunc binds a generic Go function over stats.Data — []float64 or
// *ndarray.Array — given its two instantiations, overSlice (D = []float64)
// and overArray (D = *ndarray.Array). The parameters where the two differ are
// the data: Ruby passes an Array or an NDArray there. When every data argument
// is an NDArray the NDArray instantiation runs on them unchanged; otherwise
// the slice one runs, with each NDArray read row-major. A variadic data
// parameter takes the remaining positional arguments, one sample each. The
// other parameters convert as for any Go function (see invoke).
func dataFunc(overSlice, overArray any) rb.Func {
	fs, fa := reflect.ValueOf(overSlice), reflect.ValueOf(overArray)
	ts := fs.Type()
	isData := make([]bool, ts.NumIn())
	for i := range isData {
		isData[i] = ts.In(i) != fa.Type().In(i)
	}
	return func(c *rb.Call) (any, error) {
		allArrays := true
		for i, a := range c.Args {
			if dataArg(isData, ts, i) {
				if _, ok := a.Go().(*nd.Array); !ok {
					allArrays = false
				}
			}
		}
		fn := fs
		if allArrays {
			fn = fa
		}
		return invoke(fn, c, c.Args)
	}
}

// dataArg reports whether positional argument i lands on a data parameter.
func dataArg(isData []bool, ft reflect.Type, i int) bool {
	p := 0
	for j := range isData {
		if filled(ft.In(j)) {
			continue
		}
		if p == i || (ft.IsVariadic() && j == ft.NumIn()-1) {
			return isData[j]
		}
		p++
	}
	return false
}
