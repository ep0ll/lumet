package namedcontext

import (
	"context"

	"github.com/ep0ll/lumet/context/loader"
	"github.com/ep0ll/lumet/frontend"
	"github.com/moby/buildkit/client/llb"
	dockerspec "github.com/moby/docker-image-spec/specs-go/v1"
	"github.com/pkg/errors"
)

const maxDepth = 10

// Source implements dexfile.NamedContext.
type Source struct {
	input            string
	name             string
	nameWithPlatform string
	client           frontend.Client
	opt              ContextOpt
}

func (s *Source) Load(ctx context.Context) (*llb.State, *dockerspec.DockerOCIImage, error) {
	return s.load(ctx, 0)
}

func (s *Source) load(ctx context.Context, depth int) (*llb.State, *dockerspec.DockerOCIImage, error) {
	if depth > maxDepth {
		return nil, nil, errors.Errorf("context recursion limit exceeded (possible cycle): %s", s.input)
	}

	return loader.Load(ctx, loader.Request{
		Input:            s.input,
		Name:             s.name,
		NameWithPlatform: s.nameWithPlatform,
		Client:           s.client,
		Opt:              s.opt,
		Depth:            depth,
		Retry: func(newInput string) (*llb.State, *dockerspec.DockerOCIImage, error) {
			next := *s
			next.input = newInput
			return next.load(ctx, depth+1)
		},
	})
}
