package ruby

// The class DataFrame binds go-dataframe/dataframe, pandas' DataFrame and
// Series in pure Go:
//
//	df = DataFrame.new(city: ["Lyon", "Paris", "Lyon"], pop: [522, 2103, 530])
//	df.group_by("city").mean
//	df[df["pop"] > 525]
//	DataFrame.read_csv(File.read("data.csv"), parse_dates: ["day"])
//
// Bound, by Ruby name (each calls the Go function or method in parentheses;
// "=" marks a default, "kw" an argument that may also be given by keyword):
//
// DataFrame, class methods:
//
//	new(hash) / new(name: data, ...)         (New, Col; columns in the order given)
//	read_csv(text, **opts)                   (ReadCSV)
//	read_csv_file(path, **opts)              (ReadCSVFile)
//	from_array(ndarray, names = [])          (FromArray)
//	from_series(*series)                     (FromSeries)
//	concat(*frames, axis: 0, ignore_index: false, inner: false)  (ConcatWith)
//
// DataFrame, instance methods:
//
//	columns length size shape dtypes index   (Columns Len Len Shape DTypes Index)
//	[](name) [](names) [](mask)              (Column, Select, Filter)
//	column(name) column_at(i) series         (Column ColumnAt Series)
//	head(n = 5) tail(n = 5)                  (Head Tail)
//	describe copy transpose t                (Describe Copy T T)
//	count mean sum std var min max median    (the column reductions)
//	isna notna ffill bfill                   (IsNA NotNA FFill BFill)
//	shift(n = 1) round(decimals = 0)         (Shift Round)
//	drop(*names) select(*names)              (Drop Select)
//	filter(mask)                             (Filter: a bool Series or Array)
//	sort_values(by, ascending: true)         (SortValues; by and ascending
//	                                          a value or an Array)
//	sort_index(ascending: true)              (SortIndex)
//	set_index(*names) reset_index(drop: false) with_index(index)
//	reindex(labels) reindex_columns(*names) set_columns(*names)
//	rename(hash)                             (Rename)
//	with_column(name, data)                  (WithColumn)
//	astype(dtype) / astype(hash)             (AsTypeAll / AsType)
//	dropna(*subset, how: :any, thresh: nil, axis: 0)
//	                                         (DropNA DropNAAll DropNAThresh DropNAColumns)
//	fillna(value) / fillna(hash)             (FillNA / FillNAColumns)
//	group_by(*keys) groupby(*keys)           (GroupBy)
//	merge(right, how:, on:, left_on:, right_on:, suffixes:, indicator:)  (Merge)
//	join(other, how: "left", lsuffix: "", rsuffix: "")  (Join)
//	pivot(index, columns, values)            (Pivot; kw)
//	pivot_table(index, columns, values, aggfunc = "mean")  (PivotTable; kw)
//	melt(id_vars: [], value_vars: [])        (Melt)
//	stack unstack to_period(freq)            (Stack Unstack ToPeriod)
//	query(expr, **vars) eval(expr, **vars)   (Query Eval)
//	iloc(rows, *cols) loc(rows, *cols)       (Iloc Loc; see Selectors)
//	at(label, column) iat(i, j)              (At IAt)
//	apply(name = "") { |row| ... }           (Apply; row is a Hash)
//	resample(rule, on: nil)                  (Resample / ResampleOn)
//	+ - * / add sub mul div                  (Add Sub Mul Div)
//	values to_ndarray                        (Values) -> NDArray
//	to_h                                     {name => Array}, in column order
//	to_csv(**opts) write_csv(path, **opts)   (WriteCSV WriteCSVFile)
//	to_s inspect to_html display             (String String HTML Display)
//
// DataFrame::Series:
//
//	Series.new(data, name: "", index: nil)   (NewSeries / NewSeriesIndexed)
//	name length size count dtype index       (Name Len Len Count DType Index)
//	[](i) [](mask) [](range)                 (At, Filter, Iloc)
//	get(label)                               (Get)
//	values to_ndarray floats to_a valid      (Values Values Floats Items Valid)
//	mean sum std var(ddof = 1) min max median prod nunique
//	                                         (Mean SumValue Std VarDDOF Min Max ...)
//	quantile(q = 0.5) agg(name)              (Quantile Agg)
//	head(n = 5) tail(n = 5) unique value_counts copy to_frame reset_index
//	isna notna dropna fillna(v) ffill bfill  (IsNA NotNA DropNA FillNA FFill BFill)
//	shift(n = 1) diff(n = 1) pct_change(n = 1)
//	cumsum cumprod cummax cummin abs neg round(decimals = 0)
//	rename(name) astype(dtype) filter(mask) reindex(labels) with_index(index)
//	sort_values(ascending: true) sort_index(ascending: true)
//	between(lo, hi) isin(*values)            (Between IsIn)
//	+ - * / % ** add sub mul div mod pow floordiv     (Add Sub ... FloorDiv)
//	> < >= <= gt lt ge le eq ne              (Gt Lt Ge Le Eq Ne) -> a bool Series
//	& | ^ ~ invert                           (And Or Xor Not Not)
//	map { |v| ... }                          (Map)
//	rolling(window, min_periods: nil)        (Rolling / RollingMin)
//	iloc(sel) loc(sel)                       (Iloc Loc)
//	dt                                       (Dt) -> DataFrame::DatetimeFields
//	to_datetime(layout = nil) to_period(freq) to_timestamp unstack
//	to_s inspect to_html display             (String String HTML Display)
//
// DataFrame::GroupBy: agg(col: funcs, ...) (Agg, columns in order), mean sum
// count min max median std var first last nunique size ngroups n_groups
// select(*cols) [](*cols) transform(col, fn) groups ([[key, frame], ...]),
// to_s inspect.
//
// DataFrame::Resampler: agg mean sum count min max median std first last size.
//
// DataFrame::Rolling: mean sum min max std median.
//
// DataFrame::Index: Index.new(data, name: ""), Index.range(n); to_a labels
// length size name names nlevels range? label(i) [](i) get(label) values
// dtype level(k) rename(*names) to_s inspect display.
//
// DataFrame::DatetimeFields: year month day hour minute second day_of_week
// dayofweek day_of_year dayofyear normalize strftime(layout).
//
// read_csv keywords (the CSVOption of each): sep (CSVSeparator), header
// (CSVHeader), names (CSVNames), usecols (CSVUseCols), nrows (CSVNRows),
// skiprows (CSVSkipRows), na_values (CSVNAValues), keep_default_na
// (CSVKeepDefaultNA), parse_dates (CSVParseDates), date_format
// (CSVDateFormat), index_col (CSVIndexCol for a name, CSVIndexColumn for an
// Integer), dtype (CSVDTypes, a Hash), true_values (CSVTrueValues),
// false_values (CSVFalseValues), skip_blank_lines (CSVSkipBlankLines),
// comment (CSVComment), quotechar (CSVQuote; "" for none). to_csv keywords:
// sep, header, index, na_rep, line_terminator.
//
// Values: a column from a Ruby Array is NewSeries over its elements as
// []any, so its dtype is inferred as pandas infers it (Integers int64, Floats
// float64, Strings string, true/false bool, Times datetime, nil missing); an
// NDArray is passed as it is (the same array) and a Series as it is. A Time
// crosses as a time.Time and comes back as a UTC Time; a dtype is a String
// ("int64") or a Symbol (:float64); a Symbol is a String.
//
// Selectors (iloc, loc, Series#[]): an Integer (a position, or for loc a
// label), an Array (positions, labels, or bools as a mask), a bool Series (a
// mask), a Range (positions start...stop or start..stop, for loc an
// inclusive label range a..b, open with nil), or nil / :all (everything).
//
// The keyword columns of DataFrame.new and GroupBy#agg keep their order:
// both are defined in Ruby, taking **kw and passing kw.to_a on, because a
// Go method receives keywords as a map.
//
// rbgo's operator syntax does not reach methods of host-bound objects for
// the arithmetic operators, == and unary minus (rbgo dispatches them to Ruby
// objects only): `s + 1` raises a TypeError there, while `s.+(1)`, `s.add(1)`
// and the comparisons `s > 1` work. Use the named methods (add, sub, mul,
// div, mod, pow, neg, eq, ne) until rbgo dispatches them.
//
// Not bound: the Arrow and Parquet subpackages, dfplot, NewCategorical and
// Categorical, NewSeriesBitmap, NewSeriesMasked and ValidityBitmap (a nil in
// an Array marks a missing value instead), MapFloat (map covers it),
// DataFrame.Apply over Go closures other than the block, GroupBy.Apply and
// ApplyColumns (they take Go functions returning frames), Copy-free storage
// access, Index.Display's siblings HTML, Timestamp, Must, the display limits
// MaxRows/MinRows/MaxColumns and the error values.

