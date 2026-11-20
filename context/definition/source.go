package definition

import (
	"context"
	"fmt"
	"slices"
	"sync"

	"github.com/moby/buildkit/client/llb"
	gwclient "github.com/moby/buildkit/frontend/gateway/client"
	"github.com/moby/buildkit/solver/pb"
)

// source is immutable except for its lazily loaded, memoized ignore patterns.
type source struct {
	sourceMap *llb.SourceMap
	filename  string
	data      []byte
	ignoreFn  func(context.Context) ([]string, error)
	warnFn    func(context.Context, string, gwclient.WarnOpts)

	mu       sync.Mutex
	loaded   bool
	patterns []string
}

// NewSource constructs a definition source. A nil ignoreFn means this
// definition has no associated ignore-file convention.
func NewSource(
	sourceMap *llb.SourceMap,
	ignoreFn func(context.Context) ([]string, error),
	warnFn func(context.Context, string, gwclient.WarnOpts),
) Source {
	var filename string
	var data []byte
	if sourceMap != nil {
		filename = sourceMap.Filename
		data = slices.Clone(sourceMap.Data)
	}
	return &source{
		sourceMap: sourceMap,
		filename:  filename,
		data:      data,
		ignoreFn:  ignoreFn,
		warnFn:    warnFn,
	}
}

func (s *source) Data() []byte {
	if s == nil {
		return nil
	}
	return slices.Clone(s.data)
}

func (s *source) Filename() string {
	if s == nil {
		return ""
	}
	return s.filename
}

func (s *source) SourceMap() *llb.SourceMap {
	if s == nil {
		return nil
	}
	return s.sourceMap
}

// Sources implements the historical spelling used by context/lumet.
func (s *source) Sources() *llb.SourceMap { return s.SourceMap() }

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

// IgnorePatterns loads associated patterns once. Failed loads are not cached,
// so a transient gateway or context failure can be retried.
func (s *source) IgnorePatterns(ctx context.Context) ([]string, error) {
	if s == nil {
		return nil, fmt.Errorf("definition source is nil")
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
