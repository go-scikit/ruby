package ruby

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	rb "github.com/go-embedded-ruby/ruby"
)

// The examples of DataFrame and Plot run without raising.
func TestExamplesDataFramePlot(t *testing.T) {
	for _, name := range []string{"dataframe.rb", "plot.rb"} {
		path := filepath.Join("examples", name)
		src, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		var out bytes.Buffer
		in := rb.New(rb.WithStdout(&out), rb.WithStderr(&out))
		if err := Register(in); err != nil {
			t.Fatal(err)
		}
		if _, err := in.Eval(string(src), path, 1); err != nil {
			t.Errorf("%s: %v\n%s", path, err, out.String())
			continue
		}
		if out.Len() == 0 {
			t.Errorf("%s printed nothing", path)
		}
		t.Logf("%s:\n%s", path, out.String())
	}
}
