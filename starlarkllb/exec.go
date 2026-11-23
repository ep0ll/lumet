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

func (e *ExecState) String() string        { return "<llb.ExecState>" }
func (e *ExecState) Type() string          { return "llb.ExecState" }
func (e *ExecState) Freeze()               {}
func (e *ExecState) Truth() starlark.Bool  { return true }
func (e *ExecState) Hash() (uint32, error) { return 0, fmt.Errorf("unhashable type: llb.ExecState") }

func (e *ExecState) AttrNames() []string {
	return []string{
		"root",
		"add_mount",
		"get_mount",
	}
}

func (e *ExecState) Attr(name string) (starlark.Value, error) {
	switch name {
	case "root":
		return starlark.NewBuiltin("root", e.root), nil
	case "add_mount":
		return starlark.NewBuiltin("add_mount", e.addMount), nil
	case "get_mount":
		return starlark.NewBuiltin("get_mount", e.getMount), nil
	default:
		return nil, nil
	}
}

func (e *ExecState) root(_ *starlark.Thread, fn *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	if err := starlark.UnpackArgs(fn.Name(), args, kwargs); err != nil {
		return nil, err
	}
	return &State{state: e.exec.Root()}, nil
}

func (e *ExecState) addMount(_ *starlark.Thread, fn *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	var target string
	var source starlark.Value
	if err := starlark.UnpackArgs(fn.Name(), args, kwargs, "target", &target, "source", &source); err != nil {
		return nil, err
	}

	srcState, err := AsState(source)
	if err != nil {
		return nil, err
	}

	// TODO: MountOption support (Readonly, SourcePath, AsPersistentCacheDir, Tmpfs, ...)
	return &State{state: e.exec.AddMount(target, srcState)}, nil
}

func (e *ExecState) getMount(_ *starlark.Thread, fn *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	var target string
	if err := starlark.UnpackArgs(fn.Name(), args, kwargs, "target", &target); err != nil {
		return nil, err
	}
	return &State{state: e.exec.GetMount(target)}, nil
}

func (e *ExecState) Underlying() llb.ExecState {
	return e.exec
}
