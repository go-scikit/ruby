package ruby

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/go-dataframe/dataframe"
	rb "github.com/go-embedded-ruby/ruby"
	nd "github.com/go-ndarray/ndarray"
	"github.com/go-plot/plot"
)

// plotIs checks a Ruby program ending with a Plot::Figure draws the very
// figure build draws in Go: the same SVG bytes.
func plotIs(t *testing.T, in *rb.Interp, src string, build func() *plot.Figure) {
	t.Helper()
	v, err := in.Eval(src, "test.rb", 1)
	if err != nil {
		t.Fatalf("%s: %v", src, err)
	}
	f, ok := v.Go().(*plot.Figure)
	if !ok {
		t.Fatalf("%s: %s is not a Plot::Figure", src, v.Inspect())
	}
	svg, err := v.Call("to_svg")
	if err != nil {
		t.Fatal(err)
	}
	want := plSVG(build())
	if svg.Go().(string) != string(want) || !bytes.Equal(plSVG(f), want) {
		t.Errorf("%s: the SVG differs from Go's", src)
	}
}

// one is a 1x1 figure and its Axes, as Plot.subplots gives them.
func one(opts ...plot.FigOption) (*plot.Figure, *plot.Axes) {
	f, axs := plot.Subplots(1, 1, opts...)
	return f, axs[0][0]
}

