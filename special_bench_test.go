package ruby

import "testing"

func BenchmarkSpecialUfunc(b *testing.B) {
	in := newInterp(&testing.T{})
	in.Eval("x = NDArray.linspace(0.1, 5, 100_000); y = NDArray.linspace(0.5, 2, 100_000)", "b.rb", 1)
	for _, src := range []string{"Special.gamma(x)", "Special.beta(x, y)", "Special.jv(2.5, x)"} {
		b.Run(src, func(b *testing.B) {
			for b.Loop() {
				if _, err := in.Eval(src, "b.rb", 1); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}
