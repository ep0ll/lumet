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

func (s *State) String() string          { return "<llb.State>" }
func (s *State) Type() string            { return "llb.State" }
func (s *State) Freeze()                 {}
func (s *State) Truth() starlark.Bool    { return true }
func (s *State) Hash() (uint32, error)   { return 0, fmt.Errorf("unhashable type: llb.State") }

func (s *State) AttrNames() []string {
	return []string{
		"dir",
		"add_env",
		"del_env",
		"reset",
		"user",
		"hostname",
		"run",
		"file",
		"copy",
		"with",
		// more will be added (platform, network, security, etc.)
	}
}

func (s *State) Attr(name string) (starlark.Value, error) {
	switch name {
	case "dir":
		return starlark.NewBuiltin("dir", s.dir), nil
	case "add_env":
		return starlark.NewBuiltin("add_env", s.addEnv), nil
	case "del_env":
		return starlark.NewBuiltin("del_env", s.delEnv), nil
	case "reset":
		return starlark.NewBuiltin("reset", s.reset), nil
	case "user":
		return starlark.NewBuiltin("user", s.user), nil
	case "hostname":
		return starlark.NewBuiltin("hostname", s.hostname), nil
	case "run":
		return starlark.NewBuiltin("run", s.run), nil
	case "file":
		return starlark.NewBuiltin("file", s.file), nil
	case "copy":
		return starlark.NewBuiltin("copy", s.copy), nil
	case "with":
		return starlark.NewBuiltin("with", s.with), nil
	default:
		return nil, nil
	}
}

func (s *State) dir(_ *starlark.Thread, fn *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	var path string
	if err := starlark.UnpackArgs(fn.Name(), args, kwargs, "path", &path); err != nil {
		return nil, err
	}
	return &State{state: s.state.Dir(path)}, nil
}

func (s *State) addEnv(_ *starlark.Thread, fn *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	var key, value string
	if err := starlark.UnpackArgs(fn.Name(), args, kwargs, "key", &key, "value", &value); err != nil {
		return nil, err
	}
	return &State{state: s.state.AddEnv(key, value)}, nil
}

func (s *State) delEnv(_ *starlark.Thread, fn *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	var key string
	if err := starlark.UnpackArgs(fn.Name(), args, kwargs, "key", &key); err != nil {
		return nil, err
	}
	return &State{state: s.state.DelEnv(key)}, nil
}

func (s *State) reset(_ *starlark.Thread, fn *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	if err := starlark.UnpackArgs(fn.Name(), args, kwargs); err != nil {
		return nil, err
	}
	return &State{state: s.state.Reset()}, nil
}

func (s *State) user(_ *starlark.Thread, fn *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	var name string
	if err := starlark.UnpackArgs(fn.Name(), args, kwargs, "name", &name); err != nil {
		return nil, err
	}
	return &State{state: s.state.User(name)}, nil
}

func (s *State) hostname(_ *starlark.Thread, fn *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	var name string
	if err := starlark.UnpackArgs(fn.Name(), args, kwargs, "name", &name); err != nil {
		return nil, err
	}
	return &State{state: s.state.Hostname(name)}, nil
}

func (s *State) run(_ *starlark.Thread, fn *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	var cmd starlark.Value
	if err := starlark.UnpackArgs(fn.Name(), args, kwargs, "cmd", &cmd); err != nil {
		return nil, err
	}

	var runOpts []llb.RunOption

	switch v := cmd.(type) {
	case starlark.String:
		runOpts = append(runOpts, llb.Args([]string{"/bin/sh", "-c", string(v)}))
	case *starlark.List:
		argv := make([]string, 0, v.Len())
		iter := v.Iterate()
		defer iter.Done()
		var elem starlark.Value
		for iter.Next(&elem) {
			s, ok := elem.(starlark.String)
			if !ok {
				return nil, fmt.Errorf("run: expected list of strings")
			}
			argv = append(argv, string(s))
		}
		runOpts = append(runOpts, llb.Args(argv))
	default:
		return nil, fmt.Errorf("run: expected string or list of strings, got %s", cmd.Type())
	}

	return &ExecState{exec: s.state.Run(runOpts...)}, nil
}

func (s *State) file(_ *starlark.Thread, fn *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	// Expect a FileAction (from fileop.go)
	var action starlark.Value
	if err := starlark.UnpackArgs(fn.Name(), args, kwargs, "action", &action); err != nil {
		return nil, err
	}
	fa, ok := action.(*FileAction)
	if !ok {
		return nil, fmt.Errorf("file: expected FileAction")
	}
	return &State{state: s.state.File(fa.action)}, nil
}

func (s *State) copy(_ *starlark.Thread, fn *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	var src starlark.Value
	var dest string
	if err := starlark.UnpackArgs(fn.Name(), args, kwargs, "src", &src, "dest", &dest); err != nil {
		return nil, err
	}

	srcState, err := AsState(src)
	if err != nil {
		return nil, err
	}

	// Simple copy of whole state for now. Full Copy API is richer.
	return &State{state: s.state.File(llb.Copy(srcState, "/", dest))}, nil
}

func (s *State) with(_ *starlark.Thread, fn *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	// Accept a list of StateOptions or individual options later.
	// For now this is a placeholder that returns self.
	return s, nil
}

// Underlying returns the real llb.State.
func (s *State) Underlying() llb.State {
	return s.state
}
