package ruby

import (
	"reflect"

	rb "github.com/go-embedded-ruby/ruby"
	nd "github.com/go-ndarray/ndarray"
	"github.com/go-numerics/integrate"
)

// registerIntegrate defines Integrate, go-numerics/integrate with the
// integrand or the right-hand side as the block:
//
//	Integrate.quad(a, b, **options) { |x| f(x) }            # => QuadResult (value, abs_err, …)
//	Integrate.dblquad(a, b, g, h) { |y, x| f(y, x) }        # g, h: Floats or Procs of x
//	Integrate.tplquad(a, b, g, h, q, r) { |z, y, x| ... }   # q, r: Floats or Procs of (x, y)
//	Integrate.nquad([[a, b], [c, d]]) { |x| f(x) }          # x an NDArray
//	Integrate.fixed_quad(a, b, n = 5) { |x| f(x) }
//	Integrate.romberg(a, b, **options) { |x| f(x) }
//	Integrate.solve_ivp([t0, t1], y0, **options) { |t, y| dydt }   # => Solution
//	Integrate.odeint(y0, t, **options) { |t, y| dydt }      # => [NDArray, OdeintInfo]
//	Integrate.trapezoid(y, x: nil, dx: 1.0, axis: -1)       # also simpson
//	Integrate.cumulative_trapezoid(y, x: nil, dx: 1.0, initial: false, axis: -1)
//	Integrate.romb(y, dx: 1.0)
//	Integrate.gauss_legendre(n)                             # => [nodes, weights]
//
// The right-hand side gets t as a Float and y as an NDArray, as in the Go
// API (and SciPy's solve_ivp; odeint takes the same order, SciPy's
// tfirst=True). The keywords are the fields of the Go option struct.
func registerIntegrate(in *rb.Interp) error {
	d := &definer{in: in}
	m := d.module("Integrate")
	if d.err != nil {
		return d.err
	}
	d.structClass(m, "QuadResult", integrate.QuadResult{})
	d.structClass(m, "Solution", integrate.Solution{})
	d.structClass(m, "OdeintInfo", integrate.OdeintInfo{})
	odeSol := d.class(m, "ODESolution")
	if odeSol != nil {
		odeSol.Bind(&integrate.ODESolution{})
		bindMethods(odeSol, reflect.TypeFor[*integrate.ODESolution](), map[string]bool{"EvalInto": true})
	}
	d.fn(m, "quad", quad)
	d.fn(m, "dblquad", dblquad)
	d.fn(m, "tplquad", tplquad)
	d.fn(m, "nquad", nquad)
	d.fn(m, "fixed_quad", fixedQuad)
	d.fn(m, "romberg", romberg)
	d.fn(m, "solve_ivp", solveIVP)
	d.fn(m, "odeint", odeint)
	d.fn(m, "trapezoid", sampled(integrate.Trapezoid, integrate.TrapezoidND))
	d.fn(m, "simpson", sampled(integrate.Simpson, integrate.SimpsonND))
	d.fn(m, "cumulative_trapezoid", cumulativeTrapezoid)
	d.fn(m, "romb", romb)
	d.fn(m, "gauss_legendre", func(c *rb.Call) (any, error) {
		if err := arity(c, 1, 1); err != nil {
			return nil, err
		}
		n, err := toInt(c.Args[0].Go())
		if err != nil {
			return nil, err
		}
		x, w := integrate.GaussLegendre(int(n))
		return []any{vec(x), vec(w)}, nil
	})
	return d.err
}

func init() {
	converters[reflect.TypeFor[integrate.Range]()] = func(v any) (reflect.Value, error) {
		b, err := toFloats(v)
		if err != nil || len(b) != 2 {
			return reflect.Value{}, typeError(v, "a range [lo, hi]")
		}
		return reflect.ValueOf(integrate.ConstRange(b[0], b[1])), nil
	}
}

func quad(c *rb.Call) (any, error) {
	call, err := blockAndArgs(c, 2)
	if err != nil {
		return nil, err
	}
	ab, err := floatArgs(c)
	if err != nil {
		return nil, err
	}
	o, err := optsFrom[integrate.QuadOptions](c.Kwargs)
	if err != nil {
		return nil, err
	}
	g := &guard{}
	res, err := integrate.Quad(fn[func(float64) float64](g, call), ab[0], ab[1], o)
	return finish(g, res, err)
}

