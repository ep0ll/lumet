package starlarkllb

import (
	"fmt"
	"os"

	"github.com/moby/buildkit/client/llb"
	"go.starlark.net/starlark"
)

// FileAction wraps llb.FileAction.
type FileAction struct {
	action *llb.FileAction
}

var (
	_ starlark.Value    = (*FileAction)(nil)
	_ starlark.HasAttrs = (*FileAction)(nil)
)

func (f *FileAction) String() string        { return "<llb.FileAction>" }
func (f *FileAction) Type() string          { return "llb.FileAction" }
func (f *FileAction) Freeze()               {}
func (f *FileAction) Truth() starlark.Bool  { return true }
func (f *FileAction) Hash() (uint32, error) { return 0, fmt.Errorf("unhashable type: llb.FileAction") }

func (f *FileAction) AttrNames() []string {
	return []string{"copy", "mkdir", "mkfile", "rm", "symlink"}
}

func (f *FileAction) Attr(name string) (starlark.Value, error) {
	switch name {
	case "copy":
		return starlark.NewBuiltin("copy", f.copy), nil
	case "mkdir":
		return starlark.NewBuiltin("mkdir", f.mkdir), nil
	case "mkfile":
		return starlark.NewBuiltin("mkfile", f.mkfile), nil
	case "rm":
		return starlark.NewBuiltin("rm", f.rm), nil
	case "symlink":
		return starlark.NewBuiltin("symlink", f.symlink), nil
	default:
		return nil, nil
	}
}

// Constructors ------------------------------------------------------------

func mkdirBuiltin(_ *starlark.Thread, fn *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	var path string
	var mode int = 0755
	if err := starlark.UnpackArgs(fn.Name(), args, kwargs, "path", &path, "mode?", &mode); err != nil {
		return nil, err
	}
	return &FileAction{action: llb.Mkdir(path, os.FileMode(mode))}, nil
}

func mkfileBuiltin(_ *starlark.Thread, fn *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	var path, data string
	var mode int = 0644
	if err := starlark.UnpackArgs(fn.Name(), args, kwargs, "path", &path, "data", &data, "mode?", &mode); err != nil {
		return nil, err
	}
	return &FileAction{action: llb.Mkfile(path, os.FileMode(mode), []byte(data))}, nil
}

func rmBuiltin(_ *starlark.Thread, fn *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	var path string
	if err := starlark.UnpackArgs(fn.Name(), args, kwargs, "path", &path); err != nil {
		return nil, err
	}
	return &FileAction{action: llb.Rm(path)}, nil
}

func copyActionBuiltin(_ *starlark.Thread, fn *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	var input starlark.Value
	var src, dest string
	if err := starlark.UnpackArgs(fn.Name(), args, kwargs, "input", &input, "src", &src, "dest", &dest); err != nil {
		return nil, err
	}

	var copyInput llb.CopyInput
	switch v := input.(type) {
	case *State:
		copyInput = v.state
	case *FileAction:
		copyInput = v.action
	default:
		return nil, fmt.Errorf("copy: expected State or FileAction as input")
	}

	return &FileAction{action: llb.Copy(copyInput, src, dest)}, nil
}

// Methods on FileAction ---------------------------------------------------

func (f *FileAction) copy(_ *starlark.Thread, fn *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	var input starlark.Value
	var src, dest string
	if err := starlark.UnpackArgs(fn.Name(), args, kwargs, "input", &input, "src", &src, "dest", &dest); err != nil {
		return nil, err
	}

	var copyInput llb.CopyInput
	switch v := input.(type) {
	case *State:
		copyInput = v.state
	case *FileAction:
		copyInput = v.action
	default:
		return nil, fmt.Errorf("copy: expected State or FileAction")
	}

	return &FileAction{action: f.action.Copy(copyInput, src, dest)}, nil
}

func (f *FileAction) mkdir(_ *starlark.Thread, fn *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	var path string
	var mode int = 0755
	if err := starlark.UnpackArgs(fn.Name(), args, kwargs, "path", &path, "mode?", &mode); err != nil {
		return nil, err
	}
	return &FileAction{action: f.action.Mkdir(path, os.FileMode(mode))}, nil
}

func (f *FileAction) mkfile(_ *starlark.Thread, fn *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	var path, data string
	var mode int = 0644
	if err := starlark.UnpackArgs(fn.Name(), args, kwargs, "path", &path, "data", &data, "mode?", &mode); err != nil {
		return nil, err
	}
	return &FileAction{action: f.action.Mkfile(path, os.FileMode(mode), []byte(data))}, nil
}

func (f *FileAction) rm(_ *starlark.Thread, fn *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	var path string
	if err := starlark.UnpackArgs(fn.Name(), args, kwargs, "path", &path); err != nil {
		return nil, err
	}
	return &FileAction{action: f.action.Rm(path)}, nil
}

func (f *FileAction) symlink(_ *starlark.Thread, fn *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	var oldpath, newpath string
	if err := starlark.UnpackArgs(fn.Name(), args, kwargs, "oldpath", &oldpath, "newpath", &newpath); err != nil {
		return nil, err
	}
	return &FileAction{action: f.action.Symlink(oldpath, newpath)}, nil
}
