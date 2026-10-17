package ruby

import (
	"os"
	"sort"
	"strings"
	"testing"
)

// bindings lists every Ruby name the bindings define: "Module.function",
// "Class.class_method" and "Class#method", for the modules and classes
// Register makes.
func bindings(t *testing.T) []string {
	in := newInterp(t)
	src := `
out = []
walk = lambda do |mod|
  mod.singleton_methods(false).each { |m| out << "#{mod}.#{m}" }
  mod.instance_methods(false).each { |m| out << "#{mod}##{m}" } if mod.is_a?(Class)
  mod.constants(false).each do |c|
    v = mod.const_get(c)
    walk.(v) if v.is_a?(Module) && v.name.start_with?("#{mod}::")
  end
end
[Stats, Special, Optimize, Integrate, DataFrame, Plot].each { |m| walk.(m) }
out`
	v, err := in.Eval(src, "names.rb", 1)
	if err != nil {
		t.Fatal(err)
	}
	var names []string
	for _, n := range v.Go().([]any) {
		names = append(names, n.(string))
	}
	sort.Strings(names)
	return names
}

// The README lists what is bound, every name of it: a binding added without
// its line in the README fails here.
func TestREADMEListsEveryBinding(t *testing.T) {
	readme, err := os.ReadFile("README.md")
	if err != nil {
		t.Fatal(err)
	}
	text := string(readme)
	missing := 0
	for _, n := range bindings(t) {
		// The README groups names by class: "Stats::Normal#pdf" is listed as
		// `pdf` in the Stats::Normal entry, so look for the bare name in it.
		owner, name := n, n
		if i := strings.LastIndexAny(n, ".#"); i >= 0 {
			owner, name = n[:i], n[i+1:]
		}
		section := sectionOf(text, owner)
		if !strings.Contains(section, "`"+name+"`") {
			if missing < 40 {
				t.Errorf("README does not list %s", n)
			}
			missing++
		}
	}
	if missing > 0 {
		t.Errorf("%d names missing in all", missing)
	}
}

// sectionOf is the README's entry for a module or class: the lines from its
// "#### Owner" heading to the next heading.
func sectionOf(text, owner string) string {
	h := "#### " + owner + "\n"
	i := strings.Index(text, h)
	if i < 0 {
		return ""
	}
	rest := text[i+len(h):]
	if j := strings.Index(rest, "\n#"); j >= 0 {
		rest = rest[:j]
	}
	return rest
}

// TestPrintBindings prints the names grouped as the README lists them; run
// it with -v -run TestPrintBindings when the bindings change.
func TestPrintBindings(t *testing.T) {
	groups := map[string][]string{}
	var owners []string
	for _, n := range bindings(t) {
		i := strings.LastIndexAny(n, ".#")
		owner := n[:i]
		if _, ok := groups[owner]; !ok {
			owners = append(owners, owner)
		}
		groups[owner] = append(groups[owner], "`"+n[i+1:]+"`")
	}
	var b strings.Builder
	for _, o := range owners {
		b.WriteString("#### " + o + "\n\n" + strings.Join(groups[o], " ") + "\n\n")
	}
	t.Log("\n" + b.String())
}
