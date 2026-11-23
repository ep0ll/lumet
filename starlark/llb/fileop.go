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

func (v *FileActionValue) String() string        { return "<llb.file_action>" }
func (v *FileActionValue) Type() string          { return "llb.file_action" }
func (v *FileActionValue) Freeze()               {}
func (v *FileActionValue) Truth() starlark.Bool  { return starlark.Bool(v != nil && v.action != nil) }
func (v *FileActionValue) Hash() (uint32, error) { return 0, fmt.Errorf("unhashable: llb.file_action") }

func (v *FileActionValue) Attr(name string) (starlark.Value, error) {
		if v == nil || v.action == nil { return nil, fmt.Errorf("llb.file_action: nil receiver") }
	action := v.action
	switch name {
	case "mkdir":
		return boundBuiltin("file_action.mkdir", func(_ *starlark.Thread, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
			if err := arity("file_action.mkdir", args, kwargs, 2, -1); err != nil { return nil, err }
			path, err := stringAt("file_action.mkdir", args, 0); if err != nil { return nil, err }
			mode, err := intAt("file_action.mkdir", args, 1); if err != nil { return nil, err }
			options, err := optionsFromArgs[buildkitllb.MkdirOption](args, 2, "mkdir_option")
			if err != nil { return nil, err }
			return WrapFileAction(action.Mkdir(path, os.FileMode(mode), options...)), nil
		}), nil
	case "mkfile":
		return boundBuiltin("file_action.mkfile", func(_ *starlark.Thread, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
			if err := arity("file_action.mkfile", args, kwargs, 3, -1); err != nil { return nil, err }
			path, err := stringAt("file_action.mkfile", args, 0); if err != nil { return nil, err }
			mode, err := intAt("file_action.mkfile", args, 1); if err != nil { return nil, err }
			data, err := bytesAt("file_action.mkfile", args, 2); if err != nil { return nil, err }
			options, err := optionsFromArgs[buildkitllb.MkfileOption](args, 3, "mkfile_option")
			if err != nil { return nil, err }
			return WrapFileAction(action.Mkfile(path, os.FileMode(mode), data, options...)), nil
		}), nil
	case "symlink":
		return boundBuiltin("file_action.symlink", func(_ *starlark.Thread, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
			if err := arity("file_action.symlink", args, kwargs, 2, -1); err != nil { return nil, err }
			oldPath, err := stringAt("file_action.symlink", args, 0); if err != nil { return nil, err }
			newPath, err := stringAt("file_action.symlink", args, 1); if err != nil { return nil, err }
			options, err := optionsFromArgs[buildkitllb.SymlinkOption](args, 2, "symlink_option")
			if err != nil { return nil, err }
			return WrapFileAction(action.Symlink(oldPath, newPath, options...)), nil
		}), nil
	case "rm":
		return boundBuiltin("file_action.rm", func(_ *starlark.Thread, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
			if err := arity("file_action.rm", args, kwargs, 1, -1); err != nil { return nil, err }
			path, err := stringAt("file_action.rm", args, 0); if err != nil { return nil, err }
			options, err := optionsFromArgs[buildkitllb.RmOption](args, 1, "rm_option")
			if err != nil { return nil, err }
			return WrapFileAction(action.Rm(path, options...)), nil
		}), nil
	case "copy":
		return boundBuiltin("file_action.copy", func(_ *starlark.Thread, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
			if err := arity("file_action.copy", args, kwargs, 3, -1); err != nil { return nil, err }
			input, err := copyInputFromValue(args[0]); if err != nil { return nil, err }
			src, err := stringAt("file_action.copy", args, 1); if err != nil { return nil, err }
			dest, err := stringAt("file_action.copy", args, 2); if err != nil { return nil, err }
			options, err := optionsFromArgs[buildkitllb.CopyOption](args, 3, "copy_option")
			if err != nil { return nil, err }
			return WrapFileAction(action.Copy(input, src, dest, options...)), nil
		}), nil
	case "with_state":
		return boundBuiltin("file_action.with_state", func(_ *starlark.Thread, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
			if err := arity("file_action.with_state", args, kwargs, 1, 1); err != nil { return nil, err }
			state, err := stateFromValue(args[0]); if err != nil { return nil, err }
			return wrapCopyInput(action.WithState(state)), nil
		}), nil
	default:
		return nil, nil
	}
}

