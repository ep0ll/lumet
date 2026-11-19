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

// Source resolves a registered context lazily.
type Source struct {
	input            string
	name             string
	nameWithPlatform string
	client           frontend.Client
	opt              ContextOpt
}

func (s *Source) Load(ctx context.Context) (*llb.State, *dockerspec.DockerOCIImage, error) {
	if s == nil {
		return nil, nil, errors.New("named context source is nil")
	}
	if s.client == nil {
		return nil, nil, errors.New("named context source has no frontend client")
	}

	// SetOpt mutates BuildOpts. Use an independent client view for each load
	// so concurrent named-context loads cannot race while following redirects.
	client := s.client.Clone()
	if client == nil {
		return nil, nil, errors.New("failed to clone frontend client for named context")
	}
	return s.load(ctx, 0, client)
}

func (s *Source) load(ctx context.Context, depth int, client frontend.Client) (*llb.State, *dockerspec.DockerOCIImage, error) {
	if depth > maxDepth {
		return nil, nil, errors.Errorf("context recursion limit exceeded (possible cycle): %s", s.input)
	}

	return loader.Load(ctx, loader.Request{
		Input:            s.input,
		Name:             s.name,
		NameWithPlatform: s.nameWithPlatform,
		Client:           client,
		Opt:              s.opt,
		Depth:            depth,
		Retry: func(newInput string) (*llb.State, *dockerspec.DockerOCIImage, error) {
			next := *s
			next.input = newInput
			return next.load(ctx, depth+1, client)
		},
	})
}
