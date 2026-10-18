package ruby

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/go-dataframe/dataframe"
	rb "github.com/go-embedded-ruby/ruby"
	nd "github.com/go-ndarray/ndarray"
)

// dfSame compares a Ruby result (converted to Go) with the Go one: frames,
// series, indexes and group-bys by their full printed form and type, dtypes
// by name, instants by their nanoseconds, slices element by element, and
// everything else as same does.
func dfSame(got, want any) bool {
	switch w := want.(type) {
	case dataframe.DType:
		return got == w.String()
	case time.Time:
		v, ok := got.(rb.Value)
		if !ok || v.ClassName() != "Time" {
			return false
		}
		s, _ := v.Call("to_i")
		ns, _ := v.Call("nsec")
		off, _ := v.Call("utc_offset")
		return s.Go() == w.Unix() && ns.Go() == int64(w.Nanosecond()) && off.Go() == int64(0)
	case fmt.Stringer:
		g, ok := got.(fmt.Stringer)
		return ok && reflect.TypeOf(g) == reflect.TypeOf(w) && g.String() == w.String()
	case map[string][]byte:
		g, ok := got.(map[any]any)
		if !ok || len(g) != len(w) {
			return false
		}
		for k, v := range w {
			if g[k] != string(v) {
				return false
			}
		}
		return true
	}
	wv := reflect.ValueOf(want)
	if want != nil && wv.Kind() == reflect.Slice && wv.Type().Elem().Kind() != reflect.Float64 {
		g, ok := got.([]any)
		if !ok || len(g) != wv.Len() {
			return false
		}
		for i := range g {
			if !dfSame(g[i], wv.Index(i).Interface()) {
				return false
			}
		}
		return true
	}
	return same(got, want)
}

func dfIs(t *testing.T, in *rb.Interp, src string, want any) {
	t.Helper()
	got := eval(t, in, src)
	if !dfSame(got, want) {
		t.Errorf("%s\n  ruby: %#v\n  go:   %#v", src, got, want)
	}
}

// dfFixture holds the frames every test shares, built the same way in Go and
// in Ruby.
type dfFixture struct {
	df, num, long, ts, multi *dataframe.DataFrame
	s, f, b, str, day        *dataframe.Series
}

const dfSetup = `
df = DataFrame.new(city: ["Lyon", "Paris", "Lyon"], pop: [522, 2103, 530], x: [1.5, nil, 2.0])
num = df.select("pop", "x")
long = DataFrame.new("k" => ["a", "a", "b", "b"], "c" => ["x", "y", "x", "y"], "v" => [1, 2, 3, 4])
ts = DataFrame.read_csv("day,v\n2024-01-01,1\n2024-01-02,2\n2024-01-09,5\n", parse_dates: ["day"], index_col: "day")
multi = long.set_index("k", "c")
s = df["pop"]
f = df["x"]
b = s > 525
str = df["city"]
day = ts.reset_index["day"]
nil
`

func newDFFixture(t *testing.T) (*rb.Interp, dfFixture) {
	t.Helper()
	in := newInterp(t)
	eval(t, in, dfSetup)
	m := must[*dataframe.DataFrame](t)
	ms := must[*dataframe.Series](t)
	var x dfFixture
	x.df = m(dataframe.New(
		dataframe.Col("city", []any{"Lyon", "Paris", "Lyon"}),
		dataframe.Col("pop", []any{int64(522), int64(2103), int64(530)}),
		dataframe.Col("x", []any{1.5, nil, 2.0})))
	x.num = m(x.df.Select("pop", "x"))
	x.long = m(dataframe.New(
		dataframe.Col("k", []any{"a", "a", "b", "b"}),
		dataframe.Col("c", []any{"x", "y", "x", "y"}),
		dataframe.Col("v", []any{int64(1), int64(2), int64(3), int64(4)})))
	x.ts = m(dataframe.ReadCSV(strings.NewReader("day,v\n2024-01-01,1\n2024-01-02,2\n2024-01-09,5\n"),
		dataframe.CSVIndexCol("day"), dataframe.CSVParseDates("day")))
	x.multi = m(x.long.SetIndex("k", "c"))
	x.s = ms(x.df.Column("pop"))
	x.f = ms(x.df.Column("x"))
	x.b = ms(x.s.Gt(int64(525)))
	x.str = ms(x.df.Column("city"))
	x.day = ms(x.ts.ResetIndex(false).Column("day"))
	return in, x
}

