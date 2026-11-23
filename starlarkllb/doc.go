// Package starlarkllb provides Starlark bindings for the BuildKit LLB API.
//
// It is a thin wrapper around github.com/moby/buildkit/client/llb.
// Starlark code can construct LLB graphs using a Python-like dialect and
// the host Go application can extract the real llb.State / llb.Definition
// for solving.
//
// Basic usage from the host:
//
//	import (
//		"go.starlark.net/starlark"
//		"github.com/ep0ll/lumet/starlarkllb"
//	)
//
//	predeclared := starlark.StringDict{}
//	starlarkllb.Register(predeclared)
//
//	thread := &starlark.Thread{Name: "build"}
//	globals, err := starlark.ExecFile(thread, "build.star", nil, predeclared)
//	if err != nil { ... }
//
//	st, err := starlarkllb.AsState(globals["result"])
//
// Example Starlark (build.star):
//
//	load("llb", "llb")   # or just use the predeclared "llb"
//
//	st = llb.image("golang:1.22")
//	st = st.dir("/src").add_env("CGO_ENABLED", "0")
//	st = st.run("go build -o /out ./cmd/app").root()
//
//	result = st
//
// The package is intentionally incomplete. New methods and constructors
// should follow the same pattern used for State / ExecState / image / scratch.
package starlarkllb
