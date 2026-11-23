package starlarkllb

import (
	"fmt"

	"github.com/moby/buildkit/client/llb"
	"go.starlark.net/starlark"
)

// mergeBuiltin implements llb.merge(states...)
func mergeBuiltin(_ *starlark.Thread, fn *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	if len(kwargs) > 0 {
		return nil, fmt.Errorf("%s: unexpected keyword arguments", fn.Name())
	}
	if args.Len() == 0 {
		return nil, fmt.Errorf("%s: at least one state required", fn.Name())
	}

	states := make([]llb.State, 0, args.Len())
	for i := 0; i < args.Len(); i++ {
		st, err := AsState(args.Index(i))
		if err != nil {
			return nil, err
		}
		states = append(states, st)
	}

	return &State{state: llb.Merge(states)}, nil
}

// diffBuiltin implements llb.diff(lower, upper)
func diffBuiltin(_ *starlark.Thread, fn *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	var lower, upper starlark.Value
	if err := starlark.UnpackArgs(fn.Name(), args, kwargs, "lower", &lower, "upper", &upper); err != nil {
		return nil, err
	}

	low, err := AsState(lower)
	if err != nil {
		return nil, err
	}
	high, err := AsState(upper)
	if err != nil {
		return nil, err
	}

	return &State{state: llb.Diff(low, high)}, nil
}
