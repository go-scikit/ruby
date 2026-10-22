package ruby

import (
	"reflect"
	"strings"
	"testing"

	rb "github.com/go-embedded-ruby/ruby"
	nd "github.com/go-ndarray/ndarray"
	"github.com/go-statistics/stats"
)

// errorCases are calls that must raise: wrong counts, wrong types, unknown
// keywords, errors of the Go library, and errors in the block.
var errorCases = []struct{ src, class, msg string }{
	// Arity and blocks.
	{"Optimize.minimize", "ArgumentError", "wrong number of arguments (given 0, expected 1)"},
	{"Optimize.minimize([1])", "ArgumentError", "no block given"},
	{"Optimize.brentq(1, 2, 3) { }", "ArgumentError", "given 3, expected 2"},
	{"Integrate.fixed_quad(0) { }", "ArgumentError", "given 1, expected 2..3"},
	{"Integrate.fixed_quad(0, 1)", "ArgumentError", "no block given"},
	// Argument types.
	{"Optimize.minimize('x') { }", "TypeError", "an Array of Floats"},
	{"Optimize.brentq('a', 2) { }", "TypeError", "into Float"},
	{"Optimize.newton('a') { }", "TypeError", "into Float"},
	{"Optimize.root('x') { }", "TypeError", "an Array of Floats"},
	{"Optimize.fsolve('x') { }", "TypeError", "an Array of Floats"},
	{"Optimize.least_squares('x') { }", "TypeError", "an Array of Floats"},
	{"Optimize.leastsq('x') { }", "TypeError", "an Array of Floats"},
	{"Optimize.curve_fit('x', [1]) { }", "TypeError", "an Array of Floats"},
	{"Optimize.curve_fit([1], 'y') { }", "TypeError", "an Array of Floats"},
	{"Optimize.curve_fit([1], [1], p0: 'p') { }", "TypeError", "an Array of Floats"},
	{"Optimize.linprog('c')", "TypeError", "an Array of Floats"},
	{"Optimize.linprog([1], a_ub: 'a')", "TypeError", "an Array of Floats"},
	{"Optimize.linprog([1], bounds: [[1]])", "TypeError", "a bound [lo, hi]"},
	{"Optimize.linprog([1], bounds: [['a', 1]])", "TypeError", "into Float"},
	{"Optimize.linprog([1], bounds: [[1, 'b']])", "TypeError", "into Float"},
	{"Optimize.brute('r') { }", "TypeError", "into Array"},
	{"Optimize.brute([[1]]) { }", "TypeError", "a range [lo, hi] or [lo, hi, step]"},
	{"Optimize.brute([[1, 'b']]) { }", "TypeError", "into Float"},
	{"Optimize.fminbound(0, 'a') { }", "TypeError", "into Float"},
	{"Optimize.fminbound(0, 1, xtol: 'a') { }", "TypeError", "into Float"},
	{"Optimize.fminbound(0, 1, maxfun: 1.5) { }", "TypeError", "into Integer"},
	{"Optimize.bracket(0, 'a') { }", "TypeError", "into Float"},
	{"Optimize.approx_fprime('x', 1) { }", "TypeError", "an Array of Floats"},
	{"Optimize.approx_fprime([1], 'e') { }", "TypeError", "into Float"},
	{"Optimize.root_scalar(method: 1) { }", "TypeError", "into String"},
	{"Optimize.least_squares([1], m: 'm') { }", "TypeError", "into Integer"},
	{"Integrate.quad('a', 1) { }", "TypeError", "into Float"},
	{"Integrate.dblquad('a', 1, 0, 1) { }", "TypeError", "into Float"},
	{"Integrate.dblquad(0, 1, 'g', 1) { }", "TypeError", "a Float or a Proc"},
	{"Integrate.dblquad(0, 1, 0, 'h') { }", "TypeError", "a Float or a Proc"},
	{"Integrate.tplquad('a', 1, 0, 1, 0, 1) { }", "TypeError", "into Float"},
	{"Integrate.tplquad(0, 1, 'g', 1, 0, 1) { }", "TypeError", "a Float or a Proc"},
	{"Integrate.tplquad(0, 1, 0, 1, 'q', 1) { }", "TypeError", "a Float or a Proc"},
	{"Integrate.nquad('r') { }", "TypeError", "into Array"},
	{"Integrate.nquad([[1]]) { }", "TypeError", "a range [lo, hi]"},
	{"Integrate.fixed_quad('a', 1) { }", "TypeError", "into Float"},
	{"Integrate.fixed_quad(0, 1, 1.5) { }", "TypeError", "into Integer"},
	{"Integrate.romberg('a', 1) { }", "TypeError", "into Float"},
	{"Integrate.solve_ivp('s', [1]) { }", "TypeError", "into Array"},
	{"Integrate.solve_ivp([0, 1], 'y') { }", "TypeError", "an Array of Floats"},
	{"Integrate.odeint('y', [1]) { }", "TypeError", "an Array of Floats"},
	{"Integrate.odeint([1], 't') { }", "TypeError", "an Array of Floats"},
	{"Integrate.trapezoid", "ArgumentError", "wrong number of arguments"},
	{"Integrate.trapezoid('y')", "TypeError", "an Array of Floats"},
	{"Integrate.trapezoid([1], x: 'x')", "TypeError", "an Array of Floats"},
	{"Integrate.trapezoid([1], dx: 'x')", "TypeError", "into Float"},
	{"Integrate.trapezoid([1], axis: 'x')", "TypeError", "into Integer"},
	{"Integrate.trapezoid([1], nope: 1)", "ArgumentError", "unknown keyword: :nope (known: x, dx, axis)"},
	{"Integrate.cumulative_trapezoid", "ArgumentError", "wrong number of arguments"},
	{"Integrate.cumulative_trapezoid([1], nope: 1)", "ArgumentError", "unknown keyword: :nope"},
	{"Integrate.cumulative_trapezoid([1], initial: 1)", "TypeError", "true or false"},
	{"Integrate.cumulative_trapezoid('y')", "TypeError", "an Array of Floats"},
	{"Integrate.romb", "ArgumentError", "wrong number of arguments"},
	{"Integrate.romb([1], nope: 1)", "ArgumentError", "unknown keyword"},
	{"Integrate.romb('y')", "TypeError", "an Array of Floats"},
	{"Integrate.gauss_legendre", "ArgumentError", "wrong number of arguments"},
	{"Integrate.gauss_legendre(1.5)", "TypeError", "into Integer"},
	// Keywords.
	{"Optimize.minimize([1], nope: 1) { |x| 0 }", "ArgumentError", "unknown keyword: :nope"},
	{"Optimize.minimize([1], method: :nope) { |x| 0 }", "ArgumentError", "unknown method"},
	{"Optimize.minimize([1], jac: 1) { |x| 0 }", "TypeError", "a Proc"},
	{"Optimize.minimize([1], jac: BasicObject.new) { |x| 0 }", "TypeError", "a Proc"},
	{"Optimize.minimize_scalar(method: :nope) { |x| 0 }", "ArgumentError", "unknown scalar_method"},
	{"Optimize.minimize_scalar(nope: 1) { |x| 0 }", "ArgumentError", "unknown keyword"},
	{"Optimize.minimize_scalar(1) { |x| 0 }", "ArgumentError", "given 1, expected 0"},
	{"Optimize.brentq(0, 1, nope: 1) { |x| 0 }", "ArgumentError", "unknown keyword"},
	{"Optimize.newton(1.0, fprime: 1) { |x| 0 }", "TypeError", "a Proc"},
	{"Optimize.newton(1.0, fprime2: 1) { |x| 0 }", "TypeError", "a Proc"},
	{"Optimize.newton(1.0, nope: 1) { |x| 0 }", "ArgumentError", "unknown keyword"},
	{"Optimize.root_scalar(fprime: 1) { |x| 0 }", "TypeError", "a Proc"},
	{"Optimize.root_scalar(nope: 1) { |x| 0 }", "ArgumentError", "unknown keyword"},
	{"Optimize.root([1], jac: 1) { |x| [0] }", "TypeError", "a Proc"},
	{"Optimize.root([1], nope: 1) { |x| [0] }", "ArgumentError", "unknown keyword"},
	{"Optimize.least_squares([1], jac: 1) { |x| [0] }", "TypeError", "a Proc"},
	{"Optimize.least_squares([1], method: :nope) { |x| [0] }", "ArgumentError", "unknown ls_method"},
	{"Optimize.least_squares([1], nope: 1) { |x| [0] }", "ArgumentError", "unknown keyword"},
	{"Optimize.leastsq([1], jac: 1) { |x| [0] }", "TypeError", "a Proc"},
	{"Optimize.leastsq([1], nope: 1) { |x| [0] }", "ArgumentError", "unknown keyword"},
	{"Optimize.curve_fit([1], [1], nope: 1) { |x, a| a }", "ArgumentError", "unknown keyword"},
	{"Optimize.linprog([1], nope: 1)", "ArgumentError", "unknown keyword"},
	{"Optimize.brute([[0, 1]], nope: 1) { |x| 0 }", "ArgumentError", "unknown keyword"},
	{"Optimize.fminbound(0, 1, nope: 1) { |x| 0 }", "ArgumentError", "unknown keyword: :nope (known: xtol, maxfun)"},
	{"Integrate.quad(0, 1, nope: 1) { |x| x }", "ArgumentError", "unknown keyword"},
	{"Integrate.dblquad(0, 1, 0, 1, nope: 1) { |y, x| x }", "ArgumentError", "unknown keyword"},
	{"Integrate.tplquad(0, 1, 0, 1, 0, 1, nope: 1) { |z, y, x| x }", "ArgumentError", "unknown keyword"},
	{"Integrate.nquad([[0, 1]], nope: 1) { |x| 0 }", "ArgumentError", "unknown keyword"},
	{"Integrate.romberg(0, 1, nope: 1) { |x| x }", "ArgumentError", "unknown keyword"},
	{"Integrate.solve_ivp([0, 1], [1], nope: 1) { |t, y| y }", "ArgumentError", "unknown keyword"},
	{"Integrate.odeint([1], [0, 1], nope: 1) { |t, y| y }", "ArgumentError", "unknown keyword"},
	// Errors of the Go library.
	{"Optimize.minimize([]) { |x| 0 }", "ArgumentError", "optimize"},
	{"Optimize.minimize_scalar(method: :bounded) { |x| 0 }", "ArgumentError", "optimize"},
	{"Optimize.brentq(0, 1) { |x| 1 }", "ArgumentError", "optimize"},
	{"Optimize.newton(1.0, max_iter: -1) { |x| x }", "ArgumentError", "optimize"},
	{"Optimize.root_scalar(method: 'nope') { |x| x }", "ArgumentError", "optimize"},
	{"Optimize.root([]) { |x| [] }", "ArgumentError", "optimize"},
	{"Optimize.least_squares([], m: 0) { |x| [] }", "ArgumentError", "optimize"},
	{"Optimize.leastsq([], m: 0) { |x| [] }", "ArgumentError", "optimize"},
	{"Optimize.curve_fit([1, 2], [1]) { |x, a| a }", "ArgumentError", "optimize"},
	{"Optimize.linprog([])", "ArgumentError", "optimize"},
	{"Optimize.brute([]) { |x| 0 }", "ArgumentError", "optimize"},
	{"Optimize.fminbound(1, 0) { |x| x }", "ArgumentError", "optimize"},
	{"Optimize.bracket(0, 1) { |x| -x }", "ArgumentError", "optimize"},
	{"Integrate.quad(0, 1, limit: -1) { |x| x }", "ArgumentError", "integrate"},
	{"Integrate.romberg(0.01, 1, div_max: 1) { |x| Math.sin(1 / x) }", "ArgumentError", "DivMax exceeded"},
	{"Integrate.solve_ivp([0, 1], []) { |t, y| y }", "TypeError", "nil"},
	{"Integrate.odeint([1], []) { |t, y| y }", "ArgumentError", "integrate"},
	{"Integrate.trapezoid([1, 2], x: [1])", "ArgumentError", "integrate"},
	{"Integrate.simpson(NDArray.zeros(2, 3), axis: 5)", "ArgumentError", "integrate"},
	{"Integrate.cumulative_trapezoid([1, 2], x: [1])", "ArgumentError", "integrate"},
	{"Integrate.cumulative_trapezoid(NDArray.zeros(2, 3), axis: 5)", "ArgumentError", "integrate"},
	{"Integrate.romb([1, 2, 3, 4])", "ArgumentError", "integrate"},
	{"Integrate.fixed_quad(0, 1, 0) { |x| x }", "ArgumentError", "integrate"},
	// Errors in the block, and what it returns.
	{"Optimize.minimize([1]) { |x| raise IOError, 'in block' }", "IOError", "in block"},
	{"Optimize.minimize([1]) { |x| 'not a number' }", "TypeError", "into Float"},
	{"Optimize.root([1]) { |x| raise IOError, 'in block' }", "IOError", "in block"},
	{"Optimize.root([1]) { |x| 'v' }", "TypeError", "an Array of Floats"},
	{"Optimize.root([1, 2]) { |x| [1] }", "ArgumentError", "the block returned 1 values, expected 2"},
	{"Optimize.least_squares([1]) { |x| raise IOError, 'first call' }", "IOError", "first call"},
	{"Optimize.least_squares([1]) { |x| 'v' }", "TypeError", "an Array of Floats"},
	{"Integrate.quad(0, 1) { |x| raise IOError, 'q' }", "IOError", "q"},
	{"Integrate.dblquad(0, 1, ->(x) { raise IOError, 'lim' }, 1) { |y, x| x }", "IOError", "lim"},
	{"Integrate.fixed_quad(0, 1) { |x| raise IOError, 'fq' }", "IOError", "fq"},
	{"Integrate.romberg(0, 1) { |x| raise IOError, 'r' }", "IOError", "r"},
	{"Integrate.solve_ivp([0, 1], [1]) { |t, y| raise IOError, 'ivp' }", "IOError", "ivp"},
	{"Integrate.odeint([1], [0, 1]) { |t, y| raise IOError, 'ode' }", "IOError", "ode"},
	{"Integrate.nquad([[0, 1]]) { |x| raise IOError, 'n' }", "IOError", "n"},
	{"Integrate.tplquad(0, 1, 0, 1, 0, 1) { |z, y, x| raise IOError, 't' }", "IOError", "t"},
	{"Optimize.minimize_scalar { |x| raise IOError, 'ms' }", "IOError", "ms"},
	{"Optimize.brentq(0, 1) { |x| raise IOError, 'bq' }", "IOError", "bq"},
	{"Optimize.newton(1.0) { |x| raise IOError, 'nw' }", "IOError", "nw"},
	{"Optimize.root_scalar(bracket: [0, 1]) { |x| raise IOError, 'rs' }", "IOError", "rs"},
	{"Optimize.leastsq([1], m: 1) { |x| raise IOError, 'lq' }", "IOError", "lq"},
	{"Optimize.curve_fit([1, 2], [1, 2], p0: [1]) { |x, a| raise IOError, 'cf' }", "IOError", "cf"},
	{"Optimize.brute([[0, 1]]) { |x| raise IOError, 'br' }", "IOError", "br"},
	{"Optimize.fminbound(0, 1) { |x| raise IOError, 'fb' }", "IOError", "fb"},
	{"Optimize.bracket(0, 1) { |x| raise IOError, 'bk' }", "IOError", "bk"},
	{"Optimize.approx_fprime([1], 1e-6) { |x| raise IOError, 'af' }", "IOError", "af"},
	{"Optimize.fsolve([1]) { |x| raise IOError, 'fs' }", "IOError", "fs"},
	{"Optimize.brute([[0, 1]], workers: 2) { |x| 0 }", "ArgumentError", "workers: must be 1"},
	{"Optimize.curve_fit([1], [1], p0: [1], workers: 2) { |x, a| a }", "ArgumentError", "workers: must be 1"},
}

