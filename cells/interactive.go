package cells

import (
	"fmt"
	"regexp"
	"sort"
	"strings"

	rb "github.com/go-embedded-ruby/ruby"
	nd "github.com/go-ndarray/ndarray"
)

// An interactive session (go-notebook's `notebook irb`) evaluates inputs one
// by one in the same interpreter as cells do, but as irb does: each input may
// assign any local again, `_` holds the value of the last one, and the value
// is shown as its inspect.

// Eval evaluates src, one input of an interactive session that starts at
// line of file, and sets the local `_` to its value, as irb does. It returns
// the value's inspect and, when the value shows itself (a Plot figure), its
// MIME bundle — an NDArray shows its inspect only, as in a cell.
//
// An exception returns an *Error whose text is the message and class, then
// the backtrace frames in file (twelve at most); its Class says which
// exception it was (SystemExit for exit). `_` is left as it was.
func (s *Session) Eval(src, file string, line int) (inspect string, display map[string][]byte, err error) {
	val, err := s.in.Eval(src, file, line)
	if err != nil {
		return "", nil, interactiveError(err, file)
	}
	_ = s.in.Set("_", val) // a Value is set as itself, which cannot fail
	if d, ok := val.Go().(interface{ Display() map[string][]byte }); ok {
		if _, isArray := d.(*nd.Array); !isArray {
			display = d.Display()
		}
	}
	return val.Inspect(), display, nil
}

// Class is the class of the Ruby exception behind the error ("" for an error
// that is not one).
func (e *Error) Class() string { return e.class }

// interactiveError renders an exception as irb shows it: "message (Class)",
// then the frames in file.
func interactiveError(err error, file string) *Error {
	e, ok := err.(*rb.Error)
	if !ok {
		return &Error{msg: err.Error()}
	}
	var b strings.Builder
	fmt.Fprintf(&b, "%s (%s)", e.Message, e.Class)
	frames(&b, e.Backtrace, file)
	return &Error{msg: b.String(), class: e.Class}
}

// frames writes the backtrace frames that are in file, maxFrames at most.
func frames(b *strings.Builder, backtrace []string, file string) {
	n := 0
	for _, f := range backtrace {
		if !strings.HasPrefix(f, file+":") {
			continue
		}
		if n++; n <= maxFrames {
			b.WriteString("\n\tfrom " + f)
		}
	}
	if n > maxFrames {
		fmt.Fprintf(b, "\n\t... and %d more frames", n-maxFrames)
	}
}

// reservedWords are Ruby's keywords, which complete like names (IRB's
// RubyLex::RESERVED_WORDS).
var reservedWords = []string{
	"__ENCODING__", "__LINE__", "__FILE__", "BEGIN", "END", "alias", "and",
	"begin", "break", "case", "class", "def", "defined?", "do", "else", "elsif",
	"end", "ensure", "false", "for", "if", "in", "module", "next", "nil", "not",
	"or", "redo", "rescue", "retry", "return", "self", "super", "then", "true",
	"undef", "unless", "until", "when", "while", "yield",
}

// The forms of the word before the cursor, after IRB's RegexpCompletor
// (irb 1.16, completion.rb).
var (
	reString   = regexp.MustCompile(`^(.*["'` + "`" + `])\.([^.]*)$`)
	reRegexp   = regexp.MustCompile(`^(.*/)\.([^.]*)$`)
	reArray    = regexp.MustCompile(`^([^\]]*\])\.([^.]*)$`)
	reHash     = regexp.MustCompile(`^([^}]*\})\.([^.]*)$`)
	reSymbol   = regexp.MustCompile(`^(:[^:.]+)$`)
	reAbsConst = regexp.MustCompile(`^::([A-Z][^:.()]*)$`)
	reConst    = regexp.MustCompile(`^([A-Z].*)::([^:.]*)$`)
	reSymCall  = regexp.MustCompile(`^(:[^:.]+)(\.|::)([^.]*)$`)
	reNumber   = regexp.MustCompile(`^(-?(?:0[dbo])?[0-9_]+(?:\.[0-9_]+)?(?:(?:[eE][+-]?[0-9]+)?i?|r)?)(\.|::)([^.]*)$`)
	reHex      = regexp.MustCompile(`^(-?0x[0-9a-fA-F_]+)(\.|::)([^.]*)$`)
	reGlobal   = regexp.MustCompile(`^(\$[^.]*)$`)
	reCall     = regexp.MustCompile(`^([^.:"].*)(\.|::)([^.]*)$`)
	reDot      = regexp.MustCompile(`^\.([^.]*)$`)
	reBlank    = regexp.MustCompile(`^\s*$`)

	// constPath is a receiver completion may evaluate: a constant path
	// (Stats::Normal), which reads constants and calls nothing.
	constPath = regexp.MustCompile(`^(::)?[A-Z]\w*(::[A-Z]\w*)*$`)
)

