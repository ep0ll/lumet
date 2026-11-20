// Package lumet retains the historical name for the generic build-definition
// source contract. New code should import context/definition.
package lumet

import (
	"context"
	"errors"

	"github.com/ep0ll/lumet/context/definition"
	"github.com/moby/buildkit/client/llb"
	gwclient "github.com/moby/buildkit/frontend/gateway/client"
)

// Source is an alias retained for code that used the earlier context/lumet
// package.
type Source = definition.Source

// NewSource creates a frontend-neutral source with source-map diagnostics
// and optional lazy ignore-pattern loading.
func NewSource(
	sourceMap *llb.SourceMap,
	ignoreFn func(context.Context) ([]string, error),
	warnFn func(context.Context, string, gwclient.WarnOpts),
) Source {
	return definition.NewSource(sourceMap, ignoreFn, warnFn)
}

// BuildDefinition no longer has a global implementation. A definition source
// must be selected for each frontend invocation.
//
// Deprecated: use context.Client.BuildDefinition with a definition.Loader.
func BuildDefinition() (Source, error) {
	return nil, errors.New("BuildDefinition requires an explicit definition loader")
}