func TestErrors(t *testing.T) {
	in := newInterp(t)
	for _, c := range errorCases {
		evalErr(t, in, c.src, c.class, c.msg)
	}
}

// A break in the block leaves the Go algorithm and is the call's value, as in
// Ruby.
func TestBreakOutOfTheBlock(t *testing.T) {
	in := newInterp(t)
	for _, src := range []string{
		"Optimize.minimize([1.0]) { |x| break :early }",
		"Optimize.brute([[0, 1]]) { |x| break :early }", // from Brute's own goroutine
		"catch(:early) { Optimize.brute([[0, 1]]) { |x| throw :early, :early } }",
	} {
		if got := eval(t, in, src); got != rb.Symbol("early") {
			t.Fatalf("%s = %#v", src, got)
		}
	}
}

// Register fails, and says why, when a name it defines is taken.
func TestRegisterConflicts(t *testing.T) {
	for _, pre := range []string{"Stats = 1", "Special = 1", "Optimize = 1", "Integrate = 1",
		"module Stats; Normal = 1; end", "module Stats; RNG = 1; end", "module Stats; GaussianKDE = 1; end",
		"module Stats; MultivariateNormal = 1; end", "module Integrate; ODESolution = 1; end", "module Stats; TestResult = 1; end"} {
		in := rb.New()
		if _, err := in.Eval(pre, "pre.rb", 1); err != nil {
			t.Fatal(err)
		}
		err := Register(in)
		if err == nil || !strings.Contains(err.Error(), "already defined") {
			t.Errorf("%s: Register = %v", pre, err)
		}
	}
}

