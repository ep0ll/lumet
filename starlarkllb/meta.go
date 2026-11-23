package starlarkllb

import (
	"go.starlark.net/starlark"

	"github.com/moby/buildkit/client/llb"
)

// meta-related helpers and option builders will live here.
// Currently we expose the most common constraint helpers as module-level functions.

func withCustomNameBuiltin(_ *starlark.Thread, fn *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	var name string
	if err := starlark.UnpackArgs(fn.Name(), args, kwargs, "name", &name); err != nil {
		return nil, err
	}
	// Return a callable that can later be applied, or just document that
	// constraints are applied via State methods for now.
	// Full ConstraintsOpt support requires a richer option object system.
	_ = llb.WithCustomName(name)
	return starlark.None, nil
}

// Additional meta helpers (ProgressGroup, Platform, Require, etc.) can be
// added following the same pattern once the option object system is richer.
