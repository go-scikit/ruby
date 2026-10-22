package ruby

import (
	"math"
	"testing"

	nd "github.com/go-ndarray/ndarray"
	"github.com/go-numerics/optimize"
)

func rosen(x []float64) float64 {
	// float64(...) rounds each product, as Ruby does: without it the Go
	// compiler may fuse a multiply and an add (FMA on arm64) and the two
	// sides would differ by the fused rounding, not by the binding.
	a, b := 1-x[0], x[1]-float64(x[0]*x[0])
	return float64(a*a) + float64(100*b*b)
}

const rosenRuby = "{ |x| a = 1 - x[0]; b = x[1] - x[0] * x[0]; a * a + 100 * b * b }"

func rosenGrad(x, g []float64) {
	g[0] = float64(-2*(1-x[0])) - float64(400*x[0]*(x[1]-float64(x[0]*x[0])))
	g[1] = 200 * (x[1] - float64(x[0]*x[0]))
}

const rosenGradRuby = "->(x) { [-2 * (1 - x[0]) - 400 * x[0] * (x[1] - x[0] * x[0]), 200 * (x[1] - x[0] * x[0])] }"

func cube(x float64) float64 { return float64(x*x*x) - float64(2*x) - 5 }

const cubeRuby = "{ |x| x * x * x - 2 * x - 5 }"

