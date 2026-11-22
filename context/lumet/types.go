package lumet

import (
	"context"

	"github.com/moby/buildkit/client/llb"
	"github.com/moby/buildkit/frontend/gateway/client"
	"github.com/moby/buildkit/solver/pb"
)

type Source interface {
	IgnorePatterns(ctx context.Context) ([]string, error)
	Location(r []*pb.Range) llb.ConstraintsOpt
	WarnSources(context.Context, string, client.WarnOpts)
	Sources() *llb.SourceMap
}

func BuildDefinition() (Source, error) {
	return nil, nil
}
