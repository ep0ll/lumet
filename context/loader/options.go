package loader

import (
	"github.com/moby/buildkit/client/llb"
	"github.com/opencontainers/go-digest"
	ocispecs "github.com/opencontainers/image-spec/specs-go/v1"
)

const maxContextDepth = 10

// ContextOpt configures a named-context source.
type ContextOpt struct {
	NoIgnorePatterns bool
	AsyncLocalOpts   func() []llb.LocalOption
	Platform         *ocispecs.Platform
	ResolveMode      string
	CaptureDigest    *digest.Digest
}
