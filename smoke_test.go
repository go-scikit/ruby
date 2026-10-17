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
	} {
		v, err := in.Eval(src, "smoke.rb", 1)
		if err != nil {
			t.Errorf("%s: %v", src, err)
			continue
		}
		t.Logf("%s => %s", src, v.Inspect())
	}
}
