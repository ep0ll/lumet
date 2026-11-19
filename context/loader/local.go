package loader

import (
	"bytes"
	"context"

	"github.com/ep0ll/lumet/context/llbutil"
	"github.com/ep0ll/lumet/context/localoutput"
	"github.com/ep0ll/lumet/frontend"
	"github.com/moby/buildkit/client/llb"
	gwclient "github.com/moby/buildkit/frontend/gateway/client"
	dockerspec "github.com/moby/docker-image-spec/specs-go/v1"
	"github.com/moby/patternmatcher/ignorefile"
	"github.com/pkg/errors"
)

func loadLocal(ctx context.Context, name string, req Request) (*llb.State, *dockerspec.DockerOCIImage, error) {
	sessionID, _ := frontend.GetLocalSession(req.Client, name)

	excludes, err := readIgnoreFile(ctx, name, sessionID, req)
	if err != nil {
		return nil, nil, err
	}

	st := llb.NewState(&localoutput.AsyncOutput{
		Name: name,
		NameWithPlatform: req.NameWithPlatform,
		SessionID: sessionID,
		SharedKey: req.Opt.SharedKey,
		Excludes: excludes,
		Extra: req.Opt.AsyncLocalOpts,
	})
	return &st, nil, nil
}

func readIgnoreFile(ctx context.Context, name, sessionID string, req Request) ([]string, error) {
	if req.Opt.NoIgnorePatterns {
		return nil, nil
	}

	probe := llb.Local(name,
		llb.SessionID(sessionID),
		llb.FollowPaths([]string{DefaultIgnoreName}),
		llb.SharedKeyHint("context:"+req.NameWithPlatform+"-"+DefaultIgnoreName),
		llb.WithCustomName(label(req.NameWithPlatform, "load "+DefaultIgnoreName)),
		llb.Differ(llb.DiffNone, false),
	)
	def, err := probe.Marshal(ctx, llb.WithCaps(req.Client.BuildOpts().Caps))
	if err != nil {
		return nil, errors.Wrap(err, "marshal local ignore-file probe")
	}
	res, err := req.Client.Solve(ctx, gwclient.SolveRequest{
		Evaluate: true,
		Definition: def.ToPB(),
	})
	if err != nil {
		return nil, errors.Wrap(err, "solve local ignore-file probe")
	}
	ref, err := res.SingleRef()
	if err != nil {
		return nil, errors.Wrap(err, "resolve local ignore-file probe")
	}

	dt, err := llbutil.ReadFile(ctx, ref, DefaultIgnoreName)
	if err != nil {
		if llbutil.IsNotFound(err) {
			return nil, nil
		}
		return nil, errors.Wrapf(err, "read %s", DefaultIgnoreName)
	}
	if len(dt) == 0 {
		return nil, nil
	}

	patterns, err := ignorefile.ReadAll(bytes.NewBuffer(dt))
	if err != nil {
		return nil, errors.Wrapf(err, "failed parsing %s", DefaultIgnoreName)
	}
	return patterns, nil
}
