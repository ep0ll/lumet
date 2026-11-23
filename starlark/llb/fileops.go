package llb

import (
	"fmt"
	"os"

	buildkitllb "github.com/moby/buildkit/client/llb"
	"go.starlark.net/starlark"
)

func fileBuiltins() map[string]starlark.Value {
	return map[string]starlark.Value{
		"Mkdir": builtin("Mkdir", mkdirBuiltin),
		"Mkfile": builtin("Mkfile", mkfileBuiltin),
		"Symlink": builtin("Symlink", symlinkBuiltin),
		"Rm": builtin("Rm", rmBuiltin),
		"Copy": builtin("Copy", copyBuiltin),
	}
}

func mkdirBuiltin(_ *starlark.Thread, _ *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	if err := arity("Mkdir", args, kwargs, 2, -1); err != nil { return nil, err }
	path, err := stringAt("Mkdir", args, 0); if err != nil { return nil, err }
	mode, err := intAt("Mkdir", args, 1); if err != nil { return nil, err }
	opts, err := optionsFromArgs[buildkitllb.MkdirOption](args, 2, "MkdirOption"); if err != nil { return nil, err }
	return &FileActionValue{action: buildkitllb.Mkdir(path, os.FileMode(mode), opts...)}, nil
}

func mkfileBuiltin(_ *starlark.Thread, _ *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	if err := arity("Mkfile", args, kwargs, 3, -1); err != nil { return nil, err }
	path, err := stringAt("Mkfile", args, 0); if err != nil { return nil, err }
	mode, err := intAt("Mkfile", args, 1); if err != nil { return nil, err }
	data, err := bytesAt("Mkfile", args, 2); if err != nil { return nil, err }
	opts, err := optionsFromArgs[buildkitllb.MkfileOption](args, 3, "MkfileOption"); if err != nil { return nil, err }
	return &FileActionValue{action: buildkitllb.Mkfile(path, os.FileMode(mode), data, opts...)}, nil
}

func symlinkBuiltin(_ *starlark.Thread, _ *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	if err := arity("Symlink", args, kwargs, 2, -1); err != nil { return nil, err }
	oldPath, err := stringAt("Symlink", args, 0); if err != nil { return nil, err }
	newPath, err := stringAt("Symlink", args, 1); if err != nil { return nil, err }
	opts, err := optionsFromArgs[buildkitllb.SymlinkOption](args, 2, "SymlinkOption"); if err != nil { return nil, err }
	return &FileActionValue{action: buildkitllb.Symlink(oldPath, newPath, opts...)}, nil
}

func rmBuiltin(_ *starlark.Thread, _ *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	if err := arity("Rm", args, kwargs, 1, -1); err != nil { return nil, err }
	path, err := stringAt("Rm", args, 0); if err != nil { return nil, err }
	opts, err := optionsFromArgs[buildkitllb.RmOption](args, 1, "RmOption"); if err != nil { return nil, err }
	return &FileActionValue{action: buildkitllb.Rm(path, opts...)}, nil
}

func copyBuiltin(_ *starlark.Thread, _ *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	if err := arity("Copy", args, kwargs, 3, -1); err != nil { return nil, err }
	input, err := copyInputFromValue(args[0]); if err != nil { return nil, err }
	src, err := stringAt("Copy", args, 1); if err != nil { return nil, err }
	dest, err := stringAt("Copy", args, 2); if err != nil { return nil, err }
	opts, err := optionsFromArgs[buildkitllb.CopyOption](args, 3, "CopyOption"); if err != nil { return nil, err }
	return &FileActionValue{action: buildkitllb.Copy(input, src, dest, opts...)}, nil
}

func bytesAt(name string, args starlark.Tuple, index int) ([]byte, error) {
	switch value := args[index].(type) {
	case starlark.Bytes:
		return []byte(value), nil
	case starlark.String:
		return []byte(string(value)), nil
	default:
		return nil, fmt.Errorf("%s: argument %d: got %s, want bytes or string", name, index+1, valueType(args[index]))
	}
}

func copyInputFromValue(value starlark.Value) (buildkitllb.CopyInput, error) {
	switch v := value.(type) {
	case *StateValue:
		if v == nil { break }
		return v.state, nil
	case *copyInputValue:
		if v == nil || v.input == nil { break }
		return v.input, nil
	default:
		return nil, fmt.Errorf("got %s, want llb.State or llb.CopyInput", valueType(value))
	}
	return nil, fmt.Errorf("invalid nil LLB copy input")
}

