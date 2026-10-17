# Integrate: the integrand, or the right-hand side, is the block.
q = Integrate.quad(0, Float::INFINITY) { |x| Math.exp(-x * x) }
puts "∫ exp(-x²) over [0, ∞) = #{q.value} (√π/2 = #{Math.sqrt(Math::PI) / 2})"

puts Integrate.dblquad(0, 1, 0, ->(x) { x }) { |y, x| x * y }.value

# A damped oscillator: y'' + 0.2 y' + y = 0.
sol = Integrate.solve_ivp([0, 10], [1.0, 0.0], t_eval: NDArray.linspace(0, 10, 6).to_a) { |t, y| [y[1], -0.2 * y[1] - y[0]] }
p sol.y_array
p Integrate.trapezoid(NDArray.linspace(0, 1, 11) ** 2, dx: 0.1)