func TestPlotAxes(t *testing.T) {
	in := newInterp(t)
	x := []float64{1, 2, 3, 4}
	y := []float64{1, 4, 9, 16}
	eval(t, in, `X = [1, 2, 3, 4]; Y = [1, 4, 9, 16]`)
	plotIs(t, in, `fig, ax = Plot.subplots(figsize: [5, 3], dpi: 80)
		ax.plot(Y)
		ax.plot(X, Y, alpha: 0.5, clip: false, color: "C2", ec: "k", fc: "w", label: "sq",
			linestyle: "--", linewidth: 2, marker: "o", markersize: 4, mew: 1, z_order: 3)
		ax.legend(loc: "upper left", fontsize: 9)
		ax.grid
		ax.grid(false)
		ax.grid(true)
		ax.title = "T"
		ax.set_title("Title $x^2$")
		ax.set_xlabel("x"); ax.xlabel = "xx"
		ax.set_ylabel("y"); ax.ylabel = "yy"
		fig`, func() *plot.Figure {
		f, ax := one(plot.DPI(80), plot.FigSize(5, 3))
		ax.Plot(nil, y)
		ax.Plot(x, y, plot.Alpha(0.5), plot.Clip(false), plot.Color("C2"), plot.EdgeColor("k"), plot.FaceColor("w"),
			plot.Label("sq"), plot.LineStyle("--"), plot.LineWidth(2), plot.Marker("o"), plot.MarkerSize(4),
			plot.MarkerEdgeWidth(1), plot.ZOrder(3))
		ax.Legend(plot.FontSize(9), plot.Loc("upper left"))
		ax.Grid(true)
		ax.Grid(false)
		ax.Grid(true)
		ax.SetTitle("T")
		ax.SetTitle("Title $x^2$")
		ax.SetXLabel("x")
		ax.SetXLabel("xx")
		ax.SetYLabel("y")
		ax.SetYLabel("yy")
		return f
	})
	plotIs(t, in, `fig, ax = Plot.subplots
		m = ax.scatter(X, Y, c: [1, 2, 3, 4], cmap: "plasma", s: [10, 20, 30, 40], vmax: 4, vmin: 0)
		fig.colorbar(m, ax)
		m2 = ax.scatter(X, Y, colors: ["red", "blue", "green", "k"], edgecolors: "k", log_norm: false)
		fig.colorbar(ax.scatter(X, Y, c: NDArray.linspace(1, 4, 4), log_norm: true), ax: ax)
		raise "scatter without c gives a Mappable" unless m2.nil?
		fig`, func() *plot.Figure {
		f, ax := one()
		m := ax.Scatter(x, y, plot.CValues([]float64{1, 2, 3, 4}), plot.Cmap("plasma"), plot.Sizes([]float64{10, 20, 30, 40}),
			plot.Vmax(4), plot.Vmin(0))
		f.Colorbar(m, ax)
		ax.Scatter(x, y, plot.Colors("red", "blue", "green", "k"), plot.EdgeColor("k"))
		lin, _ := nd.FromData([]float64{1, 2, 3, 4}, 4)
		f.Colorbar(ax.Scatter(x, y, plot.CValues(lin), plot.LogNorm()), ax)
		return f
	})
	plotIs(t, in, `fig, ax = Plot.subplots
		ax.hist([1, 2, 2, 3, 3, 3, 4], bins: 3, density: true, edgecolor: "k")
		ax.hist([1, 2, 2, 3], bins: [0, 2, 4], color: "red", alpha: 0.5)
		ax.bar(X, Y, align: "edge", bottom: 1, colors: ["red", "green", "blue", "k"], width: 0.5)
		ax.barh(X, Y, height: 0.4, left: 2, color: [0.2, 0.4, 0.6])
		ax.bar(X, Y, colors: "red", color: [1, 0, 0, 0.5])
		fig`, func() *plot.Figure {
		f, ax := one()
		ax.Hist([]float64{1, 2, 2, 3, 3, 3, 4}, plot.Bins(3), plot.Density(), plot.EdgeColor("k"))
		ax.Hist([]float64{1, 2, 2, 3}, plot.BinEdges(0, 2, 4), plot.Color("red"), plot.Alpha(0.5))
		ax.Bar(x, y, plot.Align("edge"), plot.Bottom(1), plot.Colors("red", "green", "blue", "k"), plot.Width(0.5))
		ax.Barh(x, y, plot.Width(0.4), plot.Bottom(2), plot.RGBA(0.2, 0.4, 0.6, 1))
		ax.Bar(x, y, plot.RGBA(1, 0, 0, 0.5), plot.Colors("red"))
		return f
	})
	plotIs(t, in, `fig, ax = Plot.subplots
		ax.fill_between(X, Y)
		ax.fill_between(X, Y, [0, 1, 2, 3], where: [true, false, true, true], alpha: 0.3, label: "band")
		ax.errorbar(X, Y, capsize: 3, ecolor: "red", xerr: [0.1, 0.2, 0.1, 0.2], yerr: 0.5)
		ax.axhline
		ax.axhline(2, color: "k", ls: "--", lw: 1)
		ax.axvline(1.5)
		ax.text(1, 2, "hi", fontsize: 12, ha: "center", rotation: 45, va: "top")
		ax.annotate("pt", [2, 4], [3, 8], arrowprops: {})
		ax.annotate("p", [1, 1])
		ax.annotate("q", [1, 1], [2, 2], arrow: true, arrowprops: nil)
		fig`, func() *plot.Figure {
		f, ax := one()
		ax.FillBetween(x, y, nil)
		ax.FillBetween(x, y, []float64{0, 1, 2, 3}, plot.Alpha(0.3), plot.Label("band"), plot.Where([]bool{true, false, true, true}))
		ax.ErrorBar(x, y, plot.CapSize(3), plot.ErrorColor("red"), plot.XErr([]float64{0.1, 0.2, 0.1, 0.2}), plot.YErr(0.5))
		ax.Axhline(0)
		ax.Axhline(2, plot.Color("k"), plot.LineStyle("--"), plot.LineWidth(1))
		ax.Axvline(1.5)
		ax.Text(1, 2, "hi", plot.FontSize(12), plot.HA("center"), plot.Rotation(45), plot.VA("top"))
		ax.Annotate("pt", 2, 4, 3, 8, plot.Arrow())
		ax.Annotate("p", 1, 1, 1, 1)
		ax.Annotate("q", 1, 1, 2, 2, plot.Arrow())
		return f
	})
	z := [][]float64{{1, 2, 3}, {4, 5, 6}, {7, 8, 10}}
	rgb, _ := nd.FromData([]float64{1, 0, 0, 0, 1, 0, 0, 0, 1, 1, 1, 1}, 2, 2, 3)
	plotIs(t, in, `fig, (a, b, c) = Plot.subplots(1, 3)
		Z = [[1, 2, 3], [4, 5, 6], [7, 8, 10]]
		m = a.imshow(Z, aspect: "auto", cmap: "gray", extent: [0, 1, 0, 1], origin: "lower")
		fig.colorbar(m, a)
		b.imshow([[[1, 0, 0], [0, 1, 0]], [[0, 0, 1], [1, 1, 1]]])
		c.contour(Z, levels: 4)
		cs = c.contourf([0, 1, 2], [0, 1, 2], Z, levels: [0, 5, 10], alpha: 0.5)
		fig.colorbar(cs, c)
		fig`, func() *plot.Figure {
		f, axs := plot.Subplots(1, 3)
		a, b, c := axs[0][0], axs[0][1], axs[0][2]
		m := a.Imshow(z, plot.Aspect("auto"), plot.Cmap("gray"), plot.Extent(0, 1, 0, 1), plot.Origin("lower"))
		f.Colorbar(m, a)
		b.Imshow(rgb)
		c.Contour(nil, nil, z, plot.NLevels(4))
		cs := c.Contourf([]float64{0, 1, 2}, []float64{0, 1, 2}, z, plot.Alpha(0.5), plot.Levels(0, 5, 10))
		f.Colorbar(cs, c)
		return f
	})
	plotIs(t, in, `fig, ax = Plot.subplots
		ax.plot(X, Y)
		ax.set_xlim(0, 10); ax.xlim = [0, 5]; ax.xlim(1, 6)
		ax.set_ylim([1, 20]); ax.ylim = [0, 30]; ax.ylim(1, 40)
		ax.set_xscale("log"); ax.xscale = "linear"
		ax.set_yscale("linear"); ax.yscale = "log"
		ax.set_xticks([1, 2], ["a", "b"])
		ax.set_yticks([1, 10])
		ax.set_aspect(:equal); ax.aspect = 2; ax.set_aspect("auto"); ax.set_aspect(1)
		t = ax.twinx
		t.plot(X, [4, 3, 2, 1], color: "red")
		raise "xlim" unless ax.xlim == [1.0, 6.0] && ax.ylim == [1.0, 40.0]
		fig`, func() *plot.Figure {
		f, ax := one()
		ax.Plot(x, y)
		ax.SetXLim(0, 10)
		ax.SetXLim(0, 5)
		ax.SetXLim(1, 6)
		ax.SetYLim(1, 20)
		ax.SetYLim(0, 30)
		ax.SetYLim(1, 40)
		ax.SetXScale("log")
		ax.SetXScale("linear")
		ax.SetYScale("linear")
		ax.SetYScale("log")
		ax.SetXTicks([]float64{1, 2}, "a", "b")
		ax.SetYTicks([]float64{1, 10})
		ax.SetAspect(1)
		ax.SetAspect(2)
		ax.SetAspect(0)
		ax.SetAspect(1)
		tw := ax.TwinX()
		tw.Plot(x, []float64{4, 3, 2, 1}, plot.Color("red"))
		return f
	})
	// Data as an NDArray and as a DataFrame::Series.
	plotIs(t, in, `fig, ax = Plot.subplots
		ax.plot(NDArray.linspace(0, 1, 5), DataFrame::Series.new([1, 3, 2, 5, 4]))
		ax.plot(DataFrame::Series.new([1.5, 2.5]))
		fig`, func() *plot.Figure {
		f, ax := one()
		lin, _ := nd.FromData([]float64{0, 0.25, 0.5, 0.75, 1}, 5)
		s, _ := dataframe.NewSeries("", []any{int64(1), int64(3), int64(2), int64(5), int64(4)})
		sf, _ := s.Floats()
		ax.Plot(lin, sf)
		ax.Plot(nil, []float64{1.5, 2.5})
		return f
	})
}

