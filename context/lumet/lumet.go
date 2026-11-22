package lumet

import (
	"context"

	"github.com/moby/buildkit/client/llb"
	"github.com/moby/buildkit/frontend/gateway/client"
	"github.com/ep0ll/lumet/context/llbutil"
	"github.com/moby/buildkit/util/flightcontrol"
	"github.com/pkg/errors"
)

// fileResult is cached per (state digest, filename). Found=false caches misses,
// so probing absent candidates doesn't re-read on every call.
type fileResult struct {
	Data  []byte
	Found bool
}

var (
	fileCache flightcontrol.CachedGroup[*fileResult]
)

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
	digest, err := llbutil.StateDigest(ctx, st, c)
	if err != nil {
		return nil, "", err
	}

	if len(candidates) == 0 {
		candidates = DefaultCandidates()
	}

	for _, name := range candidates {
		key := digest.String() + "\x00" + name

		res, err := fileCache.Do(ctx, key, func(ctx context.Context) (*fileResult, error) {
			ref, err := llbutil.SingleRef(ctx, st, c)
			if err != nil {
				return nil, err
			}
			dt, err := llbutil.ReadFile(ctx, ref, name)
			if err != nil {
				if llbutil.IsFileTooLarge(err) {
					return nil, err
				}
				return &fileResult{}, nil
			}
			return &fileResult{Data: dt, Found: true}, nil
		})
		if err != nil {
			return nil, "", errors.Wrapf(err, "failed to read %s", name)
		}
		if res.Found {
			return res.Data, name, nil
		}
	}
	return nil, "", errors.Errorf("no definition found (tried %v)", candidates)
}

// UnderLumet returns a new state that contains the entire contents of `st`
// under the directory ".lumet".
//
// Equivalent to:
//
//	Scratch
//	└── .lumet/          ← everything that was previously at the root of st
func UnderLumet(st llb.State) llb.State {
	return llb.Scratch().File(
		llb.Copy(st, "/", ".lumet", &llb.CopyInfo{
			CopyDirContentsOnly: true,
			CreateDestPath:      true,
		}),
	)
}
