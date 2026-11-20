package maincontext

import (
	"context"
	"fmt"
	"strings"

	"github.com/ep0ll/lumet/context/llbutil"
	"github.com/ep0ll/lumet/context/source"
	"github.com/ep0ll/lumet/frontend"
	"github.com/moby/buildkit/client/llb"
)

// BuildKitLoader resolves one source and applies optional scoping and
// exclusions. It never loads or interprets an ignore file itself.
type BuildKitLoader struct {
	Resolver source.Resolver
}

// Load implements Loader.
func (l BuildKitLoader) Load(ctx context.Context, client frontend.Client, opts Options) (*llb.State, error) {
	spec := opts.Source
	if spec.Kind == "" {
		spec = source.Local("context")
	}
	if err := spec.Validate(); err != nil {
		return nil, fmt.Errorf("main context source: %w", err)
	}
	subDir, err := cleanSubDir(opts.SubDir)
	if err != nil {
		return nil, fmt.Errorf("main context subdirectory: %w", err)
	}

		excludes := append([]string(nil), opts.ExcludePatterns...)
	if opts.ApplyIgnorePatterns {
		excludes = append(excludes, opts.IgnorePatterns...)
	}

	followPaths, err := source.CleanPaths(opts.FollowPaths)
	if err != nil {
		return nil, fmt.Errorf("main context follow paths: %w", err)
	}
	resolveOptions := opts.Options.Clone()
	resolveOptions.FollowPaths = followPaths
	resolveOptions.ExcludePatterns = nil
	if spec.Kind == source.KindLocal {
		resolveOptions.ExcludePatterns = excludes
	}
	resolved, err := source.ResolveUsing(ctx, client, opts.Resolver, l.Resolver, spec, resolveOptions)
	if err != nil {
		return nil, fmt.Errorf("resolve main context: %w", err)
	}
	state := *resolved.State

	// Apply exclusions against the original context root before scoping it.
	// Local sources receive exclusions directly on the lazy local output;
	// other source kinds use a root copy operation.
	if len(excludes) != 0 && spec.Kind != source.KindLocal {
		filtered := llb.Scratch().File(llb.Copy(state, "/", "/", &llb.CopyInfo{
			CopyDirContentsOnly: true,
			ExcludePatterns:     excludes,
		}))
		state = filtered
	}

	// Scoping is deliberately a separate operation so exclusions retain the
	// same root-relative semantics regardless of the selected subdirectory.
	if subDir != "" {
		scoped := llb.Scratch().File(llb.Copy(state, "/"+subDir, "/", &llb.CopyInfo{
			CopyDirContentsOnly: true,
		}))
		state = scoped
	}

	return &state, nil
}

func cleanSubDir(value string) (string, error) {
	value = strings.TrimSpace(value)
	if value == "" || value == "." {
		return "", nil
	}
	return llbutil.CleanRelativePath(value)
}
