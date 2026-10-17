package llbutil

import (
	"github.com/moby/buildkit/client/llb"
	"github.com/moby/buildkit/frontend/gateway/client"
)

func WithInternalName(name string) llb.ConstraintsOpt {
	return llb.WithCustomName("[internal] " + name)
}

func MarshalOpts(c client.BuildOpts) []llb.ConstraintsOpt {
	return []llb.ConstraintsOpt{llb.WithCaps(c.Caps)}
}
