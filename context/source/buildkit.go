package source

import (
	"context"
	"fmt"
	"net/url"
	"path"
	"strings"

	"github.com/ep0ll/lumet/context/archive"
	"github.com/ep0ll/lumet/context/loader"
	"github.com/ep0ll/lumet/frontend"
	"github.com/moby/buildkit/client/llb"
	"github.com/moby/buildkit/frontend/dockerui"
	gwclient "github.com/moby/buildkit/frontend/gateway/client"
	dockerspec "github.com/moby/docker-image-spec/specs-go/v1"
)

func resolveLocal(ctx context.Context, client frontend.Client, spec Spec, opts Options) (*Resolved, error) {
	return resolveWithLoader(ctx, client, spec, opts, "local:"+spec.Reference)
}

func resolveGit(_ context.Context, client frontend.Client, spec Spec, opts Options) (*Resolved, error) {
	state, err := loader.ResolveGit(client, spec.Reference, opts.KeepGitDir)
	if err != nil {
		return nil, err
	}
	return &Resolved{Spec: spec, State: state}, nil
}

func resolveHTTP(ctx context.Context, client frontend.Client, spec Spec, _ Options) (*Resolved, error) {
	parsed, err := url.Parse(spec.Reference)
	if err != nil {
		return nil, fmt.Errorf("invalid HTTP(S) source %q: %w", spec.Reference, err)
	}
	if (!strings.EqualFold(parsed.Scheme, "http") && !strings.EqualFold(parsed.Scheme, "https")) || parsed.Host == "" {
		return nil, fmt.Errorf("invalid HTTP(S) URL %q", spec.Reference)
	}

	state, filename, ok := dockerui.DetectHTTPContext(spec.Reference)
	if !ok {
		return nil, fmt.Errorf("invalid HTTP(S) source %q", spec.Reference)
	}
	resolved := &Resolved{Spec: spec, State: state, Filename: filename}
	solved, err := resolved.Solve(ctx, client)
	if err != nil {
		return nil, fmt.Errorf("inspect HTTP source %q: %w", spec.Reference, err)
	}
	header, err := solved.Reference.ReadFile(ctx, gwclient.ReadRequest{
		Filename: filename,
		Range:    &gwclient.FileRange{Length: 1024},
	})
	if err != nil {
		return nil, fmt.Errorf("read HTTP source header %q: %w", spec.Reference, err)
	}

	if archive.IsArchive(header) {
		unpacked := llb.Scratch().File(llb.Copy(*state, path.Join("/", filename), "/", &llb.CopyInfo{
			AttemptUnpack: true,
		}))
		return &Resolved{Spec: spec, State: &unpacked}, nil
	}
	return resolved, nil
}

func resolveInput(ctx context.Context, client frontend.Client, spec Spec, opts Options) (*Resolved, error) {
	return resolveWithLoader(ctx, client, spec, opts, "input:"+spec.Reference)
}

func resolveImage(ctx context.Context, client frontend.Client, spec Spec, opts Options) (*Resolved, error) {
	return resolveWithLoader(ctx, client, spec, opts, "docker-image:"+strings.TrimPrefix(spec.Reference, "//"))
}

func resolveOCI(ctx context.Context, client frontend.Client, spec Spec, opts Options) (*Resolved, error) {
	return resolveWithLoader(ctx, client, spec, opts, "oci-layout:"+strings.TrimPrefix(spec.Reference, "//"))
}

// resolveWithLoader adapts the independent source model to the established
// context loader. This keeps source dispatch, image metadata, redirects, and
// lazy local evaluation in one place instead of maintaining parallel loaders.
func resolveWithLoader(
	ctx context.Context,
	client frontend.Client,
	spec Spec,
	opts Options,
	input string,
) (*Resolved, error) {
	name := opts.Name
	if name == "" {
		name = "buildkit-context"
		if spec.Kind == KindLocal {
			name = spec.Reference
		}
	}
	nameWithPlatform := opts.NameWithPlatform
	if nameWithPlatform == "" {
		nameWithPlatform = name
	}
	localOptions := append([]llb.LocalOption(nil), opts.LocalOptions...)

	var load func(frontend.Client, string, int) (*llb.State, *dockerspec.DockerOCIImage, error)
	load = func(activeClient frontend.Client, activeInput string, depth int) (*llb.State, *dockerspec.DockerOCIImage, error) {
		if depth > 10 {
			return nil, nil, fmt.Errorf("context source redirection limit exceeded for %q", spec.Reference)
		}
		request := loader.Request{
			Input:            activeInput,
			Name:             name,
			NameWithPlatform: nameWithPlatform,
			Client:           activeClient,
			Opt: loader.ContextOpt{
				NoIgnorePatterns: true,
				AsyncLocalOpts: func() []llb.LocalOption {
					return append([]llb.LocalOption(nil), localOptions...)
				},
				FollowPaths:      append([]string(nil), opts.FollowPaths...),
				ExcludePatterns:  append([]string(nil), opts.ExcludePatterns...),
				Platform:         opts.Platform,
				ResolveMode:      opts.ResolveMode,
				CaptureDigest:    opts.CaptureDigest,
				SharedKey:        opts.SharedKey,
			},
		}
		request.Retry = func(newInput string) (*llb.State, *dockerspec.DockerOCIImage, error) {
			nextClient := frontend.SetOpt(activeClient, "context:"+nameWithPlatform, newInput)
			return load(nextClient, newInput, depth+1)
		}
		return loader.Load(ctx, request)
	}

	state, image, err := load(client, input, 0)
	if err != nil {
		return nil, err
	}
	if state == nil {
		return nil, fmt.Errorf("source loader returned an empty state")
	}
	return &Resolved{Spec: spec, State: state, Image: image}, nil
}
