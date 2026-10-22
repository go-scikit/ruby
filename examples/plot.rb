# Plot: matplotlib's figures and axes, from go-plot.
# Run with the go-scikit bindings registered (ruby.Register).

x = NDArray.linspace(0, 2 * Math::PI, 50).to_a   # an NDArray works as data too
fig, (left, right) = Plot.subplots(1, 2, figsize: [9, 3.5])

left.plot(x, x.map { |v| Math.sin(v) }, label: "sin", color: "C0", linewidth: 2)
left.plot(x, x.map { |v| Math.cos(v) }, label: "cos", linestyle: "--")
left.axhline(0, color: "k", lw: 0.5)
left.legend(loc: "upper right")
left.title = "Trigonometry"
left.xlabel = "$x$ (rad)"
left.grid

z = (0...20).map { |i| (0...20).map { |j| Math.sin(i / 3.0) * Math.cos(j / 4.0) } }
m = right.imshow(z, cmap: "viridis", origin: "lower")
fig.colorbar(m, right)
right.set_title("A field")

fig.tight_layout
p fig                                   # #<Plot::Figure 900x350>
svg = fig.to_svg                        # File.write("fig.svg", svg) saves it
puts "#{svg.bytesize} bytes of SVG, #{fig.to_png.bytesize} bytes of PNG"
puts fig.display.keys.inspect           # what a notebook shows

# Bars and a histogram from a DataFrame column.
df = DataFrame.new(group: %w[a b c], value: [3, 7, 5])
fig2, ax = Plot.subplots
ax.bar([0, 1, 2], df["value"], colors: ["C0", "C1", "C2"])
ax.set_xticks([0, 1, 2], df["group"].to_a)
puts fig2.inspect
