package starlarkllb

import (
	"go.starlark.net/starlark"
	"go.starlark.net/starlarkstruct"
)

// Module returns the top-level "llb" Starlark module.
// It is a thin binding over github.com/moby/buildkit/client/llb.
func Module() *starlarkstruct.Module {
	return &starlarkstruct.Module{
		Name: "llb",
		Members: starlark.StringDict{
			// Constructors
			"image":   starlark.NewBuiltin("image", imageBuiltin),
			"scratch": starlark.NewBuiltin("scratch", scratchBuiltin),

			// Helpers
			"shlex": starlark.NewBuiltin("shlex", shlexBuiltin),

			// More will be added (local, git, http, merge, diff, etc.)
		},
	}
}

// Register adds the llb module into a predeclared dict (typical usage).
func Register(predeclared starlark.StringDict) {
	if predeclared == nil {
		return
	}
	predeclared["llb"] = Module()
}

// Must be called once from the host application that embeds Starlark.
// Example:
//
//	predeclared := starlark.StringDict{}
//	starlarkllb.Register(predeclared)
//	thread := &starlark.Thread{Name: "main"}
//	globals, err := starlark.ExecFile(thread, "build.star", nil, predeclared)
