package loader

import (
	"context"
	"fmt"
	"strings"

	"github.com/ep0ll/lumet/context/llbutil"
	"github.com/ep0ll/lumet/context/localoutput"
	"github.com/ep0ll/lumet/frontend"
	"github.com/moby/buildkit/client/llb"
	gwclient "github.com/moby/buildkit/frontend/gateway/client"
	dockerspec "github.com/moby/docker-image-spec/specs-go/v1"
)

func loadLocal(ctx context.Context, name string, req Request) (*llb.State, *dockerspec.DockerOCIImage, error) {
	sessionID, _ := frontend.GetLocalSession(req.Client, name)

	followPaths, err := llbutil.CleanRelativePaths(req.Opt.FollowPaths)
	if err != nil {
		return nil, nil, fmt.Errorf("invalid local context follow paths: %w", err)
	}
	excludes := append([]string(nil), req.Opt.ExcludePatterns...)
	if !req.Opt.NoIgnorePatterns {
		patterns, err := readIgnoreFile(ctx, name, sessionID, req)
		if err != nil {
			return nil, nil, err
		}
		excludes = append(excludes, patterns...)
	}

	state := llb.NewState(&localoutput.AsyncOutput{
		Name:             name,
		NameWithPlatform: req.NameWithPlatform,
		SessionID:        sessionID,
		FollowPaths:      followPaths,
		Excludes:         excludes,
		Extra:            req.Opt.AsyncLocalOpts,
		SharedKey:        req.Opt.SharedKey,
	})
	return &state, nil, nil
}

func readIgnoreFile(ctx context.Context, name, sessionID string, req Request) ([]string, error) {
	names := req.Opt.IgnoreFiles
	if len(names) == 0 {
		names = []string{DefaultIgnoreName}
	}
	names, err := cleanIgnoreNames(names)
	if err != nil {
		return nil, err
	}
	if len(names) == 0 {
		return nil, nil
	}

	probe := llb.Local(name,
		llb.SessionID(sessionID),
		llb.FollowPaths(names),
		llb.SharedKeyHint("context:"+req.NameWithPlatform+"-ignore-"+strings.Join(names, "-")),
		llb.WithCustomName(label(req.NameWithPlatform, "load context ignore files")),
		llb.Differ(llb.DiffNone, false),
	)
	definition, err := probe.Marshal(ctx, llbutil.MarshalOpts(req.Client.BuildOpts())...)
	if err != nil {
		return nil, fmt.Errorf("marshal local ignore-file probe: %w", err)
	}
	result, err := req.Client.Solve(ctx, gwclient.SolveRequest{
		Evaluate:   true,
		Definition: definition.ToPB(),
	})
	if err != nil {
		return nil, fmt.Errorf("solve local ignore-file probe: %w", err)
	}
	ref, err := result.SingleRef()
	if err != nil {
		return nil, fmt.Errorf("resolve local ignore-file reference: %w", err)
	}

	for _, filename := range names {
		data, err := llbutil.ReadFile(ctx, ref, filename)
		if llbutil.IsNotFound(err) {
			continue
		}
		if err != nil {
			return nil, fmt.Errorf("read ignore file %q: %w", filename, err)
		}

		patterns, err := llbutil.ParseIgnoreFile(req.Opt.IgnoreParser, filename, data)
		if err != nil {
			return nil, fmt.Errorf("parse ignore file %q: %w", filename, err)
		}
		return patterns, nil
	}
	return nil, nil
}

func cleanIgnoreNames(names []string) ([]string, error) {
	filtered := make([]string, 0, len(names))
	for _, name := range names {
		if strings.TrimSpace(name) != "" {
			filtered = append(filtered, name)
		}
	}
	return llbutil.CleanRelativePaths(filtered)
}
