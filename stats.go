package ruby

import (
	"fmt"
	"reflect"

	rb "github.com/go-embedded-ruby/ruby"
	nd "github.com/go-ndarray/ndarray"
	"github.com/go-ndarray/ndarray/random"
	"github.com/go-statistics/stats"
)

// statsFuncs are the functions of Stats over data, each the two
// instantiations of the go-statistics function it is named after.
var statsFuncs = []struct {
	name         string
	overS, overA any
	goName       string
}{
	// Descriptive statistics.
	{"mean", stats.Mean[[]float64], stats.Mean[*nd.Array], "Mean"},
	{"average", stats.Average[[]float64], stats.Average[*nd.Array], "Average"},
	{"var", stats.Var[[]float64], stats.Var[*nd.Array], "Var"},
	{"std", stats.Std[[]float64], stats.Std[*nd.Array], "Std"},
	{"median", stats.Median[[]float64], stats.Median[*nd.Array], "Median"},
	{"gmean", stats.GMean[[]float64], stats.GMean[*nd.Array], "GMean"},
	{"hmean", stats.HMean[[]float64], stats.HMean[*nd.Array], "HMean"},
	{"pmean", stats.PMean[[]float64], stats.PMean[*nd.Array], "PMean"},
	{"skew", stats.Skew[[]float64], stats.Skew[*nd.Array], "Skew"},
	{"kurtosis", stats.Kurtosis[[]float64], stats.Kurtosis[*nd.Array], "Kurtosis"},
	{"moment", stats.Moment[[]float64], stats.Moment[*nd.Array], "Moment"},
	{"sem", stats.SEM[[]float64], stats.SEM[*nd.Array], "SEM"},
	{"variation", stats.Variation[[]float64], stats.Variation[*nd.Array], "Variation"},
	{"iqr", stats.IQR[[]float64], stats.IQR[*nd.Array], "IQR"},
	{"median_abs_deviation", stats.MedianAbsDeviation[[]float64], stats.MedianAbsDeviation[*nd.Array], "MedianAbsDeviation"},
	{"quantile", stats.Quantile[[]float64], stats.Quantile[*nd.Array], "Quantile"},
	{"quantiles", stats.Quantiles[[]float64], stats.Quantiles[*nd.Array], "Quantiles"},
	{"percentile", stats.Percentile[[]float64], stats.Percentile[*nd.Array], "Percentile"},
	{"percentiles", stats.Percentiles[[]float64], stats.Percentiles[*nd.Array], "Percentiles"},
	{"tmean", stats.TMean[[]float64], stats.TMean[*nd.Array], "TMean"},
	{"tvar", stats.TVar[[]float64], stats.TVar[*nd.Array], "TVar"},
	{"tstd", stats.TStd[[]float64], stats.TStd[*nd.Array], "TStd"},
	{"tsem", stats.TSem[[]float64], stats.TSem[*nd.Array], "TSem"},
	{"tmin", stats.TMin[[]float64], stats.TMin[*nd.Array], "TMin"},
	{"tmax", stats.TMax[[]float64], stats.TMax[*nd.Array], "TMax"},
	{"trim_mean", stats.TrimMean[[]float64], stats.TrimMean[*nd.Array], "TrimMean"},
	{"trimboth", stats.TrimBoth[[]float64], stats.TrimBoth[*nd.Array], "TrimBoth"},
	{"trim1", stats.Trim1[[]float64], stats.Trim1[*nd.Array], "Trim1"},
	{"zscore", stats.ZScore[[]float64], stats.ZScore[*nd.Array], "ZScore"},
	{"rankdata", stats.RankData[[]float64], stats.RankData[*nd.Array], "RankData"},
	{"describe", stats.Describe[[]float64], stats.Describe[*nd.Array], "Describe"},
	{"mode", stats.Mode[[]float64], stats.Mode[*nd.Array], "Mode"},
	{"entropy", stats.Entropy[[]float64], stats.Entropy[*nd.Array], "Entropy"},
	{"differential_entropy", stats.DifferentialEntropy[[]float64], stats.DifferentialEntropy[*nd.Array], "DifferentialEntropy"},
	{"histogram", stats.Histogram[[]float64], stats.Histogram[*nd.Array], "Histogram"},
	{"histogram_bin_edges", stats.HistogramBinEdges[[]float64], stats.HistogramBinEdges[*nd.Array], "HistogramBinEdges"},
	{"cumfreq", stats.CumFreq[[]float64], stats.CumFreq[*nd.Array], "CumFreq"},
	{"relfreq", stats.RelFreq[[]float64], stats.RelFreq[*nd.Array], "RelFreq"},
	{"zmap", stats.ZMap[[]float64, []float64], stats.ZMap[*nd.Array, *nd.Array], "ZMap"},
	{"binned_statistic", stats.BinnedStatistic[[]float64, []float64], stats.BinnedStatistic[*nd.Array, *nd.Array], "BinnedStatistic"},
	// Tests.
	{"ttest_1samp", stats.TTest1Samp[[]float64], stats.TTest1Samp[*nd.Array], "TTest1Samp"},
	{"ttest_ind", stats.TTestInd[[]float64], stats.TTestInd[*nd.Array], "TTestInd"},
	{"ttest_rel", stats.TTestRel[[]float64], stats.TTestRel[*nd.Array], "TTestRel"},
	{"mannwhitneyu", stats.MannWhitneyU[[]float64], stats.MannWhitneyU[*nd.Array], "MannWhitneyU"},
	{"wilcoxon", stats.Wilcoxon[[]float64], stats.Wilcoxon[*nd.Array], "Wilcoxon"},
	{"ks_1samp", stats.KSTest1Samp[[]float64], stats.KSTest1Samp[*nd.Array], "KSTest1Samp"},
	{"ks_2samp", stats.KSTest2Samp[[]float64], stats.KSTest2Samp[*nd.Array], "KSTest2Samp"},
	{"shapiro", stats.ShapiroWilk[[]float64], stats.ShapiroWilk[*nd.Array], "ShapiroWilk"},
	{"normaltest", stats.Normaltest[[]float64], stats.Normaltest[*nd.Array], "Normaltest"},
	{"skewtest", stats.SkewTest[[]float64], stats.SkewTest[*nd.Array], "SkewTest"},
	{"kurtosistest", stats.KurtosisTest[[]float64], stats.KurtosisTest[*nd.Array], "KurtosisTest"},
	{"jarque_bera", stats.JarqueBera[[]float64], stats.JarqueBera[*nd.Array], "JarqueBera"},
	{"anderson", stats.AndersonDarling[[]float64], stats.AndersonDarling[*nd.Array], "AndersonDarling"},
	{"pearsonr", stats.Pearsonr[[]float64], stats.Pearsonr[*nd.Array], "Pearsonr"},
	{"spearmanr", stats.Spearmanr[[]float64], stats.Spearmanr[*nd.Array], "Spearmanr"},
	{"kendalltau", stats.Kendalltau[[]float64], stats.Kendalltau[*nd.Array], "Kendalltau"},
	{"linregress", stats.Linregress[[]float64], stats.Linregress[*nd.Array], "Linregress"},
	{"chisquare", stats.ChiSquare[[]float64], stats.ChiSquare[*nd.Array], "ChiSquare"},
	{"power_divergence", stats.PowerDivergence[[]float64], stats.PowerDivergence[*nd.Array], "PowerDivergence"},
	{"f_oneway", stats.FOneway[[]float64], stats.FOneway[*nd.Array], "FOneway"},
	{"kruskal", stats.KruskalWallis[[]float64], stats.KruskalWallis[*nd.Array], "KruskalWallis"},
	{"levene", stats.Levene[[]float64], stats.Levene[*nd.Array], "Levene"},
	{"bartlett", stats.Bartlett[[]float64], stats.Bartlett[*nd.Array], "Bartlett"},
}

