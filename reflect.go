package ruby

import (
	"errors"
	"fmt"
	"reflect"
	"strings"

	rb "github.com/go-embedded-ruby/ruby"
	nd "github.com/go-ndarray/ndarray"
	"github.com/go-ndarray/ndarray/random"
)

// This file binds Go types by reflection: a result struct becomes a Ruby class
// with a reader per field, and the methods of a Go type whose parameters and
// results convert are bound as they are, under their snake_case names.

// definer defines modules and classes, keeping the first error so a register
// function can define everything and check once.
type definer struct {
	in  *rb.Interp
	err error
}

func (d *definer) module(name string) *rb.Module {
	m, err := d.in.DefineModule(name)
	d.keep(err)
	return m
}

// class defines a class inside parent; nil (and the error kept) when the
// name is taken by something else.
func (d *definer) class(parent *rb.Module, name string) *rb.Module {
	c, err := parent.DefineClass(name, nil)
	d.keep(err)
	return c
}

func (d *definer) keep(err error) {
	if d.err == nil {
		d.err = err
	}
}

// fn defines a module function.
// Every binding is guarded: its Go algorithm may call back into Ruby.
func (d *definer) fn(m *rb.Module, name string, f rb.Func) { m.DefineSingletonMethod(name, guarded(f)) }

// structClass defines the class name inside parent for the Go struct type of
// sample (a value, not a pointer): values of the type and pointers to it
// become its instances, each exported field a reader, each convertible method
// a method, plus to_h and inspect.
func (d *definer) structClass(parent *rb.Module, name string, sample any) *rb.Module {
	c := d.class(parent, name)
	if c == nil {
		return nil
	}
	t := reflect.TypeOf(sample)
	c.Bind(sample)
	c.Bind(reflect.New(t).Interface())
	fields := readableFields(t)
	for _, f := range fields {
		f := f
		c.DefineMethod(snakeName(f.Name), func(call *rb.Call) (any, error) {
			if err := noArgs(call); err != nil {
				return nil, err
			}
			return out(structOf(call.Self).FieldByIndex(f.Index)), nil
		})
	}
	c.DefineMethod("to_h", func(call *rb.Call) (any, error) {
		v := structOf(call.Self)
		pairs := make([]any, len(fields))
		for i, f := range fields {
			pairs[i] = []any{rb.Symbol(snakeName(f.Name)), out(v.FieldByIndex(f.Index))}
		}
		list, _ := call.Interp.ValueOf(pairs)
		return list.Call("to_h")
	})
	c.DefineMethod("inspect", func(call *rb.Call) (any, error) {
		v := structOf(call.Self)
		parts := make([]string, len(fields))
		for i, f := range fields {
			fv, _ := call.Interp.ValueOf(out(v.FieldByIndex(f.Index)))
			parts[i] = snakeName(f.Name) + "=" + fv.Inspect()
		}
		return "#<" + c.Name() + " " + strings.Join(parts, ", ") + ">", nil
	})
	bindMethods(c, reflect.PointerTo(t), nil)
	return c
}

// structOf is the struct value behind self, a T or a *T.
func structOf(self rb.Value) reflect.Value {
	return reflect.Indirect(reflect.ValueOf(self.Go()))
}

// readableFields are the exported fields that convert to Ruby.
func readableFields(t reflect.Type) []reflect.StructField {
	var out []reflect.StructField
	for _, f := range reflect.VisibleFields(t) {
		if f.IsExported() && !f.Anonymous && f.Type.Kind() != reflect.Func {
			out = append(out, f)
		}
	}
	return out
}

func noArgs(c *rb.Call) error {
	if len(c.Args) > 0 || len(c.Kwargs) > 0 {
		return argError("wrong number of arguments (given %d, expected 0)", len(c.Args)+len(c.Kwargs))
	}
	return nil
}

// bindMethods binds every method of t (a pointer type, so the value methods
// are included) whose parameters and results convert, except those named in
// skip, under its snake_case name. It returns the Ruby names bound.
func bindMethods(c *rb.Module, t reflect.Type, skip map[string]bool) []string {
	var names []string
	for i := range t.NumMethod() {
		m := t.Method(i)
		if skip[m.Name] || !convertible(m.Type, 1) {
			continue
		}
		name := snakeName(m.Name)
		names = append(names, name)
		c.DefineMethod(name, guarded(func(call *rb.Call) (any, error) {
			recv := reflect.ValueOf(call.Self.Go())
			if recv.Kind() != reflect.Pointer {
				p := reflect.New(recv.Type())
				p.Elem().Set(recv)
				recv = p
			}
			return invoke(recv.MethodByName(m.Name), call, call.Args)
		}))
	}
	return names
}