// statsErrorCases are the Stats calls that must raise.
var statsErrorCases = []struct{ src, class, msg string }{
	{"Stats.mean", "ArgumentError", "wrong number of arguments (given 0, expected 1)"},
	{"Stats.mean([1], [2])", "ArgumentError", "wrong number of arguments (given 2, expected 1)"},
	{"Stats.mean('x')", "TypeError", "an Array of Floats"},
	{"Stats.mean([1], nope: 1)", "ArgumentError", "unknown keyword: :nope"},
	{"Stats.mean([1], ddof: 'x')", "TypeError", "into Float"},
	{"Stats.trim_mean([1, 2], 0.1, extra: 1)", "ArgumentError", "unknown keyword: :extra"},
	{"Stats.mean([1], random_state: 1)", "ArgumentError", "unknown keyword: :random_state"},
	{"Stats.mean([])", "ArgumentError", "stats"},
	{"Stats::RNG.new(1).normal", "ArgumentError", "wrong number of arguments (given 0, expected 2+)"},
	{"Stats.f_oneway([1, 2], 'x')", "TypeError", "an Array of Floats"},
	{"Stats.ks_1samp([1, 2])", "ArgumentError", "no block given"},
	{"Stats.ks_1samp([1, 2]) { |x| raise IOError, 'cdf' }", "IOError", "cdf"},
	{"Stats.ks_1samp([1, 2]) { |x| 'p' }", "TypeError", "into Float"},
	{"Stats.fisher_exact([[1, 2], [3]])", "ArgumentError", "expected an Array of 2 elements"},
	{"Stats::Normal.new(1, 2, 3)", "ArgumentError", "wrong number of arguments (given 3, expected 0..2)"},
	{"Stats::Normal.new(1, loc: 2)", "ArgumentError", "loc given twice"},
	{"Stats::Normal.new('a')", "TypeError", "into Float"},
	{"Stats::Normal.new(nope: 1)", "ArgumentError", "unknown keyword: :nope (known: loc, scale)"},
	{"Stats::Normal.new(scale: 'a')", "TypeError", "into Float"},
	{"Stats::Gamma.new", "ArgumentError", "missing keyword: :a"},
	{"Stats::Normal.new(scale: -1)", "ArgumentError", "stats"},
	{"Stats::Normal.new.pdf", "ArgumentError", "wrong number of arguments (given 0, expected 1)"},
	{"Stats::Normal.new.pdf('x')", "TypeError", "into Float or NDArray"},
	{"Stats::Normal.new.pdf(['x'])", "TypeError", "into Float"},
	{"Stats::Normal.new.mean(1)", "ArgumentError", "wrong number of arguments (given 1, expected 0)"},
	{"Stats::Normal.new.loc(1)", "ArgumentError", "wrong number of arguments (given 1, expected 0)"},
	{"Stats::Normal.new.moment", "ArgumentError", "wrong number of arguments (given 0, expected 1)"},
	{"Stats::Normal.new.moment(1.5)", "TypeError", "into Integer"},
	{"Stats::Normal.new.rvs(2, random_state: -1)", "ArgumentError", "non-negative"},
	{"Stats::Normal.new.rvs(2, random_state: 'x')", "TypeError", "a seed or a Stats::RNG"},
	{"Stats::Normal.new.rvs('x')", "TypeError", "into Integer"},
	{"Stats::Normal.new.rvs(-1, random_state: 1)", "ArgumentError", ""},
	{"Stats::Normal.fit", "ArgumentError", "wrong number of arguments"},
	{"Stats::Normal.fit([1, 2], nope: 1)", "ArgumentError", "unknown keyword: :nope (known: fixed)"},
	{"Stats::Normal.fit([1, 2], fixed: 1)", "TypeError", "into Hash"},
	{"Stats::Normal.fit([1, 2], fixed: {1 => 2})", "TypeError", "into String"},
	{"Stats::Normal.fit([1, 2], fixed: {nope: 2})", "ArgumentError", "unknown parameter nope (known: loc, scale)"},
	{"Stats::Normal.fit([1, 2], fixed: {loc: 'x'})", "TypeError", "into Float"},
	{"Stats::Normal.fit('x')", "TypeError", "an Array of Floats"},
	{"Stats::Normal.fit([])", "ArgumentError", "stats"},
	{"Stats::RNG.new(1, 2)", "ArgumentError", "wrong number of arguments"},
	{"Stats::RNG.new(-1)", "ArgumentError", "non-negative"},
	{"Stats::RNG.new(1.5)", "ArgumentError", "non-negative"},
	{"Stats::GaussianKDE.new([1, 2, 3]).evaluate", "ArgumentError", "wrong number of arguments"},
	{"Stats::GaussianKDE.new([1, 2, 3]).evaluate('x')", "TypeError", "an Array of Floats"},
	{"Stats::GaussianKDE.new(NDArray.from([1, 2, 3, 4, 5, 6.5], [2, 3])).evaluate([1, 2, 3])", "ArgumentError", "stats"},
	{"Stats.bootstrap", "ArgumentError", "wrong number of arguments"},
	{"Stats.bootstrap([[1, 2]])", "ArgumentError", "no block given"},
	{"Stats.bootstrap('x') { }", "TypeError", "an Array of samples"},
	{"Stats.bootstrap(['x']) { }", "TypeError", "an Array of Floats"},
	{"Stats.bootstrap([[1, 2]], random_state: 'x') { }", "TypeError", "a seed"},
	{"Stats.bootstrap([[1, 2]], nope: 1) { }", "ArgumentError", "unknown keyword"},
	{"Stats.bootstrap([[1, 2, 3]], n_resamples: 9) { |s| raise IOError, 'stat' }", "IOError", "stat"},
	{"Stats.bootstrap([[1, 2, 3]], n_resamples: 9) { |s| 'x' }", "TypeError", "into Float"},
	{"Stats.bootstrap([[1]]) { |s| s.mean }", "ArgumentError", "stats"},
	{"Stats::MultivariateNormal.new('m', NDArray.eye(2))", "TypeError", "an Array of Floats"},
	{"Stats::MultivariateNormal.new([0], NDArray.eye(2))", "ArgumentError", "stats"},
	{"Stats::MultivariateNormal.new([0, 0], NDArray.eye(2), nope: 1)", "ArgumentError", "unknown keyword"},
	{"Stats::MultivariateNormal.new([0, 0])", "ArgumentError", "wrong number of arguments (given 1, expected 2)"},
	{"Stats::MultivariateNormal.new([0, 0], NDArray.eye(2), 3)", "ArgumentError", "wrong number of arguments (given 3, expected 2)"},
	{"Stats::Normal.new.pdf(1, nope: 2)", "ArgumentError", "unknown keyword: :nope"},
}