// statsPlain are the functions of Stats that are not generic over the data.
var statsPlain = []struct {
	name, goName string
	fn           any
}{
	{"chi2_contingency", "Chi2Contingency", stats.Chi2Contingency},
	{"fisher_exact", "FisherExact", stats.FisherExact},
	{"fit_multivariate_normal", "FitMultivariateNormal", stats.FitMultivariateNormal},
}

// statsResults are the result types, each a class of Stats with a reader per
// field.
var statsResults = []struct {
	name   string
	sample any
}{
	{"TestResult", stats.TestResult{}},
	{"TTestResult", stats.TTestResult{}},
	{"RankTestResult", stats.RankTestResult{}},
	{"KSResult", stats.KSResult{}},
	{"PearsonResult", stats.PearsonResult{}},
	{"LinregressResult", stats.LinregressResult{}},
	{"AndersonResult", stats.AndersonResult{}},
	{"Chi2ContingencyResult", stats.Chi2ContingencyResult{}},
	{"DescribeResult", stats.DescribeResult{}},
	{"ModeResult", stats.ModeResult{}},
	{"HistogramResult", stats.HistogramResult{}},
	{"FreqResult", stats.FreqResult{}},
	{"BinnedResult", stats.BinnedResult{}},
	{"ConfidenceInterval", stats.ConfidenceInterval{}},
	{"BootstrapResult", stats.BootstrapResult{}},
	{"CDFEstimate", stats.CDFEstimate{}},
}

