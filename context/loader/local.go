package loader

import (
	"bytes"
	"context"
	"fmt"
	"path"
	"strings"

	"github.com/ep0ll/lumet/context/llbutil"
	"github.com/ep0ll/lumet/context/localoutput"
	"github.com/ep0ll/lumet/frontend"
	"github.com/moby/buildkit/client/llb"
	gwclient "github.com/moby/buildkit/frontend/gateway/client"
	dockerspec "github.com/moby/docker-image-spec/specs-go/v1"
	"github.com/moby/patternmatcher/ignorefile"
)

func loadLocal(ctx context.Context, name string, req Request) (*llb.State, *dockerspec.DockerOCIImage, error) {
	sessionID, _ := frontend.GetLocalSession(req.Client, name)

	excludes, err := readIgnoreFile(ctx, name, sessionID, req)
	if err != nil {
		return nil, nil, err
	}

	st := llb.NewState(&localoutput.AsyncOutput{
		Name:             name,
		NameWithPlatform: req.NameWithPlatform,
		SessionID:        sessionID,
		SharedKey:        req.Opt.SharedKey,
		Excludes:         excludes,
		Extra:            req.Opt.AsyncLocalOpts,
	})
	return &st, nil, nil
}

func readIgnoreFile(ctx context.Context, name, sessionID string, req Request) ([]string, error) {
	if req.Opt.NoIgnorePatterns {
		return nil, nil
	}

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
		llb.SharedKeyHint("context:"+req.NameWithPlatform+"-ignore-"+joinNames(names)),
		llb.WithCustomName(label(req.NameWithPlatform, "load context ignore files")),
		llb.Differ(llb.DiffNone, false),
	)
	definition, err := probe.Marshal(ctx, llbutil.MarshalOpts(req.Client.BuildOpts())...)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal local ignore-file probe: %w", err)
	}
	result, err := req.Client.Solve(ctx, gwclient.SolveRequest{
		Evaluate:   true,
		Definition: definition.ToPB(),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to solve local ignore-file probe: %w", err)
	}
	ref, err := result.SingleRef()
	if err != nil {
		return nil, fmt.Errorf("failed to resolve local ignore-file reference: %w", err)
	}

	for _, name := range names {
		data, err := llbutil.ReadFile(ctx, ref, name)
		if llbutil.IsNotFound(err) {
			continue
		}
		if err != nil {
			return nil, fmt.Errorf("failed to read ignore file %q: %w", name, err)
		}

		var patterns []string
		if req.Opt.IgnoreParser != nil {
			patterns, err = req.Opt.IgnoreParser(name, data)
		} else {
			patterns, err = ignorefile.ReadAll(bytes.NewReader(data))
		}
		if err != nil {
			return nil, fmt.Errorf("failed to parse ignore file %q: %w", name, err)
		}
		return patterns, nil
	}
	return nil, nil
}

func cleanIgnoreNames(names []string) ([]string, error) {
	seen := make(map[string]struct{}, len(names))
	cleaned := make([]string, 0, len(names))
	for _, name := range names {
		name = strings.TrimSpace(strings.ReplaceAll(name, "\\", "/"))
		if name == "" {
			continue
		}
		clean := path.Clean(name)
		if path.IsAbs(clean) || clean == "." || clean == ".." || strings.HasPrefix(clean, "../") {
			return nil, fmt.Errorf("invalid ignore-file path %q: path must remain inside the context root", name)
		}
		if _, ok := seen[clean]; ok {
			continue
		}
		seen[clean] = struct{}{}
		cleaned = append(cleaned, clean)
	}
	return cleaned, nil
}

func joinNames(names []string) string {
	return strings.Join(names, "-")
}