func TestStatsErrors(t *testing.T) {
	in := newInterp(t)
	for _, c := range statsErrorCases {
		evalErr(t, in, c.src, c.class, c.msg)
	}
}

func TestBlockRequired(t *testing.T) {
	in := newInterp(t)
	for _, src := range []string{
		"Integrate.quad(0, 1)", "Integrate.dblquad(0, 1, 0, 1)", "Integrate.tplquad(0, 1, 0, 1, 0, 1)", "Integrate.nquad([[0, 1]])",
		"Integrate.romberg(0, 1)", "Integrate.solve_ivp([0, 1], [1])", "Integrate.odeint([1], [0, 1])",
		"Optimize.newton(1.0)", "Optimize.minimize_scalar", "Optimize.root([1])", "Optimize.fsolve([1])", "Optimize.least_squares([1])",
		"Optimize.leastsq([1])", "Optimize.curve_fit([1], [1])", "Optimize.brute([[0, 1]])", "Optimize.fminbound(0, 1)",
		"Optimize.bracket(0, 1)", "Optimize.approx_fprime([1], 1e-6)", "Optimize.root_scalar", "Optimize.brentq(0, 1)",
	} {
		evalErr(t, in, src, "ArgumentError", "no block given")
	}
	evalErr(t, in, "Optimize.linprog", "ArgumentError", "wrong number of arguments")
}

