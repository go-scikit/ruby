package ruby

import (
	"fmt"
	"reflect"
	"sort"
	"strings"
	"testing"

	rb "github.com/go-embedded-ruby/ruby"
	nd "github.com/go-ndarray/ndarray"
	"github.com/go-ndarray/ndarray/random"
	"github.com/go-statistics/stats"
)

var (
	sampleX = []float64{2.1, 3.4, 1.9, 5.6, 4.4, 3.3, 2.8, 4.9, 3.7, 2.2}
	sampleY = []float64{1.2, 2.9, 2.2, 4.8, 4.1, 2.5, 3.1, 3.9, 3.0, 1.8}
	sampleZ = []float64{3.1, 4.2, 3.9, 5.0, 4.8, 4.4, 3.6, 5.2, 4.1, 3.3}
)

// Markers for the data arguments of a statsFuncs case.
type dataX struct{}
type dataY struct{}
type dataZ struct{}

// statsCases call each function of statsFuncs: rubyArgs is what follows the
// data in Ruby, goArgs the Go arguments with the data as markers, opts the Go
// option struct the Ruby keywords stand for (nil: none).
var statsCases = []struct {
	name, ruby string
	goArgs     []any
}{
	{"mean", "X", []any{dataX{}, (*stats.MeanOptions)(nil)}},
	{"mean", "X, weights: [1, 2, 3, 4, 5, 6, 7, 8, 9, 10], ddof: 1, omit_nan: true", []any{dataX{}, &stats.MeanOptions{Weights: []float64{1, 2, 3, 4, 5, 6, 7, 8, 9, 10}, Ddof: 1, OmitNaN: true}}},
	{"average", "X, weights: [1, 2, 3, 4, 5, 6, 7, 8, 9, 10]", []any{dataX{}, &stats.MeanOptions{Weights: []float64{1, 2, 3, 4, 5, 6, 7, 8, 9, 10}}}},
	{"var", "X, ddof: 1", []any{dataX{}, &stats.MeanOptions{Ddof: 1}}},
	{"std", "X", []any{dataX{}, (*stats.MeanOptions)(nil)}},
	{"median", "X", []any{dataX{}, (*stats.MedianOptions)(nil)}},
	{"gmean", "X", []any{dataX{}, (*stats.PowerMeanOptions)(nil)}},
	{"hmean", "X, weights: [1, 1, 1, 1, 1, 2, 2, 2, 2, 2]", []any{dataX{}, &stats.PowerMeanOptions{Weights: []float64{1, 1, 1, 1, 1, 2, 2, 2, 2, 2}}}},
	{"pmean", "X, 3", []any{dataX{}, 3.0, (*stats.PowerMeanOptions)(nil)}},
	{"skew", "X, unbiased: true", []any{dataX{}, &stats.ShapeOptions{Unbiased: true}}},
	{"kurtosis", "X, pearson: true", []any{dataX{}, &stats.ShapeOptions{Pearson: true}}},
	{"moment", "X, 3, center: 0.5", []any{dataX{}, 3, &stats.MomentOptions{Center: ptr(0.5)}}},
	{"sem", "X, ddof: 0.0", []any{dataX{}, &stats.DdofOptions{Ddof: ptr(0.0)}}},
	{"variation", "X", []any{dataX{}, (*stats.DdofOptions)(nil)}},
	{"iqr", "X, range: [10, 90], method: :nearest", []any{dataX{}, &stats.IQROptions{Range: [2]float64{10, 90}, Method: stats.QuantileNearest}}},
	{"median_abs_deviation", "X, scale: 1.4826", []any{dataX{}, &stats.MADOptions{Scale: 1.4826}}},
	{"quantile", "X, 0.3", []any{dataX{}, 0.3, (*stats.QuantileOptions)(nil)}},
	{"quantiles", "X, [0.1, 0.5, 0.9], method: :weibull", []any{dataX{}, []float64{0.1, 0.5, 0.9}, &stats.QuantileOptions{Method: stats.QuantileWeibull}}},
	{"percentile", "X, 75", []any{dataX{}, 75.0, (*stats.QuantileOptions)(nil)}},
	{"percentiles", "X, [25, 75]", []any{dataX{}, []float64{25, 75}, (*stats.QuantileOptions)(nil)}},
	{"tmean", "X, limits: {lower: 2, upper: 5}", []any{dataX{}, &stats.TrimmedOptions{Limits: &stats.Limits{Lower: 2, Upper: 5}}}},
	{"tvar", "X", []any{dataX{}, (*stats.TrimmedOptions)(nil)}},
	{"tstd", "X", []any{dataX{}, (*stats.TrimmedOptions)(nil)}},
	{"tsem", "X", []any{dataX{}, (*stats.TrimmedOptions)(nil)}},
	{"tmin", "X, 2.5, exclusive: true", []any{dataX{}, 2.5, &stats.BoundOptions{Exclusive: true}}},
	{"tmax", "X, 4.5", []any{dataX{}, 4.5, (*stats.BoundOptions)(nil)}},
	{"trim_mean", "X, 0.1", []any{dataX{}, 0.1}},
	{"trimboth", "X, 0.1", []any{dataX{}, 0.1}},
	{"trim1", "X, 0.2, :left", []any{dataX{}, 0.2, stats.TailLeft}},
	{"zscore", "X", []any{dataX{}, (*stats.DdofOptions)(nil)}},
	{"rankdata", "X, :min", []any{dataX{}, stats.RankMin}},
	{"describe", "X", []any{dataX{}, (*stats.ShapeOptions)(nil)}},
	{"mode", "[1, 2, 2, 3]", []any{[]float64{1, 2, 2, 3}, (*stats.NaNOptions)(nil)}},
	{"entropy", "[0.2, 0.3, 0.5], base: 2", []any{[]float64{0.2, 0.3, 0.5}, &stats.EntropyOptions{Base: 2}}},
	{"differential_entropy", "X", []any{dataX{}, (*stats.DiffEntropyOptions)(nil)}},
	{"histogram", "X, bins: 4, range: [1, 6], density: true", []any{dataX{}, &stats.HistogramOptions{Bins: 4, Range: &[2]float64{1, 6}, Density: true}}},
	{"histogram_bin_edges", "X, rule: :sturges", []any{dataX{}, &stats.HistogramOptions{Rule: stats.BinsSturges}}},
	{"cumfreq", "X, num_bins: 4", []any{dataX{}, &stats.FreqOptions{NumBins: 4}}},
	{"relfreq", "X", []any{dataX{}, (*stats.FreqOptions)(nil)}},
	{"zmap", "X, Y", []any{dataX{}, dataY{}, (*stats.DdofOptions)(nil)}},
	{"binned_statistic", "X, Y, statistic: :median, bins: 3", []any{dataX{}, dataY{}, &stats.BinnedOptions{Statistic: stats.BinMedian, Bins: 3}}},
	{"ttest_1samp", "X, 3.0, alternative: :greater", []any{dataX{}, 3.0, &stats.TTestOptions{Alternative: stats.Greater}}},
	{"ttest_ind", "X, Y, welch: true, trim: 0.1", []any{dataX{}, dataY{}, &stats.TTestIndOptions{Welch: true, Trim: 0.1}}},
	{"ttest_rel", "X, Y", []any{dataX{}, dataY{}, (*stats.TTestOptions)(nil)}},
	{"mannwhitneyu", "X, Y, method: :asymptotic", []any{dataX{}, dataY{}, &stats.MannWhitneyUOptions{Method: stats.MethodAsymptotic}}},
	{"wilcoxon", "X, Y, alternative: :less", []any{dataX{}, dataY{}, &stats.WilcoxonOptions{Alternative: stats.Less}}},
	{"ks_2samp", "X, Y", []any{dataX{}, dataY{}, (*stats.KSOptions)(nil)}},
	{"shapiro", "X", []any{dataX{}}},
	{"normaltest", "X + Y", []any{append(append([]float64{}, sampleX...), sampleY...)}},
	{"skewtest", "X + Y", []any{append(append([]float64{}, sampleX...), sampleY...), (*stats.SkewTestOptions)(nil)}},
	{"kurtosistest", "X + Y + Z", []any{append(append(append([]float64{}, sampleX...), sampleY...), sampleZ...), (*stats.SkewTestOptions)(nil)}},
	{"jarque_bera", "X", []any{dataX{}}},
	{"anderson", "X, :expon", []any{dataX{}, stats.ADExpon}},
	{"pearsonr", "X, Y", []any{dataX{}, dataY{}, (*stats.CorrelationOptions)(nil)}},
	{"spearmanr", "X, Y, alternative: :greater", []any{dataX{}, dataY{}, &stats.CorrelationOptions{Alternative: stats.Greater}}},
	{"kendalltau", "X, Y, variant: :c", []any{dataX{}, dataY{}, &stats.KendallOptions{Variant: stats.TauC}}},
	{"linregress", "X, Y", []any{dataX{}, dataY{}, (*stats.CorrelationOptions)(nil)}},
	{"chisquare", "[10, 20, 30], expected: [20, 20, 20]", []any{[]float64{10, 20, 30}, &stats.ChiSquareOptions{Expected: []float64{20, 20, 20}}}},
	{"power_divergence", "[10, 20, 30], 0.5", []any{[]float64{10, 20, 30}, 0.5, (*stats.ChiSquareOptions)(nil)}},
	{"f_oneway", "X, Y, Z, welch: true", []any{&stats.FOnewayOptions{Welch: true}, dataX{}, dataY{}, dataZ{}}},
	{"kruskal", "X, Y, Z", []any{dataX{}, dataY{}, dataZ{}}},
	{"levene", "X, Y, Z, center: :mean", []any{&stats.LeveneOptions{Center: stats.CenterMean}, dataX{}, dataY{}, dataZ{}}},
	{"bartlett", "X, Y", []any{dataX{}, dataY{}}},
}

