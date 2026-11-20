// Package definition provides frontend-neutral build-definition metadata.
package definition

import (
	"context"
	"slices"
	"sync"

	"github.com/moby/buildkit/client/llb"
	gwclient "github.com/moby/buildkit/frontend/gateway/client"
	"github.com/moby/buildkit/solver/pb"
)

// Source represents a loaded build definition and the diagnostics associated
// with it. IgnorePatterns is lazy because frontends do not always need a
// filtered build context.
type Source interface {
	IgnorePatterns(context.Context) ([]string, error)
	Location([]*pb.Range) llb.ConstraintsOpt
	WarnSources(context.Context, string, gwclient.WarnOpts)
	Sources() *llb.SourceMap
}

type source struct {
	sourceMap *llb.SourceMap
	ignoreFn  func(context.Context) ([]string, error)
	warnFn    func(context.Context, string, gwclient.WarnOpts)

	mu       sync.Mutex
	loaded   bool
	patterns []string
}

// NewSource creates a build-definition source. Ignore patterns are cached
// only after a successful read, allowing transient failures and cancellation
// to be retried by a later caller.
func NewSource(
	sourceMap *llb.SourceMap,
	ignoreFn func(context.Context) ([]string, error),
	warnFn func(context.Context, string, gwclient.WarnOpts),
) Source {
	return &source{
		sourceMap: sourceMap,
		ignoreFn:  ignoreFn,
		warnFn:   warnFn,
	}
}

func (s *source) IgnorePatterns(ctx context.Context) ([]string, error) {
	if s == nil {
		return nil, nil
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if s.loaded {
		return slices.Clone(s.patterns), nil
	}
	if s.ignoreFn == nil {
		s.loaded = true
		return nil, nil
	}

	patterns, err := s.ignoreFn(ctx)
	if err != nil {
		return nil, err
	}
	s.patterns = slices.Clone(patterns)
	s.loaded = true
	return slices.Clone(s.patterns), nil
}

func (s *source) Location(ranges []*pb.Range) llb.ConstraintsOpt {
	if s == nil || s.sourceMap == nil {
		return nil
	}
	return s.sourceMap.Location(ranges)
}

func (s *source) WarnSources(ctx context.Context, message string, opts gwclient.WarnOpts) {
	if s == nil || s.warnFn == nil {
		return
	}
	s.warnFn(ctx, message, opts)
}

func (s *source) Sources() *llb.SourceMap {
	if s == nil {
		return nil
	}
	return s.sourceMap
}
