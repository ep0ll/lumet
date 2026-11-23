package llb

import (
	"fmt"
	"os"
	"time"

	buildkitllb "github.com/moby/buildkit/client/llb"
	"go.starlark.net/starlark"
)

// FileActionValue wraps BuildKit's persistent, chainable FileAction value.
type FileActionValue struct {
	action *buildkitllb.FileAction
}

// WrapFileAction adapts a native BuildKit FileAction.
func WrapFileAction(action *buildkitllb.FileAction) *FileActionValue {
	return &FileActionValue{action: action}
}

// Unwrap returns the native FileAction.
func (v *FileActionValue) Unwrap() *buildkitllb.FileAction {
	if v == nil { return nil }
	return v.action
}

func (v *FileActionValue) String() string        { return "<llb.FileAction>" }
func (v *FileActionValue) Type() string          { return "llb.FileAction" }
func (v *FileActionValue) Freeze()               {}
func (v *FileActionValue) Truth() starlark.Bool  { return starlark.Bool(v != nil && v.action != nil) }
func (v *FileActionValue) Hash() (uint32, error) { return 0, fmt.Errorf("unhashable: llb.FileAction") }

func (v *FileActionValue) Attr(name string) (starlark.Value, error) {
	if v == nil || v.action == nil { return nil, fmt.Errorf("llb.FileAction: nil receiver") }
	action := v.action
	switch name {
	case "Mkdir", "mkdir":
		return boundBuiltin("FileAction.Mkdir", func(_ *starlark.Thread, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
			if err := arity("FileAction.Mkdir", args, kwargs, 2, -1); err != nil { return nil, err }
			path, err := stringAt("FileAction.Mkdir", args, 0); if err != nil { return nil, err }
			mode, err := intAt("FileAction.Mkdir", args, 1); if err != nil { return nil, err }
			options, err := optionsFromArgs[buildkitllb.MkdirOption](args, 2, "MkdirOption")
			if err != nil { return nil, err }
			return WrapFileAction(action.Mkdir(path, os.FileMode(mode), options...)), nil
		}), nil
	case "Mkfile", "mkfile":
		return boundBuiltin("FileAction.Mkfile", func(_ *starlark.Thread, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
			if err := arity("FileAction.Mkfile", args, kwargs, 3, -1); err != nil { return nil, err }
			path, err := stringAt("FileAction.Mkfile", args, 0); if err != nil { return nil, err }
			mode, err := intAt("FileAction.Mkfile", args, 1); if err != nil { return nil, err }
			data, err := bytesAt("FileAction.Mkfile", args, 2); if err != nil { return nil, err }
			options, err := optionsFromArgs[buildkitllb.MkfileOption](args, 3, "MkfileOption")
			if err != nil { return nil, err }
			return WrapFileAction(action.Mkfile(path, os.FileMode(mode), data, options...)), nil
		}), nil
	case "Symlink", "symlink":
		return boundBuiltin("FileAction.Symlink", func(_ *starlark.Thread, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
			if err := arity("FileAction.Symlink", args, kwargs, 2, -1); err != nil { return nil, err }
			oldPath, err := stringAt("FileAction.Symlink", args, 0); if err != nil { return nil, err }
			newPath, err := stringAt("FileAction.Symlink", args, 1); if err != nil { return nil, err }
			options, err := optionsFromArgs[buildkitllb.SymlinkOption](args, 2, "SymlinkOption")
			if err != nil { return nil, err }
			return WrapFileAction(action.Symlink(oldPath, newPath, options...)), nil
		}), nil
	case "Rm", "rm":
		return boundBuiltin("FileAction.Rm", func(_ *starlark.Thread, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
			if err := arity("FileAction.Rm", args, kwargs, 1, -1); err != nil { return nil, err }
			path, err := stringAt("FileAction.Rm", args, 0); if err != nil { return nil, err }
			options, err := optionsFromArgs[buildkitllb.RmOption](args, 1, "RmOption")
			if err != nil { return nil, err }
			return WrapFileAction(action.Rm(path, options...)), nil
		}), nil
	case "Copy", "copy":
		return boundBuiltin("FileAction.Copy", func(_ *starlark.Thread, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
			if err := arity("FileAction.Copy", args, kwargs, 3, -1); err != nil { return nil, err }
			input, err := copyInputFromValue(args[0]); if err != nil { return nil, err }
			src, err := stringAt("FileAction.Copy", args, 1); if err != nil { return nil, err }
			dest, err := stringAt("FileAction.Copy", args, 2); if err != nil { return nil, err }
			options, err := optionsFromArgs[buildkitllb.CopyOption](args, 3, "CopyOption")
			if err != nil { return nil, err }
			return WrapFileAction(action.Copy(input, src, dest, options...)), nil
		}), nil
	case "WithState", "with_state":
		return boundBuiltin("FileAction.WithState", func(_ *starlark.Thread, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
			if err := arity("FileAction.WithState", args, kwargs, 1, 1); err != nil { return nil, err }
			state, err := stateFromValue(args[0]); if err != nil { return nil, err }
			return wrapCopyInput(action.WithState(state)), nil
		}), nil
	default:
		return nil, nil
	}
}