func TestDataFrameConstruction(t *testing.T) {
	in, x := newDFFixture(t)
	m := must[*dataframe.DataFrame](t)
	ms := must[*dataframe.Series](t)
	dfIs(t, in, `df`, x.df)
	dfIs(t, in, `long`, x.long)
	dfIs(t, in, `DataFrame.new({"b" => [1], "a" => ["z"]}, c: [true])`, m(dataframe.New(
		dataframe.Col("b", []any{int64(1)}), dataframe.Col("a", []any{"z"}), dataframe.Col("c", []any{true}))))
	dfIs(t, in, `DataFrame.new([[:a, NDArray.linspace(0, 1, 3)], ["s", df["pop"]]])`, m(dataframe.New(
		dataframe.Col("a", must[*nd.Array](t)(nd.FromData([]float64{0, 0.5, 1}, 3))), dataframe.Col("s", x.s))))
	dfIs(t, in, `DataFrame.new`, m(dataframe.New()))
	dfIs(t, in, `DataFrame.new.columns`, m(dataframe.New()).Columns())
	dfIs(t, in, `DataFrame.new.series`, m(dataframe.New()).Series())
	dfIs(t, in, `DataFrame::Series.new([1]).valid`, ms(dataframe.NewSeries("", []any{int64(1)})).Valid())
	evalErr(t, in, `DataFrame.new(t: [Time.utc(2020, 1, 2, 3, 4, 5), :sym])`, "ArgumentError", "")
	evalErr(t, in, `DataFrame.new(a: [1, 2], b: [1])`, "ArgumentError", "length")
	evalErr(t, in, `DataFrame.new(1)`, "TypeError", "Integer into Hash")
	evalErr(t, in, `DataFrame.new([1])`, "TypeError", "Integer into a [name, value] pair")
	evalErr(t, in, `DataFrame.new([[1, [2]]])`, "TypeError", "Integer into String")

	tm := time.Date(2020, 1, 2, 3, 4, 5, 6000, time.UTC)
	dfIs(t, in, `DataFrame.new(t: [Time.utc(2020, 1, 2, 3, 4, 5, 6)])`, m(dataframe.New(dataframe.Col("t", []any{tm}))))
	dfIs(t, in, `DataFrame.new(t: [Time.utc(2020, 1, 2, 3, 4, 5, 6)])["t"][0]`, tm)

	csv := "a;b;c\n# note\n1;x;2020-01-02\nNA;y;2020-02-03\n3;;2020-03-04\n"
	dfIs(t, in, `DataFrame.read_csv("a,b\n1,2\n")`, m(dataframe.ReadCSV(strings.NewReader("a,b\n1,2\n"))))
	if err := in.Set("csv", csv); err != nil {
		t.Fatal(err)
	}
	dfIs(t, in, `DataFrame.read_csv(csv, sep: ";", comment: "#", parse_dates: "c", date_format: "%Y-%m-%d",
		na_values: ["x"], keep_default_na: true, usecols: [:a, :b, :c], nrows: 3, quotechar: "'", skip_blank_lines: false)`,
		m(dataframe.ReadCSV(strings.NewReader(csv), dataframe.CSVComment('#'), dataframe.CSVDateFormat("%Y-%m-%d"),
			dataframe.CSVKeepDefaultNA(true), dataframe.CSVNAValues("x"), dataframe.CSVNRows(3), dataframe.CSVParseDates("c"),
			dataframe.CSVQuote('\''), dataframe.CSVSeparator(';'), dataframe.CSVSkipBlankLines(false), dataframe.CSVUseCols("a", "b", "c"))))
	tf := "h\nskip\nq,r\nyes,1\nno,2\n"
	if err := in.Set("tf", tf); err != nil {
		t.Fatal(err)
	}
	dfIs(t, in, `DataFrame.read_csv(tf, skiprows: 2, true_values: "yes", false_values: ["no"], dtype: {"r" => :float64}, index_col: 1)`,
		m(dataframe.ReadCSV(strings.NewReader(tf), dataframe.CSVDTypes(map[string]dataframe.DType{"r": dataframe.Float64}),
			dataframe.CSVFalseValues("no"), dataframe.CSVIndexColumn(1), dataframe.CSVSkipRows(2), dataframe.CSVTrueValues("yes"))))
	dfIs(t, in, `DataFrame.read_csv("1,2\n3,4\n", header: false, names: ["p", "q"], quotechar: "")`,
		m(dataframe.ReadCSV(strings.NewReader("1,2\n3,4\n"), dataframe.CSVHeader(false), dataframe.CSVNames("p", "q"), dataframe.CSVQuote(0))))
	dir := t.TempDir()
	path := filepath.Join(dir, "a.csv")
	if err := os.WriteFile(path, []byte("a,b\n1,2\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := in.Set("path", path); err != nil {
		t.Fatal(err)
	}
	dfIs(t, in, `DataFrame.read_csv_file(path, sep: nil)`, m(dataframe.ReadCSVFile(path, dataframe.CSVSeparator(0))))
	for _, c := range []struct{ src, class, msg string }{
		{`DataFrame.read_csv`, "ArgumentError", "given 0, expected 1"},
		{`DataFrame.read_csv(1)`, "TypeError", "Integer into String"},
		{`DataFrame.read_csv("a", bogus: 1)`, "ArgumentError", "unknown keyword: :bogus (known: comment,"},
		{`DataFrame.read_csv("a", sep: ";;")`, "ArgumentError", "single character"},
		{`DataFrame.read_csv("a", sep: 1)`, "TypeError", "Integer into String"},
		{`DataFrame.read_csv("a", header: 1)`, "TypeError", "true or false"},
		{`DataFrame.read_csv("a", names: 1)`, "TypeError", "String or an Array"},
		{`DataFrame.read_csv("a", names: [1])`, "TypeError", "Integer into String"},
		{`DataFrame.read_csv("a", nrows: "x")`, "TypeError", "String into Integer"},
		{`DataFrame.read_csv("a", date_format: 1)`, "TypeError", "Integer into String"},
		{`DataFrame.read_csv("a", index_col: 1.5)`, "TypeError", "Float into String"},
		{`DataFrame.read_csv("a", dtype: {"a" => :nope})`, "ArgumentError", "nope"},
		{`DataFrame.read_csv("a", dtype: {"a" => 1})`, "TypeError", "Integer into a dtype"},
		{`DataFrame.read_csv("a,b\n1,2\n3,4,5\n")`, "ArgumentError", "line"},
		{`DataFrame.read_csv_file(path + ".none")`, "ArgumentError", "a.csv.none"}, // the system's own words differ
		{`DataFrame.read_csv_file`, "ArgumentError", "given 0, expected 1"},
	} {
		evalErr(t, in, c.src, c.class, c.msg)
	}

	a := must[*nd.Array](t)(nd.FromData([]float64{1, 2, 3, 4}, 2, 2))
	if err := in.Set("arr", a); err != nil {
		t.Fatal(err)
	}
	dfIs(t, in, `DataFrame.from_array(arr, ["p", "q"])`, m(dataframe.FromArray(a, "p", "q")))
	dfIs(t, in, `DataFrame.from_array(arr)`, m(dataframe.FromArray(a)))
	evalErr(t, in, `DataFrame.from_array(arr, "p")`, "ArgumentError", "")
	evalErr(t, in, `DataFrame.from_array([1])`, "TypeError", "Array into NDArray")
	evalErr(t, in, `DataFrame.from_array(arr, [1])`, "TypeError", "Integer into String")
	evalErr(t, in, `DataFrame.from_array(arr, x: 1)`, "ArgumentError", "unknown keyword: :x")
	evalErr(t, in, `DataFrame.from_array`, "ArgumentError", "given 0, expected 1..2")

	dfIs(t, in, `DataFrame.from_series(s, f)`, m(dataframe.FromSeries(x.s, x.f)))
	evalErr(t, in, `DataFrame.from_series(1)`, "TypeError", "Integer into DataFrame::Series")
	evalErr(t, in, `DataFrame.from_series(s, x: 1)`, "ArgumentError", "unknown keyword: :x")

	dfIs(t, in, `DataFrame.concat(df, num)`, m(dataframe.Concat(x.df, x.num)))
	dfIs(t, in, `DataFrame.concat(df, num, ignore_index: true, inner: true)`,
		m(dataframe.ConcatWith(dataframe.ConcatOptions{IgnoreIndex: true, Inner: true}, x.df, x.num)))
	dfIs(t, in, `DataFrame.concat(df, long, axis: 1)`, m(dataframe.ConcatWith(dataframe.ConcatOptions{Axis: 1}, x.df, x.long)))
	evalErr(t, in, `DataFrame.concat`, "ArgumentError", "nothing to concatenate")
	evalErr(t, in, `DataFrame.concat(df, 1)`, "TypeError", "Integer into DataFrame")
	evalErr(t, in, `DataFrame.concat(df, bogus: 1)`, "ArgumentError", "unknown keyword: :bogus")
}

func TestDataFrameMethods(t *testing.T) {
	in, x := newDFFixture(t)
	m := must[*dataframe.DataFrame](t)
	ms := must[*dataframe.Series](t)
	df, num, long, ts := x.df, x.num, x.long, x.ts
	r, c := df.Shape()
	disp := map[string][]byte{}
	for k, v := range df.Display() {
		disp[k] = v
	}
	for _, cs := range []struct {
		src  string
		want any
	}{
		{`df.columns`, df.Columns()},
		{`df.length`, df.Len()},
		{`df.size`, df.Len()},
		{`df.shape`, []int{r, c}},
		{`df.describe`, df.Describe()},
		{`df.copy`, df.Copy()},
		{`df.count`, df.Count()},
		{`df.mean`, df.Mean()},
		{`df.sum`, df.Sum()},
		{`df.std`, df.Std()},
		{`df.var`, df.Var()},
		{`df.min`, df.Min()},
		{`df.max`, df.Max()},
		{`df.median`, df.Median()},
		{`df.isna`, df.IsNA()},
		{`df.notna`, df.NotNA()},
		{`df.ffill`, df.FFill()},
		{`df.bfill`, df.BFill()},
		{`df.transpose`, df.T()},
		{`df.t`, df.T()},
		{`df.drop("x")`, m(df.Drop("x"))},
		{`df.drop(["x", "pop"])`, m(df.Drop("x", "pop"))},
		{`df.select(:city, :x)`, m(df.Select("city", "x"))},
		{`df.set_index("city")`, m(df.SetIndex("city"))},
		{`multi`, x.multi},
		{`multi.reset_index`, x.multi.ResetIndex(false)},
		{`multi.reset_index(drop: true)`, x.multi.ResetIndex(true)},
		{`multi.reset_index(true)`, x.multi.ResetIndex(true)},
		{`df.sort_index(ascending: false)`, df.SortIndex(false)},
		{`df.sort_index`, df.SortIndex(true)},
		{`df.reindex([2, 0, 7])`, m(df.Reindex([]any{int64(2), int64(0), int64(7)}))},
		{`df.reindex_columns("x", "q")`, df.ReindexColumns("x", "q")},
		{`df.set_columns("a", "b", "c")`, m(df.SetColumns("a", "b", "c"))},
		{`df.column("pop")`, m2(ms)(df.Column("pop"))},
		{`df.column_at(2)`, ms(df.ColumnAt(2))},
		{`df.series`, df.Series()},
		{`df.iat(1, 1)`, int64(2103)},
		{`df.iat(-1, 0)`, "Lyon"},
		{`df.at(2, "city")`, "Lyon"},
		{`df.at(1, "x")`, nil},
		{`num.values`, must[*nd.Array](t)(num.Values())},
		{`num.to_ndarray`, must[*nd.Array](t)(num.Values())},
		{`df.to_s`, df.String()},
		{`df.inspect`, df.String()},
		{`df.to_html`, df.HTML()},
		{`df.display`, disp},
		{`df.index`, df.Index()},
		{`multi.index`, x.multi.Index()},
		{`df.with_column("y", [1, 2, 3])`, m(df.WithColumn("y", []any{int64(1), int64(2), int64(3)}))},
		{`df.with_column("y", 7)`, m(df.WithColumn("y", int64(7)))},
		{`df.with_column("pop", df["x"])`, m(df.WithColumn("pop", x.f))},
		{`df.with_index(DataFrame::Index.new(["a", "b", "c"], name: "n"))`,
			m(df.WithIndex(must[*dataframe.Index](t)(dataframe.NewIndex("n", []any{"a", "b", "c"}))))},
		{`long.pivot("k", "c", "v")`, m(long.Pivot("k", "c", "v"))},
		{`long.pivot(index: "k", columns: "c", values: "v")`, m(long.Pivot("k", "c", "v"))},
		{`long.pivot_table("c", "k", "v")`, m(long.PivotTable("c", "k", "v", "mean"))},
		{`long.pivot_table("c", "k", "v", aggfunc: "sum")`, m(long.PivotTable("c", "k", "v", "sum"))},
		{`num.stack`, ms(num.Stack())},
		{`multi.unstack`, m(x.multi.Unstack())},
		{`ts.to_period("M")`, m(ts.ToPeriod("M"))},
		{`df.head`, df.Head(5)},
		{`df.head(1)`, df.Head(1)},
		{`df.head(n: 2)`, df.Head(2)},
		{`df.tail(1)`, df.Tail(1)},
		{`df.tail`, df.Tail(5)},
		{`num.round`, num.Round(0)},
		{`num.div(7).round(decimals: 2)`, m(num.Div(int64(7))).Round(2)},
		{`df.shift`, df.Shift(1)},
		{`df.shift(periods: -1)`, df.Shift(-1)},
		{`num.add(1)`, m(num.Add(int64(1)))},
		{`num.+(num)`, m(num.Add(num))},
		{`num.sub(0.5)`, m(num.Sub(0.5))},
		{`num.-(1)`, m(num.Sub(int64(1)))},
		{`num.mul(2)`, m(num.Mul(int64(2)))},
		{`num.*(num)`, m(num.Mul(num))},
		{`num.div(4)`, m(num.Div(int64(4)))},
		{`num./(num)`, m(num.Div(num))},
		// The methods written by hand.
		{`df["pop"]`, ms(df.Column("pop"))},
		{`df[:city]`, ms(df.Column("city"))},
		{`df[["x", :city]]`, m(df.Select("x", "city"))},
		{`df[b]`, m(df.Filter(x.b))},
		{`df[[true, false, true]]`, m(df.Filter([]bool{true, false, true}))},
		{`df.filter(b)`, m(df.Filter(x.b))},
		{`df.filter([false, false, true])`, m(df.Filter([]bool{false, false, true}))},
		{`df.dtypes`, map[string]string{"city": "string", "pop": "int64", "x": "float64"}},
		{`df.dtypes.keys`, []string{"city", "pop", "x"}},
		{`df.to_h.keys`, df.Columns()},
		{`df.to_h["x"]`, x.f.Items()},
		{`ts.to_h["v"]`, []any{int64(1), int64(2), int64(5)}},
		{`ts.reset_index.to_h["day"]`, x.day.Items()},
		{`df.sort_values("pop")`, m(df.SortValues([]string{"pop"}))},
		{`df.sort_values(["city", "pop"], ascending: [true, false])`, m(df.SortValues([]string{"city", "pop"}, true, false))},
		{`df.sort_values(:x, ascending: false)`, m(df.SortValues([]string{"x"}, false))},
		{`df.group_by("city").mean`, m(m3(df.GroupBy("city")).Mean())},
		{`df.groupby(["city"]).sum`, m(m3(df.GroupBy("city")).Sum())},
		{`df.dropna`, m(df.DropNA())},
		{`df.dropna("pop")`, m(df.DropNA("pop"))},
		{`df.dropna(how: :all)`, m(df.DropNAAll())},
		{`df.dropna(how: "any")`, m(df.DropNA())},
		{`df.dropna(:x, thresh: 1)`, m(df.DropNAThresh(1, "x"))},
		{`df.dropna(axis: 1)`, df.DropNAColumns()},
		{`df.dropna(axis: 0)`, m(df.DropNA())},
		{`df.fillna(0)`, df.FillNA(int64(0))},
		{`df.fillna("x" => 9.5)`, m(df.FillNAColumns(map[string]any{"x": 9.5}))},
		{`df.fillna(x: 9.5)`, m(df.FillNAColumns(map[string]any{"x": 9.5}))},
		{`df.astype(:string)`, m(df.AsTypeAll(dataframe.String))},
		{`df.astype("pop" => "float64")`, m(df.AsType(map[string]dataframe.DType{"pop": dataframe.Float64}))},
		{`df.astype(pop: :float32)`, m(df.AsType(map[string]dataframe.DType{"pop": dataframe.Float32}))},
		{`df.rename("pop" => "p")`, df.Rename(map[string]string{"pop": "p"})},
		{`df.rename(pop: "p", x: :y)`, df.Rename(map[string]string{"pop": "p", "x": "y"})},
		{`df.merge(long.rename(k: "city"))`, m(df.Merge(long.Rename(map[string]string{"k": "city"}), dataframe.MergeOptions{}))},
		{`df.merge(DataFrame.new(c: ["Lyon"], q: [1]), how: "left", left_on: ["city"], right_on: ["c"], indicator: true, suffixes: ["_l", "_r"])`,
			m(df.Merge(m(dataframe.New(dataframe.Col("c", []any{"Lyon"}), dataframe.Col("q", []any{int64(1)}))),
				dataframe.MergeOptions{How: "left", LeftOn: []string{"city"}, RightOn: []string{"c"}, Indicator: true, Suffixes: [2]string{"_l", "_r"}}))},
		{`df.join(num, lsuffix: "_a", rsuffix: "_b")`, m(df.Join(num, "left", "_a", "_b"))},
		{`df.join(num.rename(pop: "p", x: "q"), how: "inner")`, m(df.Join(num.Rename(map[string]string{"pop": "p", "x": "q"}), "inner", "", ""))},
		{`long.melt(id_vars: ["k"], value_vars: "v")`, m(long.Melt([]string{"k"}, []string{"v"}))},
		{`long.melt(id_vars: :k)`, m(long.Melt([]string{"k"}, nil))},
		{`long.melt`, m(long.Melt(nil, nil))},
		{`df.query("pop > 525")`, m(df.Query("pop > 525"))},
		{`df.query("pop > @lo and city == @c", lo: 500, c: :Lyon)`, m(df.Query("pop > @lo and city == @c", map[string]any{"lo": int64(500), "c": "Lyon"}))},
		{`df.eval("pop * 2")`, ms(df.Eval("pop * 2"))},
		{`df.eval("pop * @k", k: 1.5)`, ms(df.Eval("pop * @k", map[string]any{"k": 1.5}))},
		{`df.iloc(0)`, m(df.Iloc(dataframe.Pos(0)))},
		{`df.iloc([2, 0], [1])`, m(df.Iloc(dataframe.Pos(2, 0), dataframe.Pos(1)))},
		{`df.iloc(0...2)`, m(df.Iloc(dataframe.Slice(0, 2)))},
		{`df.iloc(1..2)`, m(df.Iloc(dataframe.Slice(1, 3)))},
		{`df.iloc(1..)`, m(df.Iloc(dataframe.SliceFrom(1)))},
		{`df.iloc(..1)`, m(df.Iloc(dataframe.Slice(0, 2)))},
		{`df.iloc(nil, 0..1)`, m(df.Iloc(dataframe.All(), dataframe.Slice(0, 2)))},
		{`df.iloc(:all, [true, false, true])`, m(df.Iloc(dataframe.All(), dataframe.Mask([]bool{true, false, true})))},
		{`df.iloc(b)`, m(df.Iloc(dataframe.Mask(x.b)))},
		{`df.loc(1)`, m(df.Loc(dataframe.Labels(int64(1))))},
		{`df.loc([2, 0], ["x"])`, m(df.Loc(dataframe.Labels(int64(2), int64(0)), "x"))},
		{`df.loc(0..1, :city)`, m(df.Loc(dataframe.LabelRange(int64(0), int64(1)), "city"))},
		{`df.loc(1.., :city)`, m(df.Loc(dataframe.LabelRange(int64(1), nil), "city"))},
		{`df.set_index("city").loc("Paris")`, m(m(df.SetIndex("city")).Loc(dataframe.Labels("Paris")))},
		{`ts.loc(Time.utc(2024, 1, 2)..)`, m(ts.Loc(dataframe.LabelRange(time.Date(2024, 1, 2, 0, 0, 0, 0, time.UTC), nil)))},
		{`df.loc(b, "pop")`, m(df.Loc(dataframe.Mask(x.b), "pop"))},
		{`df.loc([false, true, true])`, m(df.Loc(dataframe.Mask([]bool{false, true, true})))},
		{`df.apply { |r| r["pop"] * 2 }`, ms(df.Apply("", func(r dataframe.Row) any { return r.Get("pop").(int64) * 2 }))},
		{`df.apply("q") { |r| r.keys.join }`, ms(df.Apply("q", func(r dataframe.Row) any { return "citypopx" }))},
		{`df.apply("q") { |r| r["x"] }`, ms(df.Apply("q", func(r dataframe.Row) any { return r.Get("x") }))},
		{`ts.resample("W").sum`, m(m4(ts.Resample("W")).Sum())},
		{`ts.reset_index.resample("2D", on: "day").mean`, m(m4(ts.ResetIndex(false).ResampleOn("day", "2D")).Mean())},
	} {
		dfIs(t, in, cs.src, cs.want)
	}

	var buf bytes.Buffer
	if err := df.WriteCSV(&buf); err != nil {
		t.Fatal(err)
	}
	dfIs(t, in, `df.to_csv`, buf.String())
	buf.Reset()
	if err := df.WriteCSV(&buf, dataframe.CSVWriteHeader(false), dataframe.CSVWriteIndex(false), dataframe.CSVWriteLineTerminator("\r\n"),
		dataframe.CSVWriteNARep("NA"), dataframe.CSVWriteSeparator(';')); err != nil {
		t.Fatal(err)
	}
	dfIs(t, in, `df.to_csv(header: false, index: false, line_terminator: "\r\n", na_rep: "NA", sep: ";")`, buf.String())
	dir := t.TempDir()
	if err := in.Set("out", filepath.Join(dir, "o.csv")); err != nil {
		t.Fatal(err)
	}
	eval(t, in, `df.write_csv(out, index: false)`)
	buf.Reset()
	_ = df.WriteCSV(&buf, dataframe.CSVWriteIndex(false))
	if got, _ := os.ReadFile(filepath.Join(dir, "o.csv")); string(got) != buf.String() {
		t.Errorf("write_csv wrote %q, want %q", got, buf.String())
	}
}

// m2, m3 and m4 are must for the other result types.
func m2(ms func(*dataframe.Series, error) *dataframe.Series) func(*dataframe.Series, error) *dataframe.Series {
	return ms
}

func m3(g *dataframe.GroupBy, err error) *dataframe.GroupBy {
	if err != nil {
		panic(err)
	}
	return g
}

func m4(r *dataframe.Resampler, err error) *dataframe.Resampler {
	if err != nil {
		panic(err)
	}
	return r
}

func TestDataFrameSeries(t *testing.T) {
	in, x := newDFFixture(t)
	ms := must[*dataframe.Series](t)
	s, f, b, str, day := x.s, x.f, x.b, x.str, x.day
	mf := func(v *nd.Array, err error) *nd.Array { return must[*nd.Array](t)(v, err) }
	agg := func(v any, err error) any { return must[any](t)(v, err) }
	fl := func(v []float64, err error) []float64 { return must[[]float64](t)(v, err) }
	dt, err := day.Dt()
	if err != nil {
		t.Fatal(err)
	}
	rollMean := func(r *dataframe.Rolling, err error) *dataframe.Series {
		return must[*dataframe.Rolling](t)(r, err).Mean()
	}
	disp := map[string][]byte{}
	for k, v := range s.Display() {
		disp[k] = v
	}
	for _, c := range []struct {
		src  string
		want any
	}{
		{`DataFrame::Series.new([1, 2, 3])`, ms(dataframe.NewSeries("", []any{int64(1), int64(2), int64(3)}))},
		{`DataFrame::Series.new([1.5, nil], name: :a)`, ms(dataframe.NewSeries("a", []any{1.5, nil}))},
		{`DataFrame::Series.new(NDArray.linspace(0, 1, 3), name: "a")`, ms(dataframe.NewSeries("a", mf(nd.FromData([]float64{0, 0.5, 1}, 3))))},
		{`DataFrame::Series.new([1, 2], index: ["a", "b"])`, ms(dataframe.NewSeriesIndexed("", []any{int64(1), int64(2)},
			must[*dataframe.Index](t)(dataframe.NewIndex("", []any{"a", "b"}))))},
		{`DataFrame::Series.new([1, 2], index: DataFrame::Index.range(2))`, ms(dataframe.NewSeriesIndexed("", []any{int64(1), int64(2)}, dataframe.NewRangeIndex(2)))},
		{`s.name`, s.Name()},
		{`s.length`, s.Len()},
		{`s.size`, s.Len()},
		{`f.count`, f.Count()},
		{`s.dtype`, s.DType()},
		{`s.index`, s.Index()},
		{`s.values`, mf(s.Values())},
		{`s.to_ndarray`, mf(s.Values())},
		{`s.floats`, fl(s.Floats())},
		{`s.to_a`, s.Items()},
		{`day.to_a`, day.Items()},
		{`f.valid`, f.Valid()},
		{`s.get(1)`, agg(s.Get(int64(1)))},
		{`s.mean`, s.Mean()},
		{`s.sum`, s.SumValue()},
		{`f.sum`, f.SumValue()},
		{`s.std`, s.Std()},
		{`s.var`, s.VarDDOF(1)},
		{`s.var(ddof: 0)`, s.VarDDOF(0)},
		{`s.var(0)`, s.VarDDOF(0)},
		{`s.min`, s.Min()},
		{`s.max`, s.Max()},
		{`day.max`, day.Max()},
		{`s.median`, s.Median()},
		{`s.prod`, s.Prod()},
		{`str.nunique`, str.NUnique()},
		{`s.quantile`, s.Quantile(0.5)},
		{`s.quantile(0.25)`, s.Quantile(0.25)},
		{`s.quantile(q: 0.75)`, s.Quantile(0.75)},
		{`s.agg("sum")`, agg(s.Agg("sum"))},
		{`s.agg(:mean)`, agg(s.Agg("mean"))},
		{`s.head`, s.Head(5)},
		{`s.head(1)`, s.Head(1)},
		{`s.tail(2)`, s.Tail(2)},
		{`str.unique`, str.Unique()},
		{`str.value_counts`, str.ValueCounts()},
		{`s.copy`, s.Copy()},
		{`s.to_frame`, s.ToFrame()},
		{`s.reset_index`, s.ResetIndex()},
		{`f.isna`, f.IsNA()},
		{`f.notna`, f.NotNA()},
		{`f.dropna`, f.DropNA()},
		{`f.fillna(0)`, ms(f.FillNA(int64(0)))},
		{`f.ffill`, f.FFill()},
		{`f.bfill`, f.BFill()},
		{`s.shift`, s.Shift(1)},
		{`s.shift(periods: 2)`, s.Shift(2)},
		{`s.diff`, ms(s.Diff(1))},
		{`s.pct_change(2)`, ms(s.PctChange(2))},
		{`s.cumsum`, ms(s.CumSum())},
		{`s.cumprod`, ms(s.CumProd())},
		{`s.cummax`, ms(s.CumMax())},
		{`s.cummin`, ms(s.CumMin())},
		{`s.neg.abs`, ms(ms(s.Neg()).Abs())},
		{`s.send(:-@)`, ms(s.Neg())},
		{`f.round`, f.Round(0)},
		{`s.div(7).round(decimals: 3)`, ms(s.Div(int64(7))).Round(3)},
		{`s.rename("p")`, s.Rename("p")},
		{`s.astype(:float32)`, ms(s.AsType(dataframe.Float32))},
		{`s.reindex([2, 0])`, ms(s.Reindex([]any{int64(2), int64(0)}))},
		{`s.with_index(DataFrame::Index.new(["a", "b", "c"]))`, ms(s.SetIndex(must[*dataframe.Index](t)(dataframe.NewIndex("", []any{"a", "b", "c"}))))},
		{`s.sort_values`, s.SortValues(true)},
		{`s.sort_values(ascending: false)`, s.SortValues(false)},
		{`s.sort_values.sort_index(ascending: false)`, s.SortValues(true).SortIndex(false)},
		{`s.sort_index`, s.SortIndex(true)},
		{`s.between(500, 600)`, ms(s.Between(int64(500), int64(600)))},
		{`str.isin("Lyon", "Rome")`, ms(str.IsIn("Lyon", "Rome"))},
		{`str.isin(["Paris"])`, ms(str.IsIn("Paris"))},
		{`s.isin(522, 1)`, ms(s.IsIn(int64(522), int64(1)))},
		{`day.dt.year`, dt.Year()},
		{`day.dt.month`, dt.Month()},
		{`day.dt.day`, dt.Day()},
		{`day.dt.hour`, dt.Hour()},
		{`day.dt.minute`, dt.Minute()},
		{`day.dt.second`, dt.Second()},
		{`day.dt.day_of_week`, dt.DayOfWeek()},
		{`day.dt.dayofweek`, dt.DayOfWeek()},
		{`day.dt.day_of_year`, dt.DayOfYear()},
		{`day.dt.dayofyear`, dt.DayOfYear()},
		{`day.dt.normalize`, dt.Normalize()},
		{`day.dt.strftime("%d/%m")`, dt.Strftime("%d/%m")},
		{`day.to_period("M")`, ms(day.ToPeriod("M"))},
		{`day.to_period("M").to_timestamp`, ms(ms(day.ToPeriod("M")).ToTimestamp())},
		{`multi["v"].unstack`, must[*dataframe.DataFrame](t)(ms(x.multi.Column("v")).Unstack())},
		{`s.to_s`, s.String()},
		{`s.inspect`, s.String()},
		{`s.to_html`, s.HTML()},
		{`s.display`, disp},
		{`s.+(1)`, ms(s.Add(int64(1)))},
		{`s.add(s)`, ms(s.Add(s))},
		{`str.add("!")`, ms(str.Add("!"))},
		{`s.-(1.5)`, ms(s.Sub(1.5))},
		{`s.sub(s)`, ms(s.Sub(s))},
		{`s.*(2)`, ms(s.Mul(int64(2)))},
		{`s.mul(f)`, ms(s.Mul(f))},
		{`s./(2)`, ms(s.Div(int64(2)))},
		{`s.div(f)`, ms(s.Div(f))},
		{`s.%(7)`, ms(s.Mod(int64(7)))},
		{`s.mod(7)`, ms(s.Mod(int64(7)))},
		{`s.**(2)`, ms(s.Pow(int64(2)))},
		{`s.pow(0.5)`, ms(s.Pow(0.5))},
		{`s.floordiv(7)`, ms(s.FloorDiv(int64(7)))},
		{`s > 525`, ms(s.Gt(int64(525)))},
		{`s.gt(525)`, ms(s.Gt(int64(525)))},
		{`s < 525`, ms(s.Lt(int64(525)))},
		{`s.lt(s)`, ms(s.Lt(s))},
		{`s >= 530`, ms(s.Ge(int64(530)))},
		{`s.ge(530)`, ms(s.Ge(int64(530)))},
		{`s <= 530`, ms(s.Le(int64(530)))},
		{`s.le(530)`, ms(s.Le(int64(530)))},
		{`str.eq("Lyon")`, ms(str.Eq("Lyon"))},
		{`str.ne(:Lyon)`, ms(str.Ne("Lyon"))},
		{`day > Time.utc(2024, 1, 1)`, ms(day.Gt(time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)))},
		{`b & s.lt(600)`, ms(b.And(ms(s.Lt(int64(600)))))},
		{`b | s.lt(600)`, ms(b.Or(ms(s.Lt(int64(600)))))},
		{`b ^ s.lt(600)`, ms(b.Xor(ms(s.Lt(int64(600)))))},
		{`~b`, ms(b.Not())},
		{`b.invert`, ms(b.Not())},
		// The methods written by hand.
		{`s[1]`, agg(s.At(1))},
		{`s[-1]`, agg(s.At(-1))},
		{`day[0]`, agg(day.At(0))},
		{`f[1]`, nil},
		{`s[b]`, ms(s.Filter(b))},
		{`s[[true, false, false]]`, ms(s.Filter([]bool{true, false, false}))},
		{`s[0..1]`, ms(s.Iloc(dataframe.Slice(0, 2)))},
		{`s[[2, 0]]`, ms(s.Iloc(dataframe.Pos(2, 0)))},
		{`s.filter(b)`, ms(s.Filter(b))},
		{`s.filter([false, true, true])`, ms(s.Filter([]bool{false, true, true}))},
		{`s.map { |v| v * 2 }`, ms(s.Map(func(v any) any { return v.(int64) * 2 }))},
		{`f.map { |v| v.nil? ? "none" : "v" }`, ms(f.Map(func(v any) any {
			if v == nil {
				return "none"
			}
			return "v"
		}))},
		{`day.map { |v| v.year }`, ms(day.Map(func(v any) any { return int64(v.(time.Time).Year()) }))},
		{`s.rolling(2).mean`, rollMean(s.Rolling(2))},
		{`s.rolling(2, min_periods: 1).mean`, rollMean(s.RollingMin(2, 1))},
		{`s.rolling(2).sum`, must[*dataframe.Rolling](t)(s.Rolling(2)).Sum()},
		{`s.rolling(2).min`, must[*dataframe.Rolling](t)(s.Rolling(2)).Min()},
		{`s.rolling(2).max`, must[*dataframe.Rolling](t)(s.Rolling(2)).Max()},
		{`s.rolling(2).std`, must[*dataframe.Rolling](t)(s.Rolling(2)).Std()},
		{`s.rolling(2).median`, must[*dataframe.Rolling](t)(s.Rolling(2)).Median()},
		{`s.iloc([0, 2])`, ms(s.Iloc(dataframe.Pos(0, 2)))},
		{`s.loc(1..2)`, ms(s.Loc(dataframe.LabelRange(int64(1), int64(2))))},
		{`s.loc(b)`, ms(s.Loc(dataframe.Mask(b)))},
		{`DataFrame::Series.new(["2024-03-04"]).to_datetime`, ms(dataframe.ToDatetime(ms(dataframe.NewSeries("", []any{"2024-03-04"}))))},
		{`DataFrame::Series.new(["04/03/2024"]).to_datetime("%d/%m/%Y")`,
			ms(dataframe.ToDatetime(ms(dataframe.NewSeries("", []any{"04/03/2024"})), "%d/%m/%Y"))},
	} {
		dfIs(t, in, c.src, c.want)
	}
}

