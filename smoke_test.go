package ruby

import (
	"bytes"
	"testing"

	rb "github.com/go-embedded-ruby/ruby"
)

func TestSmoke(t *testing.T) {
	var out bytes.Buffer
	in := rb.New(rb.WithStdout(&out), rb.WithStderr(&out))
	if err := Register(in); err != nil {
		t.Fatal(err)
	}
	for _, src := range []string{
		`Stats::Normal.new(loc: 0, scale: 1).pdf(0.5)`,
		`Stats::Normal.new.pdf(NDArray.linspace(-1, 1, 3))`,
		`Stats::Normal.new(1, 2)`,
		`Stats::Normal.new(1, 2).interval(0.95)`,
		`Stats.mean([1, 2, 3, 4])`,
		`Stats.mean([1, 2, 3, 4], weights: [1, 1, 1, 5])`,
		`Stats.ttest_ind([1, 2, 3, 4], [2, 3, 4, 5.5], welch: true)`,
		`Stats.ttest_ind([1, 2, 3, 4], [2, 3, 4, 5.5], alternative: :less).pvalue`,
		`Stats.f_oneway([1, 2, 3], [2, 3, 4], [5, 6, 7])`,
		`Stats::Normal.fit([1, 2, 3, 4.5])`,
		`Stats::Gamma.fit([1, 2, 3, 4.5], fixed: {loc: 0})`,
		`Stats::Normal.new.rvs(3, random_state: 42)`,
		`Stats::Binomial.new(n: 10, p: 0.3).pmf([0, 1, 2])`,
		`Stats.ks_1samp([0.1, -0.3, 0.5]) { |x| Stats::Normal.new.cdf(x) }`,
		`Stats.describe(NDArray.arange(10))`,
		`Stats::RNG.new(42).standard_normal(2, 2)`,
		`Stats::GaussianKDE.new([1, 2, 3, 4, 7]).evaluate([2, 3])`,
		`Stats.bootstrap([[1, 2, 3, 4, 5, 6]], random_state: 1, n_resamples: 99) { |s| s.mean }`,
		`Stats::MultivariateNormal.new([0, 0], NDArray.eye(2)).pdf([0, 0])`,
		`Stats.pearsonr([1, 2, 3, 4], [1, 3, 2, 4]).confidence_interval(0.9)`,
		`Stats::Normal.new.to_h`,
		`Special.gamma(5)`,
		`Special.gamma(NDArray.arange(1, 5))`,
		`Special.beta_inc(2, 3, [0.1, 0.5])`,
		`Special.jv(NDArray.arange(3).reshape(3, 1), NDArray.linspace(0, 1, 2))`,
		`Special.jn_zeros(0, 3)`,
		`Special.log_gamma_sign(-0.5)`,
		`Special.softmax([1, 2, 3])`,
		`Special.comb(5, 2)`,
		`Special.in(1, 0.5)`,
		`Optimize.minimize([1.0, 1.0]) { |x| (x[0] - 3)**2 + (x[1] + 1)**2 }`,
		`Optimize.minimize([1.0, 1.0], method: :nelder_mead) { |x| (x[0] - 3)**2 + (x[1] + 1)**2 }.x`,
		`Optimize.minimize([1.0], method: :lbfgsb, bounds: [[2, nil]]) { |x| x[0]**2 }.x`,
		`Optimize.brentq(0, 2) { |x| x * x - 2 }.root`,
		`Optimize.newton(1.0, fprime: ->(x) { 2 * x }) { |x| x * x - 2 }.root`,
		`Optimize.root_scalar(method: "brentq", bracket: [0, 2]) { |x| x * x - 2 }.root`,
		`Optimize.root([1.0, 1.0]) { |x| [x[0] + x[1] - 3, x[0] - x[1] - 1] }.x`,
		`Optimize.curve_fit([0, 1, 2, 3], [1, 2.7, 7.4, 20.1], p0: [1, 1]) { |x, a, b| a * Math.exp(b * x) }.popt`,
		`Optimize.linprog([-1, -2], a_ub: NDArray.from([1, 1], [1, 2]), b_ub: [4], bounds: [[0, nil], [0, 3]]).x`,
		`Optimize.minimize_scalar { |x| (x - 2)**2 }.x`,
		`Optimize.least_squares([1.0, 1.0]) { |x| [x[0] - 1, x[1] - 2, x[0] + x[1] - 3] }.x`,
		`Integrate.quad(0, 1) { |x| x * x }`,
		`Integrate.quad(0, Float::INFINITY) { |x| Math.exp(-x) }.value`,
		`Integrate.dblquad(0, 1, 0, ->(x) { x }) { |y, x| x * y }.value`,
		`Integrate.solve_ivp([0, 1], [1.0], t_eval: [0, 0.5, 1]) { |t, y| [-y[0]] }.y`,
		`Integrate.odeint([1.0], [0, 0.5, 1]) { |t, y| [-y[0]] }`,
		`Integrate.trapezoid([1, 2, 3])`,
		`Integrate.simpson(NDArray.arange(6).reshape(2, 3), axis: 1)`,
		`Integrate.gauss_legendre(3)`,
	} {
		v, err := in.Eval(src, "smoke.rb", 1)
		if err != nil {
			t.Errorf("%s: %v", src, err)
			continue
		}
		t.Logf("%s => %s", src, v.Inspect())
	}
}
