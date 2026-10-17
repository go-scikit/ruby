# DataFrame: pandas' DataFrame and Series, from go-dataframe.
# Run with the go-scikit bindings registered (ruby.Register).

df = DataFrame.new(
  city: ["Lyon", "Paris", "Lyon", "Nice", "Paris"],
  year: [2020, 2020, 2021, 2021, 2021],
  pop:  [516.1, 2148.3, 522.2, 342.7, 2133.1]
)
puts df
puts df.shape.inspect          # [5, 3]
puts df.dtypes.inspect         # {"city" => "string", "year" => "int64", "pop" => "float64"}

# Columns are Series; comparisons give masks, and a mask selects rows.
big = df[df["pop"] > 500]
puts big
puts df["pop"].mean

# Group, aggregate, sort.
puts df.group_by("city").agg(pop: [:mean, :max], year: :count)
puts df.sort_values(["year", "pop"], ascending: [true, false]).head(3)

# Arithmetic on a Series uses the named methods (add, mul, ...).
puts df["pop"].mul(1000).round.to_a.inspect

# Reading CSV text, with pandas' options.
csv = <<~CSV
  day,visits
  2024-01-01,10
  2024-01-02,
  2024-01-09,7
CSV
visits = DataFrame.read_csv(csv, parse_dates: ["day"], index_col: "day")
puts visits.fillna(0).resample("W").sum
puts visits.index[0]           # 2024-01-01 00:00:00 UTC

# A Series built row by row, and the data back as an NDArray.
ratio = df.apply("per_year") { |row| row["pop"] / row["year"] }
puts ratio.values.inspect
puts df.to_csv(index: false)
