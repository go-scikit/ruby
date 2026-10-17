package ruby

// The module Plot binds go-plot/plot, matplotlib's object model in pure Go:
//
//	fig, ax = Plot.subplots(figsize: [6, 4])
//	ax.plot([1, 2, 3], [1, 4, 9], color: "C1", label: "squares", marker: "o")
//	ax.legend
//	ax.title = "growth"
//	File.write("fig.svg", fig.to_svg)
//
// Bound, by Ruby name (each calls the Go function or method in parentheses):
//
//	Plot.figure(figsize: [w, h], dpi: d)          (NewFigure, FigSize, DPI)
//	Plot.subplots(rows = 1, cols = 1, figsize:, dpi:, squeeze: true)
//	                                              (Subplots) -> [fig, axes]
//	Plot.values(data)                             (Values) -> NDArray
//	Plot::Figure.new(figsize:, dpi:)              (NewFigure)
//	Figure#width #height #dpi                     (the fields)
//	Figure#pixel_size                             (PixelSize) -> [w, h]
//	Figure#axes                                   (Axes)
//	Figure#add_axes(l, b, w, h) / ([l, b, w, h])  (AddAxes)
//	Figure#subplots(rows = 1, cols = 1, squeeze: true)  (Figure.Subplots)
//	Figure#tight_layout                           (TightLayout) -> true/false
//	Figure#colorbar(mappable, ax) / (m, ax: ax)   (Colorbar) -> Axes
//	Figure#to_svg #to_png                         (SaveSVG, SavePNG) -> String
//	Figure#save_svg(path) #save_png(path)         (SaveSVG, SavePNG)
//	Figure#savefig(path)                          (by extension: .svg or .png)
//	Figure#display                                (Display) -> {mime => bytes}
//	Figure#inspect #to_s                          "#<Plot::Figure 640x480>"
//	Axes#plot(y) / (x, y, **opts)                 (Plot)
//	Axes#scatter(x, y, **opts)                    (Scatter) -> Mappable or nil
//	Axes#hist(x, **opts)                          (Hist)
//	Axes#bar(x, height, **opts)                   (Bar)
//	Axes#barh(y, width, **opts)                   (Barh)
//	Axes#fill_between(x, y1, y2 = nil, **opts)    (FillBetween)
//	Axes#errorbar(x, y, **opts)                   (ErrorBar)
//	Axes#axhline(y = 0, **opts) #axvline(x = 0)   (Axhline, Axvline)
//	Axes#text(x, y, s, **opts)                    (Text)
//	Axes#annotate(s, xy, xytext = xy, **opts)     (Annotate; xy are [x, y])
//	Axes#imshow(data, **opts)                     (Imshow) -> Mappable
//	Axes#contour(z) / (x, y, z, **opts)           (Contour) -> Mappable
//	Axes#contourf(z) / (x, y, z, **opts)          (Contourf) -> Mappable
//	Axes#legend(**opts)                           (Legend)
//	Axes#grid(on = true)                          (Grid)
//	Axes#set_title(s) #title=                     (SetTitle)
//	Axes#set_xlabel(s) #xlabel= #set_ylabel #ylabel=     (SetXLabel, SetYLabel)
//	Axes#set_xlim(lo, hi) / ([lo, hi]) #xlim= #set_ylim #ylim=  (SetXLim, SetYLim)
//	Axes#xlim #ylim -> [lo, hi]; xlim(lo, hi) sets (XLim, SetXLim, ...)
//	Axes#set_xscale(s) #xscale= #set_yscale #yscale=     (SetXScale, SetYScale)
//	Axes#set_xticks(locs, labels = nil) #set_yticks      (SetXTicks, SetYTicks)
//	Axes#set_aspect(r) #aspect=  (a number, :equal for 1, :auto for 0)
//	Axes#twinx                                    (TwinX)
//	Axes#figure                                   (Figure)
//	Plot::Mappable#inspect
//
// Keyword options map to the plot.Option constructors by their snake_case
// name, read from the functions themselves; case and underscores aside, so
// line_width: and linewidth: are one: align alpha arrow aspect bin_edges bins
// bottom c_values cap_size clip cmap color colors density edge_color
// error_color extent face_color font_size ha label levels line_style
// line_width loc log_norm marker marker_edge_width marker_size n_levels origin
// rgba rotation sizes va vmax vmin where width x_err y_err z_order; and
// matplotlib's spellings lw ls ms mew fc ec edgecolors ecolor c (c_values) s
// (sizes) clip_on height (width) left (bottom) arrowprops (arrow, when
// truthy). An option without an argument (density, arrow, log_norm) takes
// true (false leaves it out); one with several takes an Array (extent: [l, r,
// b, t], rgba: [r, g, b, a]); bins: and levels: take an Integer (Bins,
// NLevels) or an Array (BinEdges, Levels); color: takes a String or an Array
// [r, g, b] / [r, g, b, a] (RGBA). Options are applied in the alphabetical
// order of their keywords. An unknown keyword is an ArgumentError naming the
// known ones, and what go-plot panics on (an unknown colour, marker or line
// style, data of mismatched lengths) is an ArgumentError too.
//
// Data is an Array of numbers, an NDArray (passed as it is) or a
// DataFrame::Series (its Floats); 2-D data (imshow, contour) an NDArray or an
// Array of Arrays, and 3-D data (an RGB image) an NDArray or nested Arrays.
//
// Plot.subplots squeezes as matplotlib does: a 1x1 grid gives the Axes
// itself, one row or one column a flat Array, any other grid an Array of
// rows; squeeze: false always gives the Array of rows.
//
// Not bound: Figure.Draw and Figure.Image (a renderer and a raster, types Ruby
// has no use for; to_png gives the pixels), the fields Facecolor, Text and
// SubplotParams, and Mappable's Cmap and Norm.

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"sort"
	"strings"

	"github.com/go-dataframe/dataframe"
	rb "github.com/go-embedded-ruby/ruby"
	nd "github.com/go-ndarray/ndarray"
	"github.com/go-plot/plot"
)

