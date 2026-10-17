package ruby

//go:generate go run ./internal/genenums

// enumValue is one value of an enumerated Go type and its Ruby symbol.
type enumValue struct {
	name  string
	value any
}