// Complete returns the completions of target, the word before the cursor,
// in the session's state — the candidates IRB's default completor
// (RegexpCompletor) gives: the methods of a literal's class after a dot, of
// a local, global or instance variable or a constant path's value after a
// dot or ::, the constants of a module after ::, the global variables after
// $, and otherwise the locals, the methods and instance variables of main,
// the constants and the keywords. Each candidate is the whole word, receiver
// included ("x.size"), sorted for a bare name.
//
// Unlike IRB, it evaluates nothing the word does not name: a receiver is a
// variable or a constant path, never an expression, so pressing Tab never
// calls a method of the input. A variable's own #methods is called, as IRB
// does.
func (s *Session) Complete(target string) []string {
	switch {
	case reString.MatchString(target):
		m := reString.FindStringSubmatch(target)
		return selectMessage(m[1], m[2], s.names("String.instance_methods"), ".")
	case reRegexp.MatchString(target):
		m := reRegexp.FindStringSubmatch(target)
		return selectMessage(m[1], m[2], s.names("Regexp.instance_methods"), ".")
	case reArray.MatchString(target):
		m := reArray.FindStringSubmatch(target)
		return selectMessage(m[1], m[2], s.names("Array.instance_methods"), ".")
	case reHash.MatchString(target):
		m := reHash.FindStringSubmatch(target)
		return selectMessage(m[1], m[2], union(s.names("Hash.instance_methods"), s.names("Proc.instance_methods")), ".")
	case reSymbol.MatchString(target):
		// IRB lists Symbol.all_symbols, which rbgo does not have; a symbol
		// is completed from nothing.
		return nil
	case reAbsConst.MatchString(target):
		m := reAbsConst.FindStringSubmatch(target)
		var out []string
		for _, c := range s.topConstants() {
			if strings.HasPrefix(c, m[1]) {
				out = append(out, "::"+c)
			}
		}
		return out
	case reConst.MatchString(target):
		m := reConst.FindStringSubmatch(target)
		if !constPath.MatchString(m[1]) {
			return nil
		}
		cands := union(s.names(m[1]+".constants"), s.names(m[1]+".methods"))
		sort.Strings(cands)
		return selectMessage(m[1], m[2], cands, "::")
	case reSymCall.MatchString(target):
		m := reSymCall.FindStringSubmatch(target)
		return selectMessage(m[1], m[3], s.names("Symbol.instance_methods"), m[2])
	case reNumber.MatchString(target):
		m := reNumber.FindStringSubmatch(target)
		return selectMessage(m[1], m[3], s.names("("+m[1]+").methods"), m[2])
	case reHex.MatchString(target):
		m := reHex.FindStringSubmatch(target)
		return selectMessage(m[1], m[3], s.names("("+m[1]+").methods"), m[2])
	case reGlobal.MatchString(target):
		var out []string
		for _, g := range s.names("global_variables") {
			if strings.Contains(g, target) {
				out = append(out, g)
			}
		}
		return out
	case reCall.MatchString(target):
		m := reCall.FindStringSubmatch(target)
		return selectMessage(m[1], m[3], s.receiverNames(m[1], m[2]), m[2])
	case reDot.MatchString(target):
		m := reDot.FindStringSubmatch(target)
		cands := s.names("String.instance_methods(true)")
		sort.Strings(cands)
		return selectMessage("", m[1], cands, ".")
	case reBlank.MatchString(target):
		return nil
	}
	cands := union(s.names("Kernel.instance_method(:methods).bind_call(self)"),
		s.names("Kernel.instance_method(:private_methods).bind_call(self)"),
		s.in.Locals(),
		s.names("Kernel.instance_method(:instance_variables).bind_call(self)"),
		s.topConstants(), reservedWords)
	var out []string
	for _, c := range cands {
		if strings.HasPrefix(c, target) {
			out = append(out, c)
		}
	}
	sort.Strings(out)
	return out
}

// receiverNames are the candidates after receiver and sep (IRB's "variable.func"
// form): the methods of a variable's or a constant path's value, and its
// constants after ::. Anything else — a method call, an expression — has none.
func (s *Session) receiverNames(receiver, sep string) []string {
	vars := union(s.names("global_variables"), []string{"true", "false", "nil"}, s.in.Locals(),
		s.names("Kernel.instance_method(:instance_variables).bind_call(self)"))
	known := false
	for _, v := range vars {
		known = known || v == receiver
	}
	if !known && !constPath.MatchString(receiver) {
		return nil
	}
	var cands []string
	if sep == "::" {
		cands = s.names("->(r) { r.kind_of?(Module) ? r.constants : [] }.call(" + receiver + ")")
	}
	return union(cands, s.names("("+receiver+").methods"))
}

// topConstants are the constants of the top level. (rbgo's Object.constants
// is empty: Module.constants, which MRI also gives the top level's
// constants from, is used instead.)
func (s *Session) topConstants() []string { return s.names("Module.constants") }

// names evaluates expr, which gives an Array of names (Symbols or Strings),
// and returns them as strings; an error or another value gives none. The
// expressions completion builds assign no local, so the session's locals are
// left as they were.
func (s *Session) names(expr string) []string {
	v, err := s.in.Eval("(begin; ("+expr+").map(&:to_s); rescue Exception; []; end)", "(complete)", 1)
	if err != nil {
		return nil
	}
	list, _ := v.Go().([]any)
	out := make([]string, 0, len(list))
	for _, e := range list {
		if str, ok := e.(string); ok {
			out = append(out, str)
		}
	}
	return out
}

// selectMessage keeps the candidates that start with message and that can be
// typed after a dot, prefixed with the receiver and the separator (IRB's
// select_message).
func selectMessage(receiver, message string, cands []string, sep string) []string {
	var out []string
	for _, c := range cands {
		// An operator or a name that starts otherwise is left out, as IRB's
		// select_message maps them to nil.
		if strings.HasPrefix(c, message) && c != "" && isIdentStart(c[0]) {
			out = append(out, receiver+sep+c)
		}
	}
	return out
}

func isIdentStart(c byte) bool {
	return c == '_' || c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z'
}

// union is the lists joined, each name once, in order of first appearance.
func union(lists ...[]string) []string {
	seen := map[string]bool{}
	var out []string
	for _, l := range lists {
		for _, e := range l {
			if !seen[e] {
				seen[e] = true
				out = append(out, e)
			}
		}
	}
	return out
}