// dist is a distribution of Stats: its Ruby class name, its zero value, its
// fit (the two instantiations of FitXxx; nil when go-statistics has none)
// and whether it is discrete.
type dist struct {
	name     string
	zero     any
	fitS     any
	fitA     any
	discrete bool
}

var distributions = []dist{
	{"Beta", stats.Beta{}, stats.FitBeta[[]float64], stats.FitBeta[*nd.Array], false},
	{"Cauchy", stats.Cauchy{}, stats.FitCauchy[[]float64], stats.FitCauchy[*nd.Array], false},
	{"Chi", stats.Chi{}, stats.FitChi[[]float64], stats.FitChi[*nd.Array], false},
	{"ChiSquared", stats.ChiSquared{}, stats.FitChiSquared[[]float64], stats.FitChiSquared[*nd.Array], false},
	{"Exponential", stats.Exponential{}, stats.FitExponential[[]float64], stats.FitExponential[*nd.Array], false},
	{"F", stats.F{}, stats.FitF[[]float64], stats.FitF[*nd.Array], false},
	{"Gamma", stats.Gamma{}, stats.FitGamma[[]float64], stats.FitGamma[*nd.Array], false},
	{"Gumbel", stats.Gumbel{}, stats.FitGumbel[[]float64], stats.FitGumbel[*nd.Array], false},
	{"HalfNormal", stats.HalfNormal{}, stats.FitHalfNormal[[]float64], stats.FitHalfNormal[*nd.Array], false},
	{"InverseGamma", stats.InverseGamma{}, stats.FitInverseGamma[[]float64], stats.FitInverseGamma[*nd.Array], false},
	{"Laplace", stats.Laplace{}, stats.FitLaplace[[]float64], stats.FitLaplace[*nd.Array], false},
	{"LogNormal", stats.LogNormal{}, stats.FitLogNormal[[]float64], stats.FitLogNormal[*nd.Array], false},
	{"Logistic", stats.Logistic{}, stats.FitLogistic[[]float64], stats.FitLogistic[*nd.Array], false},
	{"Lomax", stats.Lomax{}, stats.FitLomax[[]float64], stats.FitLomax[*nd.Array], false},
	{"NoncentralChiSquared", stats.NoncentralChiSquared{}, nil, nil, false},
	{"Normal", stats.Normal{}, stats.FitNormal[[]float64], stats.FitNormal[*nd.Array], false},
	{"Pareto", stats.Pareto{}, stats.FitPareto[[]float64], stats.FitPareto[*nd.Array], false},
	{"Rayleigh", stats.Rayleigh{}, stats.FitRayleigh[[]float64], stats.FitRayleigh[*nd.Array], false},
	{"Reciprocal", stats.Reciprocal{}, stats.FitReciprocal[[]float64], stats.FitReciprocal[*nd.Array], false},
	{"StudentT", stats.StudentT{}, stats.FitStudentT[[]float64], stats.FitStudentT[*nd.Array], false},
	{"Triangular", stats.Triangular{}, stats.FitTriangular[[]float64], stats.FitTriangular[*nd.Array], false},
	{"Uniform", stats.Uniform{}, stats.FitUniform[[]float64], stats.FitUniform[*nd.Array], false},
	{"VonMises", stats.VonMises{}, stats.FitVonMises[[]float64], stats.FitVonMises[*nd.Array], false},
	{"Weibull", stats.Weibull{}, stats.FitWeibull[[]float64], stats.FitWeibull[*nd.Array], false},
	{"Bernoulli", stats.Bernoulli{}, nil, nil, true},
	{"Binomial", stats.Binomial{}, nil, nil, true},
	{"DiscreteUniform", stats.DiscreteUniform{}, nil, nil, true},
	{"Geometric", stats.Geometric{}, nil, nil, true},
	{"Hypergeometric", stats.Hypergeometric{}, nil, nil, true},
	{"NegativeBinomial", stats.NegativeBinomial{}, nil, nil, true},
	{"Poisson", stats.Poisson{}, nil, nil, true},
	{"Zipf", stats.Zipf{}, nil, nil, true},
}