func TestOptimizeMatchesGo(t *testing.T) {
	in := newInterp(t)
	x0 := []float64{-1.2, 1}
	m := must[*optimize.Result](t)
	check(t, in, "Optimize.minimize([-1.2, 1]) "+rosenRuby, m(optimize.Minimize(optimize.Problem{Func: rosen}, x0, optimize.BFGS, nil)))
	for _, meth := range []struct {
		sym string
		m   optimize.Method
	}{{":nelder_mead", optimize.NelderMead}, {":powell", optimize.Powell}, {":cg", optimize.CG}, {"'L-BFGS-B'", optimize.LBFGSB}} {
		check(t, in, "Optimize.minimize([-1.2, 1], method: "+meth.sym+", max_iter: 400) "+rosenRuby,
			m(optimize.Minimize(optimize.Problem{Func: rosen}, x0, meth.m, &optimize.Options{MaxIter: 400})))
	}
	check(t, in, "Optimize.minimize([-1.2, 1], jac: "+rosenGradRuby+") "+rosenRuby,
		m(optimize.Minimize(optimize.Problem{Func: rosen, Grad: rosenGrad}, x0, optimize.BFGS, nil)))
	check(t, in, "Optimize.minimize([3.0, 3.0], method: :lbfgsb, bounds: [[2, nil], [nil, 2.5]]) "+rosenRuby,
		m(optimize.Minimize(optimize.Problem{Func: rosen}, []float64{3, 3}, optimize.LBFGSB, &optimize.Options{Bounds: []optimize.Bound{{Lo: 2, Hi: math.Inf(1)}, {Lo: math.Inf(-1), Hi: 2.5}}})))

	sq := func(x float64) float64 { return (x - 2) * (x - 2) }
	check(t, in, "Optimize.minimize_scalar { |x| (x - 2) * (x - 2) }", must[*optimize.ScalarResult](t)(optimize.MinimizeScalar(sq, optimize.Brent, nil)))
	check(t, in, "Optimize.minimize_scalar(method: :bounded, bounds: [0, 1]) { |x| (x - 2) * (x - 2) }",
		must[*optimize.ScalarResult](t)(optimize.MinimizeScalar(sq, optimize.Bounded, &optimize.ScalarOptions{Bounds: &optimize.Bound{Lo: 0, Hi: 1}})))

	r := must[*optimize.RootScalarResult](t)
	for name, f := range map[string]func(func(float64) float64, float64, float64, *optimize.BracketOptions) (*optimize.RootScalarResult, error){
		"brentq": optimize.Brentq, "brenth": optimize.Brenth, "bisect": optimize.Bisect, "ridder": optimize.Ridder,
	} {
		check(t, in, "Optimize."+name+"(2, 3) "+cubeRuby, r(f(cube, 2, 3, nil)))
		check(t, in, "Optimize."+name+"(2, 3, xtol: 1e-6) "+cubeRuby, r(f(cube, 2, 3, &optimize.BracketOptions{Xtol: 1e-6})))
	}
	dcube := func(x float64) float64 { return float64(3*x*x) - 2 }
	check(t, in, "Optimize.newton(2.0, fprime: ->(x) { 3 * x * x - 2 }, fprime2: ->(x) { 6 * x }) "+cubeRuby,
		r(optimize.Newton(cube, 2, &optimize.NewtonOptions{Fprime: dcube, Fprime2: func(x float64) float64 { return 6 * x }})))
	check(t, in, "Optimize.newton(2.0, x1: 2.5) "+cubeRuby, r(optimize.Newton(cube, 2, &optimize.NewtonOptions{X1: 2.5, X1Set: true})))
	check(t, in, "Optimize.root_scalar(method: 'brentq', bracket: [2, 3]) "+cubeRuby, r(optimize.RootScalar(cube, "brentq", &optimize.RootScalarOptions{Bracket: []float64{2, 3}})))
	check(t, in, "Optimize.root_scalar(x0: 2.0, fprime: ->(x) { 3 * x * x - 2 }) "+cubeRuby, r(optimize.RootScalar(cube, "", &optimize.RootScalarOptions{X0: 2, X0Set: true, Fprime: dcube})))
	check(t, in, "Optimize.root_scalar(method: :secant, x0: 2.0, x1: 3.0) "+cubeRuby, r(optimize.RootScalar(cube, "secant", &optimize.RootScalarOptions{X0: 2, X0Set: true, X1: 3, X1Set: true})))

	sys := func(x, f []float64) {
		d := x[0] - x[1]
		f[0] = x[0] + float64(0.5*d*d*d) - 1
		e := x[1] - x[0]
		f[1] = float64(0.5*e*e*e) + x[1]
	}
	sysRuby := "{ |x| d = x[0] - x[1]; [x[0] + 0.5 * d * d * d - 1, 0.5 * (x[1] - x[0]) * (x[1] - x[0]) * (x[1] - x[0]) + x[1]] }"
	sysJac := func(x, j []float64) {
		d := x[0] - x[1]
		j[0], j[1] = 1+float64(1.5*d*d), -1.5*d*d
		j[2], j[3] = -1.5*d*d, 1+float64(1.5*d*d)
	}
	sysJacRuby := "->(x) { d = x[0] - x[1]; [1 + 1.5 * d * d, -1.5 * d * d, -1.5 * d * d, 1 + 1.5 * d * d] }"
	rr := must[*optimize.RootResult](t)
	check(t, in, "Optimize.root([0, 0]) "+sysRuby, rr(optimize.Root(optimize.SystemProblem{Func: sys}, []float64{0, 0}, nil)))
	check(t, in, "Optimize.fsolve([0, 0], jac: "+sysJacRuby+", xtol: 1e-10) "+sysRuby, rr(optimize.Fsolve(optimize.SystemProblem{Func: sys, Jac: sysJac}, []float64{0, 0}, &optimize.RootOptions{Xtol: 1e-10})))

	resid := func(x, f []float64) { f[0], f[1], f[2] = x[0]-1, x[1]-2, float64(x[0]*x[1])-2.5 }
	residRuby := "{ |x| [x[0] - 1, x[1] - 2, x[0] * x[1] - 2.5] }"
	check(t, in, "Optimize.least_squares([0.5, 0.5]) "+residRuby, must[*optimize.LSResult](t)(optimize.LeastSquares(optimize.LSProblem{Func: resid, M: 3}, []float64{0.5, 0.5}, optimize.TRF, nil)))
	check(t, in, "Optimize.least_squares([0.5, 0.5], m: 3, method: :lm, xtol: 1e-12) "+residRuby, must[*optimize.LSResult](t)(optimize.LeastSquares(optimize.LSProblem{Func: resid, M: 3}, []float64{0.5, 0.5}, optimize.LM, &optimize.LSOptions{Xtol: 1e-12})))
	residJac := func(x, j []float64) { j[0], j[1], j[2], j[3], j[4], j[5] = 1, 0, 0, 1, x[1], x[0] }
	check(t, in, "Optimize.least_squares([0.5, 0.5], jac: ->(x) { [1, 0, 0, 1, x[1], x[0]] }) "+residRuby, must[*optimize.LSResult](t)(optimize.LeastSquares(optimize.LSProblem{Func: resid, Jac: residJac, M: 3}, []float64{0.5, 0.5}, optimize.TRF, nil)))
	check(t, in, "Optimize.leastsq([0.5, 0.5], ftol: 1e-10) "+residRuby, must[*optimize.LeastsqResult](t)(optimize.Leastsq(optimize.LSProblem{Func: resid, M: 3}, []float64{0.5, 0.5}, &optimize.LeastsqOptions{Ftol: 1e-10})))

	xd, _ := nd.FromData([]float64{0, 1, 2, 3, 4}, 5)
	yd, _ := nd.FromData([]float64{1.1, 2.6, 7.6, 20.0, 54.9}, 5)
	model := func(x float64, p []float64) float64 { return p[0] * math.Exp(p[1]*x) }
	check(t, in, "Optimize.curve_fit([0, 1, 2, 3, 4], NDArray.from([1.1, 2.6, 7.6, 20.0, 54.9], [5]), p0: [1, 1]) { |x, a, b| a * Math.exp(b * x) }",
		must[*optimize.CurveFitResult](t)(optimize.CurveFit(model, xd, yd, []float64{1, 1}, nil)))
	check(t, in, "Optimize.curve_fit([0, 1, 2, 3, 4], [1.1, 2.6, 7.6, 20.0, 54.9], p0: [1, 1], bounds: [[0, 10], [0, 2]], method: :trf) { |x, a, b| a * Math.exp(b * x) }",
		must[*optimize.CurveFitResult](t)(optimize.CurveFit(model, xd, yd, []float64{1, 1}, &optimize.CurveFitOptions{Bounds: []optimize.Bound{{Lo: 0, Hi: 10}, {Lo: 0, Hi: 2}}, Method: optimize.TRF})))

	aub, _ := nd.FromData([]float64{1, 1, 1, 3}, 2, 2)
	bub, _ := nd.FromData([]float64{4, 6}, 2)
	check(t, in, "Optimize.linprog([-1, -2], a_ub: NDArray.from([1, 1, 1, 3], [2, 2]), b_ub: [4, 6], bounds: [[0, nil], [0, nil]], method: :interior_point)",
		must[*optimize.LPResult](t)(optimize.Linprog([]float64{-1, -2}, optimize.LPConstraints{AUb: aub, BUb: bub, Bounds: []optimize.Bound{{Lo: 0, Hi: math.Inf(1)}, {Lo: 0, Hi: math.Inf(1)}}}, &optimize.LPOptions{Method: optimize.LPInteriorPoint})))
	check(t, in, "Optimize.linprog([-1, -2], a_ub: NDArray.from([1, 1, 1, 3], [2, 2]), b_ub: [4, 6])",
		must[*optimize.LPResult](t)(optimize.Linprog([]float64{-1, -2}, optimize.LPConstraints{AUb: aub, BUb: bub}, nil)))

	check(t, in, "Optimize.brute([[-2, 2], [-1, 3, 0.5]], ns: 9) "+rosenRuby,
		must[*optimize.BruteResult](t)(optimize.Brute(rosen, []optimize.Range{{Lo: -2, Hi: 2}, {Lo: -1, Hi: 3, Step: 0.5}}, &optimize.BruteOptions{Ns: 9})))
	check(t, in, "Optimize.fminbound(0, 1) { |x| (x - 2) * (x - 2) }", must[*optimize.ScalarResult](t)(optimize.Fminbound(sq, 0, 1, 1e-5, 500)))
	check(t, in, "Optimize.fminbound(0, 3, xtol: 1e-8, maxfun: 50) { |x| (x - 2) * (x - 2) }", must[*optimize.ScalarResult](t)(optimize.Fminbound(sq, 0, 3, 1e-8, 50)))
	xs, fs, nfev, err := optimize.Bracket(sq, 0, 1)
	if err != nil {
		t.Fatal(err)
	}
	check(t, in, "Optimize.bracket(0, 1) { |x| (x - 2) * (x - 2) }", []any{xs[:], fs[:], nfev})
	check(t, in, "Optimize.approx_fprime([-1.2, 1], 1e-7) "+rosenRuby, optimize.ApproxFprime(rosen, []float64{-1.2, 1}, 1e-7))
}
