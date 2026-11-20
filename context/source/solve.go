package source

import (
	"context"
	"fmt"

	"github.com/ep0ll/lumet/context/llbutil"
	"github.com/ep0ll/lumet/frontend"
	"github.com/moby/buildkit/client/llb"
	gwclient "github.com/moby/buildkit/frontend/gateway/client"
	"github.com/moby/buildkit/solver/pb"
	"github.com/opencontainers/go-digest"
)

// Solved is an evaluated source reference plus the LLB definition used to
// produce it. It can be reused to read several files without solving again.
type Solved struct {
	Definition *llb.Definition
	Reference  gwclient.Reference
	Vertex     digest.Digest
}

// Solve evaluates the resolved state once and retains its reference.
func (r *Resolved) Solve(ctx context.Context, client frontend.Client) (*Solved, error) {
	if r == nil || r.State == nil {
		return nil, fmt.Errorf("resolved source has no LLB state")
	}

	r.mu.Lock()
	defer r.mu.Unlock()
	if r.solved != nil {
		return r.solved, nil
	}

	definition, err := r.State.Marshal(ctx, llbutil.MarshalOpts(client.BuildOpts())...)
	if err != nil {
		return nil, fmt.Errorf("marshal source state: %w", err)
	}
	vertex, err := definition.Head()
	if err != nil {
		return nil, fmt.Errorf("get source vertex: %w", err)
	}
	result, err := client.Solve(ctx, gwclient.SolveRequest{Definition: definition.ToPB()})
	if err != nil {
		return nil, fmt.Errorf("solve source state: %w", err)
	}
	ref, err := result.SingleRef()
	if err != nil {
		return nil, fmt.Errorf("get source reference: %w", err)
	}
	r.solved = &Solved{Definition: definition, Reference: ref, Vertex: vertex}
	return r.solved, nil
}

// ReadFile reads a bounded file from the solved source.
func (s *Solved) ReadFile(ctx context.Context, filename string) ([]byte, error) {
	if s == nil || s.Reference == nil {
		return nil, fmt.Errorf("solved source reference is nil")
	}
	return llbutil.ReadFile(ctx, s.Reference, filename)
}

// Warn forwards a warning using this source's LLB vertex.
func (s *Solved) Warn(ctx context.Context, client frontend.Client, message string, opts gwclient.WarnOpts) error {
	if s == nil {
		return fmt.Errorf("solved source is nil")
	}
	return client.Warn(ctx, s.Vertex, message, opts)
}

// SourceInfo returns source-map metadata suitable for a BuildKit warning.
func SourceInfo(sourceMap *llb.SourceMap) *pb.SourceInfo {
	if sourceMap == nil {
		return nil
	}
	info := &pb.SourceInfo{
		Data:     sourceMap.Data,
		Filename: sourceMap.Filename,
		Language: sourceMap.Language,
	}
	if sourceMap.Definition != nil {
		info.Definition = sourceMap.Definition.ToPB()
	}
	return info
}
