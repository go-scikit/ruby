package ruby

//go:generate go run ./internal/gen

// enumValue is one value of an enumerated Go type and its Ruby symbol.
type enumValue struct {
	name  string
	value any
}

// namedFunc is a Go function and its Go name.
type namedFunc struct {
	name string
	fn   any
}
