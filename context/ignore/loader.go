package ignore

import (
	"context"
	"fmt"

	"github.com/ep0ll/lumet/context/llbutil"
	"github.com/ep0ll/lumet/context/source"
	"github.com/ep0ll/lumet/frontend"
)

// BuildKitLoader loads ignore files from an independent BuildKit source or
// delegates to an already-loaded definition.
type BuildKitLoader struct {
	Resolver source.Resolver
}

// Load implements Loader.
func (l BuildKitLoader) Load(ctx context.Context, client frontend.Client, opts Options) ([]string, error) {
	if opts.Definition != nil {
		patterns, err := opts.Definition.IgnorePatterns(ctx)
		if err != nil {
			return nil, fmt.Errorf("load definition ignore patterns: %w", err)
		}
		return patterns, nil
	}
	if err := opts.Source.Validate(); err != nil {
		return nil, fmt.Errorf("ignore source: %w", err)
	}
	files := opts.Files
	if len(files) == 0 && opts.Source.Kind == source.KindLocal {
		files = []string{".dockerignore"}
	}
	files, err := cleanPaths(files)
	if err != nil {
		return nil, fmt.Errorf("ignore-file paths: %w", err)
	}

	resolver := opts.Resolver
	if resolver == nil {
		resolver = l.Resolver
	}
	if resolver == nil {
		resolver = source.DefaultResolver()
	}

	followPaths := append(append([]string(nil), opts.FollowPaths...), files...)
	followPaths, err = cleanPaths(followPaths)
	if err != nil {
		return nil, fmt.Errorf("ignore source follow paths: %w", err)
	}

	resolveOptions := opts.Options.Clone()
	resolveOptions.FollowPaths = followPaths
	resolveOptions.ExcludePatterns = nil
	resolved, err := resolver.Resolve(ctx, client, opts.Source, resolveOptions)
	if err != nil {
		return nil, fmt.Errorf("resolve ignore-file source: %w", err)
	}
	if len(files) == 0 && resolved.Filename != "" {
		files, err = cleanPaths([]string{resolved.Filename})
		if err != nil {
			return nil, fmt.Errorf("resolved ignore-file path: %w", err)
		}
	}
	if len(files) == 0 {
		return nil, fmt.Errorf("ignore-file path is required for source %q", opts.Source.String())
	}

	solved, err := resolved.Solve(ctx, client)
	if err != nil {
		return nil, fmt.Errorf("load ignore-file source: %w", err)
	}
	patterns, _, err := ReadPatterns(ctx, solved, files, opts.Parser)
	if err != nil {
		return nil, fmt.Errorf("read ignore files: %w", err)
	}
	return patterns, nil
}

// ReadPatterns reads and parses the first existing ignore file. Missing
// candidates are optional; other read and parse failures are returned.
func ReadPatterns(ctx context.Context, solved *source.Solved, files []string, parser Parser) ([]string, string, error) {
	for _, filename := range files {
		data, err := solved.ReadFile(ctx, filename)
		if llbutil.IsNotFound(err) {
			continue
		}
		if err != nil {
			return nil, "", fmt.Errorf("read ignore file %q: %w", filename, err)
		}
		patterns, err := Parse(parser, filename, data)
		if err != nil {
			return nil, "", fmt.Errorf("parse ignore file %q: %w", filename, err)
		}
		return patterns, filename, nil
	}
	return nil, "", nil
}

// Parse parses an ignore file with the supplied parser or Moby's default.
func Parse(parser Parser, filename string, data []byte) ([]string, error) {
	return llbutil.ParseIgnoreFile(parser, filename, data)
}

func cleanPaths(paths []string) ([]string, error) {
	return llbutil.CleanRelativePaths(paths)
}