import (
	"bytes"
	"reflect"
	"strings"
	"time"

	"github.com/go-dataframe/dataframe"
	rb "github.com/go-embedded-ruby/ruby"
	nd "github.com/go-ndarray/ndarray"
)

var dtypeType = reflect.TypeFor[dataframe.DType]()

func init() {
	// A dtype is named by a String or a Symbol: :int64, "float", "str".
	converters[dtypeType] = func(v any) (reflect.Value, error) {
		s, err := toString(v)
		if err != nil {
			return reflect.Value{}, typeError(v, "a dtype")
		}
		dt, err := dataframe.ParseDType(s)
		if err != nil {
			return reflect.Value{}, goError(err)
		}
		return reflect.ValueOf(dt), nil
	}
}

// dfPrelude defines in Ruby what must see the keywords in order.
const dfPrelude = `
class DataFrame
  def self.new(data = nil, **columns)
    data = data.to_a if data.is_a?(Hash)
    _from_pairs(data, columns.to_a)
  end

  class GroupBy
    def agg(spec = nil, **columns)
      spec = spec.to_a if spec.is_a?(Hash)
      _agg(spec, columns.to_a)
    end
  end

  class Resampler
    def agg(spec = nil, **columns)
      spec = spec.to_a if spec.is_a?(Hash)
      _agg(spec, columns.to_a)
    end
  end
end
`

// dfEnv is what the bindings of one interpreter share: the class Time.
type dfEnv struct {
	time rb.Value
}

// dfM is a Go method bound under a Ruby name: kw names its fixed parameters
// for keyword use, def gives defaults for the last ones.
type dfM struct {
	ruby, goName string
	kw           []string
	def          []any
}

func registerDataFrame(in *rb.Interp) error {
	d := &definer{in: in}
	frame, err := in.DefineClass("DataFrame", nil)
	d.keep(err)
	if d.err != nil {
		return d.err
	}
	series := d.class(frame, "Series")
	group := d.class(frame, "GroupBy")
	resampler := d.class(frame, "Resampler")
	rolling := d.class(frame, "Rolling")
	index := d.class(frame, "Index")
	dt := d.class(frame, "DatetimeFields")
	if d.err != nil {
		return d.err
	}
	tc, _ := in.Eval("Time", "(dataframe)", 1)
	e := &dfEnv{time: tc}
	frame.Bind(&dataframe.DataFrame{})
	series.Bind(&dataframe.Series{})
	group.Bind(&dataframe.GroupBy{})
	resampler.Bind(&dataframe.Resampler{})
	rolling.Bind(&dataframe.Rolling{})
	index.Bind(&dataframe.Index{})
	dt.Bind(dataframe.DatetimeFields{})

	e.bind(frame, dfFrameMethods)
	e.bind(series, dfSeriesMethods)
	e.bind(group, dfGroupMethods)
	e.bind(resampler, dfResamplerMethods)
	e.bind(rolling, dfRollingMethods)
	e.bind(index, dfIndexMethods)
	e.bind(dt, dfDatetimeMethods)

	frame.DefineSingletonMethod("_from_pairs", e.fromPairs)
	frame.DefineSingletonMethod("read_csv", e.readCSV)
	frame.DefineSingletonMethod("read_csv_file", e.readCSVFile)
	frame.DefineSingletonMethod("from_array", e.fromArray)
	frame.DefineSingletonMethod("from_series", e.fromSeries)
	frame.DefineSingletonMethod("concat", e.concat)
	for name, f := range map[string]func(*rb.Call) (any, error){
		"[]": e.frameIndex, "filter": e.frameFilter, "dtypes": e.dtypes, "to_h": e.toH,
		"to_csv": e.toCSV, "write_csv": e.writeCSV, "sort_values": e.sortValues,
		"group_by": e.groupBy, "groupby": e.groupBy, "dropna": e.dropNA, "fillna": e.fillNA,
		"astype": e.asType, "rename": e.rename, "merge": e.merge, "join": e.join,
		"melt": e.melt, "query": e.query, "eval": e.eval, "iloc": e.frameIloc,
		"loc": e.frameLoc, "apply": e.apply, "resample": e.resample,
	} {
		frame.DefineMethod(name, f)
	}
	series.DefineSingletonMethod("new", e.newSeries)
	for name, f := range map[string]func(*rb.Call) (any, error){
		"[]": e.seriesIndex, "filter": e.seriesFilter, "map": e.seriesMap,
		"rolling": e.rolling, "iloc": e.seriesIloc, "loc": e.seriesLoc,
		"to_datetime": e.toDatetime,
	} {
		series.DefineMethod(name, f)
	}
	group.DefineMethod("_agg", e.agg)
	group.DefineMethod("groups", e.groups)
	resampler.DefineMethod("_agg", e.agg)
	index.DefineSingletonMethod("new", e.newIndex)
	index.DefineSingletonMethod("range", e.rangeIndex)
	_, err = in.Eval(dfPrelude, "(dataframe)", 1)
	return err
}

// bind defines each method of table on cls, calling the Go method of the
// receiver.
func (e *dfEnv) bind(cls *rb.Module, table []dfM) {
	for _, m := range table {
		cls.DefineMethod(m.ruby, func(c *rb.Call) (any, error) {
			fn := reflect.ValueOf(c.Self.Go()).MethodByName(m.goName)
			args := make([]any, len(c.Args))
			for i, a := range c.Args {
				args[i] = a.Go()
			}
			return e.call(fn, m, args, c.Kwargs)
		})
	}
}

// call calls a Go function: positional arguments in order, keywords at the
// positions m.kw names, defaults for the last fixed parameters left out, and
// the remaining arguments (or a single Array) for a variadic parameter.
func (e *dfEnv) call(fn reflect.Value, m dfM, args []any, kw map[string]rb.Value) (any, error) {
	ft := fn.Type()
	fixed := ft.NumIn()
	if ft.IsVariadic() {
		fixed--
	}
	vals := make([]any, max(len(args), fixed))
	set := make([]bool, len(vals))
	copy(vals, args)
	for i := range args {
		set[i] = true
	}
	for _, k := range sortedKeys(kw) {
		i := indexOf(m.kw, k)
		if i < 0 || i >= fixed {
			known := "none"
			if len(m.kw) > 0 {
				known = strings.Join(m.kw, ", ")
			}
			return nil, argError("unknown keyword: :%s (known: %s)", k, known)
		}
		if set[i] {
			return nil, argError("%s given both by position and by keyword", k)
		}
		vals[i], set[i] = kw[k].Go(), true
	}
	for i := range fixed {
		if !set[i] {
			if j := i - (fixed - len(m.def)); j >= 0 {
				vals[i] = m.def[j]
				continue
			}
			return nil, wrongArgs(len(args), ft)
		}
	}
	if !ft.IsVariadic() && len(vals) > fixed {
		return nil, wrongArgs(len(args), ft)
	}
	in := make([]reflect.Value, 0, len(vals))
	for i := range fixed {
		v, err := dfArg(vals[i], ft.In(i))
		if err != nil {
			return nil, err
		}
		in = append(in, v)
	}
	if ft.IsVariadic() {
		tail := vals[fixed:]
		if len(tail) == 1 {
			if l, ok := tail[0].([]any); ok {
				tail = l
			}
		}
		for _, x := range tail {
			v, err := dfArg(x, ft.In(fixed).Elem())
			if err != nil {
				return nil, err
			}
			in = append(in, v)
		}
	}
	return e.results(fn.Call(in))
}

