// Package lumet contains compatibility aliases for the build-definition
// abstractions that predate the frontend-neutral context/definition package.
package lumet

import (
	"context"
	"errors"

	"github.com/ep0ll/lumet/context/definition"
	"github.com/moby/buildkit/client/llb"
	gwclient "github.com/moby/buildkit/frontend/gateway/client"
)

// Source is retained as an alias for compatibility. New frontend code should
// use context/definition.Source.
type Source = definition.Source

// NewSource creates a frontend-neutral build-definition source.
func NewSource(
	sourceMap *llb.SourceMap,
	ignoreFn func(context.Context) ([]string, error),
	warnFn func(context.Context, string, gwclient.WarnOpts),
) Source {
	return definition.NewSource(sourceMap, ignoreFn, warnFn)
}

// BuildDefinition cannot be implemented without a frontend specification and
// a BuildKit client. Use context.New(spec, client).BuildDefinition instead.
//
// Deprecated: use context.BuildClient.BuildDefinition.
func BuildDefinition() (Source, error) {
	return nil, errors.New("BuildDefinition requires a frontend spec and BuildKit client; use context.New")
}
