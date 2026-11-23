package starlarkllb

import (
	"fmt"

	"github.com/moby/buildkit/client/llb"
	"go.starlark.net/starlark"
)

// State is a Starlark-visible wrapper around llb.State.
type State struct {
	state llb.State
}

var (
	_ starlark.Value    = (*State)(nil)
	_ starlark.HasAttrs = (*State)(nil)
)

func (s *State) String() string { return "<llb.State>" }
func (s *State) Type() string   { return "llb.State" }
func (s *State) Freeze()        {}
func (s *State) Truth() starlark.Bool { return true }
func (s *State) Hash() (uint32, error) {
	return 0, fmt.Errorf("unhashable type: llb.State")
}

// AttrNames returns the list of attributes exposed to Starlark.
func (s *State) AttrNames() []string {
	return []string{
		"dir",
		"add_env",
		"reset",
		"user",
		"run",
		// future: platform, with, file, copy, merge, etc.
	}
}

// Attr implements starlark.HasAttrs.
func (s *State) Attr(name string) (starlark.Value, error) {
	switch name {
	case "dir":
		return starlark.NewBuiltin("dir", s.dir), nil
	case "add_env":
		return starlark.NewBuiltin("add_env", s.addEnv), nil
	case "reset":
		return starlark.NewBuiltin("reset", s.reset), nil
	case "user":
		return starlark.NewBuiltin("user", s.user), nil
	case "run":
		return starlark.NewBuiltin("run", s.run), nil
	default:
		return nil, nil // attribute not found
	}
}

// --- method implementations -------------------------------------------------

func (s *State) dir(thread *starlark.Thread, fn *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	var path string
	if err := starlark.UnpackArgs(fn.Name(), args, kwargs, "path", &path); err != nil {
		return nil, err
	}
	return &State{state: s.state.Dir(path)}, nil
}

func (s *State) addEnv(thread *starlark.Thread, fn *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	var key, value string
	if err := starlark.UnpackArgs(fn.Name(), args, kwargs, "key", &key, "value", &value); err != nil {
		return nil, err
	}
	return &State{state: s.state.AddEnv(key, value)}, nil
}

func (s *State) reset(thread *starlark.Thread, fn *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	if err := starlark.UnpackArgs(fn.Name(), args, kwargs); err != nil {
		return nil, err
	}
	return &State{state: s.state.Reset()}, nil
}

func (s *State) user(thread *starlark.Thread, fn *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	var name string
	if err := starlark.UnpackArgs(fn.Name(), args, kwargs, "name", &name); err != nil {
		return nil, err
	}
	return &State{state: s.state.User(name)}, nil
}

func (s *State) run(thread *starlark.Thread, fn *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	// Accept either a single string (shlex style) or a list of strings,
	// plus optional keyword args later (ro, mounts, etc.).
	var cmd starlark.Value
	if err := starlark.UnpackArgs(fn.Name(), args, kwargs, "cmd", &cmd); err != nil {
		return nil, err
	}

	var runOpts []llb.RunOption

	switch v := cmd.(type) {
	case starlark.String:
		runOpts = append(runOpts, llb.Args([]string{"/bin/sh", "-c", string(v)}))
	case *starlark.List:
		args := make([]string, 0, v.Len())
		iter := v.Iterate()
		defer iter.Done()
		var elem starlark.Value
		for iter.Next(&elem) {
			s, ok := elem.(starlark.String)
			if !ok {
				return nil, fmt.Errorf("run: expected list of strings")
			}
			args = append(args, string(s))
		}
		runOpts = append(runOpts, llb.Args(args))
	default:
		return nil, fmt.Errorf("run: expected string or list of strings, got %s", cmd.Type())
	}

	exec := s.state.Run(runOpts...)
	return &ExecState{exec: exec}, nil
}

// Underlying returns the real llb.State (for host code).
func (s *State) Underlying() llb.State {
	return s.state
}