func registerPlot(in *rb.Interp) error {
	d := &definer{in: in}
	m := d.module("Plot")
	fig := d.class(m, "Figure")
	ax := d.class(m, "Axes")
	mp := d.class(m, "Mappable")
	if d.err != nil {
		return d.err
	}
	fig.Bind(&plot.Figure{})
	ax.Bind(&plot.Axes{})
	mp.Bind(&plot.Mappable{})
	mp.DefineMethod("inspect", func(c *rb.Call) (any, error) {
		return fmt.Sprintf("#<Plot::Mappable %s>", c.Self.Go().(*plot.Mappable).Cmap.Name), nil
	})
	m.DefineSingletonMethod("figure", plNewFigure)
	fig.DefineSingletonMethod("new", plNewFigure)
	m.DefineSingletonMethod("subplots", plSubplots)
	m.DefineSingletonMethod("values", plValues)
	for name, f := range plFigureMethods {
		fig.DefineMethod(name, f)
	}
	for name, f := range plAxesMethods {
		ax.DefineMethod(name, f)
	}
	return nil
}

// plProtect runs f, turning a panic of go-plot (an invalid colour, data of
// mismatched lengths, ...) into an ArgumentError.
func plProtect(f func()) (err error) {
	defer func() {
		if r := recover(); r != nil {
			err = argError("%v", r)
		}
	}()
	f()
	return nil
}

// plNoKw rejects keyword arguments.
func plNoKw(c *rb.Call) error {
	if len(c.Kwargs) > 0 {
		return argError("unknown keyword: :%s", sortedKeys(c.Kwargs)[0])
	}
	return nil
}

// plExact checks a call has n positional arguments and no keywords.
func plExact(c *rb.Call, n int) error {
	if err := plNoKw(c); err != nil {
		return err
	}
	return arity(c, n, n)
}

func plValues(c *rb.Call) (any, error) {
	if err := plExact(c, 1); err != nil {
		return nil, err
	}
	v, err := plData(c.Args[0].Go())
	if err != nil {
		return nil, err
	}
	var f []float64
	if err := plProtect(func() { f = plot.Values(v) }); err != nil {
		return nil, err
	}
	return vec(f), nil
}

// plFigOptions reads figsize: and dpi:.
func plFigOptions(kw map[string]rb.Value) ([]plot.FigOption, error) {
	var opts []plot.FigOption
	for _, k := range sortedKeys(kw) {
		switch k {
		case "figsize":
			wh, err := toFloats(kw[k].Go())
			if err != nil {
				return nil, err
			}
			if len(wh) != 2 {
				return nil, argError("figsize must be [width, height], got %d values", len(wh))
			}
			opts = append(opts, plot.FigSize(wh[0], wh[1]))
		case "dpi":
			f, err := toFloat(kw[k].Go())
			if err != nil {
				return nil, err
			}
			opts = append(opts, plot.DPI(f))
		default:
			return nil, argError("unknown keyword: :%s (known: dpi, figsize)", k)
		}
	}
	return opts, nil
}