func dblquad(c *rb.Call) (any, error) {
	call, err := blockAndArgs(c, 4)
	if err != nil {
		return nil, err
	}
	ab, err := floatArgs(&rb.Call{Args: c.Args[:2]})
	if err != nil {
		return nil, err
	}
	g := &guard{}
	lo, err := numberOrProc[func(float64) float64](g, c.Args[2])
	if err != nil {
		return nil, err
	}
	hi, err := numberOrProc[func(float64) float64](g, c.Args[3])
	if err != nil {
		return nil, err
	}
	o, err := optsFrom[integrate.QuadOptions](c.Kwargs)
	if err != nil {
		return nil, err
	}
	res, err := integrate.DblQuad(fn[func(y, x float64) float64](g, call), ab[0], ab[1], lo, hi, o)
	return finish(g, res, err)
}

func tplquad(c *rb.Call) (any, error) {
	call, err := blockAndArgs(c, 6)
	if err != nil {
		return nil, err
	}
	ab, err := floatArgs(&rb.Call{Args: c.Args[:2]})
	if err != nil {
		return nil, err
	}
	g := &guard{}
	var lims [2]func(float64) float64
	for i := range lims {
		if lims[i], err = numberOrProc[func(float64) float64](g, c.Args[2+i]); err != nil {
			return nil, err
		}
	}
	var inner [2]func(x, y float64) float64
	for i := range inner {
		if inner[i], err = numberOrProc[func(x, y float64) float64](g, c.Args[4+i]); err != nil {
			return nil, err
		}
	}
	o, err := optsFrom[integrate.QuadOptions](c.Kwargs)
	if err != nil {
		return nil, err
	}
	res, err := integrate.TplQuad(fn[func(z, y, x float64) float64](g, call), ab[0], ab[1], lims[0], lims[1], inner[0], inner[1], o)
	return finish(g, res, err)
}

func nquad(c *rb.Call) (any, error) {
	call, err := blockAndArgs(c, 1)
	if err != nil {
		return nil, err
	}
	rv, err := convert(c.Args[0].Go(), reflect.TypeFor[[]integrate.Range]())
	if err != nil {
		return nil, err
	}
	var o *integrate.NQuadOptions
	if len(c.Kwargs) > 0 {
		q, err := optsFrom[integrate.QuadOptions](c.Kwargs)
		if err != nil {
			return nil, err
		}
		o = &integrate.NQuadOptions{Opts: []*integrate.QuadOptions{q}}
	}
	g := &guard{}
	res, err := integrate.NQuad(fn[func([]float64) float64](g, call), rv.Interface().([]integrate.Range), o)
	return finish(g, res, err)
}

func fixedQuad(c *rb.Call) (any, error) {
	if err := arity(c, 2, 3); err != nil {
		return nil, err
	}
	b, err := needBlock(c)
	if err != nil {
		return nil, err
	}
	ab, err := floatArgs(&rb.Call{Args: c.Args[:2]})
	if err != nil {
		return nil, err
	}
	n := int64(5)
	if len(c.Args) == 3 {
		if n, err = toInt(c.Args[2].Go()); err != nil {
			return nil, err
		}
	}
	g := &guard{}
	v, err := integrate.FixedQuad(fn[func(float64) float64](g, blockCaller(b)), ab[0], ab[1], int(n))
	return finish(g, v, err)
}

func romberg(c *rb.Call) (any, error) {
	call, err := blockAndArgs(c, 2)
	if err != nil {
		return nil, err
	}
	ab, err := floatArgs(c)
	if err != nil {
		return nil, err
	}
	o, err := optsFrom[integrate.RombergOptions](c.Kwargs)
	if err != nil {
		return nil, err
	}
	g := &guard{}
	v, err := integrate.Romberg(fn[func(float64) float64](g, call), ab[0], ab[1], o)
	return finish(g, v, err)
}

// odeFunc is the block as a right-hand side: |t, y| -> dy/dt.
func odeFunc(g *guard, call caller) integrate.ODEFunc {
	f := fn[func(t float64, y, dydt []float64)](g, call)
	return integrate.ODEFunc(f)
}

