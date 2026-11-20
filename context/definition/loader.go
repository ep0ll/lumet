package definition

import (
	"context"
	"fmt"

	"github.com/ep0ll/lumet/context/ignore"
	"github.com/ep0ll/lumet/context/llbutil"
	"github.com/ep0ll/lumet/context/source"
	"github.com/ep0ll/lumet/frontend"
	"github.com/moby/buildkit/client/llb"
	gwclient "github.com/moby/buildkit/frontend/gateway/client"
)

// BuildKitLoader loads definition files from any registered source kind.
type BuildKitLoader struct {
	Resolver source.Resolver
}

// Load implements Loader.
func (l BuildKitLoader) Load(ctx context.Context, client frontend.Client, opts Options) (Source, error) {
	if err := opts.Source.Validate(); err != nil {
		return nil, fmt.Errorf("build definition source: %w", err)
	}
	resolver := opts.Resolver
	if resolver == nil {
		resolver = l.Resolver
	}

	fallbacks, err := source.CleanPaths(opts.Fallbacks)
	if err != nil {
		return nil, fmt.Errorf("build definition fallback paths: %w", err)
	}
	ignoreFiles, err := source.CleanPaths(opts.IgnoreFiles)
	if err != nil {
		return nil, fmt.Errorf("build definition ignore-file paths: %w", err)
	}

	follow := append([]string(nil), opts.FollowPaths...)
	if opts.Filename != "" {
		clean, err := llbutil.CleanRelativePath(opts.Filename)
		if err != nil {
			return nil, fmt.Errorf("build definition filename: %w", err)
		}
		follow = append(follow, clean)
	}
	follow = append(follow, fallbacks...)
	follow = append(follow, ignoreFiles...)
	follow, err = source.CleanPaths(follow)
	if err != nil {
		return nil, fmt.Errorf("build definition follow paths: %w", err)
	}

	resolveOptions := opts.Options.Clone()
	resolveOptions.FollowPaths = follow
	resolveOptions.ExcludePatterns = nil
	resolved, err := source.Resolve(ctx, client, resolver, opts.Source, resolveOptions)
	if err != nil {
		return nil, fmt.Errorf("resolve build definition source: %w", err)
	}

	filename := opts.Filename
	if filename == "" {
		filename = resolved.Filename
	}
	if filename == "" {
		return nil, fmt.Errorf("build definition filename is required for source %q", opts.Source.String())
	}
	filename, err = llbutil.CleanRelativePath(filename)
	if err != nil {
		return nil, fmt.Errorf("build definition filename: %w", err)
	}

	solved, err := resolved.Solve(ctx, client)
	if err != nil {
		return nil, fmt.Errorf("load build definition source: %w", err)
	}
	data, usedFilename, err := source.ReadFirstFile(ctx, solved, append([]string{filename}, fallbacks...))
	if err != nil {
		return nil, fmt.Errorf("read build definition %q: %w", filename, err)
	}

	// Use BuildKit's source-map type directly, including the evaluated
	// definition so frontend diagnostics point back to the original source.
	sourceMap := llb.NewSourceMap(resolved.State, usedFilename, opts.Language, data)
	sourceMap.Definition = solved.Definition

	warn := func(ctx context.Context, message string, warnOpts gwclient.WarnOpts) {
		if warnOpts.Level == 0 {
			warnOpts.Level = 1
		}
		if warnOpts.SourceInfo == nil {
			warnOpts.SourceInfo = source.SourceInfo(sourceMap)
		}
		_ = client.Warn(ctx, solved.Vertex, message, warnOpts)
	}

	ignorePatterns := func(ctx context.Context) ([]string, error) {
		if len(ignoreFiles) == 0 {
			return nil, nil
		}
		patterns, _, err := ignore.ReadPatterns(ctx, solved, ignoreFiles, opts.Parser)
		if err != nil {
			return nil, fmt.Errorf("read definition ignore file: %w", err)
		}
		return patterns, nil
	}
	return NewSource(sourceMap, ignorePatterns, warn), nil
}