func indexOf(list []string, s string) int {
	for i, e := range list {
		if e == s {
			return i
		}
	}
	return -1
}

// dfArg converts a Ruby value to a parameter: an any parameter takes the
// value as dfValue makes it, the others convert as for every binding.
func dfArg(x any, t reflect.Type) (reflect.Value, error) {
	if t.Kind() == reflect.Interface {
		v := dfValue(x)
		return reflect.ValueOf(&v).Elem(), nil
	}
	return convert(x, t)
}

// dfValue is a Ruby value as go-dataframe takes data and scalars: a Symbol
// is a string, a Time a time.Time, an Array a []any of such values; the rest
// (numbers, strings, booleans, nil, NDArrays, Series) is the value itself.
func dfValue(x any) any {
	switch v := x.(type) {
	case rb.Symbol:
		return string(v)
	case []any:
		out := make([]any, len(v))
		for i, e := range v {
			out[i] = dfValue(e)
		}
		return out
	case rb.Value:
		if v.ClassName() == "Time" {
			s, _ := v.Call("to_i")
			ns, _ := v.Call("nsec")
			return time.Unix(s.Go().(int64), ns.Go().(int64)).UTC()
		}
	}
	return x
}

// results converts what a Go method returned (see out).
func (e *dfEnv) results(vals []reflect.Value) (any, error) {
	if n := len(vals); n > 0 && vals[n-1].Type() == errorType {
		if err, _ := vals[n-1].Interface().(error); err != nil {
			return nil, goError(err)
		}
		vals = vals[:n-1]
	}
	if len(vals) == 1 {
		return e.out(vals[0]), nil
	}
	list := make([]any, len(vals))
	for i, v := range vals {
		list[i] = e.out(v)
	}
	return list, nil
}

// out converts a Go result for Ruby: a time.Time is a UTC Time, a DType its
// name, slices element by element (nil gives []); the rest as every binding converts it.
func (e *dfEnv) out(v reflect.Value) any {
	if v.Kind() == reflect.Interface {
		if v.IsNil() {
			return nil
		}
		v = v.Elem()
	}
	switch x := v.Interface().(type) {
	case time.Time:
		t, _ := e.time.Call("at", x.Unix(), int64(x.Nanosecond()), rb.Symbol("nsec"))
		u, _ := t.Call("utc")
		return u
	case dataframe.DType:
		return x.String()
	case []float64, []byte:
		return out(v)
	}
	if v.Kind() == reflect.Slice { // a nil slice is an empty Array
		list := make([]any, v.Len())
		for i := range list {
			list[i] = e.out(v.Index(i))
		}
		return list
	}
	return out(v)
}

// outAny converts a Go value of unknown type.
func (e *dfEnv) outAny(x any) any { return e.out(reflect.ValueOf(&x).Elem()) }

// noKw rejects keyword arguments.
func noKw(c *rb.Call) error {
	if len(c.Kwargs) > 0 {
		return argError("unknown keyword: :%s", sortedKeys(c.Kwargs)[0])
	}
	return nil
}

// kwOnly checks the keywords are among names.
func kwOnly(c *rb.Call, names ...string) error {
	for _, k := range sortedKeys(c.Kwargs) {
		if !contains(names, k) {
			return argError("unknown keyword: :%s (known: %s)", k, strings.Join(names, ", "))
		}
	}
	return nil
}

func frameOf(c *rb.Call) *dataframe.DataFrame { return c.Self.Go().(*dataframe.DataFrame) }
func seriesOf(c *rb.Call) *dataframe.Series   { return c.Self.Go().(*dataframe.Series) }

// strs reads a String, a Symbol or an Array of them.
func strs(x any) ([]string, error) {
	if l, ok := x.([]any); ok {
		out := make([]string, len(l))
		for i, e := range l {
			s, err := toString(e)
			if err != nil {
				return nil, err
			}
			out[i] = s
		}
		return out, nil
	}
	s, err := toString(x)
	if err != nil {
		return nil, typeError(x, "String or an Array of Strings")
	}
	return []string{s}, nil
}

// argStrs reads every positional argument (or a single Array) as names.
func argStrs(c *rb.Call) ([]string, error) {
	var list []any
	for _, a := range c.Args {
		list = append(list, a.Go())
	}
	if len(list) == 1 {
		if l, ok := list[0].([]any); ok {
			list = l
		}
	}
	return strs(list)
}

// pairs reads the [name, value] pairs DataFrame.new and agg receive.
func pairs(xs ...any) ([][2]any, error) {
	var out [][2]any
	for _, x := range xs {
		if x == nil {
			continue
		}
		l, ok := x.([]any)
		if !ok {
			return nil, typeError(x, "Hash")
		}
		for _, p := range l {
			kv, ok := p.([]any)
			if !ok || len(kv) != 2 {
				return nil, typeError(p, "a [name, value] pair")
			}
			out = append(out, [2]any{kv[0], kv[1]})
		}
	}
	return out, nil
}

func (e *dfEnv) fromPairs(c *rb.Call) (any, error) {
	ps, err := pairs(c.Args[0].Go(), c.Args[1].Go())
	if err != nil {
		return nil, err
	}
	cols := make([]dataframe.Column, len(ps))
	for i, p := range ps {
		name, err := toString(p[0])
		if err != nil {
			return nil, err
		}
		cols[i] = dataframe.Col(name, dfValue(p[1]))
	}
	return e.results(reflect.ValueOf(dataframe.New).Call(colValues(cols)))
}

func colValues(cols []dataframe.Column) []reflect.Value {
	out := make([]reflect.Value, len(cols))
	for i, c := range cols {
		out[i] = reflect.ValueOf(c)
	}
	return out
}

// byte1 reads a one-character String as a byte; "" (or nil) is 0.
func byte1(x any) (byte, error) {
	if x == nil {
		return 0, nil
	}
	s, err := toString(x)
	if err != nil {
		return 0, err
	}
	if len(s) > 1 {
		return 0, argError("expected a single character, got %q", s)
	}
	if s == "" {
		return 0, nil
	}
	return s[0], nil
}

// csvOptions converts read_csv's keywords to CSV options, in keyword order.
func csvOptions(kw map[string]rb.Value) ([]dataframe.CSVOption, error) {
	var opts []dataframe.CSVOption
	for _, k := range sortedKeys(kw) {
		x := kw[k].Go()
		var o dataframe.CSVOption
		var err error
		switch k {
		case "sep", "comment", "quotechar":
			var b byte
			b, err = byte1(x)
			o = map[string]func(byte) dataframe.CSVOption{
				"sep": dataframe.CSVSeparator, "comment": dataframe.CSVComment, "quotechar": dataframe.CSVQuote,
			}[k](b)
		case "header", "keep_default_na", "skip_blank_lines":
			b, ok := x.(bool)
			if !ok {
				return nil, typeError(x, "true or false")
			}
			o = map[string]func(bool) dataframe.CSVOption{
				"header": dataframe.CSVHeader, "keep_default_na": dataframe.CSVKeepDefaultNA,
				"skip_blank_lines": dataframe.CSVSkipBlankLines,
			}[k](b)
		case "names", "usecols", "na_values", "parse_dates", "true_values", "false_values":
			var s []string
			s, err = strs(x)
			o = map[string]func(...string) dataframe.CSVOption{
				"names": dataframe.CSVNames, "usecols": dataframe.CSVUseCols, "na_values": dataframe.CSVNAValues,
				"parse_dates": dataframe.CSVParseDates, "true_values": dataframe.CSVTrueValues,
				"false_values": dataframe.CSVFalseValues,
			}[k](s...)
		case "nrows", "skiprows":
			var n int64
			n, err = toInt(x)
			o = map[string]func(int) dataframe.CSVOption{
				"nrows": dataframe.CSVNRows, "skiprows": dataframe.CSVSkipRows,
			}[k](int(n))
		case "date_format":
			var s string
			s, err = toString(x)
			o = dataframe.CSVDateFormat(s)
		case "index_col":
			if i, ok := x.(int64); ok {
				o = dataframe.CSVIndexColumn(int(i))
				break
			}
			var s string
			s, err = toString(x)
			o = dataframe.CSVIndexCol(s)
		case "dtype":
			var v reflect.Value
			v, err = convert(x, reflect.TypeFor[map[string]dataframe.DType]())
			if err == nil {
				o = dataframe.CSVDTypes(v.Interface().(map[string]dataframe.DType))
			}
		default:
			return nil, argError("unknown keyword: :%s (known: comment, date_format, dtype, false_values, header, index_col, keep_default_na, na_values, names, nrows, parse_dates, quotechar, sep, skip_blank_lines, skiprows, true_values, usecols)", k)
		}
		if err != nil {
			return nil, err
		}
		opts = append(opts, o)
	}
	return opts, nil
}

