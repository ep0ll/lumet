package builder

import (
	"context"
	"runtime"
	"runtime/debug"

	"github.com/dexnore/dexfile/client"
	gwclient "github.com/moby/buildkit/frontend/gateway/client"
	"github.com/pkg/errors"
)
func Build(ctx context.Context, c gwclient.Client) (_ *gwclient.Result, err error) {
	clnt, err := client.NewClient(c)
	if err != nil {
		return nil, err
	}

	if _, ok := clnt.BuildOpts().Opts["debug"]; ok {
		defer func() {
			r := recover()
			if r == nil {
				return
			}

			stack := debug.Stack()

			var perr error
			switch v := r.(type) {
			case runtime.Error:
				perr = errors.Errorf("runtime panic: %v", v)
			case error:
				perr = errors.Wrap(v, "panic")
			default:
				perr = errors.Errorf("panic: %v (type %T)", v, v)
			}

			perr = errors.Errorf("%v\n\ngoroutine stack:\n%s", perr, stack)

			if err != nil {
				err = errors.Wrapf(err, "while handling earlier error, %v", perr)
			} else {
				err = perr
			}
		}()
	}

	return nil, err
}
