package ruby

import (
	"math"
	"reflect"

	rb "github.com/go-embedded-ruby/ruby"
	"github.com/go-numerics/optimize"
)

// registerOptimize defines Optimize, go-numerics/optimize with the objective
// as the block:
//
//	Optimize.minimize(x0, method: :bfgs, jac: ->(x) { ... }, **options) { |x| f(x) }
//	Optimize.minimize_scalar(method: :brent, bracket: [a, b]) { |x| f(x) }
//	Optimize.brentq(a, b) { |x| f(x) }   # also bisect, brenth, ridder
//	Optimize.newton(x0, fprime: ->(x) { ... }) { |x| f(x) }
//	Optimize.root_scalar(method: "brentq", bracket: [a, b]) { |x| f(x) }
//	Optimize.root(x0) { |x| [f1(x), f2(x)] }      # also fsolve
//	Optimize.least_squares(x0, m: 3) { |x| residuals }   # also leastsq
//	Optimize.curve_fit(xdata, ydata, p0: [1, 1]) { |x, a, b| model }
//	Optimize.linprog(c, a_ub: ..., b_ub: ..., bounds: [[0, nil], ...])
//	Optimize.brute([[lo, hi], ...], ns: 20) { |x| f(x) }
//	Optimize.fminbound(x1, x2) { |x| f(x) }
//	Optimize.bracket(xa, xb) { |x| f(x) }
//	Optimize.approx_fprime(x, epsilon) { |x| f(x) }
//
// Vectors reach the block as NDArrays; it returns a Float, or an Array or an
// NDArray for a vector. The keywords are the fields of the Go option struct
// (max_iter:, gtol:, bounds:, …). A bound is [lo, hi], nil for a missing side.
func registerOptimize(in *rb.Interp) error {
	d := &definer{in: in}
	m := d.module("Optimize")
	if d.err != nil {
		return d.err
	}
	for _, r := range []struct {
		name   string
		sample any
	}{
		{"Result", optimize.Result{}},
		{"ScalarResult", optimize.ScalarResult{}},
		{"RootScalarResult", optimize.RootScalarResult{}},
		{"RootResult", optimize.RootResult{}},
		{"CurveFitResult", optimize.CurveFitResult{}},
		{"LSResult", optimize.LSResult{}},
		{"LeastsqResult", optimize.LeastsqResult{}},
		{"LPResult", optimize.LPResult{}},
		{"LPDual", optimize.LPDual{}},
		{"BruteResult", optimize.BruteResult{}},
	} {
		d.structClass(m, r.name, r.sample)
	}
	d.fn(m, "minimize", minimize)
	d.fn(m, "minimize_scalar", minimizeScalar)
	for name, f := range map[string]func(func(float64) float64, float64, float64, *optimize.BracketOptions) (*optimize.RootScalarResult, error){
		"brentq": optimize.Brentq, "brenth": optimize.Brenth, "bisect": optimize.Bisect, "ridder": optimize.Ridder,
	} {
		d.fn(m, name, bracketRoot(f))
	}
	d.fn(m, "newton", newton)
	d.fn(m, "root_scalar", rootScalar)
	d.fn(m, "root", systemRoot(optimize.Root))
	d.fn(m, "fsolve", systemRoot(optimize.Fsolve))
	d.fn(m, "least_squares", leastSquares)
	d.fn(m, "leastsq", leastsq)
	d.fn(m, "curve_fit", curveFit)
	d.fn(m, "linprog", linprog)
	d.fn(m, "brute", brute)
	d.fn(m, "fminbound", fminbound)
	d.fn(m, "bracket", bracketFn)
	d.fn(m, "approx_fprime", approxFprime)
	return d.err
}