func (v *FileActionValue) AttrNames() []string {
	return SnakeCaseNames([]string{"copy", "mkdir", "mkfile", "rm", "symlink", "with_state",
		"copy", "mkdir", "mkfile", "rm", "symlink", "with_state"})
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

func (v *CopyInputValue) String() string        { return "<llb.copy_input>" }
func (v *CopyInputValue) Type() string          { return "llb.copy_input" }
func (v *CopyInputValue) Freeze()               {}
func (v *CopyInputValue) Truth() starlark.Bool  { return starlark.Bool(v != nil && v.input != nil) }
func (v *CopyInputValue) Hash() (uint32, error) { return 0, fmt.Errorf("unhashable: llb.copy_input") }

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

func (v *FileOpValue) String() string        { return "<llb.file_op>" }
func (v *FileOpValue) Type() string          { return "llb.file_op" }
func (v *FileOpValue) Freeze()               {}
func (v *FileOpValue) Truth() starlark.Bool  { return starlark.Bool(v != nil && v.op != nil) }
func (v *FileOpValue) Hash() (uint32, error) { return 0, fmt.Errorf("unhashable: llb.file_op") }

func (v *FileOpValue) Attr(name string) (starlark.Value, error) {
		if v == nil || v.op == nil { return nil, fmt.Errorf("llb.file_op: nil receiver") }
	switch name {
	case "output":
		return boundBuiltin("file_op.output", func(_ *starlark.Thread, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
			if err := arity("file_op.output", args, kwargs, 0, 0); err != nil { return nil, err }
			return WrapOutput(v.op.Output()), nil
		}), nil
	case "inputs":
		return boundBuiltin("file_op.inputs", func(_ *starlark.Thread, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
			if err := arity("file_op.inputs", args, kwargs, 0, 0); err != nil { return nil, err }
			inputs := v.op.Inputs()
			values := make([]starlark.Value, len(inputs))
			for i, input := range inputs { values[i] = WrapOutput(input) }
			return starlark.NewList(values), nil
		}), nil
	case "validate":
		return boundBuiltin("file_op.validate", func(thread *starlark.Thread, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
			if err := arity("file_op.validate", args, kwargs, 0, -1); err != nil { return nil, err }
			constraints, err := constraintsFromArgs(args, 0); if err != nil { return nil, err }
			if err := v.op.Validate(callContext(thread), constraints); err != nil { return nil, err }
			return starlark.None, nil
		}), nil
	case "marshal":
		return boundBuiltin("file_op.marshal", func(thread *starlark.Thread, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
			if err := arity("file_op.marshal", args, kwargs, 0, -1); err != nil { return nil, err }
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
	return SnakeCaseNames([]string{"inputs", "marshal", "output", "validate", "inputs", "marshal", "output", "validate"})
}

func stateFileAttr(v *StateValue, name string) (starlark.Value, bool, error) {
	if name != "file" && name != "file" { return nil, false, nil }
	return boundBuiltin("state.file", func(_ *starlark.Thread, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
		if err := arity("state.file", args, kwargs, 1, -1); err != nil { return nil, err }
		action, ok := args[0].(*FileActionValue)
		if !ok || action == nil || action.action == nil {
			return nil, fmt.Errorf("state.file: got %s, want llb.file_action", valueType(args[0]))
		}
		options, err := optionsFromArgs[buildkitllb.ConstraintsOpt](args, 1, "constraints_opt")
		if err != nil { return nil, err }
		return WrapState(v.state.File(action.action, options...)), nil
	}), true, nil
}

var stateFileAttrNames = []string{"file", "file"}

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
	members["new_file_op"] = builtin("new_file_op", newFileOpBuiltin)
	return members
}

func fileActionBuiltins() map[string]starlark.Value {
	return map[string]starlark.Value{
		"mkdir": builtin("mkdir", mkdirBuiltin),
		"mkfile": builtin("mkfile", mkfileBuiltin),
		"symlink": builtin("symlink", symlinkBuiltin),
		"rm": builtin("rm", rmBuiltin),
		"copy": builtin("copy", copyBuiltin),
	}
}

func fileOwnershipBuiltins() map[string]starlark.Value {
	return map[string]starlark.Value{
		"with_parents": boolOptionBuiltin("with_parents", "mkdir_option", buildkitllb.WithParents),
		"with_user": stringOptionBuiltin("with_user", "chown_option", buildkitllb.WithUser),
		"with_uid_gid": builtin("with_uid_gid", withUIDGIDBuiltin),
		"chmod": builtin("chmod", chmodBuiltin),
		"chown": builtin("chown", chownBuiltin),
		"filename": stringOptionBuiltin("filename", "file_info_option", buildkitllb.Filename),
		"chmod_mode": intOptionBuiltin("chmod_mode", "copy_option", func(mode int) buildkitllb.CopyOption {
			return buildkitllb.ChmodOpt{Mode: os.FileMode(mode)}
		}),
		"chmod_symbolic": stringOptionBuiltin("chmod_symbolic", "copy_option", func(mode string) buildkitllb.CopyOption {
			return buildkitllb.ChmodOpt{ModeStr: mode}
		}),
	}
}

func removeOptionBuiltins() map[string]starlark.Value {
	return map[string]starlark.Value{
		"with_allow_not_found": boolOptionBuiltin("with_allow_not_found", "rm_option", buildkitllb.WithAllowNotFound),
		"with_allow_wildcard": boolOptionBuiltin("with_allow_wildcard", "rm_option", buildkitllb.WithAllowWildcard),
	}
}

func copyOptionBuiltins() map[string]starlark.Value {
	return map[string]starlark.Value{
		"with_exclude_patterns": optionBuiltin("with_exclude_patterns", "copy_option", func(_ *starlark.Thread, args starlark.Tuple, kwargs []starlark.Tuple) (any, error) {
			if err := arity("with_exclude_patterns", args, kwargs, 1, 1); err != nil { return nil, err }
			patterns, err := stringList(args[0]); if err != nil { return nil, err }
			return buildkitllb.WithExcludePatterns(patterns), nil
		}),
	}
}

func createdTimeBuiltins() map[string]starlark.Value {
	create := func(_ *starlark.Thread, args starlark.Tuple, kwargs []starlark.Tuple) (any, error) {
		if err := arity("with_created_time", args, kwargs, 1, 1); err != nil { return nil, err }
		raw, err := stringAt("with_created_time", args, 0); if err != nil { return nil, err }
		created, err := time.Parse(time.RFC3339Nano, raw)
		if err != nil { return nil, fmt.Errorf("with_created_time: %w", err) }
		return buildkitllb.WithCreatedTime(created), nil
	}
	value := optionBuiltin("with_created_time", "created_time", create)
	return map[string]starlark.Value{"created_time": value, "with_created_time": value}
}

func mkdirBuiltin(_ *starlark.Thread, _ *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	if err := arity("mkdir", args, kwargs, 2, -1); err != nil { return nil, err }
	path, err := stringAt("mkdir", args, 0); if err != nil { return nil, err }
	mode, err := intAt("mkdir", args, 1); if err != nil { return nil, err }
	options, err := optionsFromArgs[buildkitllb.MkdirOption](args, 2, "mkdir_option")
	if err != nil { return nil, err }
	return WrapFileAction(buildkitllb.Mkdir(path, os.FileMode(mode), options...)), nil
}

func mkfileBuiltin(_ *starlark.Thread, _ *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	if err := arity("mkfile", args, kwargs, 3, -1); err != nil { return nil, err }
	path, err := stringAt("mkfile", args, 0); if err != nil { return nil, err }
	mode, err := intAt("mkfile", args, 1); if err != nil { return nil, err }
	data, err := bytesAt("mkfile", args, 2); if err != nil { return nil, err }
	options, err := optionsFromArgs[buildkitllb.MkfileOption](args, 3, "mkfile_option")
	if err != nil { return nil, err }
	return WrapFileAction(buildkitllb.Mkfile(path, os.FileMode(mode), data, options...)), nil
}

func symlinkBuiltin(_ *starlark.Thread, _ *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	if err := arity("symlink", args, kwargs, 2, -1); err != nil { return nil, err }
	oldPath, err := stringAt("symlink", args, 0); if err != nil { return nil, err }
	newPath, err := stringAt("symlink", args, 1); if err != nil { return nil, err }
	options, err := optionsFromArgs[buildkitllb.SymlinkOption](args, 2, "symlink_option")
	if err != nil { return nil, err }
	return WrapFileAction(buildkitllb.Symlink(oldPath, newPath, options...)), nil
}

func rmBuiltin(_ *starlark.Thread, _ *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	if err := arity("rm", args, kwargs, 1, -1); err != nil { return nil, err }
	path, err := stringAt("rm", args, 0); if err != nil { return nil, err }
	options, err := optionsFromArgs[buildkitllb.RmOption](args, 1, "rm_option")
	if err != nil { return nil, err }
	return WrapFileAction(buildkitllb.Rm(path, options...)), nil
}

func copyBuiltin(_ *starlark.Thread, _ *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	if err := arity("copy", args, kwargs, 3, -1); err != nil { return nil, err }
	input, err := copyInputFromValue(args[0]); if err != nil { return nil, err }
	src, err := stringAt("copy", args, 1); if err != nil { return nil, err }
	dest, err := stringAt("copy", args, 2); if err != nil { return nil, err }
	options, err := optionsFromArgs[buildkitllb.CopyOption](args, 3, "copy_option")
	if err != nil { return nil, err }
	return WrapFileAction(buildkitllb.Copy(input, src, dest, options...)), nil
}

func withUIDGIDBuiltin(_ *starlark.Thread, _ *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	if err := arity("with_uid_gid", args, kwargs, 2, 2); err != nil { return nil, err }
	uid, err := intAt("with_uid_gid", args, 0); if err != nil { return nil, err }
	gid, err := intAt("with_uid_gid", args, 1); if err != nil { return nil, err }
	return wrapOption("chown_option", buildkitllb.WithUIDGID(uid, gid)), nil
}

func chmodBuiltin(_ *starlark.Thread, _ *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	if err := arity("chmod", args, kwargs, 1, 1); err != nil { return nil, err }
	mode, err := intAt("chmod", args, 0); if err != nil { return nil, err }
	return wrapOption("file_info_option", buildkitllb.Chmod(os.FileMode(mode))), nil
}

func chownBuiltin(_ *starlark.Thread, _ *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	if err := arity("chown", args, kwargs, 2, 2); err != nil { return nil, err }
	uid, err := intAt("chown", args, 0); if err != nil { return nil, err }
	gid, err := intAt("chown", args, 1); if err != nil { return nil, err }
	return wrapOption("file_info_option", buildkitllb.Chown(uid, gid)), nil
}

func newFileOpBuiltin(_ *starlark.Thread, _ *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	if err := arity("new_file_op", args, kwargs, 2, -1); err != nil { return nil, err }
	state, err := stateFromValue(args[0]); if err != nil { return nil, err }
	action, ok := args[1].(*FileActionValue)
	if !ok || action == nil || action.action == nil { return nil, fmt.Errorf("new_file_op: got %s, want llb.file_action", valueType(args[1])) }
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
	return nil, fmt.Errorf("got %s, want llb.state or llb.copy_input", valueType(value))
}

var (
	_ starlark.Value    = (*FileActionValue)(nil)
	_ starlark.HasAttrs = (*FileActionValue)(nil)
	_ starlark.Value    = (*CopyInputValue)(nil)
	_ starlark.Value    = (*FileOpValue)(nil)
	_ starlark.HasAttrs = (*FileOpValue)(nil)
)

