package main

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// The generated files in the repository are what the generator writes now.
func TestGeneratedFilesAreCurrent(t *testing.T) {
	dir := t.TempDir()
	if err := writeAll(dir, moduleDir); err != nil {
		t.Fatal(err)
	}
	for _, f := range []string{"enums_gen.go", "special_gen.go"} {
		want, err := os.ReadFile(filepath.Join(dir, f))
		if err != nil {
			t.Fatal(err)
		}
		got, err := os.ReadFile(filepath.Join("..", "..", f))
		if err != nil {
			t.Fatal(err)
		}
		// A Windows checkout has CRLF line endings; the generator writes LF.
		if strings.ReplaceAll(string(got), "\r\n", "\n") != string(want) {
			t.Errorf("%s is out of date: run go generate", f)
		}
	}
}

func TestMain_(t *testing.T) {
	dir := t.TempDir()
	saved := os.Args
	defer func() { os.Args = saved }()
	os.Args = []string{"gen", dir}
	main()
	if _, err := os.Stat(filepath.Join(dir, "enums_gen.go")); err != nil {
		t.Fatal(err)
	}
	// A failure exits 1.
	code := 0
	exit = func(c int) { code = c }
	defer func() { exit = os.Exit }()
	os.Args = []string{"gen", filepath.Join(dir, "missing", "dir")}
	main()
	if code != 1 {
		t.Fatalf("exit code %d", code)
	}
	// With no argument it writes into the current directory: the module root
	// here, whose files TestGeneratedFilesAreCurrent shows are what it writes.
	t.Chdir(filepath.Join("..", ".."))
	os.Args = []string{"gen"}
	code = 0
	main()
	if code != 0 {
		t.Fatalf("exit code %d", code)
	}
}

func TestFailures(t *testing.T) {
	boom := errors.New("boom")
	failing := func(string) (string, error) { return "", boom }
	if err := writeAll(t.TempDir(), failing); !errors.Is(err, boom) {
		t.Fatalf("writeAll = %v", err)
	}
	// enums succeed, special fails.
	onlyEnums := func(p string) (string, error) {
		if p == "github.com/go-numerics/special" {
			return "", boom
		}
		return moduleDir(p)
	}
	if err := writeAll(t.TempDir(), onlyEnums); !errors.Is(err, boom) {
		t.Fatalf("writeAll = %v", err)
	}
	empty := t.TempDir()
	if _, err := generate(func(string) (string, error) { return empty, nil }); err == nil {
		t.Fatal("generate accepted a directory with no Go file")
	}
	if _, err := generateSpecial(func(string) (string, error) { return empty, nil }); err == nil {
		t.Fatal("generateSpecial accepted a directory with no Go file")
	}
	bad := t.TempDir()
	os.WriteFile(filepath.Join(bad, "x.go"), []byte("package x\nfunc ("), 0o644)
	os.WriteFile(filepath.Join(bad, "x_test.go"), []byte("package x"), 0o644)
	if _, err := enums(bad); err == nil {
		t.Fatal("enums accepted a file that does not parse")
	}
	if _, err := funcs(bad); err == nil {
		t.Fatal("funcs accepted a file that does not parse")
	}
	if _, err := moduleDir("example.com/not/a/module"); err == nil {
		t.Fatal("moduleDir found a module that is not required")
	}
	// The special_gen write fails when enums_gen is a directory... and
	// enums_gen fails when the directory does not exist.
	dir := t.TempDir()
	os.Mkdir(filepath.Join(dir, "special_gen.go"), 0o755)
	if err := writeAll(dir, moduleDir); err == nil {
		t.Fatal("writeAll wrote over a directory")
	}
}

func TestWords(t *testing.T) {
	for in, want := range map[string]string{
		"TwoSided": "two_sided", "LBFGSB": "lbfgsb", "RK45": "rk45", "GumbelL": "gumbel_l", "NewtonCG": "newton_cg", "DOP853": "dop853",
	} {
		if got := snake(in); got != want {
			t.Errorf("snake(%q) = %q, want %q", in, got, want)
		}
	}
	for _, c := range []struct {
		names []string
		want  string
	}{
		{[]string{"CenterMedian", "CenterMean"}, "Center"},
		{[]string{"TwoSided", "Less"}, ""},
		{[]string{"Only"}, ""},
		{[]string{"BinsCount", "BinsFD"}, "Bins"},
		{[]string{"Tau", "TauB"}, ""},
		{[]string{"Ab", "Abc"}, ""},
	} {
		if got := commonPrefix(c.names); got != c.want {
			t.Errorf("commonPrefix(%v) = %q, want %q", c.names, got, c.want)
		}
	}
}
