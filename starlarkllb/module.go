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
			// Source constructors (source.go)
			"image":   starlark.NewBuiltin("image", imageBuiltin),
			"scratch": starlark.NewBuiltin("scratch", scratchBuiltin),
			"local":   starlark.NewBuiltin("local", localBuiltin),
			"git":     starlark.NewBuiltin("git", gitBuiltin),
			"http":    starlark.NewBuiltin("http", httpBuiltin),

			// File actions (fileop.go)
			"mkdir":  starlark.NewBuiltin("mkdir", mkdirBuiltin),
			"mkfile": starlark.NewBuiltin("mkfile", mkfileBuiltin),
			"rm":     starlark.NewBuiltin("rm", rmBuiltin),
			"copy":   starlark.NewBuiltin("copy", copyActionBuiltin),

			// Merge / Diff (merge.go)
			"merge": starlark.NewBuiltin("merge", mergeBuiltin),
			"diff":  starlark.NewBuiltin("diff", diffBuiltin),

			// Helpers
			"shlex": starlark.NewBuiltin("shlex", shlexBuiltin),

			// Meta / constraints helpers (meta.go) - expanding
			"with_custom_name": starlark.NewBuiltin("with_custom_name", withCustomNameBuiltin),
		},
	}
}

// Register adds the llb module into a predeclared dict.
func Register(predeclared starlark.StringDict) {
	if predeclared == nil {
		return
	}
	predeclared["llb"] = Module()
}