func plNewFigure(c *rb.Call) (any, error) {
	if err := arity(c, 0, 0); err != nil {
		return nil, err
	}
	opts, err := plFigOptions(c.Kwargs)
	if err != nil {
		return nil, err
	}
	return plot.NewFigure(opts...), nil
}

// plGrid reads (rows = 1, cols = 1) and squeeze:, leaving the other keywords.
func plGrid(c *rb.Call) (rows, cols int, squeeze bool, rest map[string]rb.Value, err error) {
	if err = arity(c, 0, 2); err != nil {
		return
	}
	dims := []int{1, 1}
	for i, a := range c.Args {
		n, e := toInt(a.Go())
		if e != nil {
			return 0, 0, false, nil, e
		}
		dims[i] = int(n)
	}
	if dims[0] < 1 || dims[1] < 1 {
		return 0, 0, false, nil, argError("rows and cols must be positive, got %d and %d", dims[0], dims[1])
	}
	mine, rest := split(c.Kwargs, "squeeze")
	squeeze = true
	if s, ok := mine["squeeze"]; ok {
		b, ok := s.Go().(bool)
		if !ok {
			return 0, 0, false, nil, typeError(s.Go(), "true or false")
		}
		squeeze = b
	}
	return dims[0], dims[1], squeeze, rest, nil
}

// plSqueeze shapes a grid of Axes as matplotlib's squeeze=True does.
func plSqueeze(axs [][]*plot.Axes, squeeze bool) any {
	if !squeeze {
		return axs
	}
	switch {
	case len(axs) == 1 && len(axs[0]) == 1:
		return axs[0][0]
	case len(axs) == 1:
		return axs[0]
	case len(axs[0]) == 1:
		col := make([]*plot.Axes, len(axs))
		for i, r := range axs {
			col[i] = r[0]
		}
		return col
	}
	return axs
}

func plSubplots(c *rb.Call) (any, error) {
	rows, cols, squeeze, rest, err := plGrid(c)
	if err != nil {
		return nil, err
	}
	opts, err := plFigOptions(rest)
	if err != nil {
		return nil, err
	}
	f, axs := plot.Subplots(rows, cols, opts...)
	return []any{f, plSqueeze(axs, squeeze)}, nil
}

// plData converts Ruby plot data to what go-plot's data parameters take: nil,
// a float64, a []float64, an *ndarray.Array (as it is), a [][]float64 (an
// Array of Arrays of numbers) or, deeper, an *ndarray.Array of that shape. A
// DataFrame::Series gives its Floats.
func plData(v any) (any, error) {
	switch x := v.(type) {
	case nil, *nd.Array:
		return x, nil
	case int64, float64:
		return toFloat(x)
	case *dataframe.Series:
		f, err := x.Floats()
		if err != nil {
			return nil, goError(err)
		}
		return f, nil
	case []any:
		if len(x) > 0 {
			if _, nested := x[0].([]any); nested {
				return plNested(x)
			}
		}
		return toFloats(x)
	}
	return nil, typeError(v, "plot data (an Array of numbers or an NDArray)")
}

// plNested converts nested Arrays: two levels to a [][]float64, more to an
// NDArray of their shape. Every level must be rectangular.
func plNested(x []any) (any, error) {
	var shape []int
	for e := any(x); ; {
		l, ok := e.([]any)
		if !ok || len(l) == 0 {
			break
		}
		shape = append(shape, len(l))
		e = l[0]
	}
	var flat []float64
	var walk func(e any, depth int) error
	walk = func(e any, depth int) error {
		if depth == len(shape) {
			f, err := toFloat(e)
			flat = append(flat, f)
			return err
		}
		l, ok := e.([]any)
		if !ok || len(l) != shape[depth] {
			return argError("plot data must be rectangular nested Arrays of numbers")
		}
		for _, s := range l {
			if err := walk(s, depth+1); err != nil {
				return err
			}
		}
		return nil
	}
	if err := walk(x, 0); err != nil {
		return nil, err
	}
	if len(shape) == 2 {
		rows := make([][]float64, shape[0])
		for i := range rows {
			rows[i] = flat[i*shape[1] : (i+1)*shape[1]]
		}
		return rows, nil
	}
	a, _ := nd.FromData(flat, shape...) // the shape is the data's own
	return a, nil
}

