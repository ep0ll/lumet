package source

import (
	"context"
	"fmt"

	"github.com/ep0ll/lumet/context/llbutil"
	"github.com/ep0ll/lumet/frontend"
)

// Resolve selects the configured resolver or the built-in resolver and
// resolves a source using a defensive copy of its options.
func Resolve(ctx context.Context, client frontend.Client, resolver Resolver, spec Spec, opts Options) (*Resolved, error) {
	if err := spec.Validate(); err != nil {
		return nil, fmt.Errorf("source: %w", err)
	}
	if resolver == nil {
		resolver = DefaultResolver()
	}
	return resolver.Resolve(ctx, client, spec, opts.Clone())
}

// CleanPaths centralizes source-relative path normalization and validation.
func CleanPaths(paths []string) ([]string, error) {
	return llbutil.CleanRelativePaths(paths)
}
