// Package optionutil provides shared application helpers for functional options.
package optionutil

// Apply applies non-nil options in order. The option type is inferred from the
// caller's package-specific option type, avoiding repeated application loops.
func Apply[T any, O ~func(*T)](target *T, options []O) {
	for _, option := range options {
		if option != nil {
			option(target)
		}
	}
}
