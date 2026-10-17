package ruby

import (
	"math"
	"math/big"
	"reflect"
	"strings"
	"testing"

	rb "github.com/go-embedded-ruby/ruby"
	nd "github.com/go-ndarray/ndarray"
	"github.com/go-statistics/stats"
)

func TestDescribe(t *testing.T) {
	in := rb.New()
	obj, _ := in.Eval("Object.new", "x.rb", 1)
	a, _ := nd.Zeros(1)
	for x, want := range map[any]string{
		nil: "nil", true: "true", int64(1): "Integer", 1.5: "Float", complex(1, 2): "Complex",
		"s": "String", rb.Symbol("s"): "Symbol", a: "NDArray", obj: "Object", struct{}{}: "struct {}",
	} {
		if got := describe(x); got != want {
			t.Errorf("describe(%#v) = %q, want %q", x, got, want)
		}
	}
	for x, want := range map[string]any{"Integer": big.NewInt(1), "Rational": big.NewRat(1, 2), "Array": []any{}, "Hash": map[any]any{}} {
		if got := describe(want); got != x {
			t.Errorf("describe(%#v) = %q", want, got)
		}
	}
}

func TestToFloatAndFriends(t *testing.T) {
	if f, err := toFloat(new(big.Int).Lsh(big.NewInt(1), 70)); err != nil || f != math.Ldexp(1, 70) {
		t.Fatal(f, err)
	}
	if f, err := toFloat(big.NewRat(1, 4)); err != nil || f != 0.25 {
		t.Fatal(f, err)
	}
	if _, err := toFloat("x"); err == nil {
		t.Fatal("toFloat accepted a String")
	}
	if _, err := toInt(1.5); err == nil {
		t.Fatal("toInt accepted a Float")
	}
	if _, err := toFloats([]any{1.0, "x"}); err == nil {
		t.Fatal("toFloats accepted a String element")
	}
	if _, err := toFloats(3.0); err == nil {
		t.Fatal("toFloats accepted a Float")
	}
	if _, err := ndArray("x"); err == nil {
		t.Fatal("ndArray accepted a String")
	}
	if vec(nil) != nil {
		t.Fatal("vec(nil) is not nil")
	}
	if _, err := toString(1.0); err == nil {
		t.Fatal("toString accepted a Float")
	}
	if s, _ := toString(rb.Symbol("a")); s != "a" {
		t.Fatal(s)
	}
}

type convTarget struct {
	F32   float32
	I8    int8
	B     bool
	S     string
	Names []string
	Pair  [2]float64
	M     map[string]float64
	Opt   *float64
	Lim   *stats.Limits
	A     *nd.Array
	Alt   stats.Alternative
	Inner stats.Limits
	Fn    func()
}

