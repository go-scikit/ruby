package cells

import (
	"bytes"
	"errors"
	"reflect"
	"slices"
	"strings"
	"testing"

	rb "github.com/go-embedded-ruby/ruby"
)

func eval1(t *testing.T, s *Session, src string, line int) (string, map[string][]byte) {
	t.Helper()
	insp, disp, err := s.Eval(src, "(irb)", line)
	if err != nil {
		t.Fatalf("%q: %v", src, err)
	}
	return insp, disp
}

func TestEvalLikeIRB(t *testing.T) {
	var out bytes.Buffer
	s := NewWriting(&out, &out)
	if insp, disp := eval1(t, s, "x = [1, 2]", 1); insp != "[1, 2]" || disp != nil {
		t.Fatalf("%q %v", insp, disp)
	}
	// Reassignment is allowed; `_` is the last value.
	if insp, _ := eval1(t, s, "x = x.sum", 2); insp != "3" {
		t.Fatal(insp)
	}
	if insp, _ := eval1(t, s, "_ * 10", 3); insp != "30" {
		t.Fatal(insp)
	}
	if insp, _ := eval1(t, s, "_", 4); insp != "30" {
		t.Fatal(insp)
	}
	if insp, _ := eval1(t, s, "puts 'hi'", 5); insp != "nil" || out.String() != "hi\n" {
		t.Fatalf("%q %q", insp, out.String())
	}
	// Multi-line input, a method across inputs.
	eval1(t, s, "def twice(v)\n  v * 2\nend", 6)
	if insp, _ := eval1(t, s, "twice(21)", 9); insp != "42" {
		t.Fatal(insp)
	}
	// An NDArray shows its inspect only; a figure its own display.
	if insp, disp := eval1(t, s, "NDArray.arange(3)", 10); disp != nil || !strings.Contains(insp, "array(") {
		t.Fatalf("%q %v", insp, disp)
	}
	insp, disp := eval1(t, s, "fig, ax = Plot.subplots\nax.plot([1, 2], [3, 4])\nfig", 11)
	if !strings.Contains(string(disp["image/svg+xml"]), "<svg") || !strings.Contains(insp, "Figure") {
		t.Fatalf("%q %v", insp, disp)
	}
}

func TestEvalErrors(t *testing.T) {
	s := New()
	eval1(t, s, "v = 1", 1)
	_, _, err := s.Eval("def f = raise(ArgumentError, 'bad')\nf", "(irb)", 2)
	var e *Error
	if !errors.As(err, &e) {
		t.Fatalf("%T %v", err, err)
	}
	if want := "bad (ArgumentError)\n\tfrom (irb):2:in 'Object#f'\n\tfrom (irb):3:in '<main>'"; e.Error() != want || e.Class() != "ArgumentError" {
		t.Fatalf("%q (%s)", e.Error(), e.Class())
	}
	// `_` keeps the last value, not the failed input's.
	if insp, _ := eval1(t, s, "_", 4); insp != "1" {
		t.Fatal(insp)
	}
	if _, _, err := s.Eval("exit 3", "(irb)", 5); !errors.As(err, &e) || e.Class() != "SystemExit" {
		t.Fatalf("exit: %v", err)
	}
	if _, _, err := s.Eval("1 +", "(irb)", 6); !errors.As(err, &e) || e.Class() != "SyntaxError" {
		t.Fatalf("syntax: %v", err)
	}
	// A deep recursion: twelve frames, then how many more.
	_, _, err = s.Eval("def r(n) = r(n + 1)\nr(0)", "(irb)", 7)
	if !errors.As(err, &e) || e.Class() != "SystemStackError" || strings.Count(e.Error(), "\tfrom") != maxFrames || !strings.Contains(e.Error(), "more frames") {
		t.Fatalf("recursion: %v", err)
	}
	// An error that is no Ruby exception.
	if got := interactiveError(errors.New("plain"), "(irb)"); got.Error() != "plain" || got.Class() != "" {
		t.Fatal(got)
	}
	if got := interactiveError(&rb.Error{Class: "E", Message: "m", Backtrace: []string{"lib.rb:1", "(irb):3:in 'x'"}}, "(irb)"); got.Error() != "m (E)\n\tfrom (irb):3:in 'x'" {
		t.Fatal(got)
	}
}