func TestPlotFigure(t *testing.T) {
	in := newInterp(t)
	plotIs(t, in, `fig = Plot.figure(figsize: [4, 3], dpi: 50)
		a = fig.add_axes(0.1, 0.1, 0.8, 0.8)
		b = fig.add_axes([0.5, 0.5, 0.3, 0.3])
		a.plot([1, 2])
		b.plot([2, 1])
		g = fig.subplots(2, 1)
		g[0].plot([1, 2, 3])
		raise "axes" unless fig.axes.size == 4
		raise "tight" unless fig.tight_layout == true
		fig`, func() *plot.Figure {
		f := plot.NewFigure(plot.DPI(50), plot.FigSize(4, 3))
		a := f.AddAxes(0.1, 0.1, 0.8, 0.8)
		b := f.AddAxes(0.5, 0.5, 0.3, 0.3)
		a.Plot(nil, []float64{1, 2})
		b.Plot(nil, []float64{2, 1})
		g := f.Subplots(2, 1)
		g[0][0].Plot(nil, []float64{1, 2, 3})
		f.TightLayout()
		return f
	})
	plotIs(t, in, `Plot::Figure.new`, func() *plot.Figure { return plot.NewFigure() })
	plotIs(t, in, `f = Plot::Figure.new; f.subplots(squeeze: false)[0][0].plot([1]); f`, func() *plot.Figure {
		f := plot.NewFigure()
		f.Subplots(1, 1)[0][0].Plot(nil, []float64{1})
		return f
	})

	f, ax := one()
	ax.Plot(nil, []float64{3, 1, 2})
	eval(t, in, `fig, ax = Plot.subplots; ax.plot([3, 1, 2])`)
	check(t, in, `fig.width`, f.Width)
	check(t, in, `fig.height`, f.Height)
	check(t, in, `fig.dpi`, f.DPI)
	w, h := f.PixelSize()
	check(t, in, `fig.pixel_size`, []int{w, h})
	check(t, in, `fig.inspect`, "#<Plot::Figure 640x480>")
	check(t, in, `fig.to_s`, "#<Plot::Figure 640x480>")
	check(t, in, `fig.to_png.bytes`, plPNG(f))
	disp := f.Display()
	check(t, in, `fig.display["image/svg+xml"]`, string(disp["image/svg+xml"]))
	check(t, in, `fig.display["image/png"].bytes`, disp["image/png"])
	check(t, in, `Plot.subplots(1, 1, squeeze: false)[1].size`, 1)
	check(t, in, `Plot.subplots(1, 3)[1].map(&:class).uniq.map(&:to_s)`, []string{"Plot::Axes"})
	check(t, in, `Plot.subplots(3, 1)[1].size`, 3)
	check(t, in, `Plot.subplots(2, 2)[1].map(&:size)`, []int{2, 2})
	check(t, in, `Plot.subplots(2)[1].size`, 2)
	check(t, in, `Plot.subplots[1].imshow([[1, 2], [3, 4]]).inspect`, "#<Plot::Mappable viridis>")
	if eval(t, in, `ax.figure`) != eval(t, in, `fig`) {
		t.Error("Axes#figure is not the figure")
	}
	check(t, in, `Plot.values([1, 2])`, plot.Values([]float64{1, 2}))
	check(t, in, `Plot.values(NDArray.linspace(0, 1, 3))`, []float64{0, 0.5, 1})
	check(t, in, `Plot.values(3)`, plot.Values(3.0))

	// Saving writes the bytes to_svg and to_png give.
	dir := t.TempDir()
	if err := in.Set("dir", dir); err != nil {
		t.Fatal(err)
	}
	eval(t, in, `fig.save_svg(File.join(dir, "a.svg")); fig.save_png(File.join(dir, "a.png"))
		fig.savefig(File.join(dir, "b.SVG")); fig.savefig(File.join(dir, "b.png"))`)
	for name, want := range map[string][]byte{"a.svg": plSVG(f), "b.SVG": plSVG(f), "a.png": plPNG(f), "b.png": plPNG(f)} {
		got, err := os.ReadFile(filepath.Join(dir, name))
		if err != nil || !bytes.Equal(got, want) {
			t.Errorf("%s: not the figure's bytes (%v)", name, err)
		}
	}
	evalErr(t, in, `fig.savefig(File.join(dir, "a.pdf"))`, "ArgumentError", "unknown format")
	evalErr(t, in, `fig.save_svg(File.join(dir, "no", "such", "a.svg"))`, "IOError", "a.svg")
	evalErr(t, in, `fig.save_svg(1)`, "TypeError", "String")
	evalErr(t, in, `fig.save_svg`, "ArgumentError", "given 0, expected 1")
}

