package localoutput

import (
	"context"
	"sync"

	"github.com/moby/buildkit/client/llb"
	"github.com/moby/buildkit/solver/pb"
)

// AsyncOutput is an llb.Output that materialises an llb.Local
// only when it is first used.
type AsyncOutput struct {
	Name             string
	NameWithPlatform string
	SessionID        string
	FollowPaths      []string
	Excludes         []string
	Extra            func() []llb.LocalOption
	SharedKey        string

	output llb.Output
	once   sync.Once
}

func (a *AsyncOutput) ToInput(ctx context.Context, c *llb.Constraints) (*pb.Input, error) {
	a.once.Do(a.materialise)
	return a.output.ToInput(ctx, c)
}

func (a *AsyncOutput) Vertex(ctx context.Context, c *llb.Constraints) llb.Vertex {
	a.once.Do(a.materialise)
	return a.output.Vertex(ctx, c)
}

func (a *AsyncOutput) materialise() {
	sharedKey := "context:" + a.NameWithPlatform
	if a.SharedKey != "" {
		sharedKey += "-" + a.SharedKey
	}
	opts := []llb.LocalOption{
		llb.WithCustomName("[context " + a.NameWithPlatform + "] load from client"),
		llb.SessionID(a.SessionID),
		llb.SharedKeyHint(sharedKey),
	}
	if len(a.FollowPaths) != 0 {
		opts = append(opts, llb.FollowPaths(a.FollowPaths))
	}
	if len(a.Excludes) != 0 {
		opts = append(opts, llb.ExcludePatterns(a.Excludes))
	}
	if a.Extra != nil {
		opts = append(opts, a.Extra()...)
	}

	st := llb.Local(a.Name, opts...)
	a.output = st.Output()
}
