package loader

import (
	"slices"

	"github.com/moby/buildkit/client/llb"
	"github.com/opencontainers/go-digest"
	ocispecs "github.com/opencontainers/image-spec/specs-go/v1"
)

// ContextOpt configures a named context source.
type ContextOpt struct {
	NoIgnorePatterns bool
	AsyncLocalOpts   func() []llb.LocalOption
	Platform         *ocispecs.Platform
	ResolveMode      string
	CaptureDigest    *digest.Digest
	SharedKey        string

	// IgnoreFiles overrides the conventional .dockerignore candidate for
	// local named contexts. Files are checked in order; the first existing
	// file wins. Nil uses DefaultIgnoreName.
	IgnoreFiles []string

	// IgnoreParser overrides the Moby ignore-file parser for this context.
	IgnoreParser func(filename string, data []byte) ([]string, error)
}

// Clone returns a copy whose mutable fields do not alias the original.
func (o ContextOpt) Clone() ContextOpt {
	o.IgnoreFiles = slices.Clone(o.IgnoreFiles)
	if o.Platform != nil {
		p := *o.Platform
		o.Platform = &p
	}
	return o
}