// plOptionFuncs are the plot.Option constructors; each one's keyword is its
// Go name in snake_case, read from the function itself.
var plOptionFuncs = []any{
	plot.Align, plot.Alpha, plot.Arrow, plot.Aspect, plot.BinEdges, plot.Bins,
	plot.Bottom, plot.CValues, plot.CapSize, plot.Clip, plot.Cmap, plot.Color,
	plot.Colors, plot.Density, plot.EdgeColor, plot.ErrorColor, plot.Extent,
	plot.FaceColor, plot.FontSize, plot.HA, plot.Label, plot.Levels,
	plot.LineStyle, plot.LineWidth, plot.Loc, plot.LogNorm, plot.Marker,
	plot.MarkerEdgeWidth, plot.MarkerSize, plot.NLevels, plot.Origin, plot.RGBA,
	plot.Rotation, plot.Sizes, plot.VA, plot.Vmax, plot.Vmin, plot.Where,
	plot.Width, plot.XErr, plot.YErr, plot.ZOrder,
}

// plAliases are matplotlib's spellings of options, and
// the Go constructor each stands for.
var plAliases = map[string]string{
	"lw": "LineWidth", "ls": "LineStyle", "ms": "MarkerSize", "mew": "MarkerEdgeWidth",
	"fc": "FaceColor", "ec": "EdgeColor", "edgecolors": "EdgeColor", "ecolor": "ErrorColor",
	"c": "CValues", "s": "Sizes", "clip_on": "Clip", "height": "Width", "left": "Bottom",
	"arrowprops": "Arrow",
}

// plOptions indexes the constructors by normalized keyword; plKnown lists the
// keywords for error messages.
var plOptions, plKnown = plOptionIndex()

func plOptionIndex() (map[string]reflect.Value, string) {
	byGo := map[string]reflect.Value{}
	idx := map[string]reflect.Value{}
	var names []string
	for _, f := range plOptionFuncs {
		fv := reflect.ValueOf(f)
		full := runtime.FuncForPC(fv.Pointer()).Name()
		goName := full[strings.LastIndex(full, ".")+1:]
		byGo[goName] = fv
		idx[normalize(goName)] = fv
		names = append(names, snakeName(goName))
	}
	for alias, goName := range plAliases {
		idx[normalize(alias)] = byGo[goName]
		names = append(names, alias)
	}
	sort.Strings(names)
	return idx, strings.Join(names, ", ")
}

// plOpts converts keyword arguments to plot options, in keyword order.
func plOpts(kw map[string]rb.Value) ([]plot.Option, error) {
	var opts []plot.Option
	for _, k := range sortedKeys(kw) {
		x := kw[k].Go()
		key := normalize(k)
		switch key {
		case "bins":
			if _, ok := x.(int64); !ok {
				key = "binedges"
			}
		case "levels":
			if _, ok := x.(int64); ok {
				key = "nlevels"
			}
		case "color":
			if l, ok := x.([]any); ok {
				key = "rgba"
				if len(l) == 3 {
					x = append(l, 1.0)
				}
			}
		case "arrowprops":
			x = x != nil && x != false
		}
		fn, ok := plOptions[key]
		if !ok {
			return nil, argError("unknown keyword: :%s (known: %s)", k, plKnown)
		}
		o, err := plOption(k, fn, x)
		if err != nil {
			return nil, err
		}
		if o != nil {
			opts = append(opts, o)
		}
	}
	return opts, nil
}

// plOption calls one option constructor with a keyword's value: true for one
// without parameters (false gives no option), an Array (or one value) for a
// variadic one, an Array of as many values for one of several parameters,
// plot data for an any parameter, and the value converted otherwise.
func plOption(k string, fn reflect.Value, x any) (plot.Option, error) {
	ft := fn.Type()
	var args []reflect.Value
	switch {
	case ft.NumIn() == 0:
		b, ok := x.(bool)
		if !ok {
			return nil, typeError(x, "true or false")
		}
		if !b {
			return nil, nil
		}
	case ft.IsVariadic():
		list, ok := x.([]any)
		if !ok {
			list = []any{x}
		}
		for _, e := range list {
			v, err := convert(e, ft.In(0).Elem())
			if err != nil {
				return nil, err
			}
			args = append(args, v)
		}
	case ft.NumIn() > 1:
		list, ok := x.([]any)
		if !ok || len(list) != ft.NumIn() {
			return nil, argError("%s: expected an Array of %d values", k, ft.NumIn())
		}
		for i, e := range list {
			v, err := convert(e, ft.In(i))
			if err != nil {
				return nil, err
			}
			args = append(args, v)
		}
	case ft.In(0).Kind() == reflect.Interface:
		d, err := plData(x)
		if err != nil {
			return nil, err
		}
		args = append(args, reflect.ValueOf(&d).Elem())
	default:
		v, err := convert(x, ft.In(0))
		if err != nil {
			return nil, err
		}
		args = append(args, v)
	}
	var o plot.Option
	err := plProtect(func() { o = fn.Call(args)[0].Interface().(plot.Option) })
	return o, err
}