// readArgs reads (text_or_path, **opts).
func readArgs(c *rb.Call) (string, []dataframe.CSVOption, error) {
	if err := arity(c, 1, 1); err != nil {
		return "", nil, err
	}
	s, err := toString(c.Args[0].Go())
	if err != nil {
		return "", nil, err
	}
	opts, err := csvOptions(c.Kwargs)
	return s, opts, err
}

func (e *dfEnv) readCSV(c *rb.Call) (any, error) {
	text, opts, err := readArgs(c)
	if err != nil {
		return nil, err
	}
	return e.result(dataframe.ReadCSV(strings.NewReader(text), opts...))
}

func (e *dfEnv) readCSVFile(c *rb.Call) (any, error) {
	path, opts, err := readArgs(c)
	if err != nil {
		return nil, err
	}
	return e.result(dataframe.ReadCSVFile(path, opts...))
}

// result converts a Go (value, error) pair.
func (e *dfEnv) result(v any, err error) (any, error) {
	if err != nil {
		return nil, goError(err)
	}
	return e.outAny(v), nil
}

func (e *dfEnv) fromArray(c *rb.Call) (any, error) {
	if err := noKw(c); err != nil {
		return nil, err
	}
	if err := arity(c, 1, 2); err != nil {
		return nil, err
	}
	a, ok := c.Args[0].Go().(*nd.Array)
	if !ok {
		return nil, typeError(c.Args[0].Go(), "NDArray")
	}
	var names []string
	if len(c.Args) == 2 {
		n, err := strs(c.Args[1].Go())
		if err != nil {
			return nil, err
		}
		names = n
	}
	return e.result(dataframe.FromArray(a, names...))
}

// seriesArgs reads every positional argument as a Series.
func seriesArgs(c *rb.Call) ([]*dataframe.Series, error) {
	out := make([]*dataframe.Series, len(c.Args))
	for i, a := range c.Args {
		s, ok := a.Go().(*dataframe.Series)
		if !ok {
			return nil, typeError(a.Go(), "DataFrame::Series")
		}
		out[i] = s
	}
	return out, nil
}

func (e *dfEnv) fromSeries(c *rb.Call) (any, error) {
	if err := noKw(c); err != nil {
		return nil, err
	}
	ss, err := seriesArgs(c)
	if err != nil {
		return nil, err
	}
	return e.result(dataframe.FromSeries(ss...))
}

func (e *dfEnv) concat(c *rb.Call) (any, error) {
	var opts dataframe.ConcatOptions
	if err := options(&opts, c.Kwargs); err != nil {
		return nil, err
	}
	frames := make([]*dataframe.DataFrame, len(c.Args))
	for i, a := range c.Args {
		f, ok := a.Go().(*dataframe.DataFrame)
		if !ok {
			return nil, typeError(a.Go(), "DataFrame")
		}
		frames[i] = f
	}
	return e.result(dataframe.ConcatWith(opts, frames...))
}

// mask reads a filter mask: a bool Series as it is, an Array as a []bool.
func mask(x any) (any, error) {
	if s, ok := x.(*dataframe.Series); ok {
		return s, nil
	}
	v, err := convert(x, reflect.TypeFor[[]bool]())
	if err != nil {
		return nil, err
	}
	return v.Interface(), nil
}

// isMask reports whether a [] argument is a mask: a Series, or an Array
// whose first element is a boolean.
func isMask(x any) bool {
	switch v := x.(type) {
	case *dataframe.Series:
		return true
	case []any:
		if len(v) > 0 {
			_, ok := v[0].(bool)
			return ok
		}
	}
	return false
}

func (e *dfEnv) frameIndex(c *rb.Call) (any, error) {
	if err := plExact(c, 1); err != nil {
		return nil, err
	}
	df := frameOf(c)
	x := c.Args[0].Go()
	if isMask(x) {
		return e.frameFilter(c)
	}
	if l, ok := x.([]any); ok {
		names, err := strs(l)
		if err != nil {
			return nil, err
		}
		return e.result(df.Select(names...))
	}
	name, err := toString(x)
	if err != nil {
		return nil, typeError(x, "a column name, an Array of names or a mask")
	}
	return e.result(df.Column(name))
}

func (e *dfEnv) frameFilter(c *rb.Call) (any, error) {
	if err := plExact(c, 1); err != nil {
		return nil, err
	}
	m, err := mask(c.Args[0].Go())
	if err != nil {
		return nil, err
	}
	return e.result(frameOf(c).Filter(m))
}

// orderedHash builds a Ruby Hash from pairs, in their order.
func orderedHash(c *rb.Call, ps []any) (any, error) {
	list, _ := c.Interp.ValueOf(ps)
	return list.Call("to_h")
}

func (e *dfEnv) dtypes(c *rb.Call) (any, error) {
	if err := plExact(c, 0); err != nil {
		return nil, err
	}
	df := frameOf(c)
	dts := df.DTypes()
	ps := make([]any, len(dts))
	for i, name := range df.Columns() {
		ps[i] = []any{name, dts[i].String()}
	}
	return orderedHash(c, ps)
}

func (e *dfEnv) toH(c *rb.Call) (any, error) {
	if err := plExact(c, 0); err != nil {
		return nil, err
	}
	var ps []any
	for _, s := range frameOf(c).Series() {
		ps = append(ps, []any{s.Name(), e.outAny(s.Items())})
	}
	return orderedHash(c, ps)
}

// csvWriteOptions converts to_csv's keywords.
func csvWriteOptions(kw map[string]rb.Value) ([]dataframe.CSVWriteOption, error) {
	var opts []dataframe.CSVWriteOption
	for _, k := range sortedKeys(kw) {
		x := kw[k].Go()
		switch k {
		case "sep":
			b, err := byte1(x)
			if err != nil {
				return nil, err
			}
			opts = append(opts, dataframe.CSVWriteSeparator(b))
		case "header", "index":
			b, ok := x.(bool)
			if !ok {
				return nil, typeError(x, "true or false")
			}
			f := dataframe.CSVWriteHeader
			if k == "index" {
				f = dataframe.CSVWriteIndex
			}
			opts = append(opts, f(b))
		case "na_rep", "line_terminator":
			s, err := toString(x)
			if err != nil {
				return nil, err
			}
			f := dataframe.CSVWriteNARep
			if k == "line_terminator" {
				f = dataframe.CSVWriteLineTerminator
			}
			opts = append(opts, f(s))
		default:
			return nil, argError("unknown keyword: :%s (known: header, index, line_terminator, na_rep, sep)", k)
		}
	}
	return opts, nil
}

