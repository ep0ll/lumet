package starlarkllb

import (
	"fmt"

	"github.com/moby/buildkit/client/llb"
	"go.starlark.net/starlark"
)

// imageBuiltin implements llb.image(ref, **opts)
func imageBuiltin(thread *starlark.Thread, fn *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	var ref string
	if err := starlark.UnpackArgs(fn.Name(), args, kwargs, "ref", &ref); err != nil {
		return nil, err
	}

	// TODO: support ImageOption kwargs (platform, resolve_mode, etc.)
	st := llb.Image(ref)
	return &State{state: st}, nil
}

// scratchBuiltin implements llb.scratch()
func scratchBuiltin(thread *starlark.Thread, fn *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	if err := starlark.UnpackArgs(fn.Name(), args, kwargs); err != nil {
		return nil, err
	}
	return &State{state: llb.Scratch()}, nil
}

// shlexBuiltin is a convenience helper that turns a shell-like string
// into a list of arguments (using the same logic as llb.Shlex).
// For now we keep it simple; later we can call the real helper if needed.
func shlexBuiltin(thread *starlark.Thread, fn *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	var s string
	if err := starlark.UnpackArgs(fn.Name(), args, kwargs, "s", &s); err != nil {
		return nil, err
	}

	// Minimal implementation – for production use github.com/google/shlex
	// or the internal buildkit helper.
	// This is just a placeholder that returns a single-element list.
	list := starlark.NewList([]starlark.Value{starlark.String(s)})
	return list, nil
}

// Helper used by host code to turn a Starlark value back into llb.State.
func AsState(v starlark.Value) (llb.State, error) {
	s, ok := v.(*State)
	if !ok {
		return llb.State{}, fmt.Errorf("expected llb.State, got %s", v.Type())
	}
	return s.state, nil
}