func (v *FileActionValue) AttrNames() []string {
	return []string{"Copy", "Mkdir", "Mkfile", "Rm", "Symlink", "WithState",
		"copy", "mkdir", "mkfile", "rm", "symlink", "with_state"}
}

// CopyInputValue carries BuildKit's sealed CopyInput interface. A Starlark
// value can only be made by binding a FileAction to a State.
type CopyInputValue struct {
	input buildkitllb.CopyInput
}

// WrapCopyInput adapts a native BuildKit CopyInput.
func WrapCopyInput(input buildkitllb.CopyInput) *CopyInputValue {
	return &CopyInputValue{input: input}
}

// Unwrap returns the native CopyInput.
func (v *CopyInputValue) Unwrap() buildkitllb.CopyInput {
	if v == nil { return nil }
	return v.input
}

func (v *CopyInputValue) String() string        { return "<llb.CopyInput>" }
func (v *CopyInputValue) Type() string          { return "llb.CopyInput" }
func (v *CopyInputValue) Freeze()               {}
func (v *CopyInputValue) Truth() starlark.Bool  { return starlark.Bool(v != nil && v.input != nil) }
func (v *CopyInputValue) Hash() (uint32, error) { return 0, fmt.Errorf("unhashable: llb.CopyInput") }

func wrapCopyInput(input buildkitllb.CopyInput) starlark.Value {
	if input == nil { return starlark.None }
	return WrapCopyInput(input)
}

// FileOpValue wraps BuildKit's file operation vertex.
type FileOpValue struct {
	op *buildkitllb.FileOp
}

// WrapFileOp adapts a native FileOp.
func WrapFileOp(op *buildkitllb.FileOp) *FileOpValue {
	return &FileOpValue{op: op}
}

// Unwrap returns the native FileOp.
func (v *FileOpValue) Unwrap() *buildkitllb.FileOp {
	if v == nil { return nil }
	return v.op
}

func (v *FileOpValue) String() string        { return "<llb.FileOp>" }
func (v *FileOpValue) Type() string          { return "llb.FileOp" }
func (v *FileOpValue) Freeze()               {}
func (v *FileOpValue) Truth() starlark.Bool  { return starlark.Bool(v != nil && v.op != nil) }
func (v *FileOpValue) Hash() (uint32, error) { return 0, fmt.Errorf("unhashable: llb.FileOp") }

