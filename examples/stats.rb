# Stats: distributions as classes, descriptive statistics, tests, fits.
rng = Stats::RNG.new(42)
x = Stats::Normal.new(loc: 10, scale: 2).rvs(200, random_state: rng)
y = Stats::Normal.new(loc: 10.6, scale: 2).rvs(200, random_state: rng)

puts "mean #{Stats.mean(x).round(3)}, std #{Stats.std(x, ddof: 1).round(3)}"
p Stats.describe(x)

t = Stats.ttest_ind(x, y, welch: true)
puts "Welch t = #{t.statistic.round(3)}, p = #{t.pvalue.round(4)}"
p t.confidence_interval(0.95)

fit = Stats::Normal.fit(x)
puts "fitted loc #{fit.loc.round(3)}, scale #{fit.scale.round(3)}"

# Elementwise over an NDArray, as SciPy.
grid = NDArray.linspace(4, 16, 5)
p Stats::Normal.new(loc: 10, scale: 2).pdf(grid)
p Stats.ks_1samp(x) { |v| fit.cdf(v) }.pvalue > 0.05
