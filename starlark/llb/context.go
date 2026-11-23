package llb

import (
	"context"

	"go.starlark.net/starlark"
)

// callContext retrieves the Go context attached to the current Starlark
// thread. If no context was supplied, it returns context.Background.
func callContext(thread *starlark.Thread) context.Context {
	if thread != nil {
		if ctx, ok := thread.Local("context").(context.Context); ok && ctx != nil {
			return ctx
		}
	}
	return context.Background()
}
