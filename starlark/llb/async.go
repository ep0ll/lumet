package llb

import (
	"context"
	"fmt"
	"sync"

	buildkitllb "github.com/moby/buildkit/client/llb"
	"go.starlark.net/starlark"
)

// State.Async defers part of an LLB graph until BuildKit resolves the state.
// The Starlark callback receives the previous state and the resolution
// constraints, and must return an llb.State.
func stateAsyncAttr(v *StateValue, name string) (starlark.Value, bool, error) {
	switch name {
	case "Async", "async":
		return boundBuiltin("State.Async", func(thread *starlark.Thread, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
			if err := arity("State.Async", args, kwargs, 1, 1); err != nil {
				return nil, err
			}
			callback, ok := args[0].(starlark.Callable)
			if !ok {
				return nil, fmt.Errorf("State.Async: got %s, want callable", valueType(args[0]))
			}

			// BuildKit may resolve an asynchronous vertex while marshaling. A
			// Starlark interpreter thread must never be entered concurrently.
			var callMu sync.Mutex
			asyncState := v.state.Async(func(ctx context.Context, previous buildkitllb.State, constraints *buildkitllb.Constraints) (buildkitllb.State, error) {
				callMu.Lock()
				defer callMu.Unlock()

				oldContext := callContext(thread)
				thread.SetLocal("context", ctx)
				defer thread.SetLocal("context", oldContext)

				result, err := starlark.Call(
					thread,
					callback,
					starlark.Tuple{WrapState(previous), WrapConstraints(*constraints)},
					nil,
				)
				if err != nil {
					return buildkitllb.State{}, err
				}
				state, err := stateFromValue(result)
				if err != nil {
					return buildkitllb.State{}, fmt.Errorf("State.Async callback: %w", err)
				}
				return state, nil
			})
			return WrapState(asyncState), nil
		}), true, nil
	default:
		return nil, false, nil
	}
}

var stateAsyncAttrNames = []string{"Async", "async"}
