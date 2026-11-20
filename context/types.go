package context

import (
	"context"

	"github.com/ep0ll/lumet/context/definition"
	"github.com/ep0ll/lumet/context/loader"
	"github.com/ep0ll/lumet/context/namedcontext"
	"github.com/moby/buildkit/client/llb"
)

// BuildClient exposes the build-definition, main-context, and named-context
// sources shared by BuildKit frontends.
type BuildClient interface {
	BuildDefinition(ctx context.Context, opts ...llb.LocalOption) (definition.Source, error)
	MainContext(ctx context.Context, opts ...llb.LocalOption) (*llb.State, error)
	NamedContext(name string, opt ContextOpt) (NamedContext, error)
	BaseContext(name string, opt ContextOpt) (NamedContext, error)
}

// NamedContext is a named source that can be loaded into an LLB state.
type NamedContext = namedcontext.NamedContext

// ContextOpt configures how a named context is resolved and loaded.
type ContextOpt = loader.ContextOpt

// BuildSpec is retained for callers that used the early context package
// scaffold. BuildClient operations now return their values directly.
type BuildSpec struct {
	Source  definition.Source
	Context *llb.State
	Named   NamedContext
}