func init() {
	converters[reflect.TypeFor[optimize.Bound]()] = func(v any) (reflect.Value, error) {
		b, err := toBound(v)
		return reflect.ValueOf(b), err
	}
	converters[reflect.TypeFor[*optimize.Bound]()] = func(v any) (reflect.Value, error) {
		b, err := toBound(v)
		return reflect.ValueOf(&b), err
	}
	converters[reflect.TypeFor[optimize.Range]()] = func(v any) (reflect.Value, error) {
		r, err := toRange(v)
		return reflect.ValueOf(r), err
	}
}

// toBound reads [lo, hi], nil for a missing side (±Infinity).
func toBound(v any) (optimize.Bound, error) {
	pair, ok := v.([]any)
	if !ok || len(pair) != 2 {
		return optimize.Bound{}, typeError(v, "a bound [lo, hi]")
	}
	side := func(x any, inf float64) (float64, error) {
		if x == nil {
			return inf, nil
		}
		return toFloat(x)
	}
	lo, err := side(pair[0], math.Inf(-1))
	if err != nil {
		return optimize.Bound{}, err
	}
	hi, err := side(pair[1], math.Inf(1))
	return optimize.Bound{Lo: lo, Hi: hi}, err
}

// toRange reads a brute-force axis: [lo, hi] (ns points) or [lo, hi, step].
func toRange(v any) (optimize.Range, error) {
	list, ok := v.([]any)
	if !ok || len(list) < 2 || len(list) > 3 {
		return optimize.Range{}, typeError(v, "a range [lo, hi] or [lo, hi, step]")
	}
	f, err := toFloats(list)
	if err != nil {
		return optimize.Range{}, err
	}
	r := optimize.Range{Lo: f[0], Hi: f[1]}
	if len(f) == 3 {
		r.Step = f[2]
	}
	return r, nil
}

// errWorkers refuses workers: above 1: a Ruby block runs on one goroutine
// at a time, and the Go side would call it from several at once.
var errWorkers = rb.NewError("ArgumentError", "workers: must be 1: a Ruby block cannot be called from several goroutines at once")

// optsFrom fills an option struct from keywords, nil when there are none.
func optsFrom[O any](kw map[string]rb.Value) (*O, error) {
	if len(kw) == 0 {
		return nil, nil
	}
	o := new(O)
	if err := options(o, kw); err != nil {
		return nil, err
	}
	return o, nil
}

// finish returns a result, or the error of the callback first, then the Go
// error.
func finish(g *guard, res any, err error) (any, error) {
	if err != nil {
		return nil, goError(err)
	}
	return res, nil
}

// blockAndArgs checks the positional count and returns the block's caller.
func blockAndArgs(c *rb.Call, n int) (caller, error) {
	if err := arity(c, n, n); err != nil {
		return nil, err
	}
	b, err := needBlock(c)
	if err != nil {
		return nil, err
	}
	return blockCaller(b), nil
}

func floatArgs(c *rb.Call) ([]float64, error) {
	out := make([]float64, len(c.Args))
	for i, a := range c.Args {
		f, err := toFloat(a.Go())
		if err != nil {
			return nil, err
		}
		out[i] = f
	}
	return out, nil
}

// enumKw reads an enumerated keyword, def when it is absent.
func enumKw[E any](kw map[string]rb.Value, name string, def E) (E, error) {
	v, ok := kw[name]
	if !ok {
		return def, nil
	}
	ev, err := convert(v.Go(), reflect.TypeFor[E]())
	if err != nil {
		return def, err
	}
	return ev.Interface().(E), nil
}

func minimize(c *rb.Call) (any, error) {
	call, err := blockAndArgs(c, 1)
	if err != nil {
		return nil, err
	}
	x0, err := toFloats(c.Args[0].Go())
	if err != nil {
		return nil, err
	}
	mine, rest := split(c.Kwargs, "method", "jac")
	method, err := enumKw(mine, "method", optimize.BFGS)
	if err != nil {
		return nil, err
	}
	g := &guard{}
	p := optimize.Problem{Func: fn[func([]float64) float64](g, call)}
	if p.Grad, _, err = optionalProc[func(x, grad []float64)](g, mine, "jac"); err != nil {
		return nil, err
	}
	o, err := optsFrom[optimize.Options](rest)
	if err != nil {
		return nil, err
	}
	res, err := optimize.Minimize(p, x0, method, o)
	return finish(g, res, err)
}