func TestComplete(t *testing.T) {
	s := New()
	eval1(t, s, "xs = [3, 1, 2]\nname = 'ab'\nh = {a: 1}\n@iv = 5\n$gv = 1\nclass Shape; SIDES = 3; def self.make = new; end", 1)
	has := func(target string, want ...string) {
		t.Helper()
		got := s.Complete(target)
		for _, w := range want {
			if !slices.Contains(got, w) {
				t.Errorf("Complete(%q) = %q, missing %q", target, got, w)
			}
		}
	}
	none := func(target string) {
		t.Helper()
		if got := s.Complete(target); len(got) != 0 {
			t.Errorf("Complete(%q) = %q, want none", target, got)
		}
	}
	// Names: locals, methods of main, constants (the bindings), keywords.
	has("x", "xs")
	has("na", "name")
	has("pu", "puts")
	has("St", "Stats", "String")
	has("Sh", "Shape")
	has("whi", "while")
	has("@i", "@iv")
	if got := s.Complete("x"); !slices.IsSorted(got) {
		t.Errorf("not sorted: %q", got)
	}
	// After a dot: the methods of a variable's value.
	has("xs.so", "xs.sort", "xs.sort_by")
	has("name.up", "name.upcase")
	has("h.ke", "h.keys")
	has("@iv.ti", "@iv.times")
	has("nil.to", "nil.to_a")
	// Literals.
	has("'s'.up", "'s'.upcase")
	has("/r/.ma", "/r/.match")
	has("[1].ma", "[1].map")
	has("{}.ke", "{}.keys")
	has(":sym.le", ":sym.length")
	has("1.ti", "1.times")
	has("1.5.fl", "1.5.floor")
	has("0xff.ti", "0xff.times")
	has(".up", ".upcase")
	// Constants and their methods, after :: and after a dot.
	has("Stats::Norm", "Stats::Normal")
	has("Shape::SI", "Shape::SIDES")
	has("Shape::ma", "Shape::make")
	has("Shape.ma", "Shape.make")
	has("Stats::Normal.fi", "Stats::Normal.fit")
	has("::Sta", "::Stats")
	// :: after a variable: a module's constants and methods; another value's methods.
	eval1(t, s, "m = Stats", 10)
	has("m::Norm", "m::Normal")
	has("xs::fi", "xs::first")
	has("$g", "$gv")
	// Nothing: a blank word, a symbol, an unknown receiver, a receiver that is
	// an expression (Tab must not call it), an operator.
	none("")
	none("  ")
	none(":sy")
	none("nope.si")
	none("xs.first.ti")
	none("Kernel.exit(1)::X")
	none("Shape.make.ze")
	if got := s.Complete("xs.+"); len(got) != 0 {
		t.Errorf("operator: %q", got)
	}
	// Completing ran nothing and defined nothing.
	if got := sortedCopy(s.in.Locals()); !reflect.DeepEqual(got, []string{"_", "h", "m", "name", "xs"}) {
		t.Errorf("locals now %q", got)
	}
}

func sortedCopy(l []string) []string {
	c := slices.Clone(l)
	slices.Sort(c)
	return c
}

func TestCompleteHelpers(t *testing.T) {
	s := New()
	// An expression that does not give names gives none.
	if got := s.names("42"); len(got) != 0 {
		t.Errorf("%q", got)
	}
	if got := s.names("[:a, 1, 'b']"); !reflect.DeepEqual(got, []string{"a", "1", "b"}) {
		t.Errorf("%q", got)
	}
	if got := s.names("("); got != nil {
		t.Errorf("syntax error: %q", got)
	}
	if got := selectMessage("r", "", []string{"", "abc", "+", "9x", "Up"}, "."); !reflect.DeepEqual(got, []string{"r.abc", "r.Up"}) {
		t.Errorf("%q", got)
	}
	if got := union([]string{"a", "b"}, []string{"b", "c"}); !reflect.DeepEqual(got, []string{"a", "b", "c"}) {
		t.Errorf("%q", got)
	}
}