func plAxes(c *rb.Call) *plot.Axes  { return c.Self.Go().(*plot.Axes) }
func plFig(c *rb.Call) *plot.Figure { return c.Self.Go().(*plot.Figure) }

// plCall reads min..max positional arguments as plot data and the keywords
// as options.
func plCall(c *rb.Call, min, max int) ([]any, []plot.Option, error) {
	if err := arity(c, min, max); err != nil {
		return nil, nil, err
	}
	data := make([]any, len(c.Args))
	for i, a := range c.Args {
		d, err := plData(a.Go())
		if err != nil {
			return nil, nil, err
		}
		data[i] = d
	}
	opts, err := plOpts(c.Kwargs)
	return data, opts, err
}

// plMappable is a Mappable for Ruby: nil when go-plot gave none.
func plMappable(m *plot.Mappable) any {
	if m == nil {
		return nil
	}
	return m
}

// plDraw binds a plotting method of fixed data arity n.
func plDraw(n int, draw func(a *plot.Axes, d []any, opts []plot.Option)) rb.Func {
	return func(c *rb.Call) (any, error) {
		d, opts, err := plCall(c, n, n)
		if err != nil {
			return nil, err
		}
		return nil, plProtect(func() { draw(plAxes(c), d, opts) })
	}
}

// plMap binds a method that returns a Mappable, taking a single z or x, y, z
// (contour) or just data (imshow, scatter).
func plMap(counts []int, draw func(a *plot.Axes, d []any, opts []plot.Option) *plot.Mappable) rb.Func {
	return func(c *rb.Call) (any, error) {
		d, opts, err := plCall(c, counts[0], counts[len(counts)-1])
		if err != nil {
			return nil, err
		}
		if len(counts) == 2 && len(d) != counts[0] && len(d) != counts[1] {
			return nil, argError("wrong number of arguments (given %d, expected %d or %d)", len(d), counts[0], counts[1])
		}
		if len(d) == 1 && len(counts) == 2 {
			d = []any{nil, nil, d[0]}
		}
		var m *plot.Mappable
		if err := plProtect(func() { m = draw(plAxes(c), d, opts) }); err != nil {
			return nil, err
		}
		return plMappable(m), nil
	}
}

// plFloats reads n numbers from the positional arguments, or from a single
// Array of n.
func plFloats(c *rb.Call, n int) ([]float64, error) {
	if err := plNoKw(c); err != nil {
		return nil, err
	}
	args := make([]any, len(c.Args))
	for i, a := range c.Args {
		args[i] = a.Go()
	}
	if len(args) == 1 {
		if l, ok := args[0].([]any); ok {
			args = l
		}
	}
	if len(args) != n {
		return nil, argError("wrong number of arguments (given %d, expected %d)", len(args), n)
	}
	out := make([]float64, n)
	for i, a := range args {
		f, err := toFloat(a)
		if err != nil {
			return nil, err
		}
		out[i] = f
	}
	return out, nil
}

// plString binds a setter of one String.
func plString(set func(a *plot.Axes, s string)) rb.Func {
	return func(c *rb.Call) (any, error) {
		if err := plExact(c, 1); err != nil {
			return nil, err
		}
		s, err := toString(c.Args[0].Go())
		if err != nil {
			return nil, err
		}
		set(plAxes(c), s)
		return nil, nil
	}
}

// plSetLim binds set_xlim and its kind: (lo, hi) or ([lo, hi]).
func plSetLim(set func(a *plot.Axes, lo, hi float64)) rb.Func {
	return func(c *rb.Call) (any, error) {
		f, err := plFloats(c, 2)
		if err != nil {
			return nil, err
		}
		set(plAxes(c), f[0], f[1])
		return nil, nil
	}
}