func ptr[T any](v T) *T { return &v }

// goData replaces the data markers with the samples, as slices or arrays.
func goData(args []any, arrays bool) []reflect.Value {
	out := make([]reflect.Value, len(args))
	for i, a := range args {
		var s []float64
		switch a.(type) {
		case dataX:
			s = sampleX
		case dataY:
			s = sampleY
		case dataZ:
			s = sampleZ
		default:
			if f, ok := a.([]float64); ok && arrays && i == 0 {
				s = f
			} else {
				out[i] = reflect.ValueOf(a)
				continue
			}
		}
		if arrays {
			out[i] = reflect.ValueOf(vec(s))
		} else {
			out[i] = reflect.ValueOf(append([]float64(nil), s...))
		}
	}
	return out
}

// Every data function of Stats, on Arrays and on NDArrays, gives the go-statistics
// function's result for the same inputs, bit for bit.
func TestStatsFunctionsMatchGo(t *testing.T) {
	in := newInterp(t)
	byName := map[string]int{}
	for i, f := range statsFuncs {
		byName[f.name] = i
	}
	covered := map[string]bool{}
	for _, arrays := range []bool{false, true} {
		for _, c := range statsCases {
			f := statsFuncs[byName[c.name]]
			fn := reflect.ValueOf(f.overS)
			if arrays {
				fn = reflect.ValueOf(f.overA)
			}
			args := goData(c.goArgs, arrays)
			for i := range args {
				if !args[i].IsValid() || (args[i].Kind() == reflect.Pointer && args[i].IsNil()) {
					args[i] = reflect.Zero(fn.Type().In(min(i, fn.Type().NumIn()-1)))
				}
			}
			res := fn.Call(args)
			if err, _ := res[len(res)-1].Interface().(error); err != nil {
				t.Fatalf("%s (Go): %v", c.name, err)
			}
			want := res[0].Interface()
			if len(res) == 3 {
				want = []any{res[0].Interface(), res[1].Interface()}
			}
			src := "Stats." + c.name + "(" + c.ruby + ")"
			if arrays {
				src = strings.NewReplacer("X", "XA", "Y", "YA", "Z", "ZA").Replace(src)
				src = strings.ReplaceAll(src, "XA + YA + ZA", "NDArray.concatenate([XA, YA, ZA])")
				src = strings.ReplaceAll(src, "XA + YA", "NDArray.concatenate([XA, YA])")
			}
			setSamples(t, in)
			check(t, in, src, want)
			covered[c.name] = true
		}
	}
	for _, f := range statsFuncs {
		if !covered[f.name] && f.name != "ks_1samp" {
			t.Errorf("Stats.%s has no comparison with Go", f.name)
		}
	}
}