func TestReflectEdges(t *testing.T) {
	if r, err := results(nil); r != nil || err != nil {
		t.Fatal(r, err)
	}
	if got := out(reflect.ValueOf(stats.Less)); got != rb.Symbol("less") {
		t.Fatalf("enum out = %#v", got)
	}
	if got := out(reflect.ValueOf(stats.Alternative(99))); got != stats.Alternative(99) {
		t.Fatalf("unknown enum out = %#v", got)
	}
	if got := out(reflect.ValueOf((*stats.Normal)(nil))); got != nil {
		t.Fatalf("nil pointer out = %#v", got)
	}
}

func TestStatsOddments(t *testing.T) {
	in := newInterp(t)
	for src, want := range map[string]string{
		"Stats::RNG.new.inspect": "#<Stats::RNG>",
		"Stats::MultivariateNormal.new([0, 0], NDArray.eye(2)).inspect": "#<Stats::MultivariateNormal>",
		"Stats::GaussianKDE.new([1, 2, 3.5]).inspect":                   "#<Stats::GaussianKDE d=1 n=3>",
	} {
		if got := eval(t, in, src); got != want {
			t.Errorf("%s = %#v", src, got)
		}
	}
	if _, ok := eval(t, in, "Stats::RNG.new.random(2)").(*nd.Array); !ok {
		t.Fatal("RNG.new.random is not an NDArray")
	}
}
