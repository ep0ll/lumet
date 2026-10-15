package solver

import (
	"context"

	"github.com/dexnore/dexfile"
	gwclient "github.com/moby/buildkit/frontend/gateway/client"
)

type Solver struct {
	client dexfile.Client
}

func (s *Solver) Solve(ctx context.Context, client dexfile.Client) (res *gwclient.Result, err error) {
	return nil, nil
}
