package namedcontext

import (
	"context"

	"github.com/ep0ll/lumet/context/loader"
	"github.com/moby/buildkit/client/llb"
	dockerspec "github.com/moby/docker-image-spec/specs-go/v1"
)

// ContextOpt is shared with the loader package to keep the dependency
// direction acyclic.
type ContextOpt = loader.ContextOpt

type NamedContext interface {
	Load(ctx context.Context) (*llb.State, *dockerspec.DockerOCIImage, error)
}