// elementwise are the methods of a distribution that apply to each element
// of an NDArray or an Array, as SciPy's do.
var elementwise = map[string]bool{
	"PDF": true, "LogPDF": true, "PMF": true, "LogPMF": true, "CDF": true, "LogCDF": true,
	"SF": true, "LogSF": true, "PPF": true, "ISF": true,
}

func registerStats(in *rb.Interp) error {
	d := &definer{in: in}
	m := d.module("Stats")
	if d.err != nil {
		return d.err
	}
	for _, r := range statsResults {
		d.structClass(m, r.name, r.sample)
	}
	for _, f := range statsFuncs {
		d.fn(m, f.name, dataFunc(f.overS, f.overA))
	}
	for _, f := range statsPlain {
		fn := reflect.ValueOf(f.fn)
		d.fn(m, f.name, func(c *rb.Call) (any, error) { return invoke(fn, c, c.Args) })
	}
	for _, ds := range distributions {
		defineDist(d, m, ds)
	}
	defineRNG(d, m)
	defineKDE(d, m)
	defineMultivariate(d, m)
	d.fn(m, "bootstrap", bootstrap)
	return d.err
}

// defineDist defines a distribution class: new from its parameters
// (positional in the Go struct's field order, or by keyword; loc defaults to
// 0 and scale to 1), fit, the elementwise methods, rvs, and every other method
// of the Go type.
func defineDist(d *definer, m *rb.Module, ds dist) {
	c := d.structClass(m, ds.name, ds.zero)
	t := reflect.TypeOf(ds.zero)
	fields := readableFields(t)
	c.DefineSingletonMethod("new", func(call *rb.Call) (any, error) {
		v, err := newDist(t, fields, call.Args, call.Kwargs)
		if err != nil {
			return nil, err
		}
		return v.Interface(), nil
	})
	for name := range elementwise {
		if _, ok := reflect.PointerTo(t).MethodByName(name); !ok {
			continue
		}
		c.DefineMethod(snakeName(name), func(call *rb.Call) (any, error) {
			if err := arity(call, 1, 1); err != nil {
				return nil, err
			}
			f := reflect.ValueOf(call.Self.Go()).MethodByName(name).Interface().(func(float64) float64)
			return ufunc(call.Args[0], f)
		})
	}
	if ds.fitS != nil {
		fitS, fitA := reflect.ValueOf(ds.fitS), reflect.ValueOf(ds.fitA)
		c.DefineSingletonMethod("fit", func(call *rb.Call) (any, error) {
			if err := arity(call, 1, 1); err != nil {
				return nil, err
			}
			o, err := fitOptions(fields, call.Kwargs)
			if err != nil {
				return nil, err
			}
			x, err := data(call.Args[0].Go())
			if err != nil {
				return nil, err
			}
			fit := fitS
			if _, ok := x.(*nd.Array); ok {
				fit = fitA
			}
			return results(fit.Call([]reflect.Value{reflect.ValueOf(x), reflect.ValueOf(o)}))
		})
	}
	skip := map[string]bool{"Validate": true}
	for name := range elementwise {
		skip[name] = true
	}
	_ = bindMethods(c, reflect.PointerTo(t), skip)
}

