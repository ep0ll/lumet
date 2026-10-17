package context

import (
	"context"

	"github.com/ep0ll/lumet/context/lumet"
	"github.com/moby/buildkit/client/llb"
	dockerspec "github.com/moby/docker-image-spec/specs-go/v1"
	"github.com/opencontainers/go-digest"
	ocispecs "github.com/opencontainers/image-spec/specs-go/v1"
)

type BuildClient interface {
	BuildDefinition(ctx context.Context, opts ...llb.LocalOption) (lumet.Source, error)
	MainContext(ctx context.Context, opts ...llb.LocalOption) (*llb.State, error)
	NamedContext(name string, opt ContextOpt) (NamedContext, error)
}

type NamedContext interface {
	Load(ctx context.Context) (*llb.State, *dockerspec.DockerOCIImage, error)
}

type BuildSpec struct {
	Source lumet.Source
	Context *llb.State
	Named NamedContext
}

type ContextOpt struct {
	NoIgnorePatterns      bool
	AsyncLocalOpts func() []llb.LocalOption
	Platform       *ocispecs.Platform
	ResolveMode    string
	CaptureDigest  *digest.Digest
}