// plLim binds xlim: with no argument the limits, with two (or an Array) a
// setter.
func plLim(get func(a *plot.Axes) (float64, float64), set func(a *plot.Axes, lo, hi float64)) rb.Func {
	setter := plSetLim(set)
	return func(c *rb.Call) (any, error) {
		if len(c.Args) == 0 && len(c.Kwargs) == 0 {
			lo, hi := get(plAxes(c))
			return []any{lo, hi}, nil
		}
		return setter(c)
	}
}

// plTicks binds set_xticks(locs, labels = nil).
func plTicks(set func(a *plot.Axes, locs []float64, labels ...string)) rb.Func {
	return func(c *rb.Call) (any, error) {
		if err := plNoKw(c); err != nil {
			return nil, err
		}
		if err := arity(c, 1, 2); err != nil {
			return nil, err
		}
		locs, err := toFloats(c.Args[0].Go())
		if err != nil {
			return nil, err
		}
		var labels []string
		if len(c.Args) == 2 {
			v, err := convert(c.Args[1].Go(), reflect.TypeFor[[]string]())
			if err != nil {
				return nil, err
			}
			labels = v.Interface().([]string)
		}
		set(plAxes(c), locs, labels...)
		return nil, nil
	}
}

func plSetAspect(c *rb.Call) (any, error) {
	if err := plExact(c, 1); err != nil {
		return nil, err
	}
	var r float64
	switch x := c.Args[0].Go().(type) {
	case rb.Symbol, string:
		s, _ := toString(x)
		switch s {
		case "equal":
			r = 1
		case "auto":
			r = 0
		default:
			return nil, argError("unknown aspect %q (expected a number, :equal or :auto)", s)
		}
	default:
		f, err := toFloat(x)
		if err != nil {
			return nil, err
		}
		r = f
	}
	plAxes(c).SetAspect(r)
	return nil, nil
}

func plGridOn(c *rb.Call) (any, error) {
	if err := plNoKw(c); err != nil {
		return nil, err
	}
	if err := arity(c, 0, 1); err != nil {
		return nil, err
	}
	on := true
	if len(c.Args) == 1 {
		b, ok := c.Args[0].Go().(bool)
		if !ok {
			return nil, typeError(c.Args[0].Go(), "true or false")
		}
		on = b
	}
	plAxes(c).Grid(on)
	return nil, nil
}

// plLine binds axhline and axvline: (at = 0, **opts).
func plLine(draw func(a *plot.Axes, at float64, opts ...plot.Option)) rb.Func {
	return func(c *rb.Call) (any, error) {
		if err := arity(c, 0, 1); err != nil {
			return nil, err
		}
		at := 0.0
		if len(c.Args) == 1 {
			f, err := toFloat(c.Args[0].Go())
			if err != nil {
				return nil, err
			}
			at = f
		}
		opts, err := plOpts(c.Kwargs)
		if err != nil {
			return nil, err
		}
		return nil, plProtect(func() { draw(plAxes(c), at, opts...) })
	}
}

func plText(c *rb.Call) (any, error) {
	if err := arity(c, 3, 3); err != nil {
		return nil, err
	}
	x, err := toFloat(c.Args[0].Go())
	if err != nil {
		return nil, err
	}
	y, err := toFloat(c.Args[1].Go())
	if err != nil {
		return nil, err
	}
	s, err := toString(c.Args[2].Go())
	if err != nil {
		return nil, err
	}
	opts, err := plOpts(c.Kwargs)
	if err != nil {
		return nil, err
	}
	return nil, plProtect(func() { plAxes(c).Text(x, y, s, opts...) })
}

// plPoint reads an [x, y] pair.
func plPoint(v any) ([]float64, error) {
	p, err := toFloats(v)
	if err == nil && len(p) != 2 {
		err = argError("a point is [x, y], got %d values", len(p))
	}
	return p, err
}

func plAnnotate(c *rb.Call) (any, error) {
	if err := arity(c, 2, 3); err != nil {
		return nil, err
	}
	s, err := toString(c.Args[0].Go())
	if err != nil {
		return nil, err
	}
	xy, err := plPoint(c.Args[1].Go())
	if err != nil {
		return nil, err
	}
	txy := xy
	if len(c.Args) == 3 {
		if txy, err = plPoint(c.Args[2].Go()); err != nil {
			return nil, err
		}
	}
	opts, err := plOpts(c.Kwargs)
	if err != nil {
		return nil, err
	}
	return nil, plProtect(func() { plAxes(c).Annotate(s, xy[0], xy[1], txy[0], txy[1], opts...) })
}

