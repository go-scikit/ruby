package cells

import (
	"errors"
	"math/big"
	"os"
	"reflect"
	"strings"
	"testing"

	rb "github.com/go-embedded-ruby/ruby"
	nd "github.com/go-ndarray/ndarray"
	"github.com/go-plot/plot"
	"github.com/go-statistics/stats"
)

type shown struct{ vals []any }

func (s *shown) show(vs ...any) { s.vals = append(s.vals, vs...) }

func run(t *testing.T, s *Session, c Cell) (*shown, map[string]string) {
	t.Helper()
	sh := &shown{}
	notes := map[string]string{}
	if c.File == "" {
		c.File, c.Line = "nb.md", 3
	}
	s.Run(c, sh.show, func(n, typ string) { notes[n] = typ })
	return sh, notes
}

func TestRunExchangesVariables(t *testing.T) {
	s := New()
	xs := []float64{1, 2, 3}
	n := 10
	arr, _ := nd.FromData([]float64{1, 2}, 2)
	var m float64
	var names []string
	var a *nd.Array
	var anyv any
	sh, notes := run(t, s, Cell{
		Source:  "m = xs.sum / xs.size\nnames = %w[a b]\na = arr * K\nh = {'k' => 1}\nm + 1",
		Display: true,
		In:      []Var{{Name: "xs", Ptr: &xs}, {Name: "arr", Ptr: &arr}, {Name: "K", Ptr: &n}},
		Out:     []Var{{Name: "m", Ptr: &m}, {Name: "names", Ptr: &names}, {Name: "a", Ptr: &a}, {Name: "h", Ptr: &anyv}},
	})
	if m != 2 || !reflect.DeepEqual(names, []string{"a", "b"}) || a.Repr() != "array([10., 20.])" {
		t.Fatalf("m %v names %v a %v", m, names, a.Repr())
	}
	if !reflect.DeepEqual(anyv, map[string]any{"k": int64(1)}) {
		t.Fatalf("h = %#v", anyv)
	}
	want := map[string]string{"m": "float64", "names": "[]string", "a": "*{github.com/go-ndarray/ndarray}.Array", "h": "map[string]any"}
	if !reflect.DeepEqual(notes, want) {
		t.Fatalf("notes = %v", notes)
	}
	if len(sh.vals) != 1 || sh.vals[0] != Inspect("3.0") {
		t.Fatalf("shown %#v", sh.vals)
	}
	// A constant set again replaces it, without a warning.
	n = 3
	sh, _ = run(t, s, Cell{Source: "K * 2", Display: true, In: []Var{{Name: "K", Ptr: &n}}})
	if sh.vals[0] != Inspect("6") {
		t.Fatalf("K = %v", sh.vals)
	}
	// IfAbsent keeps the interpreter's own value; a fresh session takes it.
	stale := 99.0
	sh, _ = run(t, s, Cell{Source: "m", Display: true, In: []Var{{Name: "m", Ptr: &stale, IfAbsent: true}}})
	if sh.vals[0] != Inspect("2.0") {
		t.Fatalf("m = %v", sh.vals)
	}
	sh, _ = run(t, New(), Cell{Source: "m", Display: true, In: []Var{{Name: "m", Ptr: &stale, IfAbsent: true}}})
	if sh.vals[0] != Inspect("99.0") {
		t.Fatalf("m = %v", sh.vals)
	}
	// What a cell prints goes to the process's streams.
	run(t, s, Cell{Source: "print \"\\n\"; $stderr.print \"\\n\""})
	// Nothing shown for nil, or without Display.
	if sh, _ := run(t, s, Cell{Source: "nil", Display: true}); len(sh.vals) != 0 {
		t.Fatal("nil was shown")
	}
	if sh, _ := run(t, s, Cell{Source: "1"}); len(sh.vals) != 0 {
		t.Fatal("shown without Display")
	}
	// A figure shows itself; an NDArray by its inspect.
	sh, _ = run(t, s, Cell{Source: "fig, ax = Plot.subplots\nax.plot([1, 2], [3, 4])\nfig", Display: true})
	if _, ok := sh.vals[0].(*plot.Figure); !ok {
		t.Fatalf("figure shown as %T", sh.vals[0])
	}
	sh, _ = run(t, s, Cell{Source: "NDArray.arange(2)", Display: true})
	if sh.vals[0] != Inspect("array([0., 1.])") {
		t.Fatalf("NDArray shown as %#v", sh.vals[0])
	}
	if got := Inspect("x").Display()["text/plain"]; string(got) != "x" {
		t.Fatal(string(got))
	}
	// A variable the cell did not assign (a branch not taken) is nil.
	f := 1.5
	run(t, s, Cell{Source: "if false\n  q = 1\nend", Out: []Var{{Name: "q", Ptr: &f}}})
	if f != 0 {
		t.Fatalf("q = %v", f)
	}
}