func (e *dfEnv) toCSV(c *rb.Call) (any, error) {
	if err := arity(c, 0, 0); err != nil {
		return nil, err
	}
	opts, err := csvWriteOptions(c.Kwargs)
	if err != nil {
		return nil, err
	}
	var b bytes.Buffer
	_ = frameOf(c).WriteCSV(&b, opts...) // writing to memory cannot fail
	return b.String(), nil
}

func (e *dfEnv) writeCSV(c *rb.Call) (any, error) {
	if err := arity(c, 1, 1); err != nil {
		return nil, err
	}
	path, err := toString(c.Args[0].Go())
	if err != nil {
		return nil, err
	}
	opts, err := csvWriteOptions(c.Kwargs)
	if err != nil {
		return nil, err
	}
	if err := frameOf(c).WriteCSVFile(path, opts...); err != nil {
		return nil, rb.NewError("IOError", err.Error())
	}
	return nil, nil
}

func (e *dfEnv) sortValues(c *rb.Call) (any, error) {
	if err := kwOnly(c, "ascending"); err != nil {
		return nil, err
	}
	if err := arity(c, 1, 1); err != nil {
		return nil, err
	}
	by, err := strs(c.Args[0].Go())
	if err != nil {
		return nil, err
	}
	var asc []bool
	if a, ok := c.Kwargs["ascending"]; ok {
		x := a.Go()
		if _, isList := x.([]any); !isList {
			x = []any{x}
		}
		v, err := convert(x, reflect.TypeFor[[]bool]())
		if err != nil {
			return nil, err
		}
		asc = v.Interface().([]bool)
	}
	return e.result(frameOf(c).SortValues(by, asc...))
}

func (e *dfEnv) groupBy(c *rb.Call) (any, error) {
	if err := noKw(c); err != nil {
		return nil, err
	}
	keys, err := argStrs(c)
	if err != nil {
		return nil, err
	}
	return e.result(frameOf(c).GroupBy(keys...))
}

func (e *dfEnv) dropNA(c *rb.Call) (any, error) {
	if err := kwOnly(c, "axis", "how", "thresh"); err != nil {
		return nil, err
	}
	df := frameOf(c)
	if a, ok := c.Kwargs["axis"]; ok {
		n, err := toInt(a.Go())
		if err != nil {
			return nil, err
		}
		if n == 1 {
			return df.DropNAColumns(), nil
		}
		if n != 0 {
			return nil, argError("axis must be 0 or 1, got %d", n)
		}
	}
	subset, err := argStrs(c)
	if err != nil {
		return nil, err
	}
	if t, ok := c.Kwargs["thresh"]; ok {
		n, err := toInt(t.Go())
		if err != nil {
			return nil, err
		}
		return e.result(df.DropNAThresh(int(n), subset...))
	}
	if h, ok := c.Kwargs["how"]; ok {
		how, err := toString(h.Go())
		if err != nil {
			return nil, err
		}
		switch how {
		case "all":
			return e.result(df.DropNAAll(subset...))
		case "any":
		default:
			return nil, argError("how must be :any or :all, got %q", how)
		}
	}
	return e.result(df.DropNA(subset...))
}

// valueMap reads a Hash of column name to value.
func valueMap(x map[any]any) (map[string]any, error) {
	out := make(map[string]any, len(x))
	for k, v := range x {
		s, err := toString(k)
		if err != nil {
			return nil, err
		}
		out[s] = dfValue(v)
	}
	return out, nil
}

// hashArg reads a call's single Hash argument, given as a Hash or as
// keywords.
func hashArg(c *rb.Call) (map[any]any, bool) {
	if len(c.Args) == 0 && len(c.Kwargs) > 0 {
		h := map[any]any{}
		for k, v := range c.Kwargs {
			h[k] = v.Go()
		}
		return h, true
	}
	if len(c.Args) == 1 && len(c.Kwargs) == 0 {
		h, ok := c.Args[0].Go().(map[any]any)
		return h, ok
	}
	return nil, false
}

func (e *dfEnv) fillNA(c *rb.Call) (any, error) {
	if h, ok := hashArg(c); ok {
		m, err := valueMap(h)
		if err != nil {
			return nil, err
		}
		return e.result(frameOf(c).FillNAColumns(m))
	}
	if err := plExact(c, 1); err != nil {
		return nil, err
	}
	return frameOf(c).FillNA(dfValue(c.Args[0].Go())), nil
}

func (e *dfEnv) asType(c *rb.Call) (any, error) {
	if h, ok := hashArg(c); ok {
		v, err := convert(h, reflect.TypeFor[map[string]dataframe.DType]())
		if err != nil {
			return nil, err
		}
		return e.result(frameOf(c).AsType(v.Interface().(map[string]dataframe.DType)))
	}
	if err := plExact(c, 1); err != nil {
		return nil, err
	}
	v, err := convert(c.Args[0].Go(), dtypeType)
	if err != nil {
		return nil, err
	}
	return e.result(frameOf(c).AsTypeAll(v.Interface().(dataframe.DType)))
}

func (e *dfEnv) rename(c *rb.Call) (any, error) {
	h, ok := hashArg(c)
	if !ok {
		return nil, argError("rename takes a Hash of old name to new name")
	}
	v, err := convert(h, reflect.TypeFor[map[string]string]())
	if err != nil {
		return nil, err
	}
	return frameOf(c).Rename(v.Interface().(map[string]string)), nil
}

// otherFrame reads the single positional DataFrame of merge and join.
func otherFrame(c *rb.Call) (*dataframe.DataFrame, error) {
	if err := arity(c, 1, 1); err != nil {
		return nil, err
	}
	o, ok := c.Args[0].Go().(*dataframe.DataFrame)
	if !ok {
		return nil, typeError(c.Args[0].Go(), "DataFrame")
	}
	return o, nil
}

func (e *dfEnv) merge(c *rb.Call) (any, error) {
	right, err := otherFrame(c)
	if err != nil {
		return nil, err
	}
	var opts dataframe.MergeOptions
	if err := options(&opts, c.Kwargs); err != nil {
		return nil, err
	}
	return e.result(frameOf(c).Merge(right, opts))
}

// kwStrings reads string keywords with defaults.
func kwStrings(c *rb.Call, defaults map[string]string, names ...string) ([]string, error) {
	if err := kwOnly(c, names...); err != nil {
		return nil, err
	}
	out := make([]string, len(names))
	for i, n := range names {
		out[i] = defaults[n]
		if v, ok := c.Kwargs[n]; ok {
			s, err := toString(v.Go())
			if err != nil {
				return nil, err
			}
			out[i] = s
		}
	}
	return out, nil
}

func (e *dfEnv) join(c *rb.Call) (any, error) {
	s, err := kwStrings(c, map[string]string{"how": "left"}, "how", "lsuffix", "rsuffix")
	if err != nil {
		return nil, err
	}
	other, err := otherFrame(c)
	if err != nil {
		return nil, err
	}
	return e.result(frameOf(c).Join(other, s[0], s[1], s[2]))
}

func (e *dfEnv) melt(c *rb.Call) (any, error) {
	if err := kwOnly(c, "id_vars", "value_vars"); err != nil {
		return nil, err
	}
	if err := arity(c, 0, 0); err != nil {
		return nil, err
	}
	vars := [2][]string{}
	for i, k := range []string{"id_vars", "value_vars"} {
		if v, ok := c.Kwargs[k]; ok {
			s, err := strs(v.Go())
			if err != nil {
				return nil, err
			}
			vars[i] = s
		}
	}
	return e.result(frameOf(c).Melt(vars[0], vars[1]))
}

// exprArgs reads (expr, **vars).
func exprArgs(c *rb.Call) (string, []map[string]any, error) {
	if err := arity(c, 1, 1); err != nil {
		return "", nil, err
	}
	expr, err := toString(c.Args[0].Go())
	if err != nil {
		return "", nil, err
	}
	var vars []map[string]any
	if len(c.Kwargs) > 0 {
		m := map[string]any{}
		for k, v := range c.Kwargs {
			m[k] = dfValue(v.Go())
		}
		vars = append(vars, m)
	}
	return expr, vars, nil
}

