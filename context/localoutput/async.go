package localoutput

import (
	"context"
	"slices"
	"sync"

	"github.com/moby/buildkit/client/llb"
	"github.com/moby/buildkit/solver/pb"
)

// AsyncOutput materialises an llb.Local only when it is first used.
type AsyncOutput struct {
	Name             string
	NameWithPlatform string
	SessionID        string
	SharedKey        string
	Excludes         []string
	Extra            func() []llb.LocalOption

	output llb.Output
	once   sync.Once
}

// New creates an asynchronous local output with independent copies of
// mutable configuration.
func New(name, nameWithPlatform, sessionID, sharedKey string, excludes []string, extra func() []llb.LocalOption) *AsyncOutput {
	return &AsyncOutput{
		Name:             name,
		NameWithPlatform: nameWithPlatform,
		SessionID:        sessionID,
		SharedKey:        sharedKey,
		Excludes:         slices.Clone(excludes),
		Extra:            extra,
	}
}

func (a *AsyncOutput) ToInput(ctx context.Context, constraints *llb.Constraints) (*pb.Input, error) {
	a.once.Do(a.materialise)
	return a.output.ToInput(ctx, constraints)
}

func (a *AsyncOutput) Vertex(ctx context.Context, constraints *llb.Constraints) llb.Vertex {
	a.once.Do(a.materialise)
	return a.output.Vertex(ctx, constraints)
}

func (a *AsyncOutput) materialise() {
	opts := []llb.LocalOption{
		llb.WithCustomName("[context " + a.NameWithPlatform + "] load from client"),
		llb.SessionID(a.SessionID),
		llb.SharedKeyHint("context:" + a.NameWithPlatform + "-" + a.SharedKey),
		llb.ExcludePatterns(slices.Clone(a.Excludes)),
	}
	if a.Extra != nil {
		opts = append(opts, a.Extra()...)
	}

	state := llb.Local(a.Name, opts...)
	a.output = state.Output()
}
