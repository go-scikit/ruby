package ruby

import (
	"math"
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

// guard keeps the first error a callback met. A Go algorithm cannot be
// stopped from inside its callback, so after an error the callback answers
// NaN (or leaves its output alone) without calling Ruby again, the algorithm
// winds down, and the binding returns the error instead of its result.
type guard struct{ err error }

func (g *guard) fail(err error) {
	if g.err == nil {
		g.err = err
	}
}

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
		if outBuf {
			out := in[len(in)-1].Interface().([]float64)
			if g.err == nil {
				g.fill(out, call, args)
			}
			return nil
		}
		f := math.NaN()
		if g.err == nil {
			f = g.scalar(call, args)
		}
		return []reflect.Value{reflect.ValueOf(f)}
	})
}

func (g *guard) scalar(call caller, args []any) float64 {
	r, err := call(args...)
	if err != nil {
		g.fail(err)
		return math.NaN()
	}
	f, err := toFloat(r.Go())
	if err != nil {
		g.fail(err)
		return math.NaN()
	}
	return f
}

func (g *guard) fill(out []float64, call caller, args []any) {
	r, err := call(args...)
	if err != nil {
		g.fail(err)
		return
	}
	v, err := toFloats(r.Go())
	if err != nil {
		g.fail(err)
		return
	}
	if len(v) != len(out) {
		g.fail(argError("the block returned %d values, expected %d", len(v), len(out)))
		return
	}
	copy(out, v)
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