func TestPlotErrors(t *testing.T) {
	in := newInterp(t)
	eval(t, in, `fig, ax = Plot.subplots; s = DataFrame::Series.new(["a"])`)
	for _, c := range []struct{ src, class, msg string }{
		{`ax.plot`, "ArgumentError", "given 0, expected 1..2"},
		{`ax.plot([1], bogus: 1)`, "ArgumentError", "unknown keyword: :bogus (known: align, alpha,"},
		{`ax.plot([1], color: "nonsense")`, "ArgumentError", "not a colour"},
		{`ax.plot([1], marker: "?")`, "ArgumentError", "unknown marker"},
		{`ax.plot([1, 2], [1])`, "ArgumentError", "same length"},
		{`ax.plot(["a"])`, "TypeError", "String into Float"},
		{`ax.plot(:a)`, "TypeError", "Symbol into plot data"},
		{`ax.plot(s)`, "ArgumentError", "dtype"},
		{`ax.imshow([[1, 2], [3]])`, "ArgumentError", "rectangular"},
		{`ax.imshow([[1, 2], 3])`, "ArgumentError", "rectangular"},
		{`ax.imshow([[[1], ["a"]]])`, "TypeError", "String into Float"},
		{`ax.imshow([[]])`, "TypeError", "Array into Float"},
		{`ax.hist([1], bins: "x")`, "TypeError", "String into Float"},
		{`ax.hist([1], density: 1)`, "TypeError", "true or false"},
		{`ax.imshow([[1]], extent: [1])`, "ArgumentError", "extent: expected an Array of 4 values"},
		{`ax.imshow([[1]], extent: [1, 2, 3, "a"])`, "TypeError", "String into Float"},
		{`ax.scatter([1], [1], c: :x)`, "TypeError", "Symbol into plot data"},
		{`ax.plot([1], alpha: "a")`, "TypeError", "String into Float"},
		{`ax.hist`, "ArgumentError", "given 0, expected 1"},
		{`ax.hist([1], bogus: 1)`, "ArgumentError", "unknown keyword: :bogus"},
		{`ax.scatter([1])`, "ArgumentError", "given 1, expected 2"},
		{`ax.scatter([1], [1, 2])`, "ArgumentError", "same length"},
		{`ax.contour([[1]], [[1]])`, "ArgumentError", "given 2, expected 1 or 3"},
		{`ax.contour([1, 2], [1], [[1, 2], [3, 4]])`, "ArgumentError", "must match"},
		{`ax.fill_between([1])`, "ArgumentError", "given 1, expected 2..3"},
		{`ax.fill_between([1, 2], [1])`, "ArgumentError", "same length"},
		{`ax.axhline(1, 2)`, "ArgumentError", "given 2, expected 0..1"},
		{`ax.axhline("a")`, "TypeError", "String into Float"},
		{`ax.axhline(1, bogus: 1)`, "ArgumentError", "unknown keyword"},
		{`ax.axhline(1, ls: "~")`, "ArgumentError", "unknown line style"},
		{`ax.text(1, 2)`, "ArgumentError", "given 2, expected 3"},
		{`ax.text("a", 2, "s")`, "TypeError", "String into Float"},
		{`ax.text(1, "a", "s")`, "TypeError", "String into Float"},
		{`ax.text(1, 2, 3)`, "TypeError", "Integer into String"},
		{`ax.text(1, 2, "s", bogus: 1)`, "ArgumentError", "unknown keyword"},
		{`ax.text(1, 2, "s", color: "nonsense")`, "ArgumentError", "not a colour"},
		{`ax.annotate("s")`, "ArgumentError", "given 1, expected 2..3"},
		{`ax.annotate(1, [1, 2])`, "TypeError", "Integer into String"},
		{`ax.annotate("s", [1])`, "ArgumentError", "a point is [x, y]"},
		{`ax.annotate("s", "a")`, "TypeError", "String into an Array"},
		{`ax.annotate("s", [1, 2], [1])`, "ArgumentError", "a point is [x, y]"},
		{`ax.annotate("s", [1, 2], bogus: 1)`, "ArgumentError", "unknown keyword"},
		{`ax.annotate("s", [1, 2], color: "nonsense")`, "ArgumentError", "not a colour"},
		{`ax.legend(loc: "nowhere")`, "ArgumentError", "unknown legend location"},
		{`ax.legend(1)`, "ArgumentError", "given 1, expected 0"},
		{`ax.grid(1)`, "TypeError", "true or false"},
		{`ax.grid(true, false)`, "ArgumentError", "given 2, expected 0..1"},
		{`ax.grid(on: true)`, "ArgumentError", "unknown keyword: :on"},
		{`ax.set_title(1)`, "TypeError", "Integer into String"},
		{`ax.set_title("a", "b")`, "ArgumentError", "given 2, expected 1"},
		{`ax.set_title("a", x: 1)`, "ArgumentError", "unknown keyword: :x"},
		{`ax.set_xlim(1)`, "ArgumentError", "given 1, expected 2"},
		{`ax.set_xlim(1, "a")`, "TypeError", "String into Float"},
		{`ax.set_xlim(1, 2, x: 1)`, "ArgumentError", "unknown keyword"},
		{`ax.xlim(1)`, "ArgumentError", "given 1, expected 2"},
		{`ax.set_xticks`, "ArgumentError", "given 0, expected 1..2"},
		{`ax.set_xticks(["a"])`, "TypeError", "String into Float"},
		{`ax.set_xticks([1], [1])`, "TypeError", "Integer into String"},
		{`ax.set_xticks([1], x: 1)`, "ArgumentError", "unknown keyword"},
		{`ax.set_aspect(:square)`, "ArgumentError", "unknown aspect"},
		{`ax.set_aspect([1])`, "TypeError", "Array into Float"},
		{`ax.set_aspect`, "ArgumentError", "given 0, expected 1"},
		{`ax.twinx(1)`, "ArgumentError", "given 1, expected 0"},
		{`ax.figure(1)`, "ArgumentError", "given 1, expected 0"},
		{`fig.colorbar(1, ax)`, "TypeError", "Integer into Plot::Mappable"},
		{`fig.colorbar`, "ArgumentError", "given 0, expected 1..2"},
		{`fig.colorbar(ax.imshow([[1]]))`, "ArgumentError", "colorbar needs the Axes"},
		{`fig.colorbar(ax.imshow([[1]]), 1)`, "TypeError", "Integer into Plot::Axes"},
		{`fig.colorbar(ax.imshow([[1]]), ax, x: 1)`, "ArgumentError", "unknown keyword: :x (known: ax)"},
		{`fig.add_axes(1, 2)`, "ArgumentError", "given 2, expected 4"},
		{`fig.add_axes(1, 2, 3, "a")`, "TypeError", "String into Float"},
		{`fig.subplots(1, 1, x: 1)`, "ArgumentError", "unknown keyword: :x (known: squeeze)"},
		{`fig.subplots(0)`, "ArgumentError", "rows and cols must be positive"},
		{`fig.width(1)`, "ArgumentError", "given 1, expected 0"},
		{`fig.to_svg(x: 1)`, "ArgumentError", "unknown keyword: :x"},
		{`Plot.subplots(1, 2, 3)`, "ArgumentError", "given 3, expected 0..2"},
		{`Plot.subplots("a")`, "TypeError", "String into Integer"},
		{`Plot.subplots(squeeze: 1)`, "TypeError", "true or false"},
		{`Plot.subplots(bogus: 1)`, "ArgumentError", "unknown keyword: :bogus (known: dpi, figsize)"},
		{`Plot.figure(1)`, "ArgumentError", "given 1, expected 0"},
		{`Plot.figure(figsize: [1])`, "ArgumentError", "figsize must be [width, height]"},
		{`Plot.figure(figsize: "a")`, "TypeError", "String into an Array"},
		{`Plot.figure(dpi: "a")`, "TypeError", "String into Float"},
		{`Plot.values`, "ArgumentError", "given 0, expected 1"},
		{`Plot.values(:a)`, "TypeError", "Symbol into plot data"},
		{`Plot.values([[1, 2], [3, 4]])`, "ArgumentError", "cannot plot data"},
	} {
		evalErr(t, in, c.src, c.class, c.msg)
	}
}

// A name already taken by something else is an error of Register.
func TestPlotRegisterConflicts(t *testing.T) {
	for _, src := range []string{`class Plot; end`, `module Plot; module Figure; end; end`} {
		in := rb.New()
		if _, err := in.Eval(src, "x.rb", 1); err != nil {
			t.Fatal(err)
		}
		if err := registerPlot(in); err == nil || !strings.Contains(err.Error(), "already defined") {
			t.Errorf("%s: registerPlot = %v", src, err)
		}
	}
}