func minimizeScalar(c *rb.Call) (any, error) {
	call, err := blockAndArgs(c, 0)
	if err != nil {
		return nil, err
	}
	mine, rest := split(c.Kwargs, "method")
	method, err := enumKw(mine, "method", optimize.Brent)
	if err != nil {
		return nil, err
	}
	o, err := optsFrom[optimize.ScalarOptions](rest)
	if err != nil {
		return nil, err
	}
	g := &guard{}
	res, err := optimize.MinimizeScalar(fn[func(float64) float64](g, call), method, o)
	return finish(g, res, err)
}

func bracketRoot(f func(func(float64) float64, float64, float64, *optimize.BracketOptions) (*optimize.RootScalarResult, error)) rb.Func {
	return func(c *rb.Call) (any, error) {
		call, err := blockAndArgs(c, 2)
		if err != nil {
			return nil, err
		}
		ab, err := floatArgs(c)
		if err != nil {
			return nil, err
		}
		o, err := optsFrom[optimize.BracketOptions](c.Kwargs)
		if err != nil {
			return nil, err
		}
		g := &guard{}
		res, err := f(fn[func(float64) float64](g, call), ab[0], ab[1], o)
		return finish(g, res, err)
	}
}

// derivatives reads fprime: and fprime2: Procs.
func derivatives(g *guard, kw map[string]rb.Value) (f1, f2 func(float64) float64, rest map[string]rb.Value, err error) {
	mine, rest := split(kw, "fprime", "fprime2")
	if f1, _, err = optionalProc[func(float64) float64](g, mine, "fprime"); err != nil {
		return nil, nil, nil, err
	}
	if f2, _, err = optionalProc[func(float64) float64](g, mine, "fprime2"); err != nil {
		return nil, nil, nil, err
	}
	return f1, f2, rest, nil
}

func newton(c *rb.Call) (any, error) {
	call, err := blockAndArgs(c, 1)
	if err != nil {
		return nil, err
	}
	x0, err := toFloat(c.Args[0].Go())
	if err != nil {
		return nil, err
	}
	g := &guard{}
	f1, f2, rest, err := derivatives(g, c.Kwargs)
	if err != nil {
		return nil, err
	}
	o := &optimize.NewtonOptions{}
	if err := options(o, rest); err != nil {
		return nil, err
	}
	_, o.X1Set = rest["x1"]
	o.Fprime, o.Fprime2 = f1, f2
	res, err := optimize.Newton(fn[func(float64) float64](g, call), x0, o)
	return finish(g, res, err)
}

func rootScalar(c *rb.Call) (any, error) {
	call, err := blockAndArgs(c, 0)
	if err != nil {
		return nil, err
	}
	g := &guard{}
	f1, f2, rest, err := derivatives(g, c.Kwargs)
	if err != nil {
		return nil, err
	}
	mine, rest := split(rest, "method")
	method := ""
	if v, ok := mine["method"]; ok {
		if method, err = toString(v.Go()); err != nil {
			return nil, err
		}
	}
	o := &optimize.RootScalarOptions{}
	if err := options(o, rest); err != nil {
		return nil, err
	}
	_, o.X0Set = rest["x0"]
	_, o.X1Set = rest["x1"]
	o.Fprime, o.Fprime2 = f1, f2
	res, err := optimize.RootScalar(fn[func(float64) float64](g, call), method, o)
	return finish(g, res, err)
}

