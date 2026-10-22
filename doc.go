// Package ruby binds the go-scikit stack — go-statistics, go-numerics,
// go-dataframe and go-plot, over go-ndarray — into the pure-Go Ruby of
// go-embedded-ruby, so Ruby code gets SciPy-style statistics, special
// functions, optimisation, integration, data frames and plots with no native
// dependency:
//
//	in := rb.New()
//	ruby.Register(in)
//	in.Eval(`Stats::Normal.new(loc: 0, scale: 1).pdf(0.5)`, "x.rb", 1)
//
// Every Ruby method calls the Go function it is named after, on the same
// inputs; the results are the Go library's, bit for bit. NDArray is rbgo's
// own: an array passed in or returned is the go-ndarray array itself.
package ruby

import (
	rb "github.com/go-embedded-ruby/ruby"
)

// Register defines the modules Stats, Special, Optimize, Integrate,
// DataFrame and Plot in the interpreter.
func Register(in *rb.Interp) error {
	for _, f := range []func(*rb.Interp) error{registerStats, registerSpecial, registerOptimize, registerIntegrate, registerDataFrame, registerPlot} {
		if err := f(in); err != nil {
			return err
		}
	}
	return nil
}