func TestDataFrameGroupsAndIndexes(t *testing.T) {
	in, x := newDFFixture(t)
	m := must[*dataframe.DataFrame](t)
	ms := must[*dataframe.Series](t)
	g := m3(x.long.GroupBy("k"))
	g2 := m3(x.df.GroupBy("city", "pop"))
	r := m4(x.ts.Resample("W"))
	eval(t, in, `g = long.group_by("k"); r = ts.resample("W")`)
	ix := x.multi.Index()
	sx := must[*dataframe.Index](t)(dataframe.NewIndex("n", []any{"a", "b", "a"}))
	eval(t, in, `ix = multi.index; sx = DataFrame::Index.new(["a", "b", "a"], name: "n")`)
	keys, frames := g.Groups()
	keys2, frames2 := g2.Groups()
	var groups, groups2 []any
	for i := range keys {
		groups = append(groups, []any{keys[i], frames[i]})
	}
	for i := range keys2 {
		groups2 = append(groups2, []any{keys2[i], frames2[i]})
	}
	ixDisp := map[string][]byte{}
	for k, v := range sx.Display() {
		ixDisp[k] = v
	}
	for _, c := range []struct {
		src  string
		want any
	}{
		{`g.mean`, m(g.Mean())},
		{`g.sum`, m(g.Sum())},
		{`g.count`, m(g.Count())},
		{`g.min`, m(g.Min())},
		{`g.max`, m(g.Max())},
		{`g.median`, m(g.Median())},
		{`g.std`, m(g.Std())},
		{`g.var`, m(g.Var())},
		{`g.first`, m(g.First())},
		{`g.last`, m(g.Last())},
		{`g.nunique`, m(g.NUnique())},
		{`g.size`, g.Size()},
		{`g.ngroups`, g.NGroups()},
		{`g.n_groups`, g.NGroups()},
		{`g.select("v").sum`, m(m3(g.Select("v")).Sum())},
		{`g["v"].sum`, m(m3(g.Select("v")).Sum())},
		{`g.transform("v", "sum")`, ms(g.Transform("v", "sum"))},
		{`g.to_s`, g.String()},
		{`g.inspect`, g.String()},
		{`g.agg(v: [:sum, :max])`, m(g.Agg(dataframe.Agg("v", "sum", "max")))},
		{`g.agg({"v" => "mean"}, c: :first)`, m(g.Agg(dataframe.Agg("v", "mean"), dataframe.Agg("c", "first")))},
		{`g.groups`, groups},
		{`df.group_by("city", "pop").groups`, groups2},
		{`r.mean`, m(r.Mean())},
		{`r.sum`, m(r.Sum())},
		{`r.count`, m(r.Count())},
		{`r.min`, m(r.Min())},
		{`r.max`, m(r.Max())},
		{`r.median`, m(r.Median())},
		{`r.std`, m(r.Std())},
		{`r.first`, m(r.First())},
		{`r.last`, m(r.Last())},
		{`r.size`, r.Size()},
		{`r.agg(v: [:sum, :mean])`, m(r.Agg(dataframe.Agg("v", "sum", "mean")))},
		{`ix.to_a`, ix.Labels()},
		{`ix.labels`, ix.Labels()},
		{`ix.length`, ix.Len()},
		{`ix.size`, ix.Len()},
		{`sx.name`, sx.Name()},
		{`ix.names`, ix.Names()},
		{`ix.nlevels`, ix.NLevels()},
		{`ix.range?`, ix.IsRange()},
		{`df.index.range?`, true},
		{`ix.label(1)`, ix.Label(1)},
		{`sx[2]`, sx.Label(2)},
		{`sx.get("a")`, must[[]int](t)(sx.Get("a"))},
		{`ix.get(["a", "y"])`, must[[]int](t)(ix.Get([]any{"a", "y"}))},
		{`df.index.values`, must[*nd.Array](t)(x.df.Index().Values())},
		{`sx.dtype`, sx.DType()},
		{`ix.level(1)`, ms(ix.Level(1))},
		{`ix.rename("p", "q")`, must[*dataframe.Index](t)(ix.Rename("p", "q"))},
		{`ix.to_s`, ix.String()},
		{`ix.inspect`, ix.String()},
		{`sx.display`, ixDisp},
		{`ts.index[0]`, x.ts.Index().Label(0)},
		{`DataFrame::Index.range(3)`, dataframe.NewRangeIndex(3)},
		{`DataFrame::Index.new([1, 2])`, must[*dataframe.Index](t)(dataframe.NewIndex("", []any{int64(1), int64(2)}))},
	} {
		dfIs(t, in, c.src, c.want)
	}
}