func TestRunErrors(t *testing.T) {
	s := New()
	catch := func(c Cell) string {
		t.Helper()
		var msg string
		func() {
			defer func() {
				r := recover()
				e, ok := r.(*Error)
				if !ok {
					t.Fatalf("panic %#v", r)
				}
				if e.Error() != e.CellError() {
					t.Fatal("Error and CellError differ")
				}
				msg = e.CellError()
			}()
			run(t, s, c)
		}()
		return msg
	}
	if msg := catch(Cell{Source: "def f = raise(ArgumentError, 'bad')\n\nf", File: "nb.md", Line: 10}); !strings.HasPrefix(msg, "nb.md:10: bad (ArgumentError)") || !strings.Contains(msg, "from nb.md:12:in '<main>'") {
		t.Fatalf("msg = %q", msg)
	}
	var f float64
	if msg := catch(Cell{Source: "y = 'text'", Out: []Var{{Name: "y", Ptr: &f}}}); !strings.Contains(msg, "y: is now a string in Go, and Go cells read it as a float64") {
		t.Fatalf("msg = %q", msg)
	}
	bad := "not a local"
	if msg := catch(Cell{Source: "1", In: []Var{{Name: "a-b", Ptr: &bad}}}); !strings.Contains(msg, "not a local variable name") {
		t.Fatalf("msg = %q", msg)
	}
	if got := describe(&rb.Error{Class: "E", Message: "m", Backtrace: []string{"lib.rb:1:in 'g'", "nb.md:2:in '<main>'"}}, "nb.md"); got != "E: m\n\tfrom nb.md:2:in '<main>'" {
		t.Fatalf("describe = %q", got)
	}
	if got := describe(errors.New("plain"), "nb.md"); got != "plain" {
		t.Fatal(got)
	}
}

func TestSetConstantErrors(t *testing.T) {
	s := New()
	// A constant that cannot be removed: a frozen Object refuses.
	if _, err := s.in.Eval("module Kernel; end; class Object; def self.remove_const(*) = raise('no'); private_class_method :remove_const; end", "x.rb", 1); err != nil {
		t.Fatal(err)
	}
	n := 1
	if err := s.set(Var{Name: "Zq", Ptr: &n}); err != nil {
		t.Fatal(err)
	}
	if err := s.set(Var{Name: "Zq", Ptr: &n}); err == nil {
		t.Fatal("remove_const raised, set did not fail")
	}
}

