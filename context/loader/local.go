package loader

import (
	"bytes"
	"context"

	"github.com/ep0ll/lumet/context/localoutput"
	"github.com/moby/buildkit/client/llb"
	gwclient "github.com/moby/buildkit/frontend/gateway/client"
	dockerspec "github.com/moby/docker-image-spec/specs-go/v1"
	"github.com/moby/patternmatcher/ignorefile"
	"github.com/pkg/errors"
)

func loadLocal(ctx context.Context, name string, req Request) (*llb.State, *dockerspec.DockerOCIImage, error) {
	sessionID, _ := req.Client.GetLocalSession(name)

	excludes, err := readIgnoreFile(ctx, name, sessionID, req)
	if err != nil {
		return nil, nil, err
	}

	out := localoutput.New(name, req.NameWithPlatform, sessionID, excludes, req.Opt.AsyncLocalOpts)
	st := llb.NewState(out)
	return &st, nil, nil
}

func readIgnoreFile(ctx context.Context, name, sessionID string, req Request) ([]string, error) {
	if req.Opt.NoDexnore {
		return nil, nil
	}

	probe := llb.Local(name,
		llb.SessionID(sessionID),
		llb.FollowPaths([]string{dexfile.DefaultDexnoreName}),
		llb.SharedKeyHint("context:"+req.NameWithPlatform+"-"+dexfile.DefaultDexnoreName),
		llb.WithCustomName(label(req.NameWithPlatform, "load "+dexfile.DefaultDexnoreName)),
		llb.Differ(llb.DiffNone, false),
	)

	def, err := probe.Marshal(ctx)
	if err != nil {
		return nil, err
	}
	res, err := req.Client.Solve(ctx, gwclient.SolveRequest{
		Evaluate:   true,
		Definition: def.ToPB(),
	})
	if err != nil {
		return nil, err
	}
	ref, err := res.SingleRef()
	if err != nil {
		return nil, err
	}

	dt, _ := ref.ReadFile(ctx, gwclient.ReadRequest{Filename: dexfile.DefaultDexnoreName})
	if len(dt) == 0 {
		return nil, nil
	}

	patterns, err := ignorefile.ReadAll(bytes.NewBuffer(dt))
	if err != nil {
		return nil, errors.Wrapf(err, "failed parsing %s", dexfile.DefaultDexnoreName)
	}
	return patterns, nil
}