func (v *FileOpValue) Attr(name string) (starlark.Value, error) {
	if v == nil || v.op == nil { return nil, fmt.Errorf("llb.FileOp: nil receiver") }
	switch name {
	case "Output", "output":
		return boundBuiltin("FileOp.Output", func(_ *starlark.Thread, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
			if err := arity("FileOp.Output", args, kwargs, 0, 0); err != nil { return nil, err }
			return WrapOutput(v.op.Output()), nil
		}), nil
	case "Inputs", "inputs":
		return boundBuiltin("FileOp.Inputs", func(_ *starlark.Thread, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
			if err := arity("FileOp.Inputs", args, kwargs, 0, 0); err != nil { return nil, err }
			inputs := v.op.Inputs()
			values := make([]starlark.Value, len(inputs))
			for i, input := range inputs { values[i] = WrapOutput(input) }
			return starlark.NewList(values), nil
		}), nil
	case "Validate", "validate":
		return boundBuiltin("FileOp.Validate", func(thread *starlark.Thread, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
			if err := arity("FileOp.Validate", args, kwargs, 0, -1); err != nil { return nil, err }
			constraints, err := constraintsFromArgs(args, 0); if err != nil { return nil, err }
			if err := v.op.Validate(callContext(thread), constraints); err != nil { return nil, err }
			return starlark.None, nil
		}), nil
	case "Marshal", "marshal":
		return boundBuiltin("FileOp.Marshal", func(thread *starlark.Thread, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
			if err := arity("FileOp.Marshal", args, kwargs, 0, -1); err != nil { return nil, err }
			constraints, err := constraintsFromArgs(args, 0); if err != nil { return nil, err }
			dgst, data, metadata, sources, err := v.op.Marshal(callContext(thread), constraints)
			if err != nil { return nil, err }
			return marshalResult(dgst, data, metadata, sources), nil
		}), nil
	default:
		return nil, nil
	}
}

func (v *FileOpValue) AttrNames() []string {
	return []string{"Inputs", "Marshal", "Output", "Validate", "inputs", "marshal", "output", "validate"}
}

func stateFileAttr(v *StateValue, name string) (starlark.Value, bool, error) {
	if name != "File" && name != "file" { return nil, false, nil }
	return boundBuiltin("State.File", func(_ *starlark.Thread, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
		if err := arity("State.File", args, kwargs, 1, -1); err != nil { return nil, err }
		action, ok := args[0].(*FileActionValue)
		if !ok || action == nil || action.action == nil {
			return nil, fmt.Errorf("State.File: got %s, want llb.FileAction", valueType(args[0]))
		}
		options, err := optionsFromArgs[buildkitllb.ConstraintsOpt](args, 1, "ConstraintsOpt")
		if err != nil { return nil, err }
		return WrapState(v.state.File(action.action, options...)), nil
	}), true, nil
}

var stateFileAttrNames = []string{"File", "file"}

func fileOpBuiltins() map[string]starlark.Value {
	members := make(map[string]starlark.Value)
	groups := []map[string]starlark.Value{
		fileActionBuiltins(),
		fileOwnershipBuiltins(),
		removeOptionBuiltins(),
		copyOptionBuiltins(),
		createdTimeBuiltins(),
	}
	for _, group := range groups {
		for name, value := range group {
			if _, exists := members[name]; exists { panic("starlark/llb: duplicate file-operation builtin " + name) }
			members[name] = value
		}
	}
	members["NewFileOp"] = builtin("NewFileOp", newFileOpBuiltin)
	return members
}

func fileActionBuiltins() map[string]starlark.Value {
	return map[string]starlark.Value{
		"Mkdir": builtin("Mkdir", mkdirBuiltin),
		"Mkfile": builtin("Mkfile", mkfileBuiltin),
		"Symlink": builtin("Symlink", symlinkBuiltin),
		"Rm": builtin("Rm", rmBuiltin),
		"Copy": builtin("Copy", copyBuiltin),
	}
}