func plPlot(c *rb.Call) (any, error) {
	d, opts, err := plCall(c, 1, 2)
	if err != nil {
		return nil, err
	}
	if len(d) == 1 {
		d = []any{nil, d[0]}
	}
	return nil, plProtect(func() { plAxes(c).Plot(d[0], d[1], opts...) })
}

func plFillBetween(c *rb.Call) (any, error) {
	d, opts, err := plCall(c, 2, 3)
	if err != nil {
		return nil, err
	}
	if len(d) == 2 {
		d = append(d, nil)
	}
	return nil, plProtect(func() { plAxes(c).FillBetween(d[0], d[1], d[2], opts...) })
}

var plAxesMethods = map[string]rb.Func{
	"plot": plPlot,
	"scatter": plMap([]int{2}, func(a *plot.Axes, d []any, o []plot.Option) *plot.Mappable {
		return a.Scatter(d[0], d[1], o...)
	}),
	"hist": plDraw(1, func(a *plot.Axes, d []any, o []plot.Option) { a.Hist(d[0], o...) }),
	"bar":  plDraw(2, func(a *plot.Axes, d []any, o []plot.Option) { a.Bar(d[0], d[1], o...) }),
	"barh": plDraw(2, func(a *plot.Axes, d []any, o []plot.Option) { a.Barh(d[0], d[1], o...) }),
	"errorbar": plDraw(2, func(a *plot.Axes, d []any, o []plot.Option) {
		a.ErrorBar(d[0], d[1], o...)
	}),
	"fill_between": plFillBetween,
	"axhline":      plLine((*plot.Axes).Axhline),
	"axvline":      plLine((*plot.Axes).Axvline),
	"text":         plText,
	"annotate":     plAnnotate,
	"imshow": plMap([]int{1}, func(a *plot.Axes, d []any, o []plot.Option) *plot.Mappable {
		return a.Imshow(d[0], o...)
	}),
	"contour": plMap([]int{1, 3}, func(a *plot.Axes, d []any, o []plot.Option) *plot.Mappable {
		return a.Contour(d[0], d[1], d[2], o...)
	}),
	"contourf": plMap([]int{1, 3}, func(a *plot.Axes, d []any, o []plot.Option) *plot.Mappable {
		return a.Contourf(d[0], d[1], d[2], o...)
	}),
	"legend":     plDraw(0, func(a *plot.Axes, _ []any, o []plot.Option) { a.Legend(o...) }),
	"grid":       plGridOn,
	"set_title":  plString((*plot.Axes).SetTitle),
	"title=":     plString((*plot.Axes).SetTitle),
	"set_xlabel": plString((*plot.Axes).SetXLabel),
	"xlabel=":    plString((*plot.Axes).SetXLabel),
	"set_ylabel": plString((*plot.Axes).SetYLabel),
	"ylabel=":    plString((*plot.Axes).SetYLabel),
	"set_xscale": plString((*plot.Axes).SetXScale),
	"xscale=":    plString((*plot.Axes).SetXScale),
	"set_yscale": plString((*plot.Axes).SetYScale),
	"yscale=":    plString((*plot.Axes).SetYScale),
	"set_xlim":   plSetLim((*plot.Axes).SetXLim),
	"xlim=":      plSetLim((*plot.Axes).SetXLim),
	"set_ylim":   plSetLim((*plot.Axes).SetYLim),
	"ylim=":      plSetLim((*plot.Axes).SetYLim),
	"xlim":       plLim((*plot.Axes).XLim, (*plot.Axes).SetXLim),
	"ylim":       plLim((*plot.Axes).YLim, (*plot.Axes).SetYLim),
	"set_xticks": plTicks((*plot.Axes).SetXTicks),
	"set_yticks": plTicks((*plot.Axes).SetYTicks),
	"set_aspect": plSetAspect,
	"aspect=":    plSetAspect,
	"twinx": func(c *rb.Call) (any, error) {
		if err := plExact(c, 0); err != nil {
			return nil, err
		}
		return plAxes(c).TwinX(), nil
	},
	"figure": func(c *rb.Call) (any, error) {
		if err := plExact(c, 0); err != nil {
			return nil, err
		}
		return plAxes(c).Figure(), nil
	},
}

// plSVG and plPNG render a figure; writing to memory cannot fail.
func plSVG(f *plot.Figure) []byte {
	var b bytes.Buffer
	_ = f.SaveSVG(&b)
	return b.Bytes()
}

func plPNG(f *plot.Figure) []byte {
	var b bytes.Buffer
	_ = f.SavePNG(&b)
	return b.Bytes()
}

