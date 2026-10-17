package ruby

import (
	"math"
	"testing"

	nd "github.com/go-ndarray/ndarray"
	"github.com/go-numerics/integrate"
)

func TestIntegrateMatchesGo(t *testing.T) {
	in := newInterp(t)
	q := must[integrate.QuadResult](t)
	sq := func(x float64) float64 { return x * x }
	check(t, in, "Integrate.quad(0, 2) { |x| x * x }", q(integrate.Quad(sq, 0, 2, nil)))
	check(t, in, "Integrate.quad(0, Float::INFINITY, eps_rel: 1e-10) { |x| Math.exp(-x) }", q(integrate.Quad(func(x float64) float64 { return math.Exp(-x) }, 0, math.Inf(1), &integrate.QuadOptions{EpsRel: 1e-10})))
	check(t, in, "Integrate.quad(0, 1, weight: :cos, omega: 3) { |x| x }", q(integrate.Quad(func(x float64) float64 { return x }, 0, 1, &integrate.QuadOptions{Weight: integrate.WeightCos, Omega: 3})))

	xy := func(y, x float64) float64 { return x * y }
	check(t, in, "Integrate.dblquad(0, 1, 0, 2) { |y, x| x * y }", q(integrate.DblQuad(xy, 0, 1, integrate.Const(0), integrate.Const(2), nil)))
	check(t, in, "Integrate.dblquad(0, 1, ->(x) { x * x }, ->(x) { x }, eps_abs: 1e-12) { |y, x| x * y }",
		q(integrate.DblQuad(xy, 0, 1, func(x float64) float64 { return x * x }, func(x float64) float64 { return x }, &integrate.QuadOptions{EpsAbs: 1e-12})))
	xyz := func(z, y, x float64) float64 { return x + y + z }
	check(t, in, "Integrate.tplquad(0, 1, 0, 1, 0, ->(x, y) { x + y }) { |z, y, x| x + y + z }",
		q(integrate.TplQuad(xyz, 0, 1, integrate.Const(0), integrate.Const(1), integrate.Const2(0), func(x, y float64) float64 { return x + y }, nil)))
	check(t, in, "Integrate.nquad([[0, 1], [0, 2]]) { |x| x[0] * x[1] }",
		q(integrate.NQuad(func(x []float64) float64 { return x[0] * x[1] }, []integrate.Range{integrate.ConstRange(0, 1), integrate.ConstRange(0, 2)}, nil)))
	check(t, in, "Integrate.nquad([[0, 1]], limit: 10) { |x| x[0] }",
		q(integrate.NQuad(func(x []float64) float64 { return x[0] }, []integrate.Range{integrate.ConstRange(0, 1)}, &integrate.NQuadOptions{Opts: []*integrate.QuadOptions{{Limit: 10}}})))
	check(t, in, "Integrate.fixed_quad(0, 1) { |x| Math.sin(x) }", must[float64](t)(integrate.FixedQuad(math.Sin, 0, 1, 5)))
	check(t, in, "Integrate.fixed_quad(0, 1, 9) { |x| Math.sin(x) }", must[float64](t)(integrate.FixedQuad(math.Sin, 0, 1, 9)))
	check(t, in, "Integrate.romberg(0, 1) { |x| Math.sin(x) }", must[float64](t)(integrate.Romberg(math.Sin, 0, 1, nil)))
	check(t, in, "Integrate.romberg(0, 1, tol: 1e-10, div_max: 12) { |x| Math.sin(x) }", must[float64](t)(integrate.Romberg(math.Sin, 0, 1, &integrate.RombergOptions{Tol: 1e-10, DivMax: 12})))

	decay := func(t float64, y, dydt []float64) { dydt[0] = -0.5 * y[0]; dydt[1] = y[0] - y[1] }
	decayRuby := "{ |t, y| [-0.5 * y[0], y[0] - y[1]] }"
	check(t, in, "Integrate.solve_ivp([0, 4], [1.0, 0.0]) "+decayRuby, must[*integrate.Solution](t)(integrate.SolveIVP(decay, [2]float64{0, 4}, []float64{1, 0}, nil)))
	check(t, in, "Integrate.solve_ivp([0, 4], [1.0, 0.0], method: :radau, t_eval: [0, 1, 2, 4], r_tol: 1e-8) "+decayRuby,
		must[*integrate.Solution](t)(integrate.SolveIVP(decay, [2]float64{0, 4}, []float64{1, 0}, &integrate.IVPOptions{Method: integrate.Radau, TEval: []float64{0, 1, 2, 4}, RTol: 1e-8})))
	sol := must[*integrate.Solution](t)(integrate.SolveIVP(decay, [2]float64{0, 4}, []float64{1, 0}, &integrate.IVPOptions{DenseOutput: true}))
	check(t, in, "Integrate.solve_ivp([0, 4], [1.0, 0.0], dense_output: true) { |t, y| [-0.5 * y[0], y[0] - y[1]] }.sol.eval(1.5)", sol.Sol.Eval(1.5))
	y, info, err := integrate.Odeint(decay, []float64{1, 0}, []float64{0, 1, 2}, nil)
	if err != nil {
		t.Fatal(err)
	}
	check(t, in, "Integrate.odeint([1.0, 0.0], [0, 1, 2]) "+decayRuby, []any{y, info})
	y, info, err = integrate.Odeint(decay, []float64{1, 0}, []float64{0, 1, 2}, &integrate.OdeintOptions{RTol: 1e-10})
	if err != nil {
		t.Fatal(err)
	}
	check(t, in, "Integrate.odeint([1.0, 0.0], [0, 1, 2], r_tol: 1e-10) "+decayRuby, []any{y, info})

	ys := []float64{1, 4, 9, 16, 25}
	xs := []float64{0, 1, 2.5, 3, 4}
	f := must[float64](t)
	check(t, in, "Integrate.trapezoid([1, 4, 9, 16, 25])", f(integrate.Trapezoid(ys, nil, 1)))
	check(t, in, "Integrate.trapezoid([1, 4, 9, 16, 25], x: [0, 1, 2.5, 3, 4])", f(integrate.Trapezoid(ys, xs, 1)))
	check(t, in, "Integrate.simpson(NDArray.from([1, 4, 9, 16, 25], [5]), dx: 0.5)", f(integrate.Simpson(ys, nil, 0.5)))
	grid, _ := nd.FromData([]float64{1, 2, 3, 4, 5, 6}, 2, 3)
	check(t, in, "Integrate.trapezoid(NDArray.from([1, 2, 3, 4, 5, 6], [2, 3]), axis: 0)", must[*nd.Array](t)(integrate.TrapezoidND(grid, nil, 1, 0)))
	check(t, in, "Integrate.simpson(NDArray.from([1, 2, 3, 4, 5, 6], [2, 3]))", must[*nd.Array](t)(integrate.SimpsonND(grid, nil, 1, -1)))
	check(t, in, "Integrate.cumulative_trapezoid([1, 4, 9, 16, 25], initial: true)", must[[]float64](t)(integrate.CumulativeTrapezoid(ys, nil, 1, true)))
	check(t, in, "Integrate.cumulative_trapezoid([1, 4, 9, 16, 25], x: [0, 1, 2.5, 3, 4])", must[[]float64](t)(integrate.CumulativeTrapezoid(ys, xs, 1, false)))
	check(t, in, "Integrate.cumulative_trapezoid(NDArray.from([1, 2, 3, 4, 5, 6], [2, 3]), axis: 1, initial: true)", must[*nd.Array](t)(integrate.CumulativeTrapezoidND(grid, nil, 1, 1, true)))
	check(t, in, "Integrate.romb([1, 4, 9, 16, 25], dx: 0.25)", f(integrate.Romb(ys, 0.25)))
	gx, gw := integrate.GaussLegendre(4)
	check(t, in, "Integrate.gauss_legendre(4)", []any{gx, gw})
}
