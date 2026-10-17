package context

import (
	"context"

	"github.com/ep0ll/lumet/context/lumet"
	"github.com/ep0ll/lumet/frontend"

	"github.com/moby/buildkit/client/llb"
	"github.com/moby/buildkit/util/flightcontrol"
	"github.com/pkg/errors"
)

type buildContext struct {
	g      flightcontrol.CachedGroup[BuildSpec]
	spec   frontend.Spec
	client frontend.Client
}

func New(spec frontend.Spec, client frontend.Client) BuildClient {
	return &buildContext{
		spec:   spec,
		client: client,
	}
}

// BuildDefinition implements BuildClient.
func (b *buildContext) BuildDefinition(ctx context.Context, opts ...llb.LocalOption) (lumet.Source, error) {
	spec, err := b.g.Do(ctx, "build", func(ctx context.Context) (BuildSpec, error) {
		source, err := lumet.BuildDefinition()
		if err != nil {
			err = errors.Wrap(err, "failed to load build definition")
		}
		
		return BuildSpec{
			Source: source,
		}, err
	})
	return spec.Source, err
}

// MainContext implements BuildClient.
func (b *buildContext) MainContext(ctx context.Context, opts ...llb.LocalOption) (*llb.State, error) {
	spec, err := b.g.Do(ctx, "main", func(ctx context.Context) (BuildSpec, error) {
		return BuildSpec{}, nil
	})
	return spec.Context, err
}

// NamedContext implements BuildClient.
func (b *buildContext) NamedContext(name string, opt ContextOpt) (NamedContext, error) {
	spec, err := b.g.Do(context.Background(), "named", func(ctx context.Context) (BuildSpec, error) {
		return BuildSpec{}, nil
	})
	return spec.Named, err
}