// newDist builds a distribution value from Ruby arguments and validates it,
// which is what go-statistics' NewXxx does.
func newDist(t reflect.Type, fields []reflect.StructField, args []rb.Value, kw map[string]rb.Value) (reflect.Value, error) {
	if len(args) > len(fields) {
		return reflect.Value{}, argError("wrong number of arguments (given %d, expected 0..%d)", len(args), len(fields))
	}
	v := reflect.New(t).Elem()
	set := map[string]bool{}
	assign := func(f reflect.StructField, x rb.Value) error {
		if set[f.Name] {
			return argError("%s given twice", snakeName(f.Name))
		}
		fv, err := convert(x.Go(), f.Type)
		if err != nil {
			return err
		}
		v.FieldByIndex(f.Index).Set(fv)
		set[f.Name] = true
		return nil
	}
	for i, a := range args {
		if err := assign(fields[i], a); err != nil {
			return reflect.Value{}, err
		}
	}
	byName := fieldsByName(t)
	for _, k := range sortedKeys(kw) {
		f, ok := byName[normalize(k)]
		if !ok {
			return reflect.Value{}, argError("unknown keyword: :%s (known: %s)", k, joinNames(fields))
		}
		if err := assign(f, kw[k]); err != nil {
			return reflect.Value{}, err
		}
	}
	for _, f := range fields {
		if set[f.Name] {
			continue
		}
		switch f.Name {
		case "Loc":
		case "Scale":
			v.FieldByIndex(f.Index).SetFloat(1)
		default:
			return reflect.Value{}, argError("missing keyword: :%s", snakeName(f.Name))
		}
	}
	if err := v.Interface().(interface{ Validate() error }).Validate(); err != nil {
		return reflect.Value{}, goError(err)
	}
	return v, nil
}

func joinNames(fields []reflect.StructField) string {
	s := ""
	for i, f := range fields {
		if i > 0 {
			s += ", "
		}
		s += snakeName(f.Name)
	}
	return s
}

// fitOptions reads fit's keyword: fixed: {loc: 0} holds the named
// parameters at the given values (SciPy's floc=0), FitOptions.Fixed by Go
// field name.
func fitOptions(fields []reflect.StructField, kw map[string]rb.Value) (*stats.FitOptions, error) {
	for _, k := range sortedKeys(kw) {
		if k != "fixed" {
			return nil, argError("unknown keyword: :%s (known: fixed)", k)
		}
	}
	fixed, ok := kw["fixed"]
	if !ok {
		return nil, nil
	}
	h, isHash := fixed.Go().(map[any]any)
	if !isHash {
		return nil, typeError(fixed.Go(), "Hash")
	}
	byName := map[string]string{}
	for _, f := range fields {
		byName[normalize(f.Name)] = f.Name
	}
	o := &stats.FitOptions{Fixed: map[string]float64{}}
	for k, v := range h {
		name, err := toString(k)
		if err != nil {
			return nil, err
		}
		goName, ok := byName[normalize(name)]
		if !ok {
			return nil, argError("unknown parameter %s (known: %s)", name, joinNames(fields))
		}
		if o.Fixed[goName], err = toFloat(v); err != nil {
			return nil, err
		}
	}
	return o, nil
}

// defineRNG defines Stats::RNG, numpy.random.default_rng: the generator
// random_state takes, so a sequence of draws can be reproduced.
func defineRNG(d *definer, m *rb.Module) {
	c := d.class(m, "RNG")
	if c == nil {
		return
	}
	c.Bind(&random.Generator{})
	c.DefineSingletonMethod("new", func(call *rb.Call) (any, error) {
		if err := arity(call, 0, 1); err != nil {
			return nil, err
		}
		if len(call.Args) == 0 {
			return random.DefaultRNG(), nil
		}
		seed, err := toInt(call.Args[0].Go())
		if err != nil || seed < 0 {
			return nil, argError("seed must be a non-negative Integer")
		}
		return random.DefaultRNG(uint64(seed)), nil
	})
	for _, name := range []string{"Random", "StandardNormal", "Normal", "Uniform", "Integers", "Exponential", "StandardExponential", "Poisson", "Binomial", "Gamma", "Beta"} {
		mt, ok := reflect.TypeFor[*random.Generator]().MethodByName(name)
		if !ok {
			panic(fmt.Sprintf("random.Generator has no %s", name))
		}
		c.DefineMethod(snakeName(name), func(call *rb.Call) (any, error) {
			return invoke(reflect.ValueOf(call.Self.Go()).Method(mt.Index), call, call.Args)
		})
	}
	c.DefineMethod("inspect", func(*rb.Call) (any, error) { return "#<Stats::RNG>", nil })
}