func (e *dfEnv) query(c *rb.Call) (any, error) {
	expr, vars, err := exprArgs(c)
	if err != nil {
		return nil, err
	}
	return e.result(frameOf(c).Query(expr, vars...))
}

func (e *dfEnv) eval(c *rb.Call) (any, error) {
	expr, vars, err := exprArgs(c)
	if err != nil {
		return nil, err
	}
	return e.result(frameOf(c).Eval(expr, vars...))
}

// selector reads a row (or column) selector; positional says whether
// numbers and ranges are positions (iloc) or labels (loc).
func selector(x any, positional bool) (dataframe.Selector, error) {
	switch v := x.(type) {
	case nil:
		return dataframe.All(), nil
	case rb.Symbol:
		if v == "all" {
			return dataframe.All(), nil
		}
	case *dataframe.Series:
		return dataframe.Mask(v), nil
	case []any:
		if isMask(v) {
			m, err := mask(v)
			if err != nil {
				return dataframe.Selector{}, err
			}
			return dataframe.Mask(m), nil
		}
		if positional {
			p, err := convert(v, reflect.TypeFor[[]int]())
			if err != nil {
				return dataframe.Selector{}, err
			}
			return dataframe.Pos(p.Interface().([]int)...), nil
		}
		return dataframe.Labels(dfValue(v).([]any)...), nil
	case int64:
		if positional {
			return dataframe.Pos(int(v)), nil
		}
	case rb.Value:
		if v.ClassName() == "Range" {
			return rangeSelector(v, positional)
		}
	}
	if positional {
		return dataframe.Selector{}, typeError(x, "a position, an Array, a Range or a mask")
	}
	return dataframe.Labels(dfValue(x)), nil
}

// rangeSelector reads a Range: positions (an exclusive or inclusive end, nil
// for open) or an inclusive range of labels.
func rangeSelector(r rb.Value, positional bool) (dataframe.Selector, error) {
	b, _ := r.Call("begin")
	en, _ := r.Call("end")
	ex, _ := r.Call("exclude_end?")
	lo, hi := b.Go(), en.Go()
	if !positional {
		if ex.Go() == true {
			return dataframe.Selector{}, argError("a label range includes its end: use a..b, not a...b")
		}
		return dataframe.LabelRange(dfValue(lo), dfValue(hi)), nil
	}
	start := int64(0)
	if lo != nil {
		s, err := toInt(lo)
		if err != nil {
			return dataframe.Selector{}, err
		}
		start = s
	}
	if hi == nil {
		return dataframe.SliceFrom(int(start)), nil
	}
	stop, err := toInt(hi)
	if err != nil {
		return dataframe.Selector{}, err
	}
	if ex.Go() != true {
		stop++
	}
	return dataframe.Slice(int(start), int(stop)), nil
}

func (e *dfEnv) frameIloc(c *rb.Call) (any, error) {
	if err := noKw(c); err != nil {
		return nil, err
	}
	if err := arity(c, 1, 2); err != nil {
		return nil, err
	}
	sels := make([]dataframe.Selector, len(c.Args))
	for i, a := range c.Args {
		s, err := selector(a.Go(), true)
		if err != nil {
			return nil, err
		}
		sels[i] = s
	}
	return e.result(frameOf(c).Iloc(sels[0], sels[1:]...))
}

func (e *dfEnv) frameLoc(c *rb.Call) (any, error) {
	if err := noKw(c); err != nil {
		return nil, err
	}
	if err := arity(c, 1, 2); err != nil {
		return nil, err
	}
	rows, err := selector(c.Args[0].Go(), false)
	if err != nil {
		return nil, err
	}
	var cols []string
	if len(c.Args) == 2 {
		if cols, err = strs(c.Args[1].Go()); err != nil {
			return nil, err
		}
	}
	return e.result(frameOf(c).Loc(rows, cols...))
}

func (e *dfEnv) apply(c *rb.Call) (any, error) {
	if err := noKw(c); err != nil {
		return nil, err
	}
	if err := arity(c, 0, 1); err != nil {
		return nil, err
	}
	b, err := needBlock(c)
	if err != nil {
		return nil, err
	}
	name := ""
	if len(c.Args) == 1 {
		if name, err = toString(c.Args[0].Go()); err != nil {
			return nil, err
		}
	}
	df := frameOf(c)
	cols := df.Columns()
	var blockErr error
	s, err := df.Apply(name, func(r dataframe.Row) any {
		if blockErr != nil {
			return nil
		}
		ps := make([]any, len(cols))
		for i, col := range cols {
			ps[i] = []any{col, e.outAny(r.Get(col))}
		}
		h, _ := orderedHash(c, ps)
		v, err := b.Call(h)
		if err != nil {
			blockErr = err
			return nil
		}
		return dfValue(v.Go())
	})
	if blockErr != nil {
		return nil, blockErr
	}
	return e.result(s, err)
}

func (e *dfEnv) resample(c *rb.Call) (any, error) {
	if err := kwOnly(c, "on"); err != nil {
		return nil, err
	}
	if err := arity(c, 1, 1); err != nil {
		return nil, err
	}
	rule, err := toString(c.Args[0].Go())
	if err != nil {
		return nil, err
	}
	if on, ok := c.Kwargs["on"]; ok {
		col, err := toString(on.Go())
		if err != nil {
			return nil, err
		}
		return e.result(frameOf(c).ResampleOn(col, rule))
	}
	return e.result(frameOf(c).Resample(rule))
}

// newIndexArg reads an index: an Index as it is, else labels for NewIndex.
func newIndexArg(x any) (*dataframe.Index, error) {
	if ix, ok := x.(*dataframe.Index); ok {
		return ix, nil
	}
	ix, err := dataframe.NewIndex("", dfValue(x))
	if err != nil {
		return nil, goError(err)
	}
	return ix, nil
}

func (e *dfEnv) newSeries(c *rb.Call) (any, error) {
	if err := kwOnly(c, "index", "name"); err != nil {
		return nil, err
	}
	if err := arity(c, 1, 1); err != nil {
		return nil, err
	}
	name := ""
	if n, ok := c.Kwargs["name"]; ok {
		s, err := toString(n.Go())
		if err != nil {
			return nil, err
		}
		name = s
	}
	data := dfValue(c.Args[0].Go())
	if ixv, ok := c.Kwargs["index"]; ok {
		ix, err := newIndexArg(ixv.Go())
		if err != nil {
			return nil, err
		}
		return e.result(dataframe.NewSeriesIndexed(name, data, ix))
	}
	return e.result(dataframe.NewSeries(name, data))
}

func (e *dfEnv) seriesIndex(c *rb.Call) (any, error) {
	if err := plExact(c, 1); err != nil {
		return nil, err
	}
	x := c.Args[0].Go()
	s := seriesOf(c)
	if isMask(x) {
		return e.seriesFilter(c)
	}
	if i, ok := x.(int64); ok {
		return e.result(s.At(int(i)))
	}
	sel, err := selector(x, true)
	if err != nil {
		return nil, err
	}
	return e.result(s.Iloc(sel))
}

func (e *dfEnv) seriesFilter(c *rb.Call) (any, error) {
	if err := plExact(c, 1); err != nil {
		return nil, err
	}
	m, err := mask(c.Args[0].Go())
	if err != nil {
		return nil, err
	}
	return e.result(seriesOf(c).Filter(m))
}

func (e *dfEnv) seriesMap(c *rb.Call) (any, error) {
	if err := plExact(c, 0); err != nil {
		return nil, err
	}
	b, err := needBlock(c)
	if err != nil {
		return nil, err
	}
	var blockErr error
	s, err := seriesOf(c).Map(func(x any) any {
		if blockErr != nil {
			return nil
		}
		v, err := b.Call(e.outAny(x))
		if err != nil {
			blockErr = err
			return nil
		}
		return dfValue(v.Go())
	})
	if blockErr != nil {
		return nil, blockErr
	}
	return e.result(s, err)
}