func solveIVP(c *rb.Call) (any, error) {
	call, err := blockAndArgs(c, 2)
	if err != nil {
		return nil, err
	}
	span, err := convert(c.Args[0].Go(), reflect.TypeFor[[2]float64]())
	if err != nil {
		return nil, err
	}
	y0, err := toFloats(c.Args[1].Go())
	if err != nil {
		return nil, err
	}
	o, err := optsFrom[integrate.IVPOptions](c.Kwargs)
	if err != nil {
		return nil, err
	}
	g := &guard{}
	res, err := integrate.SolveIVP(odeFunc(g, call), span.Interface().([2]float64), y0, o)
	return finish(g, res, err)
}

func odeint(c *rb.Call) (any, error) {
	call, err := blockAndArgs(c, 2)
	if err != nil {
		return nil, err
	}
	y0, err := toFloats(c.Args[0].Go())
	if err != nil {
		return nil, err
	}
	t, err := toFloats(c.Args[1].Go())
	if err != nil {
		return nil, err
	}
	o, err := optsFrom[integrate.OdeintOptions](c.Kwargs)
	if err != nil {
		return nil, err
	}
	g := &guard{}
	y, info, err := integrate.Odeint(odeFunc(g, call), y0, t, o)
	return finish(g, []any{y, info}, err)
}

// sampledKw reads the keywords of the sample-based rules: x:, dx: (1.0) and
// axis: (-1, for an NDArray of more than one dimension).
func sampledKw(c *rb.Call, extra ...string) (x []float64, dx float64, axis int, mine map[string]rb.Value, err error) {
	mine, rest := split(c.Kwargs, append([]string{"x", "dx", "axis"}, extra...)...)
	if len(rest) > 0 {
		return nil, 0, 0, nil, argError("unknown keyword: :%s (known: x, dx, axis)", sortedKeys(rest)[0])
	}
	dx, axis = 1, -1
	if v, ok := mine["x"]; ok {
		if x, err = toFloats(v.Go()); err != nil {
			return
		}
	}
	if v, ok := mine["dx"]; ok {
		if dx, err = toFloat(v.Go()); err != nil {
			return
		}
	}
	if v, ok := mine["axis"]; ok {
		var a int64
		if a, err = toInt(v.Go()); err != nil {
			return
		}
		axis = int(a)
	}
	return
}

// sampled binds a rule over samples, its 1-d and its n-d forms: an NDArray of
// more than one dimension integrates along axis and returns an NDArray.
func sampled(oneD func(y, x []float64, dx float64) (float64, error), nD func(y *nd.Array, x []float64, dx float64, axis int) (*nd.Array, error)) rb.Func {
	return func(c *rb.Call) (any, error) {
		if err := arity(c, 1, 1); err != nil {
			return nil, err
		}
		x, dx, axis, _, err := sampledKw(c)
		if err != nil {
			return nil, err
		}
		if a, ok := c.Args[0].Go().(*nd.Array); ok && a.Ndim() > 1 {
			r, err := nD(a, x, dx, axis)
			return resultOrError(r, err)
		}
		y, err := toFloats(c.Args[0].Go())
		if err != nil {
			return nil, err
		}
		r, err := oneD(y, x, dx)
		return resultOrError(r, err)
	}
}

func cumulativeTrapezoid(c *rb.Call) (any, error) {
	if err := arity(c, 1, 1); err != nil {
		return nil, err
	}
	x, dx, axis, mine, err := sampledKw(c, "initial")
	if err != nil {
		return nil, err
	}
	initial := false
	if v, ok := mine["initial"]; ok {
		b, isBool := v.Go().(bool)
		if !isBool {
			return nil, typeError(v.Go(), "true or false")
		}
		initial = b
	}
	if a, ok := c.Args[0].Go().(*nd.Array); ok && a.Ndim() > 1 {
		r, err := integrate.CumulativeTrapezoidND(a, x, dx, axis, initial)
		return resultOrError(r, err)
	}
	y, err := toFloats(c.Args[0].Go())
	if err != nil {
		return nil, err
	}
	r, err := integrate.CumulativeTrapezoid(y, x, dx, initial)
	return resultOrError(vec(r), err)
}

func romb(c *rb.Call) (any, error) {
	if err := arity(c, 1, 1); err != nil {
		return nil, err
	}
	_, dx, _, _, err := sampledKw(c)
	if err != nil {
		return nil, err
	}
	y, err := toFloats(c.Args[0].Go())
	if err != nil {
		return nil, err
	}
	r, err := integrate.Romb(y, dx)
	return resultOrError(r, err)
}
