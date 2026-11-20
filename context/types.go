package context

import (
	"context"

	"github.com/ep0ll/lumet/context/definition"
	"github.com/ep0ll/lumet/context/ignore"
	"github.com/ep0ll/lumet/context/loader"
	"github.com/ep0ll/lumet/context/maincontext"
	"github.com/ep0ll/lumet/context/namedcontext"
	"github.com/moby/buildkit/client/llb"
)

// BuildClient exposes independent build-definition, ignore-pattern,
// main-context, and named-context operations.
type BuildClient interface {
	BuildDefinition(context.Context, ...definition.Option) (definition.Source, error)
	IgnorePatterns(context.Context, ...ignore.Option) ([]string, error)
	MainContext(context.Context, ...maincontext.Option) (*llb.State, error)
	NamedContext(string, loader.ContextOpt) (namedcontext.NamedContext, error)
	BaseContext(string, loader.ContextOpt) (namedcontext.NamedContext, error)
}

// Source is the frontend-neutral build-definition source contract.
type Source = definition.Source

// NamedContext is a named BuildKit source.
type NamedContext = namedcontext.NamedContext

// ContextOpt configures a named context.
type ContextOpt = loader.ContextOpt

var _ BuildClient = (*Client)(nil)
