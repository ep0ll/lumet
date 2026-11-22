package llbutil

import (
	"context"

	"github.com/moby/buildkit/client/llb"
	"github.com/moby/buildkit/frontend/gateway/client"
	"github.com/opencontainers/go-digest"
	"github.com/pkg/errors"
)

func StateDigest(ctx context.Context, st llb.State, c client.Client) (digest.Digest, error) {
	def, err := st.Marshal(ctx, llb.WithCaps(c.BuildOpts().Caps))
	if err != nil {
		return "", errors.Wrap(err, "marshal state")
	}
	return def.Head()
}

func SingleRef(ctx context.Context, st llb.State, c client.Client) (client.Reference, error) {
	def, err := st.Marshal(ctx, llb.WithCaps(c.BuildOpts().Caps))
	if err != nil {
		return nil, errors.Wrap(err, "marshal state")
	}
	res, err := c.Solve(ctx, client.SolveRequest{Definition: def.ToPB()})
	if err != nil {
		return nil, errors.Wrap(err, "solve state")
	}
	return res.SingleRef()
}

func ReadFileCandidates(ctx context.Context, ref client.Reference, candidates []string) (data []byte, name string, err error) {
	var lastErr error
	for _, name = range candidates {
		if dt, err := ReadFile(ctx, ref, name); err == nil {
			return dt, name, nil
		}
		lastErr = err
	}
	return nil, "", lastErr
}
