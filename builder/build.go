package builder

import (
	"context"

	"github.com/ep0ll/lumet/frontend"
	gwclient "github.com/moby/buildkit/frontend/gateway/client"
)

func Build(ctx context.Context, c gwclient.Client) (_ *gwclient.Result, err error) {
	clnt, err := frontend.NewClient(c)
	if err != nil {
		return nil, err
	}

	var debug bool
	defer func() {
		if debug {
			RecoverFromPanic(&err)
		}
	}()

	_, debug = clnt.BuildOpts().Opts["debug"]

	return nil, err
}
