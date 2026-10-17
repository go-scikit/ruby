# Special: every function of go-numerics/special, elementwise like a ufunc.
p Special.gamma(5)                               # 24.0
p Special.gamma(NDArray.arange(1, 6))            # 1, 1, 2, 6, 24
p Special.erf([0.0, 0.5, 1.0])
p Special.beta_inc(2, 3, NDArray.linspace(0, 1, 5))
# Broadcasting: orders down the rows, points along the columns.
p Special.jv(NDArray.arange(3).reshape(3, 1), NDArray.linspace(0, 2, 3))
p Special.jn_zeros(0, 3)
