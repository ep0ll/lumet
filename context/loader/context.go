package loader

import (
	"slices"

	"github.com/ep0ll/lumet/context/llbutil"
	"github.com/moby/buildkit/client/llb"
	"github.com/opencontainers/go-digest"
	ocispecs "github.com/opencontainers/image-spec/specs-go/v1"
)

// IgnoreParser parses the contents of one ignore file.
type IgnoreParser = llbutil.IgnoreParser

// ContextOpt configures a named context source.
type ContextOpt struct {
	NoIgnorePatterns bool
	AsyncLocalOpts   func() []llb.LocalOption
	Platform         *ocispecs.Platform
	ResolveMode      string
	CaptureDigest    *digest.Digest
	SharedKey        string

	// FollowPaths limits local source evaluation to these source-relative files.
	FollowPaths []string

	// ExcludePatterns excludes source-relative paths without loading an ignore file.
	ExcludePatterns []string

	// IgnoreFiles overrides the conventional .dockerignore candidate for
	// local named contexts. Candidates are checked in order. An empty slice
	// keeps the historical .dockerignore default.
	IgnoreFiles []string

	// IgnoreParser overrides the Moby ignore-file parser.
	IgnoreParser IgnoreParser
}

// Clone returns a copy whose mutable fields do not alias the original.
func (o ContextOpt) Clone() ContextOpt {
	o.FollowPaths = slices.Clone(o.FollowPaths)
	o.ExcludePatterns = slices.Clone(o.ExcludePatterns)
	o.IgnoreFiles = slices.Clone(o.IgnoreFiles)
	if o.Platform != nil {
		p := *o.Platform
		p.OSFeatures = slices.Clone(o.Platform.OSFeatures)
		o.Platform = &p
	}
	return o
}