func TestConvert(t *testing.T) {
	typ := reflect.TypeFor[convTarget]()
	field := func(name string) reflect.Type { f, _ := typ.FieldByName(name); return f.Type }
	ok := []struct {
		field string
		in    any
		want  any
	}{
		{"F32", 1.5, float32(1.5)},
		{"I8", int64(-3), int8(-3)},
		{"B", true, true},
		{"S", rb.Symbol("x"), "x"},
		{"Names", []any{"a", rb.Symbol("b")}, []string{"a", "b"}},
		{"Pair", []any{1.0, int64(2)}, [2]float64{1, 2}},
		{"M", map[any]any{rb.Symbol("a"): 1.0}, map[string]float64{"a": 1}},
		{"Opt", 0.5, ptr(0.5)},
		{"Opt", nil, (*float64)(nil)},
		{"Lim", map[any]any{"lower": int64(1)}, &stats.Limits{Lower: 1}},
		{"Alt", "two-sided", stats.TwoSided},
		{"Alt", rb.Symbol("greater"), stats.Greater},
		{"Inner", stats.Limits{Upper: 2}, stats.Limits{Upper: 2}},
	}
	for _, c := range ok {
		v, err := convert(c.in, field(c.field))
		if err != nil {
			t.Fatalf("%s: %v", c.field, err)
		}
		if !reflect.DeepEqual(v.Interface(), c.want) {
			t.Errorf("%s: %#v, want %#v", c.field, v.Interface(), c.want)
		}
	}
	bad := []struct {
		field string
		in    any
		msg   string
	}{
		{"B", int64(1), "true or false"},
		{"Names", "a", "into Array"},
		{"Names", []any{1.0}, "into String"},
		{"Pair", []any{1.0}, "expected an Array of 2 elements, got 1"},
		{"M", []any{}, "into Hash"},
		{"M", map[any]any{1.0: 1.0}, "into String"},
		{"M", map[any]any{"a": "b"}, "into Float"},
		{"Opt", "x", "into Float"},
		{"Lim", map[any]any{1.0: 1.0}, "into String"},
		{"Lim", map[any]any{"nope": 1.0}, "unknown key nope for Limits (known: lower, upper"},
		{"Lim", map[any]any{"lower": "x"}, "into Float"},
		{"Alt", 1.0, "into Symbol"},
		{"Alt", rb.Symbol("sideways"), `unknown alternative "sideways" (expected one of :two_sided, :less, :greater)`},
		{"Inner", 1.0, "into stats.Limits"},
		{"Fn", nil, "into func()"},
	}
	for _, c := range bad {
		_, err := convert(c.in, field(c.field))
		if err == nil || !strings.Contains(err.Error(), c.msg) {
			t.Errorf("%s from %#v: error %v, want one containing %q", c.field, c.in, err, c.msg)
		}
	}
	// convertMap needs String keys.
	if _, err := convertMap(map[any]any{}, reflect.TypeFor[map[int]float64]()); err == nil {
		t.Fatal("convertMap accepted an int key type")
	}
}

func TestOptions(t *testing.T) {
	in := rb.New()
	v := func(src string) rb.Value { x, _ := in.Eval(src, "x.rb", 1); return x }
	var o stats.MeanOptions
	if err := options(&o, map[string]rb.Value{"ddof": v("1"), "omit_nan": v("true")}); err != nil || o.Ddof != 1 || !o.OmitNaN {
		t.Fatal(o, err)
	}
	err := options(&o, map[string]rb.Value{"nope": v("1")})
	if err == nil || !strings.Contains(err.Error(), "unknown keyword: :nope (known: weights, ddof, omit_nan)") {
		t.Fatal(err)
	}
	if err := options(&o, map[string]rb.Value{"ddof": v("'x'")}); err == nil {
		t.Fatal("options accepted a String ddof")
	}
}

func TestSnakeName(t *testing.T) {
	for in, want := range map[string]string{
		"PValue": "pvalue", "TwoSided": "two_sided", "XArray": "x_array", "BetaIncCInv": "beta_inc_c_inv",
		"I0e": "i0e", "Log1pExp": "log1p_exp", "LogPDF": "logpdf", "HessInvArray": "hess_inv_array", "Xlog1py": "xlog1py",
	} {
		if got := snakeName(in); got != want {
			t.Errorf("snakeName(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestPlainAndConvertible(t *testing.T) {
	for typ, want := range map[reflect.Type]bool{
		reflect.TypeFor[float64]():             true,
		reflect.TypeFor[[]float64]():           true,
		reflect.TypeFor[map[string][]int]():    true,
		reflect.TypeFor[map[string]func()]():   false,
		reflect.TypeFor[func()]():              false,
		reflect.TypeFor[chan int]():            false,
		reflect.TypeFor[error]():               false,
		reflect.TypeFor[*stats.Normal]():       true,
		reflect.TypeFor[[]func(float64) int](): false,
	} {
		if got := plain(typ); got != want {
			t.Errorf("plain(%v) = %v", typ, got)
		}
	}
	for typ, want := range map[reflect.Type]bool{
		reflect.TypeFor[func(float64) (float64, error)]():      true,
		reflect.TypeFor[func(func(float64) float64) float64](): true,
		reflect.TypeFor[func(func() int) float64]():            false,
		reflect.TypeFor[func() func()]():                       false,
	} {
		if got := convertible(typ, 0); got != want {
			t.Errorf("convertible(%v) = %v", typ, got)
		}
	}
}
