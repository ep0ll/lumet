package maincontext

import (
	"github.com/moby/buildkit/client/llb"
)

// DefaultMainContext creates the conventional local context source without
// ignore filtering. Prefer BuildKitLoader for configurable source selection.
func DefaultMainContext(opts ...llb.LocalOption) *llb.State {
	localOpts := []llb.LocalOption{
		llb.SharedKeyHint("context"),
		llb.WithCustomName("[internal] load build context"),
	}
	localOpts = append(localOpts, opts...)
	state := llb.Local("context", localOpts...)
	return &state
}
