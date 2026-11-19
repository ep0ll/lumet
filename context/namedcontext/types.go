package namedcontext

import (
	"context"

	"github.com/moby/buildkit/client/llb"
	dockerspec "github.com/moby/docker-image-spec/specs-go/v1"
	"github.com/opencontainers/go-digest"
	ocispecs "github.com/opencontainers/image-spec/specs-go/v1"
)

type ContextOpt struct {
	NoIgnorePatterns      bool
	AsyncLocalOpts func() []llb.LocalOption
	Platform       *ocispecs.Platform
	ResolveMode    string
	CaptureDigest  *digest.Digest
}

type NamedContext interface {
	Load(ctx context.Context) (*llb.State, *dockerspec.DockerOCIImage, error)
}