// setSamples defines the samples as Ruby constants: X, Y, Z as Arrays and
// XA, YA, ZA as NDArrays.
func setSamples(t *testing.T, in *rb.Interp) {
	t.Helper()
	for name, s := range map[string][]float64{"X": sampleX, "Y": sampleY, "Z": sampleZ} {
		lit := fmt.Sprint(s)
		lit = strings.ReplaceAll(lit[1:len(lit)-1], " ", ", ")
		src := fmt.Sprintf("%s = [%s].freeze unless defined?(%s)\n%sA = NDArray.from(%s, [%d]) unless defined?(%sA)", name, lit, name, name, name, len(s), name)
		if _, err := in.Eval(src, "samples.rb", 1); err != nil {
			t.Fatal(err)
		}
	}
}

// distParams are valid parameters for each distribution: the Ruby keywords
// and the Go field values (loc and scale shifted off their defaults, so the
// defaults are tested by the "default" case below and not hidden).
var distParams = map[string]map[string]any{
	"Beta":                 {"A": 2.5, "B": 3.5, "Loc": 0.1, "Scale": 2.0},
	"Cauchy":               {"Loc": 0.5, "Scale": 1.5},
	"Chi":                  {"Df": 3.0, "Loc": 0.1, "Scale": 1.2},
	"ChiSquared":           {"Df": 4.0, "Loc": 0.1, "Scale": 1.2},
	"Exponential":          {"Loc": 0.2, "Scale": 1.7},
	"F":                    {"Dfn": 5.0, "Dfd": 7.0, "Loc": 0.0, "Scale": 1.0},
	"Gamma":                {"A": 2.2, "Loc": 0.1, "Scale": 1.3},
	"Gumbel":               {"Loc": 0.4, "Scale": 1.1},
	"HalfNormal":           {"Loc": 0.1, "Scale": 1.4},
	"InverseGamma":         {"A": 3.1, "Loc": 0.0, "Scale": 1.2},
	"Laplace":              {"Loc": 0.3, "Scale": 0.8},
	"LogNormal":            {"S": 0.6, "Loc": 0.0, "Scale": 1.5},
	"Logistic":             {"Loc": 0.2, "Scale": 0.9},
	"Lomax":                {"C": 2.5, "Loc": 0.0, "Scale": 1.1},
	"NoncentralChiSquared": {"Df": 3.0, "Nc": 1.5, "Loc": 0.0, "Scale": 1.0},
	"Normal":               {"Loc": 0.5, "Scale": 2.0},
	"Pareto":               {"B": 2.7, "Loc": 0.0, "Scale": 1.0},
	"Rayleigh":             {"Loc": 0.1, "Scale": 1.3},
	"Reciprocal":           {"A": 0.5, "B": 4.0, "Loc": 0.0, "Scale": 1.0},
	"StudentT":             {"Df": 6.0, "Loc": 0.2, "Scale": 1.1},
	"Triangular":           {"C": 0.3, "Loc": 0.5, "Scale": 3.0},
	"Uniform":              {"Loc": 1.0, "Scale": 4.0},
	"VonMises":             {"Kappa": 2.0, "Loc": 0.1, "Scale": 1.0},
	"Weibull":              {"C": 1.8, "Loc": 0.0, "Scale": 1.4},
	"Bernoulli":            {"P": 0.3, "Loc": 0.0},
	"Binomial":             {"N": int64(12), "P": 0.4, "Loc": 1.0},
	"DiscreteUniform":      {"Low": int64(2), "High": int64(9), "Loc": 0.0},
	"Geometric":            {"P": 0.35, "Loc": 0.0},
	"Hypergeometric":       {"Total": int64(20), "Good": int64(7), "Draws": int64(12), "Loc": 0.0},
	"NegativeBinomial":     {"N": 3.0, "P": 0.4, "Loc": 0.0},
	"Poisson":              {"Mu": 3.3, "Loc": 1.0},
	"Zipf":                 {"A": 2.5, "Loc": 0.0},
}

