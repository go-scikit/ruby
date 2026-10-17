# Optimize: the objective is the block.
rosen = ->(x) { (1 - x[0])**2 + 100 * (x[1] - x[0]**2)**2 }

res = Optimize.minimize([-1.2, 1.0], method: :bfgs) { |x| rosen.(x) }
puts "BFGS: x = #{res.x.inspect}, f = #{res.fun}, #{res.nit} iterations"

res = Optimize.minimize([3.0, 3.0], method: :lbfgsb, bounds: [[2, nil], [nil, 2.5]]) { |x| rosen.(x) }
puts "bounded: #{res.x.inspect}"

puts "root of x^3 - 2x - 5: #{Optimize.brentq(2, 3) { |x| x**3 - 2 * x - 5 }.root}"

fit = Optimize.curve_fit([0, 1, 2, 3, 4], [1.1, 2.6, 7.6, 20.0, 54.9], p0: [1, 1]) { |x, a, b| a * Math.exp(b * x) }
puts "curve_fit: a, b = #{fit.popt.inspect}"

lp = Optimize.linprog([-1, -2], a_ub: NDArray.from([1, 1, 1, 3], [2, 2]), b_ub: [4, 6], bounds: [[0, nil], [0, nil]])
puts "linprog: #{lp.x.inspect}, #{lp.message}"