func (v *FileActionValue) Attr(name string) (starlark.Value, error) {
	if v == nil || v.action == nil { return nil, fmt.Errorf("llb.FileAction: nil receiver") }
	action := v.action
	switch name {
	case "Mkdir", "mkdir":
		return boundBuiltin("FileAction.Mkdir", func(_ *starlark.Thread, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
			if err := arity("FileAction.Mkdir", args, kwargs, 2, -1); err != nil { return nil, err }
			path, err := stringAt("FileAction.Mkdir", args, 0); if err != nil { return nil, err }
			mode, err := intAt("FileAction.Mkdir", args, 1); if err != nil { return nil, err }
			opts, err := optionsFromArgs[buildkitllb.MkdirOption](args, 2, "MkdirOption"); if err != nil { return nil, err }
			return &FileActionValue{action: action.Mkdir(path, os.FileMode(mode), opts...)}, nil
		}), nil
	case "Mkfile", "mkfile":
		return boundBuiltin("FileAction.Mkfile", func(_ *starlark.Thread, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
			if err := arity("FileAction.Mkfile", args, kwargs, 3, -1); err != nil { return nil, err }
			path, err := stringAt("FileAction.Mkfile", args, 0); if err != nil { return nil, err }
			mode, err := intAt("FileAction.Mkfile", args, 1); if err != nil { return nil, err }
			data, err := bytesAt("FileAction.Mkfile", args, 2); if err != nil { return nil, err }
			opts, err := optionsFromArgs[buildkitllb.MkfileOption](args, 3, "MkfileOption"); if err != nil { return nil, err }
			return &FileActionValue{action: action.Mkfile(path, os.FileMode(mode), data, opts...)}, nil
		}), nil
	case "Symlink", "symlink":
		return boundBuiltin("FileAction.Symlink", func(_ *starlark.Thread, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
			if err := arity("FileAction.Symlink", args, kwargs, 2, -1); err != nil { return nil, err }
			oldPath, err := stringAt("FileAction.Symlink", args, 0); if err != nil { return nil, err }
			newPath, err := stringAt("FileAction.Symlink", args, 1); if err != nil { return nil, err }
			opts, err := optionsFromArgs[buildkitllb.SymlinkOption](args, 2, "SymlinkOption"); if err != nil { return nil, err }
			return &FileActionValue{action: action.Symlink(oldPath, newPath, opts...)}, nil
		}), nil
	case "Rm", "rm":
		return boundBuiltin("FileAction.Rm", func(_ *starlark.Thread, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
			if err := arity("FileAction.Rm", args, kwargs, 1, -1); err != nil { return nil, err }
			path, err := stringAt("FileAction.Rm", args, 0); if err != nil { return nil, err }
			opts, err := optionsFromArgs[buildkitllb.RmOption](args, 1, "RmOption"); if err != nil { return nil, err }
			return &FileActionValue{action: action.Rm(path, opts...)}, nil
		}), nil
	case "Copy", "copy":
		return boundBuiltin("FileAction.Copy", func(_ *starlark.Thread, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
			if err := arity("FileAction.Copy", args, kwargs, 3, -1); err != nil { return nil, err }
			input, err := copyInputFromValue(args[0]); if err != nil { return nil, err }
			src, err := stringAt("FileAction.Copy", args, 1); if err != nil { return nil, err }
			dest, err := stringAt("FileAction.Copy", args, 2); if err != nil { return nil, err }
			opts, err := optionsFromArgs[buildkitllb.CopyOption](args, 3, "CopyOption"); if err != nil { return nil, err }
			return &FileActionValue{action: action.Copy(input, src, dest, opts...)}, nil
		}), nil
	case "WithState", "with_state":
		return boundBuiltin("FileAction.WithState", func(_ *starlark.Thread, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
			if err := arity("FileAction.WithState", args, kwargs, 1, 1); err != nil { return nil, err }
			state, err := stateFromValue(args[0]); if err != nil { return nil, err }
			return &copyInputValue{input: action.WithState(state)}, nil
		}), nil
	default:
		return nil, nil
	}
}

func (v *FileActionValue) AttrNames() []string {
	return []string{"Copy", "Mkdir", "Mkfile", "Rm", "Symlink", "WithState",
		"copy", "mkdir", "mkfile", "rm", "symlink", "with_state"}
}

// copyInputValue represents BuildKit's sealed llb.CopyInput interface. A value
// is created only by FileAction.WithState, avoiding impossible external
// implementations of that interface.
type copyInputValue struct{ input buildkitllb.CopyInput }

func (v *copyInputValue) String() string        { return "<llb.CopyInput>" }
func (v *copyInputValue) Type() string          { return "llb.CopyInput" }
func (v *copyInputValue) Freeze()               {}
func (v *copyInputValue) Truth() starlark.Bool  { return starlark.Bool(v != nil && v.input != nil) }
func (v *copyInputValue) Hash() (uint32, error) { return 0, fmt.Errorf("unhashable: llb.CopyInput") }

var _ starlark.HasAttrs = (*FileActionValue)(nil)