// convertible reports whether a func type's parameters (from the first'th
// on) and results all convert: no channels or interfaces other than error,
// and no func other than a func(float64) float64 (which the block supplies).
func convertible(ft reflect.Type, first int) bool {
	for i := first; i < ft.NumIn(); i++ {
		if in := ft.In(i); !plain(in) && in != scalarFuncType {
			return false
		}
	}
	for i := range ft.NumOut() {
		if o := ft.Out(i); !plain(o) && o != errorType {
			return false
		}
	}
	return true
}

var (
	errorType      = reflect.TypeFor[error]()
	scalarFuncType = reflect.TypeFor[func(float64) float64]()
	generatorType  = reflect.TypeFor[*random.Generator]()
)

// plain reports whether values of t cross between Ruby and Go: numbers,
// strings, booleans, slices, arrays and maps of those, NDArrays, and structs
// (as opaque or bound objects).
func plain(t reflect.Type) bool {
	switch t.Kind() {
	case reflect.Func, reflect.Chan, reflect.Interface, reflect.UnsafePointer, reflect.Uintptr:
		return false
	case reflect.Slice, reflect.Array, reflect.Pointer:
		return plain(t.Elem())
	case reflect.Map:
		return plain(t.Key()) && plain(t.Elem())
	}
	return true
}

// isOptions reports whether a parameter is an option struct, *XxxOptions,
// which keyword arguments fill.
func isOptions(t reflect.Type) bool {
	return t.Kind() == reflect.Pointer && t.Elem().Kind() == reflect.Struct && strings.HasSuffix(t.Elem().Name(), "Options")
}

// filled reports whether a parameter is filled otherwise than by a positional
// argument: an option struct, the block, or the random generator.
func filled(t reflect.Type) bool { return isOptions(t) || t == scalarFuncType || t == generatorType }

// invoke calls a Go func for a Ruby call, its parameters filled in order:
//
//   - an option struct (*XxxOptions) from the keyword arguments, nil when
//     there are none;
//   - a func(float64) float64 by the block;
//   - a *random.Generator from the keyword random_state: an Integer seed
//     (numpy.random.default_rng(seed)) or a Stats::RNG; nil without one;
//   - every other parameter by the next positional argument, converted to its
//     type, and a variadic one by all the remaining arguments (a single Array
//     standing for the list too, so rvs(2, 3) and rvs([2, 3]) are one).
//
// It returns the results converted (see results).
func invoke(fn reflect.Value, c *rb.Call, args []rb.Value) (any, error) {
	ft := fn.Type()
	kw, rest := split(c.Kwargs, "random_state")
	in := make([]reflect.Value, 0, ft.NumIn())
	pos, kwUsed := 0, len(rest) == 0
	for i := range ft.NumIn() {
		t := ft.In(i)
		switch {
		case isOptions(t):
			p := reflect.Zero(t)
			if len(rest) > 0 {
				p = reflect.New(t.Elem())
				if err := options(p.Interface(), rest); err != nil {
					return nil, err
				}
				kwUsed = true
			}
			in = append(in, p)
		case t == scalarFuncType:
			b, err := needBlock(c)
			if err != nil {
				return nil, err
			}
			in = append(in, (&guard{}).makeFunc(scalarFuncType, blockCaller(b), false))
		case t == generatorType:
			g, err := generator(kw["random_state"].Go())
			if err != nil {
				return nil, err
			}
			delete(kw, "random_state")
			in = append(in, reflect.ValueOf(g))
		case ft.IsVariadic() && i == ft.NumIn()-1:
			tail := make([]any, len(args)-min(pos, len(args)))
			for j := range tail {
				tail[j] = args[pos+j].Go()
			}
			if len(tail) == 1 && t.Elem().Kind() == reflect.Int {
				if list, ok := tail[0].([]any); ok {
					tail = list
				}
			}
			for _, a := range tail {
				v, err := convert(a, t.Elem())
				if err != nil {
					return nil, err
				}
				in = append(in, v)
			}
			pos = len(args)
		default:
			if pos >= len(args) {
				return nil, wrongArgs(len(args), ft)
			}
			v, err := convert(args[pos].Go(), t)
			if err != nil {
				return nil, err
			}
			in = append(in, v)
			pos++
		}
	}
	if pos != len(args) {
		return nil, wrongArgs(len(args), ft)
	}
	if !kwUsed {
		return nil, argError("unknown keyword: :%s", sortedKeys(rest)[0])
	}
	if len(kw) > 0 {
		return nil, argError("unknown keyword: :random_state")
	}
	return results(fn.Call(in))
}