func (e *dfEnv) rolling(c *rb.Call) (any, error) {
	if err := kwOnly(c, "min_periods"); err != nil {
		return nil, err
	}
	if err := arity(c, 1, 1); err != nil {
		return nil, err
	}
	w, err := toInt(c.Args[0].Go())
	if err != nil {
		return nil, err
	}
	if mp, ok := c.Kwargs["min_periods"]; ok {
		n, err := toInt(mp.Go())
		if err != nil {
			return nil, err
		}
		return e.result(seriesOf(c).RollingMin(int(w), int(n)))
	}
	return e.result(seriesOf(c).Rolling(int(w)))
}

func (e *dfEnv) seriesSel(c *rb.Call, positional bool) (any, error) {
	if err := plExact(c, 1); err != nil {
		return nil, err
	}
	sel, err := selector(c.Args[0].Go(), positional)
	if err != nil {
		return nil, err
	}
	if positional {
		return e.result(seriesOf(c).Iloc(sel))
	}
	return e.result(seriesOf(c).Loc(sel))
}

func (e *dfEnv) seriesIloc(c *rb.Call) (any, error) { return e.seriesSel(c, true) }
func (e *dfEnv) seriesLoc(c *rb.Call) (any, error)  { return e.seriesSel(c, false) }

func (e *dfEnv) toDatetime(c *rb.Call) (any, error) {
	if err := noKw(c); err != nil {
		return nil, err
	}
	if err := arity(c, 0, 1); err != nil {
		return nil, err
	}
	var layout []string
	if len(c.Args) == 1 {
		s, err := toString(c.Args[0].Go())
		if err != nil {
			return nil, err
		}
		layout = append(layout, s)
	}
	return e.result(dataframe.ToDatetime(seriesOf(c), layout...))
}

// agg is GroupBy#agg and Resampler#agg after the Ruby prelude: [col,
// funcs] pairs, funcs a name or an Array of names.
func (e *dfEnv) agg(c *rb.Call) (any, error) {
	ps, err := pairs(c.Args[0].Go(), c.Args[1].Go())
	if err != nil {
		return nil, err
	}
	specs := make([]dataframe.AggSpec, len(ps))
	for i, p := range ps {
		col, err := toString(p[0])
		if err != nil {
			return nil, err
		}
		fns, err := strs(p[1])
		if err != nil {
			return nil, err
		}
		specs[i] = dataframe.Agg(col, fns...)
	}
	fn := reflect.ValueOf(c.Self.Go()).MethodByName("Agg")
	in := make([]reflect.Value, len(specs))
	for i, s := range specs {
		in[i] = reflect.ValueOf(s)
	}
	return e.results(fn.Call(in))
}

func (e *dfEnv) groups(c *rb.Call) (any, error) {
	if err := plExact(c, 0); err != nil {
		return nil, err
	}
	keys, frames := c.Self.Go().(*dataframe.GroupBy).Groups()
	out := make([]any, len(keys))
	for i := range keys {
		out[i] = []any{e.outAny(keys[i]), frames[i]}
	}
	return out, nil
}

func (e *dfEnv) newIndex(c *rb.Call) (any, error) {
	if err := kwOnly(c, "name"); err != nil {
		return nil, err
	}
	if err := arity(c, 1, 1); err != nil {
		return nil, err
	}
	name := ""
	if n, ok := c.Kwargs["name"]; ok {
		s, err := toString(n.Go())
		if err != nil {
			return nil, err
		}
		name = s
	}
	return e.result(dataframe.NewIndex(name, dfValue(c.Args[0].Go())))
}

func (e *dfEnv) rangeIndex(c *rb.Call) (any, error) {
	if err := plExact(c, 1); err != nil {
		return nil, err
	}
	n, err := toInt(c.Args[0].Go())
	if err != nil {
		return nil, err
	}
	return dataframe.NewRangeIndex(int(n)), nil
}

var dfFrameMethods = []dfM{
	{ruby: "columns", goName: "Columns"},
	{ruby: "length", goName: "Len"},
	{ruby: "size", goName: "Len"},
	{ruby: "shape", goName: "Shape"},
	{ruby: "describe", goName: "Describe"},
	{ruby: "copy", goName: "Copy"},
	{ruby: "count", goName: "Count"},
	{ruby: "mean", goName: "Mean"},
	{ruby: "sum", goName: "Sum"},
	{ruby: "std", goName: "Std"},
	{ruby: "var", goName: "Var"},
	{ruby: "min", goName: "Min"},
	{ruby: "max", goName: "Max"},
	{ruby: "median", goName: "Median"},
	{ruby: "isna", goName: "IsNA"},
	{ruby: "notna", goName: "NotNA"},
	{ruby: "ffill", goName: "FFill"},
	{ruby: "bfill", goName: "BFill"},
	{ruby: "transpose", goName: "T"},
	{ruby: "t", goName: "T"},
	{ruby: "drop", goName: "Drop"},
	{ruby: "select", goName: "Select"},
	{ruby: "set_index", goName: "SetIndex"},
	{ruby: "reset_index", goName: "ResetIndex", kw: []string{"drop"}, def: []any{false}},
	{ruby: "sort_index", goName: "SortIndex", kw: []string{"ascending"}, def: []any{true}},
	{ruby: "reindex", goName: "Reindex"},
	{ruby: "reindex_columns", goName: "ReindexColumns"},
	{ruby: "set_columns", goName: "SetColumns"},
	{ruby: "column", goName: "Column"},
	{ruby: "column_at", goName: "ColumnAt"},
	{ruby: "series", goName: "Series"},
	{ruby: "iat", goName: "IAt"},
	{ruby: "at", goName: "At"},
	{ruby: "values", goName: "Values"},
	{ruby: "to_ndarray", goName: "Values"},
	{ruby: "to_s", goName: "String"},
	{ruby: "inspect", goName: "String"},
	{ruby: "to_html", goName: "HTML"},
	{ruby: "display", goName: "Display"},
	{ruby: "index", goName: "Index"},
	{ruby: "with_column", goName: "WithColumn"},
	{ruby: "with_index", goName: "WithIndex"},
	{ruby: "pivot", goName: "Pivot", kw: []string{"index", "columns", "values"}},
	{ruby: "pivot_table", goName: "PivotTable", kw: []string{"index", "columns", "values", "aggfunc"}, def: []any{"mean"}},
	{ruby: "stack", goName: "Stack"},
	{ruby: "unstack", goName: "Unstack"},
	{ruby: "to_period", goName: "ToPeriod"},
	{ruby: "head", goName: "Head", kw: []string{"n"}, def: []any{int64(5)}},
	{ruby: "tail", goName: "Tail", kw: []string{"n"}, def: []any{int64(5)}},
	{ruby: "round", goName: "Round", kw: []string{"decimals"}, def: []any{int64(0)}},
	{ruby: "shift", goName: "Shift", kw: []string{"periods"}, def: []any{int64(1)}},
	{ruby: "add", goName: "Add"},
	{ruby: "+", goName: "Add"},
	{ruby: "sub", goName: "Sub"},
	{ruby: "-", goName: "Sub"},
	{ruby: "mul", goName: "Mul"},
	{ruby: "*", goName: "Mul"},
	{ruby: "div", goName: "Div"},
	{ruby: "/", goName: "Div"},
}

