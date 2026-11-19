package context

import (
	"context"

	"github.com/ep0ll/lumet/context/loader"
	"github.com/ep0ll/lumet/context/lumet"
	"github.com/ep0ll/lumet/context/namedcontext"
	"github.com/moby/buildkit/client/llb"
)

type BuildClient interface {
	BuildDefinition(ctx context.Context, opts ...llb.LocalOption) (lumet.Source, error)
	MainContext(ctx context.Context, opts ...llb.LocalOption) (*llb.State, error)
	NamedContext(name string, opt loader.ContextOpt) (namedcontext.NamedContext, error)
}

type BuildSpec struct {
	Source lumet.Source
	Context *llb.State
	Named  namedcontext.NamedContext
}
