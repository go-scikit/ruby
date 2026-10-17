package ruby

import (
	"reflect"

	rb "github.com/go-embedded-ruby/ruby"
	nd "github.com/go-ndarray/ndarray"
	"github.com/go-numerics/special"
)

// registerSpecial defines Special, every function of go-numerics/special
// under its snake_case name (Special.gamma, Special.beta_inc, Special.jv, …).
//
// A function of real arguments returning a real is a ufunc, as in SciPy: any
// real argument may be an NDArray (or an Array), the arrays broadcast against
// each other and the scalars, and the result has their shape — an NDArray, or
// an Array when every array argument was an Array. Integer parameters (an
// order n) take an Integer. The few functions of another shape are bound as
// they are: log_gamma_sign returns [value, sign], jn_zeros an NDArray, and
// softmax, log_softmax and log_sum_exp take a vector.
func registerSpecial(in *rb.Interp) error {
	d := &definer{in: in}
	m := d.module("Special")
	for _, f := range specialFuncs {
		d.fn(m, snakeName(f.name), specialFunc(f.fn))
	}
	return d.err
}

// specialFunc binds one function of package special.
func specialFunc(f any) rb.Func {
	fn := reflect.ValueOf(f)
	ft := fn.Type()
	switch {
	case isUfunc(ft):
		return func(c *rb.Call) (any, error) { return applyUfunc(fn, c) }
	case ft == reflect.TypeOf(special.Softmax):
		return func(c *rb.Call) (any, error) {
			if err := arity(c, 1, 1); err != nil {
				return nil, err
			}
			x, err := toFloats(c.Args[0].Go())
			if err != nil {
				return nil, err
			}
			return vec(f.(func(dst, x []float64) []float64)(nil, x)), nil
		}
	}
	return func(c *rb.Call) (any, error) { return invoke(fn, c, c.Args) }
}

// isUfunc reports whether a function maps reals (and integers) to one real,
// with at least one real parameter to broadcast over.
func isUfunc(ft reflect.Type) bool {
	if ft.NumOut() != 1 || ft.Out(0).Kind() != reflect.Float64 {
		return false
	}
	reals := 0
	for i := range ft.NumIn() {
		switch ft.In(i).Kind() {
		case reflect.Float64:
			reals++
		case reflect.Int, reflect.Int64:
		default:
			return false
		}
	}
	return reals > 0
}

// applyUfunc calls a ufunc elementwise over its array arguments.
func applyUfunc(fn reflect.Value, c *rb.Call) (any, error) {
	ft := fn.Type()
	if err := arity(c, ft.NumIn(), ft.NumIn()); err != nil {
		return nil, err
	}
	if len(c.Kwargs) > 0 {
		return nil, argError("unknown keyword: :%s", sortedKeys(c.Kwargs)[0])
	}
	scalars := make([]reflect.Value, ft.NumIn())
	var arrays []*nd.Array
	at := make([]int, ft.NumIn()) // index into arrays, or -1 for a scalar
	allLists := true
	for i, a := range c.Args {
		at[i] = -1
		x := a.Go()
		if ft.In(i).Kind() == reflect.Float64 {
			switch v := x.(type) {
			case *nd.Array:
				at[i] = len(arrays)
				arrays = append(arrays, v)
				allLists = false
				continue
			case []any:
				a, err := ndArray(v)
				if err != nil {
					return nil, err
				}
				at[i] = len(arrays)
				arrays = append(arrays, a)
				continue
			}
		}
		v, err := convert(x, ft.In(i))
		if err != nil {
			return nil, err
		}
		scalars[i] = v
	}
	if len(arrays) == 0 {
		return fn.Call(scalars)[0].Float(), nil
	}
	flat, shape, err := broadcast(arrays)
	if err != nil {
		return nil, err
	}
	n := len(flat[0])
	res := make([]float64, n)
	args := append([]reflect.Value(nil), scalars...)
	for k := range n {
		for i, j := range at {
			if j >= 0 {
				args[i] = reflect.ValueOf(flat[j][k])
			}
		}
		res[k] = fn.Call(args)[0].Float()
	}
	if allLists {
		list := make([]any, n)
		for k, r := range res {
			list[k] = r
		}
		return list, nil
	}
	a, _ := nd.FromData(res, shape...)
	return a, nil
}

// broadcast brings the arrays to their common shape, NumPy's rules, and
// returns their elements row-major. Multiplying by ones broadcasts exactly:
// x·1 is x for every float64, NaN and the infinities included.
func broadcast(arrays []*nd.Array) ([][]float64, []int, error) {
	common := arrays[0]
	for _, a := range arrays[1:] {
		ones, _ := nd.Ones(a.Shape()...)
		b, err := common.Mul(ones)
		if err != nil {
			return nil, nil, goError(err)
		}
		common = b
	}
	shape := common.Shape()
	ones, _ := nd.Ones(shape...)
	out := make([][]float64, len(arrays))
	for i, a := range arrays {
		b, _ := a.Mul(ones) // a broadcasts to the common shape: it was part of it
		out[i] = arrayFloats(b)
	}
	return out, shape, nil
}