var dfSeriesMethods = []dfM{
	{ruby: "name", goName: "Name"},
	{ruby: "length", goName: "Len"},
	{ruby: "size", goName: "Len"},
	{ruby: "count", goName: "Count"},
	{ruby: "dtype", goName: "DType"},
	{ruby: "index", goName: "Index"},
	{ruby: "values", goName: "Values"},
	{ruby: "to_ndarray", goName: "Values"},
	{ruby: "floats", goName: "Floats"},
	{ruby: "to_a", goName: "Items"},
	{ruby: "valid", goName: "Valid"},
	{ruby: "get", goName: "Get"},
	{ruby: "mean", goName: "Mean"},
	{ruby: "sum", goName: "SumValue"},
	{ruby: "std", goName: "Std"},
	{ruby: "var", goName: "VarDDOF", kw: []string{"ddof"}, def: []any{int64(1)}},
	{ruby: "min", goName: "Min"},
	{ruby: "max", goName: "Max"},
	{ruby: "median", goName: "Median"},
	{ruby: "prod", goName: "Prod"},
	{ruby: "nunique", goName: "NUnique"},
	{ruby: "quantile", goName: "Quantile", kw: []string{"q"}, def: []any{0.5}},
	{ruby: "agg", goName: "Agg"},
	{ruby: "head", goName: "Head", kw: []string{"n"}, def: []any{int64(5)}},
	{ruby: "tail", goName: "Tail", kw: []string{"n"}, def: []any{int64(5)}},
	{ruby: "unique", goName: "Unique"},
	{ruby: "value_counts", goName: "ValueCounts"},
	{ruby: "copy", goName: "Copy"},
	{ruby: "to_frame", goName: "ToFrame"},
	{ruby: "reset_index", goName: "ResetIndex"},
	{ruby: "isna", goName: "IsNA"},
	{ruby: "notna", goName: "NotNA"},
	{ruby: "dropna", goName: "DropNA"},
	{ruby: "fillna", goName: "FillNA"},
	{ruby: "ffill", goName: "FFill"},
	{ruby: "bfill", goName: "BFill"},
	{ruby: "shift", goName: "Shift", kw: []string{"periods"}, def: []any{int64(1)}},
	{ruby: "diff", goName: "Diff", kw: []string{"periods"}, def: []any{int64(1)}},
	{ruby: "pct_change", goName: "PctChange", kw: []string{"periods"}, def: []any{int64(1)}},
	{ruby: "cumsum", goName: "CumSum"},
	{ruby: "cumprod", goName: "CumProd"},
	{ruby: "cummax", goName: "CumMax"},
	{ruby: "cummin", goName: "CumMin"},
	{ruby: "abs", goName: "Abs"},
	{ruby: "neg", goName: "Neg"},
	{ruby: "-@", goName: "Neg"},
	{ruby: "round", goName: "Round", kw: []string{"decimals"}, def: []any{int64(0)}},
	{ruby: "rename", goName: "Rename"},
	{ruby: "astype", goName: "AsType"},
	{ruby: "reindex", goName: "Reindex"},
	{ruby: "with_index", goName: "SetIndex"},
	{ruby: "sort_values", goName: "SortValues", kw: []string{"ascending"}, def: []any{true}},
	{ruby: "sort_index", goName: "SortIndex", kw: []string{"ascending"}, def: []any{true}},
	{ruby: "between", goName: "Between"},
	{ruby: "isin", goName: "IsIn"},
	{ruby: "dt", goName: "Dt"},
	{ruby: "to_period", goName: "ToPeriod"},
	{ruby: "to_timestamp", goName: "ToTimestamp"},
	{ruby: "unstack", goName: "Unstack"},
	{ruby: "to_s", goName: "String"},
	{ruby: "inspect", goName: "String"},
	{ruby: "to_html", goName: "HTML"},
	{ruby: "display", goName: "Display"},
	{ruby: "+", goName: "Add"},
	{ruby: "add", goName: "Add"},
	{ruby: "-", goName: "Sub"},
	{ruby: "sub", goName: "Sub"},
	{ruby: "*", goName: "Mul"},
	{ruby: "mul", goName: "Mul"},
	{ruby: "/", goName: "Div"},
	{ruby: "div", goName: "Div"},
	{ruby: "%", goName: "Mod"},
	{ruby: "mod", goName: "Mod"},
	{ruby: "**", goName: "Pow"},
	{ruby: "pow", goName: "Pow"},
	{ruby: "floordiv", goName: "FloorDiv"},
	{ruby: ">", goName: "Gt"},
	{ruby: "gt", goName: "Gt"},
	{ruby: "<", goName: "Lt"},
	{ruby: "lt", goName: "Lt"},
	{ruby: ">=", goName: "Ge"},
	{ruby: "ge", goName: "Ge"},
	{ruby: "<=", goName: "Le"},
	{ruby: "le", goName: "Le"},
	{ruby: "eq", goName: "Eq"},
	{ruby: "ne", goName: "Ne"},
	{ruby: "&", goName: "And"},
	{ruby: "|", goName: "Or"},
	{ruby: "^", goName: "Xor"},
	{ruby: "~", goName: "Not"},
	{ruby: "invert", goName: "Not"},
}

var dfGroupMethods = []dfM{
	{ruby: "mean", goName: "Mean"},
	{ruby: "sum", goName: "Sum"},
	{ruby: "count", goName: "Count"},
	{ruby: "min", goName: "Min"},
	{ruby: "max", goName: "Max"},
	{ruby: "median", goName: "Median"},
	{ruby: "std", goName: "Std"},
	{ruby: "var", goName: "Var"},
	{ruby: "first", goName: "First"},
	{ruby: "last", goName: "Last"},
	{ruby: "nunique", goName: "NUnique"},
	{ruby: "size", goName: "Size"},
	{ruby: "ngroups", goName: "NGroups"},
	{ruby: "n_groups", goName: "NGroups"},
	{ruby: "select", goName: "Select"},
	{ruby: "[]", goName: "Select"},
	{ruby: "transform", goName: "Transform"},
	{ruby: "to_s", goName: "String"},
	{ruby: "inspect", goName: "String"},
}

var dfResamplerMethods = []dfM{
	{ruby: "mean", goName: "Mean"},
	{ruby: "sum", goName: "Sum"},
	{ruby: "count", goName: "Count"},
	{ruby: "min", goName: "Min"},
	{ruby: "max", goName: "Max"},
	{ruby: "median", goName: "Median"},
	{ruby: "std", goName: "Std"},
	{ruby: "first", goName: "First"},
	{ruby: "last", goName: "Last"},
	{ruby: "size", goName: "Size"},
}

var dfRollingMethods = []dfM{
	{ruby: "mean", goName: "Mean"},
	{ruby: "sum", goName: "Sum"},
	{ruby: "min", goName: "Min"},
	{ruby: "max", goName: "Max"},
	{ruby: "std", goName: "Std"},
	{ruby: "median", goName: "Median"},
}

var dfIndexMethods = []dfM{
	{ruby: "to_a", goName: "Labels"},
	{ruby: "labels", goName: "Labels"},
	{ruby: "length", goName: "Len"},
	{ruby: "size", goName: "Len"},
	{ruby: "name", goName: "Name"},
	{ruby: "names", goName: "Names"},
	{ruby: "nlevels", goName: "NLevels"},
	{ruby: "range?", goName: "IsRange"},
	{ruby: "label", goName: "Label"},
	{ruby: "[]", goName: "Label"},
	{ruby: "get", goName: "Get"},
	{ruby: "values", goName: "Values"},
	{ruby: "dtype", goName: "DType"},
	{ruby: "level", goName: "Level"},
	{ruby: "rename", goName: "Rename"},
	{ruby: "to_s", goName: "String"},
	{ruby: "inspect", goName: "String"},
	{ruby: "display", goName: "Display"},
}

var dfDatetimeMethods = []dfM{
	{ruby: "year", goName: "Year"},
	{ruby: "month", goName: "Month"},
	{ruby: "day", goName: "Day"},
	{ruby: "hour", goName: "Hour"},
	{ruby: "minute", goName: "Minute"},
	{ruby: "second", goName: "Second"},
	{ruby: "day_of_week", goName: "DayOfWeek"},
	{ruby: "dayofweek", goName: "DayOfWeek"},
	{ruby: "day_of_year", goName: "DayOfYear"},
	{ruby: "dayofyear", goName: "DayOfYear"},
	{ruby: "normalize", goName: "Normalize"},
	{ruby: "strftime", goName: "Strftime"},
}