func systemRoot(f func(optimize.SystemProblem, []float64, *optimize.RootOptions) (*optimize.RootResult, error)) rb.Func {
	return func(c *rb.Call) (any, error) {
		call, err := blockAndArgs(c, 1)
		if err != nil {
			return nil, err
		}
		x0, err := toFloats(c.Args[0].Go())
		if err != nil {
			return nil, err
		}
		g := &guard{}
		mine, rest := split(c.Kwargs, "jac")
		jac, _, err := optionalProc[func(x, jac []float64)](g, mine, "jac")
		if err != nil {
			return nil, err
		}
		o, err := optsFrom[optimize.RootOptions](rest)
		if err != nil {
			return nil, err
		}
		p := optimize.SystemProblem{Func: fn[func(x, f []float64)](g, call), Jac: jac}
		res, err := f(p, x0, o)
		return finish(g, res, err)
	}
}

// lsProblem builds the residual problem of least_squares and leastsq: m: is
// the number of residuals, and when it is not given the block is called once
// at x0 to count them.
func lsProblem(c *rb.Call, g *guard, call caller, x0 []float64) (optimize.LSProblem, map[string]rb.Value, error) {
	mine, rest := split(c.Kwargs, "m", "jac")
	p := optimize.LSProblem{Func: fn[func(x, f []float64)](g, call)}
	if v, ok := mine["m"]; ok {
		m, err := toInt(v.Go())
		if err != nil {
			return p, nil, err
		}
		p.M = int(m)
	} else {
		r, err := call(vec(x0))
		if err != nil {
			return p, nil, err
		}
		f, err := toFloats(r.Go())
		if err != nil {
			return p, nil, err
		}
		p.M = len(f)
	}
	jac, _, err := optionalProc[func(x, jac []float64)](g, mine, "jac")
	p.Jac = jac
	return p, rest, err
}

func leastSquares(c *rb.Call) (any, error) {
	call, err := blockAndArgs(c, 1)
	if err != nil {
		return nil, err
	}
	x0, err := toFloats(c.Args[0].Go())
	if err != nil {
		return nil, err
	}
	g := &guard{}
	p, rest, err := lsProblem(c, g, call, x0)
	if err != nil {
		return nil, err
	}
	mine, rest := split(rest, "method")
	method, err := enumKw(mine, "method", optimize.TRF)
	if err != nil {
		return nil, err
	}
	o, err := optsFrom[optimize.LSOptions](rest)
	if err != nil {
		return nil, err
	}
	res, err := optimize.LeastSquares(p, x0, method, o)
	return finish(g, res, err)
}

func leastsq(c *rb.Call) (any, error) {
	call, err := blockAndArgs(c, 1)
	if err != nil {
		return nil, err
	}
	x0, err := toFloats(c.Args[0].Go())
	if err != nil {
		return nil, err
	}
	g := &guard{}
	p, rest, err := lsProblem(c, g, call, x0)
	if err != nil {
		return nil, err
	}
	o, err := optsFrom[optimize.LeastsqOptions](rest)
	if err != nil {
		return nil, err
	}
	res, err := optimize.Leastsq(p, x0, o)
	return finish(g, res, err)
}

func curveFit(c *rb.Call) (any, error) {
	call, err := blockAndArgs(c, 2)
	if err != nil {
		return nil, err
	}
	x, err := ndArray(c.Args[0].Go())
	if err != nil {
		return nil, err
	}
	y, err := ndArray(c.Args[1].Go())
	if err != nil {
		return nil, err
	}
	mine, rest := split(c.Kwargs, "p0")
	var p0 []float64
	if v, ok := mine["p0"]; ok {
		if p0, err = toFloats(v.Go()); err != nil {
			return nil, err
		}
	}
	o, err := optsFrom[optimize.CurveFitOptions](rest)
	if err != nil {
		return nil, err
	}
	if o != nil && o.Workers > 1 {
		return nil, errWorkers
	}
	g := &guard{}
	model := g.makeFunc(reflect.TypeFor[func(x float64, p []float64) float64](), call, true).Interface().(func(float64, []float64) float64)
	res, err := optimize.CurveFit(model, x, y, p0, o)
	return finish(g, res, err)
}