// defineKDE defines Stats::GaussianKDE.
func defineKDE(d *definer, m *rb.Module) {
	c := d.class(m, "GaussianKDE")
	if c == nil {
		return
	}
	c.Bind(&stats.GaussianKDE{})
	newKDE := dataFunc(stats.NewGaussianKDE[[]float64], stats.NewGaussianKDE[*nd.Array])
	c.DefineSingletonMethod("new", newKDE)
	bindMethods(c, reflect.TypeFor[*stats.GaussianKDE](), map[string]bool{"Evaluate": true, "EvaluateSlice": true})
	// evaluate (and call) take points as an Array or an NDArray; SciPy's
	// kde(points).
	eval := func(call *rb.Call) (any, error) {
		k := call.Self.Go().(*stats.GaussianKDE)
		if err := arity(call, 1, 1); err != nil {
			return nil, err
		}
		x, err := ndArray(call.Args[0].Go())
		if err != nil {
			return nil, err
		}
		if k.D() == 1 {
			r, err := k.EvaluateSlice(arrayFloats(x))
			return resultOrError(vec(r), err)
		}
		r, err := k.Evaluate(x)
		return resultOrError(vec(r), err)
	}
	c.DefineMethod("evaluate", eval)
	c.DefineMethod("call", eval)
	c.DefineMethod("inspect", func(call *rb.Call) (any, error) {
		k := call.Self.Go().(*stats.GaussianKDE)
		return fmt.Sprintf("#<Stats::GaussianKDE d=%d n=%d>", k.D(), k.Dataset().Shape()[len(k.Dataset().Shape())-1]), nil
	})
}

func resultOrError(v any, err error) (any, error) {
	if err != nil {
		return nil, goError(err)
	}
	return v, nil
}

// defineMultivariate defines Stats::MultivariateNormal, Stats::Dirichlet and
// Stats::Multinomial: new is the Go constructor, and the methods are the Go
// methods.
func defineMultivariate(d *definer, m *rb.Module) {
	for _, mv := range []struct {
		name string
		ctor any
		zero any
	}{
		{"MultivariateNormal", stats.NewMultivariateNormal, &stats.MultivariateNormal{}},
		{"Dirichlet", stats.NewDirichlet, &stats.Dirichlet{}},
		{"Multinomial", stats.NewMultinomial, &stats.Multinomial{}},
	} {
		c := d.class(m, mv.name)
		if c == nil {
			return
		}
		c.Bind(mv.zero)
		ctor := reflect.ValueOf(mv.ctor)
		c.DefineSingletonMethod("new", func(call *rb.Call) (any, error) { return invoke(ctor, call, call.Args) })
		bindMethods(c, reflect.TypeOf(mv.zero), nil)
		name := mv.name
		c.DefineMethod("inspect", func(*rb.Call) (any, error) { return "#<Stats::" + name + ">", nil })
	}
}

// bootstrap is Stats.bootstrap(samples, **options) { |*samples| statistic }:
// scipy.stats.bootstrap. The block is the statistic; it is called with one
// NDArray per sample and returns a Float.
func bootstrap(c *rb.Call) (any, error) {
	if err := arity(c, 1, 1); err != nil {
		return nil, err
	}
	b, err := needBlock(c)
	if err != nil {
		return nil, err
	}
	list, ok := c.Args[0].Go().([]any)
	if !ok {
		return nil, typeError(c.Args[0].Go(), "an Array of samples")
	}
	samples := make([][]float64, len(list))
	for i, s := range list {
		if samples[i], err = toFloats(s); err != nil {
			return nil, err
		}
	}
	kw, rest := split(c.Kwargs, "random_state")
	g, err := generator(kw["random_state"])
	if err != nil {
		return nil, err
	}
	var opts *stats.BootstrapOptions
	if len(rest) > 0 {
		opts = &stats.BootstrapOptions{}
		if err := options(opts, rest); err != nil {
			return nil, err
		}
	}
	var blockErr error
	statistic := func(s ...[]float64) float64 {
		if blockErr != nil {
			return 0
		}
		args := make([]any, len(s))
		for i, x := range s {
			args[i] = vec(x)
		}
		r, err := b.Call(args...)
		if err == nil {
			var f float64
			if f, err = toFloat(r.Go()); err == nil {
				return f
			}
		}
		blockErr = err
		return 0
	}
	res, err := stats.Bootstrap(samples, statistic, g, opts)
	if blockErr != nil {
		return nil, blockErr
	}
	return resultOrError(res, err)
}