func fileOwnershipBuiltins() map[string]starlark.Value {
	return map[string]starlark.Value{
		"WithParents": boolOptionBuiltin("WithParents", "MkdirOption", buildkitllb.WithParents),
		"WithUser": stringOptionBuiltin("WithUser", "ChownOption", buildkitllb.WithUser),
		"WithUIDGID": builtin("WithUIDGID", withUIDGIDBuiltin),
		"Chmod": builtin("Chmod", chmodBuiltin),
		"Chown": builtin("Chown", chownBuiltin),
		"Filename": stringOptionBuiltin("Filename", "FileInfoOption", buildkitllb.Filename),
		"ChmodMode": intOptionBuiltin("ChmodMode", "CopyOption", func(mode int) buildkitllb.CopyOption {
			return buildkitllb.ChmodOpt{Mode: os.FileMode(mode)}
		}),
		"ChmodSymbolic": stringOptionBuiltin("ChmodSymbolic", "CopyOption", func(mode string) buildkitllb.CopyOption {
			return buildkitllb.ChmodOpt{ModeStr: mode}
		}),
	}
}

func removeOptionBuiltins() map[string]starlark.Value {
	return map[string]starlark.Value{
		"WithAllowNotFound": boolOptionBuiltin("WithAllowNotFound", "RmOption", buildkitllb.WithAllowNotFound),
		"WithAllowWildcard": boolOptionBuiltin("WithAllowWildcard", "RmOption", buildkitllb.WithAllowWildcard),
	}
}

func copyOptionBuiltins() map[string]starlark.Value {
	return map[string]starlark.Value{
		"WithExcludePatterns": optionBuiltin("WithExcludePatterns", "CopyOption", func(_ *starlark.Thread, args starlark.Tuple, kwargs []starlark.Tuple) (any, error) {
			if err := arity("WithExcludePatterns", args, kwargs, 1, 1); err != nil { return nil, err }
			patterns, err := stringList(args[0]); if err != nil { return nil, err }
			return buildkitllb.WithExcludePatterns(patterns), nil
		}),
	}
}

func createdTimeBuiltins() map[string]starlark.Value {
	create := func(_ *starlark.Thread, args starlark.Tuple, kwargs []starlark.Tuple) (any, error) {
		if err := arity("WithCreatedTime", args, kwargs, 1, 1); err != nil { return nil, err }
		raw, err := stringAt("WithCreatedTime", args, 0); if err != nil { return nil, err }
		created, err := time.Parse(time.RFC3339Nano, raw)
		if err != nil { return nil, fmt.Errorf("WithCreatedTime: %w", err) }
		return buildkitllb.WithCreatedTime(created), nil
	}
	value := optionBuiltin("WithCreatedTime", "CreatedTime", create)
	return map[string]starlark.Value{"CreatedTime": value, "WithCreatedTime": value}
}

func mkdirBuiltin(_ *starlark.Thread, _ *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	if err := arity("Mkdir", args, kwargs, 2, -1); err != nil { return nil, err }
	path, err := stringAt("Mkdir", args, 0); if err != nil { return nil, err }
	mode, err := intAt("Mkdir", args, 1); if err != nil { return nil, err }
	options, err := optionsFromArgs[buildkitllb.MkdirOption](args, 2, "MkdirOption")
	if err != nil { return nil, err }
	return WrapFileAction(buildkitllb.Mkdir(path, os.FileMode(mode), options...)), nil
}

func mkfileBuiltin(_ *starlark.Thread, _ *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	if err := arity("Mkfile", args, kwargs, 3, -1); err != nil { return nil, err }
	path, err := stringAt("Mkfile", args, 0); if err != nil { return nil, err }
	mode, err := intAt("Mkfile", args, 1); if err != nil { return nil, err }
	data, err := bytesAt("Mkfile", args, 2); if err != nil { return nil, err }
	options, err := optionsFromArgs[buildkitllb.MkfileOption](args, 3, "MkfileOption")
	if err != nil { return nil, err }
	return WrapFileAction(buildkitllb.Mkfile(path, os.FileMode(mode), data, options...)), nil
}

