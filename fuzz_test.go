package ruby

import (
	"bytes"
	"testing"

	rb "github.com/go-embedded-ruby/ruby"
)

// Whatever values a binding is handed, it answers with a result or a Ruby
// exception: the host process never panics. The values are built in Go, not
// parsed from the fuzzed bytes, so no input can make the interpreter loop.
func FuzzBindingArguments(f *testing.F) {
	f.Add(1.5, int64(3), "loc", true, uint8(0))
	f.Add(-0.0, int64(-1), "two_sided", false, uint8(7))
	f.Add(1e308, int64(1)<<62, "", true, uint8(200))
	var out bytes.Buffer
	in := rb.New(rb.WithStdout(&out), rb.WithStderr(&out))
	if err := Register(in); err != nil {
		f.Fatal(err)
	}
	stats, _ := in.Eval("Stats", "f.rb", 1)
	special, _ := in.Eval("Special", "f.rb", 1)
	normal, _ := in.Eval("Stats::Normal", "f.rb", 1)
	f.Fuzz(func(t *testing.T, x float64, n int64, s string, b bool, pick uint8) {
		vals := []any{x, n, s, b, nil, []any{x, n}, []any{x, x, x}, map[string]any{s: x}, rb.Symbol(s)}
		a := vals[int(pick)%len(vals)]
		c := vals[int(pick/9)%len(vals)]
		for _, call := range []struct {
			recv rb.Value
			name string
			args []any
		}{
			{stats, "mean", []any{a}},
			{stats, "quantile", []any{a, c}},
			{stats, "ttest_1samp", []any{a, c}},
			{stats, "describe", []any{a}},
			{special, "gamma", []any{a}},
			{special, "jv", []any{a, c}},
			{special, "comb", []any{a, c}},
			{normal, "new", []any{a, c}},
			{normal, "fit", []any{a}},
		} {
			_, _ = call.recv.Call(call.name, call.args...)
		}
	})
}
