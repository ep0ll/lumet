package lumet

import (
	"context"

	"github.com/moby/buildkit/client/llb"
	"github.com/moby/buildkit/frontend/gateway/client"
	"github.com/ep0ll/lumet/context/llbutil"
	"github.com/moby/buildkit/util/flightcontrol"
	"github.com/pkg/errors"
)

type definitionResult struct {
	Data     []byte
	Filename string
}

var definitionCache flightcontrol.CachedGroup[*definitionResult]


const (
	CandidateLPy = ".lumet/main.lpy"
	CandidatePy  = ".lumet/main.py"
)

var defaultCandidates = [...]string{CandidateLPy, CandidatePy}

func DefaultCandidates() []string {
	d := defaultCandidates
	return d[:]
}

func LoadDefinition(ctx context.Context, st llb.State, c client.Client, candidates ...string) ([]byte, string, error) {
	key, err := llbutil.StateDigest(ctx, st, c)
	if err != nil {
		return nil, "", err
	}

	if len(candidates) == 0 {
		candidates = DefaultCandidates()
	}

	res, err := definitionCache.Do(ctx, key.String(), func(ctx context.Context) (*definitionResult, error) {
		ref, err := llbutil.SingleRef(ctx, st, c)
		if err != nil {
			return nil, err
		}

		dt, name, err := llbutil.ReadFileCandidates(ctx, ref, candidates)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to read definition (tried %v)", candidates)
		}
		return &definitionResult{Data: dt, Filename: name}, nil
	})
	if err != nil {
		return nil, "", err
	}
	return res.Data, res.Filename, nil
}
