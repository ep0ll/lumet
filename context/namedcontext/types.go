package namedcontext

import (
	"context"

	"github.com/moby/buildkit/client/llb"
	dockerspec "github.com/moby/docker-image-spec/specs-go/v1"
)

type NamedContext interface {
	Load(ctx context.Context) (*llb.State, *dockerspec.DockerOCIImage, error)
}
