// Package lumet contains the source-map abstraction used by Lumet frontends.
package lumet

import (
	"context"
	"slices"
	"sync"

	"github.com/moby/buildkit/client/llb"
	gwclient "github.com/moby/buildkit/frontend/gateway/client"
	"github.com/moby/buildkit/solver/pb"
)

// Source exposes source locations, source-map metadata, warnings, and the
// ignore patterns associated with the build definition.
type Source interface {
	IgnorePatterns(ctx context.Context) ([]string, error)
	Location(ranges []*pb.Range) llb.ConstraintsOpt
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

// NewSource creates a source-map wrapper. Ignore patterns are loaded lazily
// and cached only after a successful read, so a cancelled request can retry.
func NewSource(
	sourceMap *llb.SourceMap,
	ignoreFn func(context.Context) ([]string, error),
	warnFn func(context.Context, string, gwclient.WarnOpts),
) Source {
	return &source{sourceMap: sourceMap, ignoreFn: ignoreFn, warnFn: warnFn}
}

func (s *source) IgnorePatterns(ctx context.Context) ([]string, error) {
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

func (s *source) WarnSources(ctx context.Context, message string, opt gwclient.WarnOpts) {
	if s == nil || s.warnFn == nil {
		return
	}
	s.warnFn(ctx, message, opt)
}

func (s *source) Sources() *llb.SourceMap {
	if s == nil {
		return nil
	}
	return s.sourceMap
}
