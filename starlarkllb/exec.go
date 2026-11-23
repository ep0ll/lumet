package starlarkllb

import (
	"fmt"

	"github.com/moby/buildkit/client/llb"
	"go.starlark.net/starlark"
)

// ExecState is a Starlark-visible wrapper around llb.ExecState.
type ExecState struct {
	exec llb.ExecState
}

var (
	_ starlark.Value    = (*ExecState)(nil)
	_ starlark.HasAttrs = (*ExecState)(nil)
)

func (e *ExecState) String() string { return "<llb.ExecState>" }
func (e *ExecState) Type() string   { return "llb.ExecState" }
func (e *ExecState) Freeze()        {}
func (e *ExecState) Truth() starlark.Bool { return true }
func (e *ExecState) Hash() (uint32, error) {
	return 0, fmt.Errorf("unhashable type: llb.ExecState")
}

func (e *ExecState) AttrNames() []string {
	return []string{
		"root",
		// future: add_mount, get_mount, etc.
	}
}

func (e *ExecState) Attr(name string) (starlark.Value, error) {
	switch name {
	case "root":
		return starlark.NewBuiltin("root", e.root), nil
	default:
		return nil, nil
	}
}

func (e *ExecState) root(thread *starlark.Thread, fn *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	if err := starlark.UnpackArgs(fn.Name(), args, kwargs); err != nil {
		return nil, err
	}
	return &State{state: e.exec.Root()}, nil
}

// Underlying returns the real llb.ExecState.
func (e *ExecState) Underlying() llb.ExecState {
	return e.exec
}
