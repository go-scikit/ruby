// Package cells runs the Ruby cells of a notebook: one Ruby interpreter per
// session, with the go-scikit modules registered, whose top-level locals
// persist from cell to cell, and the exchange of variables with the Go cells
// of the same notebook.
//
// It is what a go-notebook worker links when a notebook has Ruby cells; the
// generated program calls Run once per Ruby cell. It knows nothing of the
// notebook runtime: the cell's display and notes reach it through funcs.
//
// # Variables across languages
//
// A Go variable a Ruby cell reads is set as a Ruby local of the same name
// before the cell runs, converted as rbgo converts Go values (see rb.Value): a
// []float64 becomes an Array of Floats, a map a Hash, an *ndarray.Array the
// NDArray itself, any other value an opaque object that comes back as itself.
// A Go variable whose name starts with an upper-case letter becomes a Ruby
// constant.
//
// A Ruby local a cell assigns is handed back to Go after the cell runs, into
// a Go variable of the type its value has (GoValue): nil, true/false, Integer
// (int64, or *big.Int when large), Float (float64), String (string), Symbol
// (ruby.Symbol), Complex (complex128), Rational (*big.Rat), NDArray
// (*ndarray.Array, the same array), an Array whose elements are all Floats,
// all Integers, all Strings or all booleans ([]float64, []int64, []string,
// []bool; otherwise []any), a Hash with String keys (map[string]any; with
// Symbol keys map[ruby.Symbol]any; otherwise map[any]any), an object of a
// go-scikit class (the Go value it wraps: *plot.Figure, stats.TTestResult, …),
// and any other Ruby object as a ruby.Value.
package cells

import (
	"fmt"
	"io"
	"math/big"
	"os"
	"reflect"
	"strings"
	"unicode"

	rb "github.com/go-embedded-ruby/ruby"
	nd "github.com/go-ndarray/ndarray"

	scikit "github.com/go-scikit/ruby"
)

// Session is the Ruby side of one notebook worker.
type Session struct {
	in  *rb.Interp
	obj rb.Value // Object, for constants
}

// stdout and stderr write to the process's streams as they are when written,
// so a worker that redirects them per cell captures what Ruby prints.
type stdout struct{}

func (stdout) Write(p []byte) (int, error) { return os.Stdout.Write(p) }

type stderr struct{}

func (stderr) Write(p []byte) (int, error) { return os.Stderr.Write(p) }

// New starts an interpreter with the go-scikit modules, writing to the
// process's standard output and error.
func New() *Session { return NewWriting(stdout{}, stderr{}) }

// NewWriting is New writing what Ruby prints to out, and its $stderr and
// warnings to errOut: a host that captures a cell's output itself.
func NewWriting(out, errOut io.Writer) *Session {
	in := rb.New(rb.WithStdout(out), rb.WithStderr(errOut))
	_ = scikit.Register(in) // a fresh interpreter has none of their names
	obj, _ := in.Eval("Object", "(cells)", 1)
	return &Session{in: in, obj: obj}
}

// Var is a notebook variable at the boundary: its name and a pointer to the
// Go variable that holds it.
type Var struct {
	Name string
	Ptr  any
	// IfAbsent sets the Ruby local only when the interpreter does not hold
	// it already: a Ruby cell's variable, which the interpreter keeps between
	// cells but which a new worker gets back from its Go holder.
	IfAbsent bool
}

// Cell is one Ruby cell to run.
type Cell struct {
	Source string
	File   string // the notebook
	Line   int    // where Source starts in it
	// Display shows the value of the cell's final expression.
	Display bool
	// In are the variables the cell reads, Out those it assigns.
	In, Out []Var
}

// Error is a Ruby cell's failure: the exception, at its line in the
// notebook, with the backtrace frames in the notebook.
type Error struct{ msg, class string }

func (e *Error) Error() string { return e.msg }

// CellError is the failure as a notebook shows it, without a Go stack.
func (e *Error) CellError() string { return e.msg }

// Run runs a Ruby cell. show displays the value of its final expression;
// note(name, type) records the Go type each variable it assigns now has, in
// GoType's syntax, so the notebook can declare it. It panics with an *Error
// when the cell raises, which is how a worker fails a cell.
func (s *Session) Run(c Cell, show func(...any), note func(name, typ string)) {
	for _, v := range c.In {
		if err := s.set(v); err != nil {
			panic(&Error{msg: fmt.Sprintf("%s:%d: %v", c.File, c.Line, err)})
		}
	}
	val, err := s.in.Eval(c.Source, c.File, c.Line)
	if err != nil {
		panic(describe(err, c.File))
	}
	for _, v := range c.Out {
		got, _ := s.in.Lookup(v.Name) // nil when the assignment did not run
		x := GoValue(got)
		note(v.Name, GoType(x))
		if err := assign(v.Ptr, x); err != nil {
			panic(&Error{msg: fmt.Sprintf("%s:%d: %s: %v", c.File, c.Line, v.Name, err)})
		}
	}
	if c.Display && !val.IsNil() {
		if d, ok := val.Go().(interface{ Display() map[string][]byte }); ok {
			if _, isArray := d.(*nd.Array); !isArray {
				show(d)
				return
			}
		}
		show(Inspect(val.Inspect()))
	}
}

