package ruby

import (
	"reflect"

	rb "github.com/go-embedded-ruby/ruby"
)

// This file turns Ruby blocks and Procs into the Go callbacks the numerical
// libraries take: objectives, residual vectors, models, right-hand sides.

// caller calls a block or a Proc with Go arguments.
type caller func(args ...any) (rb.Value, error)

// blockCaller calls the block.
func blockCaller(b *rb.Block) caller { return b.Call }

// procCaller calls a Proc, a Method or anything else answering #call: a jac:
// or fprime: keyword, say.
func procCaller(v rb.Value) (caller, error) {
	ok, err := v.Call("respond_to?", rb.Symbol("call"))
	if err != nil || ok.Go() != true {
		return nil, typeError(v.Go(), "a Proc")
	}
	return func(args ...any) (rb.Value, error) { return v.Call("call", args...) }, nil
}

// guard builds the callbacks of one call of a binding.
type guard struct{}

// makeFunc builds a Go func of type t that calls Ruby. Its float64
// parameters reach Ruby as Floats and its []float64 parameters as NDArrays
// (copies: the Go side reuses its buffers). A func returning a float64 returns
// what Ruby returns; a func returning nothing writes what Ruby returns — an
// Array or an NDArray — into its last []float64 parameter. With splat, a
// trailing []float64 input is passed as separate Floats (curve_fit's
// |x, a, b|).
func (g *guard) makeFunc(t reflect.Type, call caller, splat bool) reflect.Value {
	outBuf := t.NumOut() == 0
	return reflect.MakeFunc(t, func(in []reflect.Value) []reflect.Value {
		inputs := in
		if outBuf {
			inputs = in[:len(in)-1]
		}
		var args []any
		for i, v := range inputs {
			switch x := v.Interface().(type) {
			case float64:
				args = append(args, x)
			case []float64:
				if splat && i == len(inputs)-1 {
					for _, e := range x {
						args = append(args, e)
					}
				} else {
					args = append(args, vec(x))
				}
			}
		}
		r, err := call(args...)
		abortOn(err)
		if outBuf {
			out := in[len(in)-1].Interface().([]float64)
			v, err := toFloats(r.Go())
			abortOn(err)
			if len(v) != len(out) {
				abortOn(argError("the block returned %d values, expected %d", len(v), len(out)))
			}
			copy(out, v)
			return nil
		}
		f, err := toFloat(r.Go())
		abortOn(err)
		return []reflect.Value{reflect.ValueOf(f)}
	})
}

// callbackAbort carries an error out of a callback, through the Go algorithm
// that called it, to the binding (see guarded).
type callbackAbort struct{ err error }

// abortOn stops the running Go algorithm when err is not nil: its callback
// cannot return an error, and answering NaN instead would let an adaptive
// algorithm run on for as long as it takes to give up — a triple quadrature
// whose block raises would call it a billion times.
func abortOn(err error) {
	if err != nil {
		panic(callbackAbort{err})
	}
}

// guarded wraps a binding whose Go algorithm calls back into Ruby: an error
// in a callback ends the algorithm and becomes the binding's error. Any other
// panic — a break or a throw in the block, unwinding to its Ruby frame — goes
// on.
func guarded(f rb.Func) rb.Func {
	return func(c *rb.Call) (res any, err error) {
		defer func() {
			if r := recover(); r != nil {
				a, ok := r.(callbackAbort)
				if !ok {
					panic(r)
				}
				res, err = nil, a.err
			}
		}()
		return f(c)
	}
}

// fn is makeFunc for a func type known at compile time.
func fn[F any](g *guard, call caller) F {
	return g.makeFunc(reflect.TypeFor[F](), call, false).Interface().(F)
}

// optionalProc reads a keyword that may hold a Proc, building the callback of
// type F from it; ok is false when the keyword is absent.
func optionalProc[F any](g *guard, kw map[string]rb.Value, name string) (f F, ok bool, err error) {
	v, present := kw[name]
	if !present {
		return f, false, nil
	}
	call, err := procCaller(v)
	if err != nil {
		return f, false, err
	}
	return fn[F](g, call), true, nil
}

// numberOrProc reads a limit of integration that may be a number or a Proc
// of the outer variables: dblquad's gfun and hfun, tplquad's qfun and rfun.
func numberOrProc[F any](g *guard, v rb.Value) (F, error) {
	if c, err := toFloat(v.Go()); err == nil {
		t := reflect.TypeFor[F]()
		return reflect.MakeFunc(t, func([]reflect.Value) []reflect.Value {
			return []reflect.Value{reflect.ValueOf(c)}
		}).Interface().(F), nil
	}
	call, err := procCaller(v)
	if err != nil {
		var zero F
		return zero, typeError(v.Go(), "a Float or a Proc")
	}
	return fn[F](g, call), nil
}