func TestDataFrameErrors(t *testing.T) {
	in, _ := newDFFixture(t)
	eval(t, in, `g = long.group_by("k")`)
	for _, c := range []struct{ src, class, msg string }{
		// The table-driven methods.
		{`df.head(1, 2)`, "ArgumentError", "given 2, expected 1"},
		{`df.head(bogus: 1)`, "ArgumentError", "unknown keyword: :bogus (known: n)"},
		{`df.columns(x: 1)`, "ArgumentError", "unknown keyword: :x (known: none)"},
		{`df.head(1, n: 2)`, "ArgumentError", "n given both by position and by keyword"},
		{`df.column`, "ArgumentError", "given 0, expected 1"},
		{`df.column("nope")`, "ArgumentError", "no such column"},
		{`df.column(1)`, "TypeError", "Integer into String"},
		{`df.drop(1)`, "TypeError", "Integer into String"},
		{`s.astype(:nope)`, "ArgumentError", "nope"},
		{`df.with_column("y", [1])`, "ArgumentError", "length"},
		// [] and filter.
		{`df[1]`, "TypeError", "a column name, an Array of names or a mask"},
		{`df[[1]]`, "TypeError", "Integer into String"},
		{`df["a", "b"]`, "ArgumentError", "given 2, expected 1"},
		{`df[[true]]`, "ArgumentError", ""},
		{`df.filter(1)`, "TypeError", "Integer into Array"},
		{`df.filter`, "ArgumentError", "given 0, expected 1"},
		{`df.dtypes(1)`, "ArgumentError", "given 1, expected 0"},
		{`df.to_h(1)`, "ArgumentError", "given 1, expected 0"},
		{`df.to_csv(1)`, "ArgumentError", "given 1, expected 0"},
		{`df.to_csv(bogus: 1)`, "ArgumentError", "unknown keyword: :bogus (known: header,"},
		{`df.to_csv(sep: 1)`, "TypeError", "Integer into String"},
		{`df.to_csv(header: 1)`, "TypeError", "true or false"},
		{`df.to_csv(na_rep: 1)`, "TypeError", "Integer into String"},
		{`df.write_csv`, "ArgumentError", "given 0, expected 1"},
		{`df.write_csv(1)`, "TypeError", "Integer into String"},
		{`df.write_csv("/no/such/dir/x.csv")`, "IOError", "/no/such/dir/x.csv"},
		{`df.write_csv("x.csv", bogus: 1)`, "ArgumentError", "unknown keyword"},
		{`df.sort_values("pop", bogus: 1)`, "ArgumentError", "unknown keyword: :bogus (known: ascending)"},
		{`df.sort_values`, "ArgumentError", "given 0, expected 1"},
		{`df.sort_values(1)`, "TypeError", "String or an Array"},
		{`df.sort_values("pop", ascending: 1)`, "TypeError", "true or false"},
		{`df.sort_values("nope")`, "ArgumentError", "no such column"},
		{`df.group_by("city", x: 1)`, "ArgumentError", "unknown keyword"},
		{`df.group_by(1)`, "TypeError", "Integer into String"},
		{`df.group_by("nope")`, "ArgumentError", "no such column"},
		{`df.dropna(bogus: 1)`, "ArgumentError", "unknown keyword: :bogus (known: axis, how, thresh)"},
		{`df.dropna(axis: "a")`, "TypeError", "String into Integer"},
		{`df.dropna(axis: 2)`, "ArgumentError", "axis must be 0 or 1"},
		{`df.dropna(1)`, "TypeError", "Integer into String"},
		{`df.dropna(thresh: "a")`, "TypeError", "String into Integer"},
		{`df.dropna(how: 1)`, "TypeError", "Integer into String"},
		{`df.dropna(how: :some)`, "ArgumentError", "how must be :any or :all"},
		{`df.dropna("nope")`, "ArgumentError", "no such column"},
		{`df.fillna({1 => 2})`, "TypeError", "Integer into String"},
		{`df.fillna("nope" => 2)`, "ArgumentError", "no such column"},
		{`df.fillna`, "ArgumentError", "given 0, expected 1"},
		{`df.astype({1 => :int64})`, "TypeError", "Integer into String"},
		{`df.astype("city" => :int64)`, "ArgumentError", ""},
		{`df.astype(1)`, "TypeError", "Integer into a dtype"},
		{`df.astype(:int64)`, "ArgumentError", ""},
		{`df.astype`, "ArgumentError", "given 0, expected 1"},
		{`df.rename(1)`, "ArgumentError", "rename takes a Hash"},
		{`df.rename({"a" => 1})`, "TypeError", "Integer into String"},
		{`df.merge(1)`, "TypeError", "Integer into DataFrame"},
		{`df.merge`, "ArgumentError", "given 0, expected 1"},
		{`df.merge(df, bogus: 1)`, "ArgumentError", "unknown keyword: :bogus"},
		{`df.merge(df, how: "sideways")`, "ArgumentError", ""},
		{`df.join(df, bogus: 1)`, "ArgumentError", "unknown keyword: :bogus (known: how, lsuffix, rsuffix)"},
		{`df.join(df, how: 1)`, "TypeError", "Integer into String"},
		{`df.join(1)`, "TypeError", "Integer into DataFrame"},
		{`df.join(df)`, "ArgumentError", ""},
		{`df.melt(bogus: 1)`, "ArgumentError", "unknown keyword: :bogus (known: id_vars, value_vars)"},
		{`df.melt(1)`, "ArgumentError", "given 1, expected 0"},
		{`df.melt(id_vars: 1)`, "TypeError", "String or an Array"},
		{`df.melt(id_vars: "nope")`, "ArgumentError", "no such column"},
		{`df.query`, "ArgumentError", "given 0, expected 1"},
		{`df.query(1)`, "TypeError", "Integer into String"},
		{`df.query("pop >")`, "ArgumentError", "bad query"},
		{`df.eval(1)`, "TypeError", "Integer into String"},
		{`df.eval("pop >")`, "ArgumentError", "bad query"},
		{`df.iloc`, "ArgumentError", "given 0, expected 1..2"},
		{`df.iloc(0, x: 1)`, "ArgumentError", "unknown keyword"},
		{`df.iloc("a")`, "TypeError", "a position, an Array, a Range or a mask"},
		{`df.iloc(:some)`, "TypeError", "a position, an Array, a Range or a mask"},
		{`df.iloc(["a"])`, "TypeError", "String into Integer"},
		{`df.iloc([true, 1])`, "TypeError", "Integer into true or false"},
		{`df.iloc("a".."b")`, "TypeError", "String into Integer"},
		{`df.iloc(0.."b")`, "TypeError", "String into Integer"},
		{`df.iloc(9)`, "ArgumentError", "out of range"},
		{`df.loc`, "ArgumentError", "given 0, expected 1..2"},
		{`df.loc(0, x: 1)`, "ArgumentError", "unknown keyword"},
		{`df.loc(0...1)`, "ArgumentError", "a label range includes its end"},
		{`df.loc(0, 1)`, "TypeError", "String or an Array"},
		{`df.loc(9)`, "ArgumentError", "not in index"},
		{`df.apply`, "ArgumentError", "no block given"},
		{`df.apply(x: 1) { }`, "ArgumentError", "unknown keyword"},
		{`df.apply(1, 2) { }`, "ArgumentError", "given 2, expected 0..1"},
		{`df.apply(1) { }`, "TypeError", "Integer into String"},
		{`df.apply { |r| raise "boom" }`, "RuntimeError", "boom"},
		{`df.apply { |r| r["pop"] == 522 ? "a" : 1 }`, "ArgumentError", ""},
		{`df.resample("D", bogus: 1)`, "ArgumentError", "unknown keyword: :bogus (known: on)"},
		{`df.resample`, "ArgumentError", "given 0, expected 1"},
		{`df.resample(1)`, "TypeError", "Integer into String"},
		{`df.resample("D", on: 1)`, "TypeError", "Integer into String"},
		{`df.resample("D")`, "ArgumentError", ""},
		{`df.resample("D", on: "city")`, "ArgumentError", ""},
		// Series.
		{`DataFrame::Series.new([1], bogus: 1)`, "ArgumentError", "unknown keyword: :bogus (known: index, name)"},
		{`DataFrame::Series.new`, "ArgumentError", "given 0, expected 1"},
		{`DataFrame::Series.new([1], name: 1)`, "TypeError", "Integer into String"},
		{`DataFrame::Series.new([1], index: [[1]])`, "ArgumentError", ""},
		{`DataFrame::Series.new([1], index: [1, 2])`, "ArgumentError", "length"},
		{`DataFrame::Series.new(1)`, "ArgumentError", "cannot make a column"},
		{`s[]`, "ArgumentError", "given 0, expected 1"},
		{`s["a"]`, "TypeError", "a position, an Array, a Range or a mask"},
		{`s[9]`, "ArgumentError", "out of range"},
		{`s[[9]]`, "ArgumentError", "out of range"},
		{`s.filter(1)`, "TypeError", "Integer into Array"},
		{`s.filter`, "ArgumentError", "given 0, expected 1"},
		{`s.filter([true])`, "ArgumentError", ""},
		{`s.map`, "ArgumentError", "no block given"},
		{`s.map(1) { }`, "ArgumentError", "given 1, expected 0"},
		{`s.map { |v| raise "boom" }`, "RuntimeError", "boom"},
		{`s.map { |v| v == 522 ? "a" : 1 }`, "ArgumentError", ""},
		{`s.rolling(2, bogus: 1)`, "ArgumentError", "unknown keyword: :bogus (known: min_periods)"},
		{`s.rolling`, "ArgumentError", "given 0, expected 1"},
		{`s.rolling("a")`, "TypeError", "String into Integer"},
		{`s.rolling(2, min_periods: "a")`, "TypeError", "String into Integer"},
		{`s.rolling(0)`, "ArgumentError", ""},
		{`s.iloc`, "ArgumentError", "given 0, expected 1"},
		{`s.iloc("a")`, "TypeError", "a position"},
		{`s.loc(9)`, "ArgumentError", "not in index"},
		{`s.iloc([9])`, "ArgumentError", "out of range"},
		{`s.to_datetime(x: 1)`, "ArgumentError", "unknown keyword"},
		{`s.to_datetime(1, 2)`, "ArgumentError", "given 2, expected 0..1"},
		{`s.to_datetime(1)`, "TypeError", "Integer into String"},
		{`str.to_datetime`, "ArgumentError", ""},
		// GroupBy, Resampler, Index.
		{`g.agg(1)`, "TypeError", "Integer into Hash"},
		{`g.agg([[1, :sum]])`, "TypeError", "Integer into String"},
		{`g.agg(v: 1)`, "TypeError", "String or an Array"},
		{`g.agg(nope: :sum)`, "ArgumentError", ""},
		{`g.groups(1)`, "ArgumentError", "given 1, expected 0"},
		{`DataFrame::Index.new([1], bogus: 1)`, "ArgumentError", "unknown keyword: :bogus (known: name)"},
		{`DataFrame::Index.new`, "ArgumentError", "given 0, expected 1"},
		{`DataFrame::Index.new([1], name: 1)`, "TypeError", "Integer into String"},
		{`DataFrame::Index.new(1)`, "ArgumentError", "cannot make a column"},
		{`DataFrame::Index.range`, "ArgumentError", "given 0, expected 1"},
		{`DataFrame::Index.range("a")`, "TypeError", "String into Integer"},
	} {
		evalErr(t, in, c.src, c.class, c.msg)
	}
}