// plSave binds a method writing the figure to the file at a path.
func plSave(render func(f *plot.Figure, path string) ([]byte, error)) rb.Func {
	return func(c *rb.Call) (any, error) {
		if err := plExact(c, 1); err != nil {
			return nil, err
		}
		path, err := toString(c.Args[0].Go())
		if err != nil {
			return nil, err
		}
		b, err := render(plFig(c), path)
		if err != nil {
			return nil, err
		}
		if err := os.WriteFile(path, b, 0o644); err != nil {
			return nil, rb.NewError("IOError", err.Error())
		}
		return nil, nil
	}
}

// plFigNoArgs binds a figure method without arguments.
func plFigNoArgs(f func(*plot.Figure) any) rb.Func {
	return func(c *rb.Call) (any, error) {
		if err := plExact(c, 0); err != nil {
			return nil, err
		}
		return f(plFig(c)), nil
	}
}

func plInspect(f *plot.Figure) any {
	w, h := f.PixelSize()
	return fmt.Sprintf("#<Plot::Figure %dx%d>", w, h)
}

func plColorbar(c *rb.Call) (any, error) {
	mine, rest := split(c.Kwargs, "ax")
	if len(rest) > 0 {
		return nil, argError("unknown keyword: :%s (known: ax)", sortedKeys(rest)[0])
	}
	if err := arity(c, 1, 2); err != nil {
		return nil, err
	}
	m, ok := c.Args[0].Go().(*plot.Mappable)
	if !ok {
		return nil, typeError(c.Args[0].Go(), "Plot::Mappable")
	}
	axv, given := mine["ax"]
	if len(c.Args) == 2 {
		axv, given = c.Args[1], true
	}
	if !given {
		return nil, argError("colorbar needs the Axes: colorbar(mappable, ax)")
	}
	ax, ok := axv.Go().(*plot.Axes)
	if !ok {
		return nil, typeError(axv.Go(), "Plot::Axes")
	}
	return plFig(c).Colorbar(m, ax), nil
}

func plAddAxes(c *rb.Call) (any, error) {
	f, err := plFloats(c, 4)
	if err != nil {
		return nil, err
	}
	return plFig(c).AddAxes(f[0], f[1], f[2], f[3]), nil
}

func plFigSubplots(c *rb.Call) (any, error) {
	rows, cols, squeeze, rest, err := plGrid(c)
	if err != nil {
		return nil, err
	}
	if len(rest) > 0 {
		return nil, argError("unknown keyword: :%s (known: squeeze)", sortedKeys(rest)[0])
	}
	return plSqueeze(plFig(c).Subplots(rows, cols), squeeze), nil
}

var plFigureMethods = map[string]rb.Func{
	"width":  plFigNoArgs(func(f *plot.Figure) any { return f.Width }),
	"height": plFigNoArgs(func(f *plot.Figure) any { return f.Height }),
	"dpi":    plFigNoArgs(func(f *plot.Figure) any { return f.DPI }),
	"pixel_size": plFigNoArgs(func(f *plot.Figure) any {
		w, h := f.PixelSize()
		return []any{w, h}
	}),
	"axes":         plFigNoArgs(func(f *plot.Figure) any { return f.Axes() }),
	"tight_layout": plFigNoArgs(func(f *plot.Figure) any { return f.TightLayout() }),
	"to_svg":       plFigNoArgs(func(f *plot.Figure) any { return string(plSVG(f)) }),
	"to_png":       plFigNoArgs(func(f *plot.Figure) any { return plPNG(f) }),
	"display":      plFigNoArgs(func(f *plot.Figure) any { return f.Display() }),
	"inspect":      plFigNoArgs(plInspect),
	"to_s":         plFigNoArgs(plInspect),
	"add_axes":     plAddAxes,
	"subplots":     plFigSubplots,
	"colorbar":     plColorbar,
	"save_svg": plSave(func(f *plot.Figure, _ string) ([]byte, error) {
		return plSVG(f), nil
	}),
	"save_png": plSave(func(f *plot.Figure, _ string) ([]byte, error) {
		return plPNG(f), nil
	}),
	"savefig": plSave(func(f *plot.Figure, path string) ([]byte, error) {
		switch strings.ToLower(filepath.Ext(path)) {
		case ".svg":
			return plSVG(f), nil
		case ".png":
			return plPNG(f), nil
		}
		return nil, argError("savefig: unknown format of %q (expected .svg or .png)", path)
	}),
}