// maxFrames bounds the backtrace an error shows: a runaway recursion has
// thousands of frames, all alike.
const maxFrames = 12

// describe renders a Ruby exception: file:line: message (Class), then the
// backtrace frames that are in the notebook.
func describe(err error, file string) *Error {
	e, ok := err.(*rb.Error)
	if !ok {
		return &Error{msg: err.Error()}
	}
	var b strings.Builder
	b.WriteString(e.Error())
	frames(&b, e.Backtrace, file)
	return &Error{msg: b.String(), class: e.Class}
}

// set hands a Go variable to Ruby.
func (s *Session) set(v Var) error {
	x := reflect.ValueOf(v.Ptr).Elem().Interface()
	if r := []rune(v.Name); len(r) > 0 && unicode.IsUpper(r[0]) {
		if def, _ := s.obj.Call("const_defined?", rb.Symbol(v.Name), false); def.Go() == true {
			if _, err := s.obj.Call("send", rb.Symbol("remove_const"), rb.Symbol(v.Name)); err != nil {
				return err
			}
		}
		_, err := s.obj.Call("const_set", rb.Symbol(v.Name), x)
		return err
	}
	if v.IfAbsent {
		if _, err := s.in.Lookup(v.Name); err == nil {
			return nil
		}
	}
	return s.in.Set(v.Name, x)
}

// Inspect is a Ruby value's inspect as a notebook shows it: plain text.
type Inspect string

// Display shows the text.
func (i Inspect) Display() map[string][]byte {
	return map[string][]byte{"text/plain": []byte(i)}
}

// GoValue is a Ruby value as a Go cell gets it; see the package
// documentation for the table.
func GoValue(v rb.Value) any {
	return narrow(v.Go())
}

func narrow(x any) any {
	switch v := x.(type) {
	case []any:
		return narrowSlice(v)
	case map[any]any:
		return narrowMap(v)
	}
	return x
}

func narrowSlice(v []any) any {
	if len(v) == 0 {
		return v
	}
	switch v[0].(type) {
	case float64:
		return collect[float64](v)
	case int64:
		return collect[int64](v)
	case string:
		return collect[string](v)
	case bool:
		return collect[bool](v)
	}
	return v
}

// collect is v as a []T when every element is a T, else v.
func collect[T any](v []any) any {
	out := make([]T, len(v))
	for i, e := range v {
		t, ok := e.(T)
		if !ok {
			return v
		}
		out[i] = t
	}
	return out
}

func narrowMap(m map[any]any) any {
	strs := make(map[string]any, len(m))
	syms := make(map[rb.Symbol]any, len(m))
	for k, e := range m {
		switch k := k.(type) {
		case string:
			strs[k] = e
		case rb.Symbol:
			syms[k] = e
		default:
			return m
		}
	}
	switch {
	case len(strs) == len(m):
		return strs
	case len(syms) == len(m):
		return syms
	}
	return m
}

// GoType names the Go type of a converted value for a generated program:
// built-in types as Go writes them, and a named type of another package as
// {import/path}.Name, which the generator turns into an import and a
// qualified name. A type it cannot name — nil, an unexported or generic type,
// a func — is any.
func GoType(x any) string {
	if x == nil {
		return "any"
	}
	if s := typeName(reflect.TypeOf(x)); s != "" {
		return s
	}
	return "any"
}

func typeName(t reflect.Type) string {
	if t == bigIntType || t == bigRatType {
		return "*{math/big}." + t.Elem().Name()
	}
	if t.Name() != "" {
		if t.PkgPath() == "" {
			return t.Name() // a predeclared type
		}
		if !token(t.Name()) {
			return "" // unexported, or a generic instance
		}
		return "{" + t.PkgPath() + "}." + t.Name()
	}
	switch t.Kind() {
	case reflect.Pointer:
		if e := typeName(t.Elem()); e != "" {
			return "*" + e
		}
	case reflect.Slice:
		if e := typeName(t.Elem()); e != "" {
			return "[]" + e
		}
	case reflect.Map:
		k, e := typeName(t.Key()), typeName(t.Elem())
		if k != "" && e != "" {
			return "map[" + k + "]" + e
		}
	case reflect.Interface:
		if t.NumMethod() == 0 {
			return "any"
		}
	}
	return ""
}

var (
	bigIntType = reflect.TypeFor[*big.Int]()
	bigRatType = reflect.TypeFor[*big.Rat]()
)

// token reports whether a type name is an exported identifier (a generic
// instance's name holds brackets).
func token(name string) bool {
	for i, r := range name {
		if i == 0 && !unicode.IsUpper(r) || !unicode.IsLetter(r) && !unicode.IsDigit(r) && r != '_' {
			return false
		}
	}
	return true
}

// assign stores a converted value in the Go variable ptr points to: as it is
// into an any, else when its type is the variable's.
func assign(ptr any, x any) error {
	dst := reflect.ValueOf(ptr).Elem()
	if x == nil {
		dst.SetZero()
		return nil
	}
	v := reflect.ValueOf(x)
	if !v.Type().AssignableTo(dst.Type()) {
		return fmt.Errorf("is now a %s in Go, and Go cells read it as a %s", GoType(x), dst.Type())
	}
	dst.Set(v)
	return nil
}