// dfSame must tell apart frames that differ in one value or in their type.
func TestDFSameIsStrict(t *testing.T) {
	a, _ := dataframe.New(dataframe.Col("a", []int64{1, 2}))
	b, _ := dataframe.New(dataframe.Col("a", []int64{1, 3}))
	s, _ := a.Column("a")
	for i, c := range []struct {
		got, want any
		same      bool
	}{
		{a, a, true},
		{a, b, false},
		{s, a, false},
		{"int64", dataframe.Int64, true},
		{"float64", dataframe.Int64, false},
		{[]any{a}, []*dataframe.DataFrame{a}, true},
		{[]any{a}, []*dataframe.DataFrame{b}, false},
		{[]any{a, a}, []*dataframe.DataFrame{a}, false},
		{1, []*dataframe.DataFrame{a}, false},
		{map[any]any{"k": "v"}, map[string][]byte{"k": []byte("v")}, true},
		{map[any]any{"k": "w"}, map[string][]byte{"k": []byte("v")}, false},
		{1, map[string][]byte{"k": []byte("v")}, false},
		{1, time.Unix(0, 0), false},
	} {
		if got := dfSame(c.got, c.want); got != c.same {
			t.Errorf("case %d: dfSame = %v", i, got)
		}
	}
}

// A name already taken by something else is an error of Register.
func TestDataFrameRegisterConflicts(t *testing.T) {
	for _, src := range []string{`module DataFrame; end`, `class DataFrame; module Series; end; end`} {
		in := rb.New()
		if _, err := in.Eval(src, "x.rb", 1); err != nil {
			t.Fatal(err)
		}
		if err := registerDataFrame(in); err == nil || !strings.Contains(err.Error(), "already defined") {
			t.Errorf("%s: registerDataFrame = %v", src, err)
		}
	}
}