func symlinkBuiltin(_ *starlark.Thread, _ *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	if err := arity("Symlink", args, kwargs, 2, -1); err != nil { return nil, err }
	oldPath, err := stringAt("Symlink", args, 0); if err != nil { return nil, err }
	newPath, err := stringAt("Symlink", args, 1); if err != nil { return nil, err }
	options, err := optionsFromArgs[buildkitllb.SymlinkOption](args, 2, "SymlinkOption")
	if err != nil { return nil, err }
	return WrapFileAction(buildkitllb.Symlink(oldPath, newPath, options...)), nil
}

func rmBuiltin(_ *starlark.Thread, _ *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	if err := arity("Rm", args, kwargs, 1, -1); err != nil { return nil, err }
	path, err := stringAt("Rm", args, 0); if err != nil { return nil, err }
	options, err := optionsFromArgs[buildkitllb.RmOption](args, 1, "RmOption")
	if err != nil { return nil, err }
	return WrapFileAction(buildkitllb.Rm(path, options...)), nil
}

func copyBuiltin(_ *starlark.Thread, _ *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	if err := arity("Copy", args, kwargs, 3, -1); err != nil { return nil, err }
	input, err := copyInputFromValue(args[0]); if err != nil { return nil, err }
	src, err := stringAt("Copy", args, 1); if err != nil { return nil, err }
	dest, err := stringAt("Copy", args, 2); if err != nil { return nil, err }
	options, err := optionsFromArgs[buildkitllb.CopyOption](args, 3, "CopyOption")
	if err != nil { return nil, err }
	return WrapFileAction(buildkitllb.Copy(input, src, dest, options...)), nil
}

func withUIDGIDBuiltin(_ *starlark.Thread, _ *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	if err := arity("WithUIDGID", args, kwargs, 2, 2); err != nil { return nil, err }
	uid, err := intAt("WithUIDGID", args, 0); if err != nil { return nil, err }
	gid, err := intAt("WithUIDGID", args, 1); if err != nil { return nil, err }
	return wrapOption("ChownOption", buildkitllb.WithUIDGID(uid, gid)), nil
}

func chmodBuiltin(_ *starlark.Thread, _ *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	if err := arity("Chmod", args, kwargs, 1, 1); err != nil { return nil, err }
	mode, err := intAt("Chmod", args, 0); if err != nil { return nil, err }
	return wrapOption("FileInfoOption", buildkitllb.Chmod(os.FileMode(mode))), nil
}

func chownBuiltin(_ *starlark.Thread, _ *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	if err := arity("Chown", args, kwargs, 2, 2); err != nil { return nil, err }
	uid, err := intAt("Chown", args, 0); if err != nil { return nil, err }
	gid, err := intAt("Chown", args, 1); if err != nil { return nil, err }
	return wrapOption("FileInfoOption", buildkitllb.Chown(uid, gid)), nil
}

func newFileOpBuiltin(_ *starlark.Thread, _ *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	if err := arity("NewFileOp", args, kwargs, 2, -1); err != nil { return nil, err }
	state, err := stateFromValue(args[0]); if err != nil { return nil, err }
	action, ok := args[1].(*FileActionValue)
	if !ok || action == nil || action.action == nil { return nil, fmt.Errorf("NewFileOp: got %s, want llb.FileAction", valueType(args[1])) }
	constraints, err := constraintsFromArgs(args[2:len(args)], 0)
	if err != nil { return nil, err }
	return WrapFileOp(buildkitllb.NewFileOp(state, action.action, *constraints)), nil
}

func copyInputFromValue(value starlark.Value) (buildkitllb.CopyInput, error) {
	switch v := value.(type) {
	case *StateValue:
		if v != nil { return v.state, nil }
	case *CopyInputValue:
		if v != nil && v.input != nil { return v.input, nil }
	}
	return nil, fmt.Errorf("got %s, want llb.State or llb.CopyInput", valueType(value))
}

var (
	_ starlark.Value    = (*FileActionValue)(nil)
	_ starlark.HasAttrs = (*FileActionValue)(nil)
	_ starlark.Value    = (*CopyInputValue)(nil)
	_ starlark.Value    = (*FileOpValue)(nil)
	_ starlark.HasAttrs = (*FileOpValue)(nil)
)

