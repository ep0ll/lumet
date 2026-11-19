package loader

import (
	"bytes"
	"context"

	"github.com/dexnore/dexfile"
	"github.com/ep0ll/lumet/context/llbutil"
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

	sharedKey := req.Client.BuildOpts().Opts["sharedkey:localdir:"+req.NameWithPlatform]
	out := localoutput.New(name, req.NameWithPlatform, sessionID, sharedKey, excludes, req.Opt.AsyncLocalOpts)
	state := llb.NewState(out)
	return &state, nil, nil
}

func readIgnoreFile(ctx context.Context, name, sessionID string, req Request) ([]string, error) {
	if req.Opt.NoIgnorePatterns {
		return nil, nil
	}

	probe := llb.Local(name,
		llb.SessionID(sessionID),
		llb.FollowPaths([]string{dexfile.DefaultDexnoreName}),
		llb.SharedKeyHint("context:"+req.NameWithPlatform+"-"+dexfile.DefaultDexnoreName),
		llb.WithCustomName(label(req.NameWithPlatform, "load "+dexfile.DefaultDexnoreName)),
		llb.Differ(llb.DiffNone, false),
	)
	def, err := probe.Marshal(ctx, llbutil.MarshalOpts(req.Client.BuildOpts())...)
	if err != nil {
		return nil, errors.Wrap(err, "failed to marshal ignore-file probe")
	}
	res, err := req.Client.Solve(ctx, gwclient.SolveRequest{
		Evaluate:   true,
		Definition: def.ToPB(),
	})
	if err != nil {
		return nil, errors.Wrap(err, "failed to solve ignore-file probe")
	}
	ref, err := res.SingleRef()
	if err != nil {
		return nil, errors.Wrap(err, "failed to get ignore-file reference")
	}

	data, err := llbutil.ReadFile(ctx, ref, dexfile.DefaultDexnoreName)
	if llbutil.IsNotFound(err) {
		return nil, nil
	}
	if err != nil {
		return nil, errors.Wrapf(err, "failed to read %s", dexfile.DefaultDexnoreName)
	}
	patterns, err := ignorefile.ReadAll(bytes.NewReader(data))
	if err != nil {
		return nil, errors.Wrapf(err, "failed parsing %s", dexfile.DefaultDexnoreName)
	}
	return patterns, nil
}