func TestGoValueAndType(t *testing.T) {
	in := rb.New()
	val := func(src string) any {
		v, err := in.Eval(src, "x.rb", 1)
		if err != nil {
			t.Fatal(err)
		}
		return GoValue(v)
	}
	big2 := new(big.Int).Lsh(big.NewInt(1), 80)
	obj, _ := in.Eval("Object.new", "x.rb", 1)
	for src, want := range map[string]struct {
		v   any
		typ string
	}{
		"nil":              {nil, "any"},
		"true":             {true, "bool"},
		"3":                {int64(3), "int64"},
		"2**80":            {big2, "*{math/big}.Int"},
		"1.5":              {1.5, "float64"},
		"'s'":              {"s", "string"},
		":s":               {rb.Symbol("s"), "{github.com/go-embedded-ruby/ruby}.Symbol"},
		"Complex(1, 2)":    {complex(1, 2), "complex128"},
		"Rational(1, 2)":   {big.NewRat(1, 2), "*{math/big}.Rat"},
		"[1.5, 2.5]":       {[]float64{1.5, 2.5}, "[]float64"},
		"[1, 2]":           {[]int64{1, 2}, "[]int64"},
		"%w[a b]":          {[]string{"a", "b"}, "[]string"},
		"[true, false]":    {[]bool{true, false}, "[]bool"},
		"[1, 'a']":         {[]any{int64(1), "a"}, "[]any"},
		"[1.5, 'a']":       {[]any{1.5, "a"}, "[]any"},
		"[]":               {[]any{}, "[]any"},
		"[nil]":            {[]any{nil}, "[]any"},
		"{'a' => 1}":       {map[string]any{"a": int64(1)}, "map[string]any"},
		"{a: 1}":           {map[rb.Symbol]any{"a": int64(1)}, "map[{github.com/go-embedded-ruby/ruby}.Symbol]any"},
		"{1 => 2}":         {map[any]any{int64(1): int64(2)}, "map[any]any"},
		"{'a' => 1, b: 2}": {map[any]any{"a": int64(1), rb.Symbol("b"): int64(2)}, "map[any]any"},
	} {
		got := val(src)
		if !reflect.DeepEqual(got, want.v) && !(want.v != nil && reflect.TypeOf(got) == reflect.TypeOf(want.v) && strings.Contains(src, "**")) {
			t.Errorf("%s: GoValue = %#v, want %#v", src, got, want.v)
		}
		if typ := GoType(got); typ != want.typ {
			t.Errorf("%s: GoType = %s, want %s", src, typ, want.typ)
		}
	}
	if typ := GoType(GoValue(obj)); typ != "{github.com/go-embedded-ruby/ruby}.Value" {
		t.Errorf("an Object: %s", typ)
	}
	type local struct{}
	for _, c := range []struct {
		x    any
		want string
	}{
		{stats.TTestResult{}, "{github.com/go-statistics/stats}.TTestResult"},
		{&plot.Figure{}, "*{github.com/go-plot/plot}.Figure"},
		{local{}, "any"},
		{func() {}, "any"},
		{map[string]func(){}, "any"},
		{map[local]int{}, "any"},
		{[]func(){}, "any"},
		{(*func())(nil), "any"},
		{[]error{}, "[]error"},
		{[]interface{ M() }{}, "any"},
		{os.Stdout, "*{os}.File"},
		{reflect.TypeFor[int](), "any"}, // its dynamic type is unexported
	} {
		if typ := GoType(c.x); typ != c.want {
			t.Errorf("GoType(%T) = %s, want %s", c.x, typ, c.want)
		}
	}
	if token("Gen[int]") || token("lower") || !token("Upper_1") {
		t.Fatal("token")
	}
	var anyv any = 1
	if err := assign(&anyv, nil); err != nil || anyv != nil {
		t.Fatal(anyv, err)
	}
}

func TestNewWriting(t *testing.T) {
	var out, errOut strings.Builder
	s := NewWriting(&out, &errOut)
	run(t, s, Cell{Source: "puts 1\nwarn 'w'"})
	if out.String() != "1\n" || errOut.String() != "w\n" {
		t.Fatalf("out %q err %q", out.String(), errOut.String())
	}
}

func TestDeepRecursionIsBounded(t *testing.T) {
	s := New()
	var msg string
	func() {
		defer func() { msg = recover().(*Error).CellError() }()
		run(t, s, Cell{Source: "def f(n) = f(n + 1)\nf(0)"})
	}()
	if !strings.Contains(msg, "stack level too deep (SystemStackError)") || !strings.Contains(msg, "more frames") || strings.Count(msg, "\n") > maxFrames+1 {
		t.Fatalf("msg = %q", msg)
	}
}