// wrongArgs is MRI's ArgumentError for a positional count that does not fit.
func wrongArgs(given int, ft reflect.Type) error {
	n := 0
	for i := range ft.NumIn() {
		if !filled(ft.In(i)) {
			n++
		}
	}
	if ft.IsVariadic() {
		return argError("wrong number of arguments (given %d, expected %d+)", given, n-1)
	}
	return argError("wrong number of arguments (given %d, expected %d)", given, n)
}

// generator reads random_state: nil (a generator seeded by the system), an
// Integer seed, or a Stats::RNG.
func generator(v any) (*random.Generator, error) {
	switch x := v.(type) {
	case nil:
		return nil, nil
	case int64:
		if x < 0 {
			return nil, argError("random_state must be a non-negative Integer")
		}
		return random.DefaultRNG(uint64(x)), nil
	case *random.Generator:
		return x, nil
	}
	return nil, typeError(v, "a seed or a Stats::RNG")
}


// results converts what a Go func returned: a final error, when not nil,
// raises; one other result is that value, several an Array.
func results(vals []reflect.Value) (any, error) {
	if n := len(vals); n > 0 && vals[n-1].Type() == errorType {
		if err, _ := vals[n-1].Interface().(error); err != nil {
			return nil, goError(err)
		}
		vals = vals[:n-1]
	}
	switch len(vals) {
	case 0:
		return nil, nil
	case 1:
		return out(vals[0]), nil
	}
	list := make([]any, len(vals))
	for i, v := range vals {
		list[i] = out(v)
	}
	return list, nil
}

// out converts one Go result for Ruby: a []float64 is an NDArray, a value of
// an enumerated type its symbol, a nil pointer or slice nil; the rest
// converts as rbgo converts it.
func out(v reflect.Value) any {
	if vals, ok := enumTable[v.Type()]; ok {
		for _, e := range vals {
			if reflect.ValueOf(e.value).Equal(v) {
				return rb.Symbol(e.name)
			}
		}
		return v.Interface()
	}
	switch v.Kind() {
	case reflect.Slice:
		if v.IsNil() {
			return nil
		}
		if f, ok := v.Interface().([]float64); ok {
			return vec(f)
		}
		list := make([]any, v.Len())
		for i := range list {
			list[i] = out(v.Index(i))
		}
		return list
	case reflect.Pointer:
		if v.IsNil() {
			return nil
		}
	}
	return v.Interface()
}

// ufunc applies f elementwise, NumPy-style: to a Float (or an Integer) it
// returns a Float, to an NDArray an NDArray of the same shape, to an Array an
// Array.
func ufunc(x rb.Value, f func(float64) float64) (any, error) {
	switch v := x.Go().(type) {
	case *nd.Array:
		return v.Map(f), nil
	case []any:
		res := make([]any, len(v))
		for i, e := range v {
			fl, err := toFloat(e)
			if err != nil {
				return nil, err
			}
			res[i] = f(fl)
		}
		return res, nil
	}
	fl, err := toFloat(x.Go())
	if err != nil {
		return nil, typeError(x.Go(), "Float or NDArray")
	}
	return f(fl), nil
}

// arity checks the number of positional arguments.
func arity(c *rb.Call, min, max int) error {
	if n := len(c.Args); n < min || n > max {
		want := fmt.Sprint(min)
		if max > min {
			want = fmt.Sprintf("%d..%d", min, max)
		}
		return argError("wrong number of arguments (given %d, expected %s)", n, want)
	}
	return nil
}

// errNoBlock is the error of a method that needs a block.
var errNoBlock = errors.New("no block given")

// needBlock returns the call's block, or the error for a method that needs one.
func needBlock(c *rb.Call) (*rb.Block, error) {
	if c.Block == nil {
		return nil, rb.NewError("ArgumentError", errNoBlock.Error()+" (yield)")
	}
	return c.Block, nil
}