// rubyKw spells Go field values as Ruby keywords.
func rubyKw(params map[string]any) string {
	var parts []string
	for _, k := range sortedFields(params) {
		parts = append(parts, fmt.Sprintf("%s: %v", snakeName(k), params[k]))
	}
	return strings.Join(parts, ", ")
}

func sortedFields(params map[string]any) []string {
	keys := make([]string, 0, len(params))
	for k := range params {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

// goDist is the Go distribution value with the parameters set directly.
func goDist(zero any, params map[string]any) reflect.Value {
	v := reflect.New(reflect.TypeOf(zero)).Elem()
	for k, p := range params {
		v.FieldByName(k).Set(reflect.ValueOf(p))
	}
	return v
}

// Every distribution method called from Ruby is the Go method on the same
// distribution, bit for bit, for a Float, an Array and an NDArray argument.
func TestDistributionsMatchGo(t *testing.T) {
	in := newInterp(t)
	points := []float64{-0.5, 0, 0.3, 1, 2.5, 4}
	probs := []float64{0.01, 0.25, 0.5, 0.9}
	for _, ds := range distributions {
		params, ok := distParams[ds.name]
		if !ok {
			t.Fatalf("no parameters for %s", ds.name)
		}
		g := goDist(ds.zero, params)
		ctor := fmt.Sprintf("Stats::%s.new(%s)", ds.name, rubyKw(params))
		eval(t, in, "d = "+ctor)
		if got := eval(t, in, "d"); !same(got, g.Interface()) {
			t.Fatalf("%s: built %#v, Go %#v", ctor, got, g.Interface())
		}
		call := func(name string, args ...any) any {
			out := g.MethodByName(name).Call(valuesOf(args))
			if len(out) == 2 {
				return []any{out[0].Interface(), out[1].Interface()}
			}
			return out[0].Interface()
		}
		for name := range elementwise {
			if !g.MethodByName(name).IsValid() {
				continue
			}
			xs := points
			if name == "PPF" || name == "ISF" {
				xs = probs
			}
			f := g.MethodByName(name).Interface().(func(float64) float64)
			want := make([]float64, len(xs))
			for i, x := range xs {
				want[i] = f(x)
				check(t, in, fmt.Sprintf("d.%s(%v)", snakeName(name), x), want[i])
			}
			arr := fmt.Sprintf("%v", xs)
			arr = strings.ReplaceAll(arr[1:len(arr)-1], " ", ", ")
			check(t, in, fmt.Sprintf("d.%s([%s])", snakeName(name), arr), want)
			check(t, in, fmt.Sprintf("d.%s(NDArray.from([%s], [2, %d]))", snakeName(name), arr, len(xs)/2), vecShape(want, 2, len(xs)/2))
		}
		for _, name := range []string{"Mean", "Var", "Std", "Skew", "Kurtosis", "Median", "Entropy", "Support"} {
			check(t, in, "d."+snakeName(name), call(name))
		}
		check(t, in, "d.moment(2)", call("Moment", 2))
		check(t, in, "d.interval(0.9)", call("Interval", 0.9))
		rvs := g.MethodByName("Rvs").Call([]reflect.Value{reflect.ValueOf(random.DefaultRNG(7)), reflect.ValueOf(2), reflect.ValueOf(3)})
		check(t, in, "d.rvs(2, 3, random_state: 7)", rvs[0].Interface())
		check(t, in, "d.rvs([2, 3], random_state: Stats::RNG.new(7))", rvs[0].Interface())
		if ds.fitS != nil {
			sample := "[0.4, 1.3, 2.2, 0.9, 1.7, 3.1, 0.6, 1.1, 2.5, 1.9]"
			x := []float64{0.4, 1.3, 2.2, 0.9, 1.7, 3.1, 0.6, 1.1, 2.5, 1.9}
			fit := reflect.ValueOf(ds.fitS).Call([]reflect.Value{reflect.ValueOf(x), reflect.ValueOf((*stats.FitOptions)(nil))})
			if err, _ := fit[1].Interface().(error); err == nil {
				check(t, in, fmt.Sprintf("Stats::%s.fit(%s)", ds.name, sample), fit[0].Interface())
				fitA := reflect.ValueOf(ds.fitA).Call([]reflect.Value{reflect.ValueOf(vec(x)), reflect.ValueOf((*stats.FitOptions)(nil))})
				check(t, in, fmt.Sprintf("Stats::%s.fit(NDArray.from(%s, [10]))", ds.name, sample), fitA[0].Interface())
			}
			fixed := reflect.ValueOf(ds.fitS).Call([]reflect.Value{reflect.ValueOf(x), reflect.ValueOf(&stats.FitOptions{Fixed: map[string]float64{"Loc": 0}})})
			if err, _ := fixed[1].Interface().(error); err == nil {
				check(t, in, fmt.Sprintf("Stats::%s.fit(%s, fixed: {loc: 0})", ds.name, sample), fixed[0].Interface())
			}
		}
	}
}

func valuesOf(args []any) []reflect.Value {
	out := make([]reflect.Value, len(args))
	for i, a := range args {
		out[i] = reflect.ValueOf(a)
	}
	return out
}

func vecShape(x []float64, shape ...int) *nd.Array {
	a, _ := nd.FromData(append([]float64(nil), x...), shape...)
	return a
}
