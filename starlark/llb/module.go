// Package llb exposes BuildKit's client/llb graph-building API to Starlark.
//
// The package deliberately wraps BuildKit values instead of reflecting over Go
// types. Starlark programs can construct immutable LLB states, compose source
// and run options, and hand the resulting StateValue back to Go.
package llb

import (
	"context"
	"fmt"
	"strings"

	"go.starlark.net/starlark"
	"go.starlark.net/starlarkstruct"
	"go.starlark.net/syntax"
)

// Module is the predeclared Starlark module named "llb".
//
// Use it with load("llb", "Image", "Scratch"), or add it to a predeclared
// environment as {"llb": Module}.
var Module = newModule()

func newModule() *starlarkstruct.Module {
	members := starlark.StringDict{}
	for name, value := range sourceBuiltins() {
		members[name] = value
	}
	for name, value := range optionBuiltins() {
		members[name] = value
	}
	for name, value := range fileBuiltins() {
		members[name] = value
	}
	for name, value := range constantValues() {
		members[name] = value
	}
	module := &starlarkstruct.Module{Name: "llb", Members: members}
	module.Freeze()
	return module
}

// Load implements starlark.Thread.Load for the built-in LLB module.
// Module names "llb" and "lumet.llb" are accepted.
func Load(_ *starlark.Thread, module string) (starlark.StringDict, error) {
	switch strings.TrimSuffix(module, ".star") {
	case "llb", "lumet.llb":
		return Module.Members, nil
	default:
		return nil, fmt.Errorf("llb: unknown Starlark module %q", module)
	}
}

// NewThread creates a Starlark thread configured to load this package's llb
// module. The optional context is stored in thread-local storage as "context"
// for context-aware LLB operations. Exec additionally cancels evaluation when
// its context is cancelled.
func NewThread(ctx context.Context) *starlark.Thread {
	thread := &starlark.Thread{Name: "lumet llb", Load: Load}
	if ctx == nil {
		return thread
	}
	thread.SetLocal("context", ctx)
	return thread
}

// Exec evaluates src as a Starlark file with the LLB module loader installed.
// predeclared may be nil. The returned globals are frozen by Starlark.
func Exec(ctx context.Context, filename string, src any, predeclared starlark.StringDict) (starlark.StringDict, error) {
	if predeclared == nil {
		predeclared = starlark.StringDict{}
	}
	thread := NewThread(ctx)
	if ctx != nil {
		stop := context.AfterFunc(ctx, func() {
			thread.Cancel("context cancelled")
		})
		defer stop()
	}
	return starlark.ExecFileOptions(&syntax.FileOptions{}, thread, filename, src, predeclared)
}

// Predeclared returns a fresh dictionary containing the "llb" module.
// The returned map may be modified without affecting future calls.
func Predeclared() starlark.StringDict {
	return starlark.StringDict{"llb": Module}
}

// boundBuiltin constructs a builtin whose implementation is bound to a Go
// value (for example, a StateValue receiver).
func boundBuiltin(name string, fn func(*starlark.Thread, starlark.Tuple, []starlark.Tuple) (starlark.Value, error)) *starlark.Builtin {
	return starlark.NewBuiltin(name, func(thread *starlark.Thread, _ *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
		return fn(thread, args, kwargs)
	})
}

// builtin is the module-level equivalent of boundBuiltin.
func builtin(name string, fn func(*starlark.Thread, *starlark.Builtin, starlark.Tuple, []starlark.Tuple) (starlark.Value, error)) *starlark.Builtin {
	return starlark.NewBuiltin(name, fn)
}

// callContext retrieves the context supplied to NewThread. A background
// context is used when a caller uses starlark.ExecFile directly.
func callContext(thread *starlark.Thread) context.Context {
	if thread != nil {
		if ctx, ok := thread.Local("context").(context.Context); ok && ctx != nil {
			return ctx
		}
	}
	return context.Background()
}
