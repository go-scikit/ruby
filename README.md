# ruby — go-scikit

**Ruby bindings to the go-scikit stack, for the pure-Go Ruby.**

The Ruby is [go-embedded-ruby/ruby](https://github.com/go-embedded-ruby/ruby)
("rbgo", a bytecode VM in pure Go, CGO=0). This module defines, in an rbgo
interpreter, Ruby modules over the Go libraries of the stack:

| Ruby | Go library | SciPy / pandas / matplotlib counterpart |
| --- | --- | --- |
| `Stats` | [go-statistics/stats](https://github.com/go-statistics/stats) v0.1.0 | `scipy.stats` |
| `Special` | [go-numerics/special](https://github.com/go-numerics/special) v0.1.0 | `scipy.special` |
| `Optimize` | [go-numerics/optimize](https://github.com/go-numerics/optimize) v0.1.0 | `scipy.optimize` |
| `Integrate` | [go-numerics/integrate](https://github.com/go-numerics/integrate) v0.1.0 | `scipy.integrate` |
| `DataFrame` | [go-dataframe/dataframe](https://github.com/go-dataframe/dataframe) v0.1.0 | `pandas` |
| `Plot` | [go-plot/plot](https://github.com/go-plot/plot) v0.1.1 | `matplotlib.pyplot` |

Arrays are rbgo's own `NDArray` ([go-ndarray](https://github.com/go-ndarray/ndarray)):
an NDArray passed to a binding, or returned by one, is the `*ndarray.Array`
itself, never a copy.

```go
import (
	rb "github.com/go-embedded-ruby/ruby"
	scikit "github.com/go-scikit/ruby"
)

in := rb.New()
scikit.Register(in)
v, _ := in.Eval(`Stats::Normal.new(loc: 0, scale: 1).pdf(NDArray.linspace(-1, 1, 3))`, "x.rb", 1)
fmt.Println(v.Inspect()) // array([0.24197072, 0.39894228, 0.24197072])
```

```ruby
d = Stats::Normal.fit(x)                         # a Stats::Normal
t = Stats.ttest_ind(a, b, welch: true)          # t.statistic, t.pvalue, t.df
res = Optimize.minimize([-1.2, 1], method: :bfgs) { |x| (1 - x[0])**2 + 100 * (x[1] - x[0]**2)**2 }
q = Integrate.quad(0, Float::INFINITY) { |x| Math.exp(-x * x) }
sol = Integrate.solve_ivp([0, 10], [1.0, 0.0]) { |t, y| [y[1], -y[0]] }
fig, ax = Plot.subplots
ax.plot(sol.t, sol.y_array[0], color: "steelblue", label: "y")
svg = fig.to_svg
```

One short program per module is in [examples/](examples/); a test runs them all.

> **Overlap with rbgo.** Since v0.7.0, rbgo defines modules `Stats`, `Special`, `Optimize` and `Integrate` of its own, over the same four libraries, with SciPy's names (`Stats.norm`, `Special.gammaln`). `Register` adds these bindings to those modules. Where both define the same name (`Special.gamma`, `Optimize.minimize`, …) this package's method replaces rbgo's, and the other names of both remain. Which of the two should own these modules is an open question, tracked with the release notes.

## The rules

- **Same code underneath.** Every Ruby method calls the Go function it is named
  after, with the same inputs; the tests compare each binding with the direct
  Go call and require the results to be identical, bit for bit (a figure:
  the same SVG bytes). No numerical code is written here.
- **Names** are snake_case, SciPy's where SciPy has one: `ttest_ind`,
  `mannwhitneyu`, `pvalue`, `logpdf`. Special's functions are the Go names in
  snake_case (`Special.beta_inc`, `Special.gamma_p_inv`).
- **Keyword arguments are the fields of the Go option struct**:
  `Stats.mean(x, weights: w, omit_nan: true)` fills `stats.MeanOptions`,
  `Optimize.minimize(x0, max_iter: 400, gtol: 1e-8)` `optimize.Options`. An
  enumerated field takes a symbol (`alternative: :less`, `method: :nelder_mead`,
  or the Go string value, `method: "L-BFGS-B"`), a bound `[lo, hi]` with `nil`
  for a missing side, a struct a Hash. An unknown keyword is an ArgumentError
  that names the known ones.
- **Data** is an Array of numbers or an NDArray (an NDArray goes to the Go
  function as it is: `stats.Data` is `[]float64 | *ndarray.Array`). Elementwise
  methods (`pdf`, `cdf`, `ppf`, … and every real function of Special) take a
  Float, an Array or an NDArray, and broadcast as NumPy does.
- **Results** are Ruby objects of a class named after the Go type
  (`Stats::TTestResult`, `Optimize::Result`, `Integrate::Solution`), with a
  reader per field, `to_h`, `inspect`, and the Go type's own methods
  (`res.confidence_interval(0.95)`). A `[]float64` comes back as an NDArray.
- **Callbacks are blocks** (or Procs, for `jac:`, `fprime:`, and integration
  limits): a vector reaches the block as an NDArray, and the block returns a
  Float, or an Array or NDArray for a vector. An exception in the block stops
  the Go algorithm at once and is raised again, unchanged, by the binding; a
  `break` or a `throw` leaves it as in Ruby. A binding that would call the block
  from several goroutines refuses `workers:` above 1.
- **Random draws** take `random_state:` — an Integer seed (NumPy's
  `default_rng(seed)`, the same numbers) or a `Stats::RNG`.

## What is bound

Every module function (`Module.name`), class method and instance method,
grouped by module and class; `TestREADMEListsEveryBinding` fails when a
binding is missing here. The argument lists, and what each name calls in Go,
are in the doc comments of [stats.go](stats.go), [special.go](special.go),
[optimize.go](optimize.go), [integrate.go](integrate.go),
[dataframe.go](dataframe.go) and [plot.go](plot.go). Every distribution
class of `Stats` (`Beta` … `Zipf`) has the same methods as `Stats::Normal`
(`pmf`/`logpmf` in place of `pdf`/`logpdf` for the discrete ones), plus its
parameters as readers.

#### DataFrame

`*` `+` `-` `/` `[]` `add` `apply` `astype` `at` `bfill` `column` `column_at` `columns` `copy` `count` `describe` `display` `div` `drop` `dropna` `dtypes` `eval` `ffill` `fillna` `filter` `group_by` `groupby` `head` `iat` `iloc` `index` `inspect` `isna` `join` `length` `loc` `max` `mean` `median` `melt` `merge` `min` `mul` `notna` `pivot` `pivot_table` `query` `reindex` `reindex_columns` `rename` `resample` `reset_index` `round` `select` `series` `set_columns` `set_index` `shape` `shift` `size` `sort_index` `sort_values` `stack` `std` `sub` `sum` `t` `tail` `to_csv` `to_h` `to_html` `to_ndarray` `to_period` `to_s` `transpose` `unstack` `values` `var` `with_column` `with_index` `write_csv` `_from_pairs` `concat` `from_array` `from_series` `new` `read_csv` `read_csv_file`

#### DataFrame::DatetimeFields

`day` `day_of_week` `day_of_year` `dayofweek` `dayofyear` `hour` `minute` `month` `normalize` `second` `strftime` `year`

#### DataFrame::GroupBy

`[]` `_agg` `agg` `count` `first` `groups` `inspect` `last` `max` `mean` `median` `min` `n_groups` `ngroups` `nunique` `select` `size` `std` `sum` `to_s` `transform` `var`

#### DataFrame::Index

`[]` `display` `dtype` `get` `inspect` `label` `labels` `length` `level` `name` `names` `nlevels` `range?` `rename` `size` `to_a` `to_s` `values` `new` `range`

#### DataFrame::Resampler

`_agg` `agg` `count` `first` `last` `max` `mean` `median` `min` `size` `std` `sum`

#### DataFrame::Rolling

`max` `mean` `median` `min` `std` `sum`

#### DataFrame::Series

`%` `&` `*` `**` `+` `-` `-@` `/` `<` `<=` `>` `>=` `[]` `^` `abs` `add` `agg` `astype` `between` `bfill` `copy` `count` `cummax` `cummin` `cumprod` `cumsum` `diff` `display` `div` `dropna` `dt` `dtype` `eq` `ffill` `fillna` `filter` `floats` `floordiv` `ge` `get` `gt` `head` `iloc` `index` `inspect` `invert` `isin` `isna` `le` `length` `loc` `lt` `map` `max` `mean` `median` `min` `mod` `mul` `name` `ne` `neg` `notna` `nunique` `pct_change` `pow` `prod` `quantile` `reindex` `rename` `reset_index` `rolling` `round` `shift` `size` `sort_index` `sort_values` `std` `sub` `sum` `tail` `to_a` `to_datetime` `to_frame` `to_html` `to_ndarray` `to_period` `to_s` `to_timestamp` `unique` `unstack` `valid` `value_counts` `values` `var` `with_index` `|` `~` `new`

#### Integrate

`cumulative_trapezoid` `dblquad` `fixed_quad` `gauss_legendre` `nquad` `odeint` `quad` `romb` `romberg` `simpson` `solve_ivp` `tplquad` `trapezoid`

#### Integrate::ODESolution

`eval` `eval_array` `t_max` `t_min`

#### Integrate::OdeintInfo

`hu` `inspect` `m_used` `message` `nfe` `nje` `nst` `t_cur` `to_h`

#### Integrate::QuadResult

`abs_err` `cycle_ier` `inspect` `n_eval` `n_intervals` `to_h` `value`

#### Integrate::Solution

`component` `inspect` `message` `n_fev` `n_fev_jac` `n_jev` `n_steps` `nlu` `sol` `status` `success` `t` `t_array` `t_events` `to_h` `y` `y_array` `y_events`

#### Optimize

`approx_fprime` `bisect` `bracket` `brenth` `brentq` `brute` `curve_fit` `fminbound` `fsolve` `least_squares` `leastsq` `linprog` `minimize` `minimize_scalar` `newton` `ridder` `root` `root_scalar`

#### Optimize::BruteResult

`axes` `fun` `inspect` `jout` `success` `to_h` `x`

#### Optimize::CurveFitResult

`cov_ok` `fvec` `inspect` `message` `nfev` `pcov` `pcov_array` `perr` `popt` `popt_array` `status` `to_h`

#### Optimize::LPDual

`inspect` `marginals` `residual` `to_h`

#### Optimize::LPResult

`con` `eq_lin` `fun` `ineq_lin` `inspect` `lower` `message` `nit` `slack` `status` `success` `to_h` `upper` `x` `x_array`

#### Optimize::LSResult

`active_mask` `cost` `fun` `grad` `inspect` `jac` `jac_array` `message` `nfev` `njev` `optimality` `status` `success` `to_h` `x` `x_array`

#### Optimize::LeastsqResult

`cov_x` `cov_x_array` `fjac` `fjac_array` `fvec` `ier` `inspect` `ipvt` `mesg` `nfev` `njev` `qtf` `success` `to_h` `x` `x_array`

#### Optimize::Result

`fun` `hess_inv` `hess_inv_array` `inspect` `jac` `message` `nfev` `nhev` `nit` `njev` `status` `success` `to_h` `x` `x_array`

#### Optimize::RootResult

`fjac` `fjac_array` `fun` `inspect` `message` `nfev` `njev` `qtf` `r` `status` `success` `to_h` `x` `x_array`

#### Optimize::RootScalarResult

`converged` `flag` `function_calls` `inspect` `iterations` `method` `root` `to_h`

#### Optimize::ScalarResult

`fun` `inspect` `message` `nfev` `nit` `status` `success` `to_h` `x`

#### Plot

`figure` `subplots` `values`

#### Plot::Axes

`annotate` `aspect=` `axhline` `axvline` `bar` `barh` `contour` `contourf` `errorbar` `figure` `fill_between` `grid` `hist` `imshow` `legend` `plot` `scatter` `set_aspect` `set_title` `set_xlabel` `set_xlim` `set_xscale` `set_xticks` `set_ylabel` `set_ylim` `set_yscale` `set_yticks` `text` `title=` `twinx` `xlabel=` `xlim` `xlim=` `xscale=` `ylabel=` `ylim` `ylim=` `yscale=`

#### Plot::Figure

`add_axes` `axes` `colorbar` `display` `dpi` `height` `inspect` `pixel_size` `save_png` `save_svg` `savefig` `subplots` `tight_layout` `to_png` `to_s` `to_svg` `width` `new`

#### Plot::Mappable

`inspect`

#### Special

`beta` `beta_inc` `beta_inc_c` `beta_inc_c_inv` `beta_inc_inv` `binom` `comb` `digamma` `ellip_e` `ellip_k` `ellip_km1` `erf` `erf_inv` `erfc` `erfc_inv` `erfcx` `expit` `factorial` `factorial2` `gamma` `gamma_p` `gamma_p_inv` `gamma_q` `gamma_q_inv` `gamma_sign` `hurwitz_zeta` `i0` `i0e` `i1` `i1e` `in` `iv` `ive` `j0` `j1` `jn` `jn_zeros` `jv` `k0` `k0e` `k1` `k1e` `kn` `kv` `kve` `log1p_exp` `log_beta` `log_expit` `log_factorial` `log_gamma` `log_gamma_sign` `log_ndtr` `log_softmax` `log_sum_exp` `logit` `ndtr` `ndtri` `perm` `polygamma` `softmax` `xlog1py` `xlogy` `y0` `y1` `yn` `yv` `zeta`

#### Stats

`anderson` `average` `bartlett` `binned_statistic` `bootstrap` `chi2_contingency` `chisquare` `cumfreq` `describe` `differential_entropy` `entropy` `f_oneway` `fisher_exact` `fit_multivariate_normal` `gmean` `histogram` `histogram_bin_edges` `hmean` `iqr` `jarque_bera` `kendalltau` `kruskal` `ks_1samp` `ks_2samp` `kurtosis` `kurtosistest` `levene` `linregress` `mannwhitneyu` `mean` `median` `median_abs_deviation` `mode` `moment` `normaltest` `pearsonr` `percentile` `percentiles` `pmean` `power_divergence` `quantile` `quantiles` `rankdata` `relfreq` `sem` `shapiro` `skew` `skewtest` `spearmanr` `std` `tmax` `tmean` `tmin` `trim1` `trim_mean` `trimboth` `tsem` `tstd` `ttest_1samp` `ttest_ind` `ttest_rel` `tvar` `var` `variation` `wilcoxon` `zmap` `zscore`

#### Stats::AndersonResult

`critical_values` `inspect` `loc` `pvalue` `scale` `significance_levels` `statistic` `to_h`

#### Stats::Bernoulli

`cdf` `entropy` `inspect` `interval` `isf` `kurtosis` `loc` `logcdf` `logpmf` `logsf` `mean` `median` `moment` `p` `pmf` `ppf` `rvs` `sf` `skew` `std` `support` `to_h` `validate` `var` `new`

#### Stats::Beta

`a` `b` `cdf` `entropy` `inspect` `interval` `isf` `kurtosis` `loc` `logcdf` `logpdf` `logsf` `mean` `median` `moment` `pdf` `ppf` `rvs` `scale` `sf` `skew` `std` `support` `to_h` `validate` `var` `fit` `new`

#### Stats::BinnedResult

`bin_number` `edges` `inspect` `statistic` `to_h`

#### Stats::Binomial

`cdf` `entropy` `inspect` `interval` `isf` `kurtosis` `loc` `logcdf` `logpmf` `logsf` `mean` `median` `moment` `n` `p` `pmf` `ppf` `rvs` `sf` `skew` `std` `support` `to_h` `validate` `var` `new`

#### Stats::BootstrapResult

`bootstrap_distribution` `confidence_interval` `inspect` `standard_error` `to_h`

#### Stats::CDFEstimate

`error` `inspect` `n_points` `to_h` `value`

#### Stats::Cauchy

`cdf` `entropy` `inspect` `interval` `isf` `kurtosis` `loc` `logcdf` `logpdf` `logsf` `mean` `median` `moment` `pdf` `ppf` `rvs` `scale` `sf` `skew` `std` `support` `to_h` `validate` `var` `fit` `new`

#### Stats::Chi

`cdf` `df` `entropy` `inspect` `interval` `isf` `kurtosis` `loc` `logcdf` `logpdf` `logsf` `mean` `median` `moment` `pdf` `ppf` `rvs` `scale` `sf` `skew` `std` `support` `to_h` `validate` `var` `fit` `new`

#### Stats::Chi2ContingencyResult

`dof` `expected` `inspect` `pvalue` `statistic` `to_h`

#### Stats::ChiSquared

`cdf` `df` `entropy` `inspect` `interval` `isf` `kurtosis` `loc` `logcdf` `logpdf` `logsf` `mean` `median` `moment` `pdf` `ppf` `rvs` `scale` `sf` `skew` `std` `support` `to_h` `validate` `var` `fit` `new`

#### Stats::ConfidenceInterval

`high` `inspect` `low` `to_h`

#### Stats::DescribeResult

`inspect` `kurtosis` `max` `mean` `min` `nobs` `skewness` `to_h` `variance`

#### Stats::Dirichlet

`alpha` `cov` `entropy` `inspect` `log_pdf_points` `logpdf` `mean` `pdf` `pdf_points` `rvs` `var` `new`

#### Stats::DiscreteUniform

`cdf` `entropy` `high` `inspect` `interval` `isf` `kurtosis` `loc` `logcdf` `logpmf` `logsf` `low` `mean` `median` `moment` `pmf` `ppf` `rvs` `sf` `skew` `std` `support` `to_h` `validate` `var` `new`

#### Stats::Exponential

`cdf` `entropy` `inspect` `interval` `isf` `kurtosis` `loc` `logcdf` `logpdf` `logsf` `mean` `median` `moment` `pdf` `ppf` `rvs` `scale` `sf` `skew` `std` `support` `to_h` `validate` `var` `fit` `new`

#### Stats::F

`cdf` `dfd` `dfn` `entropy` `inspect` `interval` `isf` `kurtosis` `loc` `logcdf` `logpdf` `logsf` `mean` `median` `moment` `pdf` `ppf` `rvs` `scale` `sf` `skew` `std` `support` `to_h` `validate` `var` `fit` `new`

#### Stats::FreqResult

`bin_size` `extra_points` `freq` `inspect` `lower_limit` `to_h`

#### Stats::Gamma

`a` `cdf` `entropy` `inspect` `interval` `isf` `kurtosis` `loc` `logcdf` `logpdf` `logsf` `mean` `median` `moment` `pdf` `ppf` `rvs` `scale` `sf` `skew` `std` `support` `to_h` `validate` `var` `fit` `new`

#### Stats::GaussianKDE

`call` `cho_cov` `covariance` `d` `dataset` `evaluate` `factor` `inspect` `integrate_box` `integrate_box_1d` `integrate_gaussian` `integrate_kde` `inv_cov` `log_det` `log_pdf_slice` `logpdf` `marginal` `n` `neff` `resample` `scotts_factor` `set_bandwidth` `silverman_factor` `weights` `new`

#### Stats::Geometric

`cdf` `entropy` `inspect` `interval` `isf` `kurtosis` `loc` `logcdf` `logpmf` `logsf` `mean` `median` `moment` `p` `pmf` `ppf` `rvs` `sf` `skew` `std` `support` `to_h` `validate` `var` `new`

#### Stats::Gumbel

`cdf` `entropy` `inspect` `interval` `isf` `kurtosis` `loc` `logcdf` `logpdf` `logsf` `mean` `median` `moment` `pdf` `ppf` `rvs` `scale` `sf` `skew` `std` `support` `to_h` `validate` `var` `fit` `new`

#### Stats::HalfNormal

`cdf` `entropy` `inspect` `interval` `isf` `kurtosis` `loc` `logcdf` `logpdf` `logsf` `mean` `median` `moment` `pdf` `ppf` `rvs` `scale` `sf` `skew` `std` `support` `to_h` `validate` `var` `fit` `new`

#### Stats::HistogramResult

`counts` `edges` `inspect` `to_h`

#### Stats::Hypergeometric

`cdf` `draws` `entropy` `good` `inspect` `interval` `isf` `kurtosis` `loc` `logcdf` `logpmf` `logsf` `mean` `median` `moment` `pmf` `ppf` `rvs` `sf` `skew` `std` `support` `to_h` `total` `validate` `var` `new`

#### Stats::InverseGamma

`a` `cdf` `entropy` `inspect` `interval` `isf` `kurtosis` `loc` `logcdf` `logpdf` `logsf` `mean` `median` `moment` `pdf` `ppf` `rvs` `scale` `sf` `skew` `std` `support` `to_h` `validate` `var` `fit` `new`

#### Stats::KSResult

`inspect` `pvalue` `statistic` `statistic_location` `statistic_sign` `to_h`

#### Stats::Laplace

`cdf` `entropy` `inspect` `interval` `isf` `kurtosis` `loc` `logcdf` `logpdf` `logsf` `mean` `median` `moment` `pdf` `ppf` `rvs` `scale` `sf` `skew` `std` `support` `to_h` `validate` `var` `fit` `new`

#### Stats::LinregressResult

`inspect` `intercept` `intercept_std_err` `pvalue` `r_value` `slope` `std_err` `to_h`

#### Stats::LogNormal

`cdf` `entropy` `inspect` `interval` `isf` `kurtosis` `loc` `logcdf` `logpdf` `logsf` `mean` `median` `moment` `pdf` `ppf` `rvs` `s` `scale` `sf` `skew` `std` `support` `to_h` `validate` `var` `fit` `new`

#### Stats::Logistic

`cdf` `entropy` `inspect` `interval` `isf` `kurtosis` `loc` `logcdf` `logpdf` `logsf` `mean` `median` `moment` `pdf` `ppf` `rvs` `scale` `sf` `skew` `std` `support` `to_h` `validate` `var` `fit` `new`

#### Stats::Lomax

`c` `cdf` `entropy` `inspect` `interval` `isf` `kurtosis` `loc` `logcdf` `logpdf` `logsf` `mean` `median` `moment` `pdf` `ppf` `rvs` `scale` `sf` `skew` `std` `support` `to_h` `validate` `var` `fit` `new`

#### Stats::ModeResult

`count` `inspect` `mode` `to_h`

#### Stats::Multinomial

`cov` `entropy` `inspect` `log_pmf_points` `logpmf` `mean` `n` `p` `pmf` `pmf_points` `rvs` `new`

#### Stats::MultivariateNormal

`cdf` `cdf_estimate` `cdf_points` `cov` `dim` `entropy` `inspect` `log_p_det` `log_pdf_points` `logcdf` `logpdf` `mean` `pdf` `pdf_points` `rank` `rvs` `new`

#### Stats::NegativeBinomial

`cdf` `entropy` `inspect` `interval` `isf` `kurtosis` `loc` `logcdf` `logpmf` `logsf` `mean` `median` `moment` `n` `p` `pmf` `ppf` `rvs` `sf` `skew` `std` `support` `to_h` `validate` `var` `new`

#### Stats::NoncentralChiSquared

`cdf` `df` `entropy` `inspect` `interval` `isf` `kurtosis` `loc` `logcdf` `logpdf` `logsf` `mean` `median` `moment` `nc` `pdf` `ppf` `rvs` `scale` `sf` `skew` `std` `support` `to_h` `validate` `var` `new`

#### Stats::Normal

`cdf` `entropy` `inspect` `interval` `isf` `kurtosis` `loc` `logcdf` `logpdf` `logsf` `mean` `median` `moment` `pdf` `ppf` `rvs` `scale` `sf` `skew` `std` `support` `to_h` `validate` `var` `fit` `new`

#### Stats::Pareto

`b` `cdf` `entropy` `inspect` `interval` `isf` `kurtosis` `loc` `logcdf` `logpdf` `logsf` `mean` `median` `moment` `pdf` `ppf` `rvs` `scale` `sf` `skew` `std` `support` `to_h` `validate` `var` `fit` `new`

#### Stats::PearsonResult

`confidence_interval` `inspect` `pvalue` `statistic` `to_h`

#### Stats::Poisson

`cdf` `entropy` `inspect` `interval` `isf` `kurtosis` `loc` `logcdf` `logpmf` `logsf` `mean` `median` `moment` `mu` `pmf` `ppf` `rvs` `sf` `skew` `std` `support` `to_h` `validate` `var` `new`

#### Stats::RNG

`beta` `binomial` `exponential` `gamma` `inspect` `integers` `normal` `poisson` `random` `standard_exponential` `standard_normal` `uniform` `new`

#### Stats::RankTestResult

`inspect` `pvalue` `statistic` `to_h` `zstatistic`

#### Stats::Rayleigh

`cdf` `entropy` `inspect` `interval` `isf` `kurtosis` `loc` `logcdf` `logpdf` `logsf` `mean` `median` `moment` `pdf` `ppf` `rvs` `scale` `sf` `skew` `std` `support` `to_h` `validate` `var` `fit` `new`

#### Stats::Reciprocal

`a` `b` `cdf` `entropy` `inspect` `interval` `isf` `kurtosis` `loc` `logcdf` `logpdf` `logsf` `mean` `median` `moment` `pdf` `ppf` `rvs` `scale` `sf` `skew` `std` `support` `to_h` `validate` `var` `fit` `new`

#### Stats::StudentT

`cdf` `df` `entropy` `inspect` `interval` `isf` `kurtosis` `loc` `logcdf` `logpdf` `logsf` `mean` `median` `moment` `pdf` `ppf` `rvs` `scale` `sf` `skew` `std` `support` `to_h` `validate` `var` `fit` `new`

#### Stats::TTestResult

`confidence_interval` `df` `inspect` `pvalue` `statistic` `to_h`

#### Stats::TestResult

`inspect` `pvalue` `statistic` `to_h`

#### Stats::Triangular

`c` `cdf` `entropy` `inspect` `interval` `isf` `kurtosis` `loc` `logcdf` `logpdf` `logsf` `mean` `median` `moment` `pdf` `ppf` `rvs` `scale` `sf` `skew` `std` `support` `to_h` `validate` `var` `fit` `new`

#### Stats::Uniform

`cdf` `entropy` `inspect` `interval` `isf` `kurtosis` `loc` `logcdf` `logpdf` `logsf` `mean` `median` `moment` `pdf` `ppf` `rvs` `scale` `sf` `skew` `std` `support` `to_h` `validate` `var` `fit` `new`

#### Stats::VonMises

`cdf` `entropy` `inspect` `interval` `isf` `kappa` `kurtosis` `loc` `logcdf` `logpdf` `logsf` `mean` `median` `moment` `pdf` `ppf` `rvs` `scale` `sf` `skew` `std` `support` `to_h` `validate` `var` `fit` `new`

#### Stats::Weibull

`c` `cdf` `entropy` `inspect` `interval` `isf` `kurtosis` `loc` `logcdf` `logpdf` `logsf` `mean` `median` `moment` `pdf` `ppf` `rvs` `scale` `sf` `skew` `std` `support` `to_h` `validate` `var` `fit` `new`

#### Stats::Zipf

`a` `cdf` `entropy` `inspect` `interval` `isf` `kurtosis` `loc` `logcdf` `logpmf` `logsf` `mean` `median` `moment` `pmf` `ppf` `rvs` `sf` `skew` `std` `support` `to_h` `validate` `var` `new`

## Notebooks

These bindings are what a ` ```ruby ` cell of a
[go-notebook](https://github.com/go-notebook/notebook) notebook (v0.3.0 and
later) runs on. A notebook can mix Go and Ruby cells, and their variables
cross both ways:

````markdown
```go
xs := []float64{2.1, 3.4, 1.9, 5.6, 4.4, 3.3, 2.8, 4.9}
```

```ruby
fit = Stats::Normal.fit(xs)   # xs is a Ruby Array of Floats here
loc, scale = fit.loc, fit.scale
```

```go
import "fmt"

fmt.Printf("loc %.3f, scale %.3f\n", loc, scale) // float64s here
```
````

Package [cells](cells/) runs those cells: one interpreter per worker, whose
top-level locals persist from cell to cell, and the exchange of variables with
the Go cells. Its documentation states the conversion rules both ways; a
notebook links it, you do not call it. The notebook's kernel requires this
module in its work module by itself, so a notebook's own `go.mod` only needs
go-notebook.

## Build tags

rbgo builds with or without its gem bindings; `-tags rbgo_core` leaves the
gems out (the language and the numeric stack only), which is what a notebook
worker or a browser page wants. These bindings work, and are tested, on both.
rbgo compiles for every GOOS/GOARCH pair Go supports except plan9 (the BSDs,
AIX, Solaris, illumos, every linux arch, wasm, android and ios included), and
so does this module; CI vets all of them, both builds. The suite runs natively
on linux/darwin/windows, amd64/arm64 and `386`, and under QEMU on riscv64,
loong64, ppc64le, s390x and 32-bit `arm`. Where modernc.org/sqlite has no port,
rbgo's SQLite3/Sequel/Bolt/Bleve/Etcd gems load but raise
`NotImplementedError` on use; these bindings do not use them.

## License

BSD-3-Clause. See [LICENSE](LICENSE).