func linprog(c *rb.Call) (any, error) {
	if err := arity(c, 1, 1); err != nil {
		return nil, err
	}
	cost, err := toFloats(c.Args[0].Go())
	if err != nil {
		return nil, err
	}
	mine, rest := split(c.Kwargs, "a_ub", "b_ub", "a_eq", "b_eq", "bounds")
	var cons optimize.LPConstraints
	if err := options(&cons, mine); err != nil {
		return nil, err
	}
	o, err := optsFrom[optimize.LPOptions](rest)
	if err != nil {
		return nil, err
	}
	res, err := optimize.Linprog(cost, cons, o)
	return finish(&guard{}, res, err)
}

func brute(c *rb.Call) (any, error) {
	call, err := blockAndArgs(c, 1)
	if err != nil {
		return nil, err
	}
	rv, err := convert(c.Args[0].Go(), reflect.TypeFor[[]optimize.Range]())
	if err != nil {
		return nil, err
	}
	o, err := optsFrom[optimize.BruteOptions](c.Kwargs)
	if err != nil {
		return nil, err
	}
	if o != nil && o.Workers > 1 {
		return nil, errWorkers
	}
	// Brute evaluates the grid on a goroutine of its own, where neither an
	// error nor a break may unwind: what the block raised, or how it left, is
	// kept and replayed here, on the caller's goroutine, once Brute is done.
	g := &guard{}
	objective := fn[func([]float64) float64](g, call)
	var pending any
	safe := func(x []float64) (f float64) {
		if pending != nil {
			return math.NaN()
		}
		defer func() {
			if r := recover(); r != nil {
				pending, f = r, math.NaN()
			}
		}()
		return objective(x)
	}
	res, err := optimize.Brute(safe, rv.Interface().([]optimize.Range), o)
	if pending != nil {
		panic(pending)
	}
	return finish(g, res, err)
}

func fminbound(c *rb.Call) (any, error) {
	call, err := blockAndArgs(c, 2)
	if err != nil {
		return nil, err
	}
	x, err := floatArgs(c)
	if err != nil {
		return nil, err
	}
	mine, rest := split(c.Kwargs, "xtol", "maxfun")
	if len(rest) > 0 {
		return nil, argError("unknown keyword: :%s (known: xtol, maxfun)", sortedKeys(rest)[0])
	}
	xtol, maxfun := 1e-5, int64(500)
	if v, ok := mine["xtol"]; ok {
		if xtol, err = toFloat(v.Go()); err != nil {
			return nil, err
		}
	}
	if v, ok := mine["maxfun"]; ok {
		if maxfun, err = toInt(v.Go()); err != nil {
			return nil, err
		}
	}
	g := &guard{}
	res, err := optimize.Fminbound(fn[func(float64) float64](g, call), x[0], x[1], xtol, int(maxfun))
	return finish(g, res, err)
}

func bracketFn(c *rb.Call) (any, error) {
	call, err := blockAndArgs(c, 2)
	if err != nil {
		return nil, err
	}
	x, err := floatArgs(c)
	if err != nil {
		return nil, err
	}
	g := &guard{}
	xs, fs, nfev, err := optimize.Bracket(fn[func(float64) float64](g, call), x[0], x[1])
	return finish(g, []any{xs[:], fs[:], nfev}, err)
}

func approxFprime(c *rb.Call) (any, error) {
	call, err := blockAndArgs(c, 2)
	if err != nil {
		return nil, err
	}
	x, err := toFloats(c.Args[0].Go())
	if err != nil {
		return nil, err
	}
	eps, err := toFloat(c.Args[1].Go())
	if err != nil {
		return nil, err
	}
	g := &guard{}
	return finish(g, vec(optimize.ApproxFprime(fn[func([]float64) float64](g, call), x, eps)), nil)
}
