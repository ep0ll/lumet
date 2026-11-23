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
	"sync"

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

// ModuleLoader resolves a non-built-in Starlark module name to its globals.
// The callback should evaluate the requested module using the supplied thread.
// NewThreadWithLoader caches successful results and detects cyclic loads.
type ModuleLoader func(thread *starlark.Thread, module string) (starlark.StringDict, error)

// NewThread creates a Starlark thread configured to load the built-in llb
// module. The optional context is stored in thread-local storage as "context"
// for context-aware LLB operations. Exec additionally cancels evaluation when
// its context is cancelled.
func NewThread(ctx context.Context) *starlark.Thread {
	return NewThreadWithLoader(ctx, nil)
}

// NewThreadWithLoader creates a Starlark thread with the built-in LLB module
// and an optional application module loader. Loaded module globals are frozen
// and cached for the lifetime of the thread.
func NewThreadWithLoader(ctx context.Context, loader ModuleLoader) *starlark.Thread {
	thread := &starlark.Thread{Name: "lumet llb"}
	if ctx != nil {
		thread.SetLocal("context", ctx)
	}
	var mu sync.Mutex
	cache := map[string]starlark.StringDict{
		"llb": Module.Members,
		"lumet.llb": Module.Members,
		"llb.star": Module.Members,
		"lumet.llb.star": Module.Members,
	}
	loading := make(map[string]bool)
	thread.Load = func(thread *starlark.Thread, module string) (starlark.StringDict, error) {
		mu.Lock()
		if globals, ok := cache[module]; ok {
			mu.Unlock()
			return globals, nil
		}
		if loading[module] {
			mu.Unlock()
			return nil, fmt.Errorf("starlark: cyclic load of module %q", module)
		}
		if loader == nil {
			mu.Unlock()
			return nil, fmt.Errorf("llb: unknown Starlark module %q", module)
		}
		loading[module] = true
		mu.Unlock()

		globals, err := loader(thread, module)

		mu.Lock()
		delete(loading, module)
		if err == nil {
			if globals == nil {
				err = fmt.Errorf("starlark: module loader returned nil globals for %q", module)
			} else {
				globals.Freeze()
				cache[module] = globals
			}
		}
		mu.Unlock()
		return globals, err
	}
	return thread
}

// ExecOptions configures one Starlark evaluation.
type ExecOptions struct {
	// FileOptions controls syntax and resolver features. The zero value uses
	// conservative Starlark defaults.
	FileOptions syntax.FileOptions

	// MaxSteps limits interpreter execution steps. Zero leaves the limit
	// disabled; this does not impose a memory limit on the Starlark program.
	MaxSteps uint64

	// Print receives messages emitted by the Starlark print() builtin.
	Print func(thread *starlark.Thread, message string)

	// Loader resolves application modules in addition to the built-in llb module.
	// The caller owns path validation and source-file access.
	Loader ModuleLoader
}

// Exec evaluates src as a Starlark file with the LLB module loader installed.
// predeclared may be nil. The returned globals are frozen by Starlark.
func Exec(ctx context.Context, filename string, src any, predeclared starlark.StringDict) (starlark.StringDict, error) {
	return ExecWithOptions(ctx, filename, src, predeclared, ExecOptions{})
}

// ExecWithOptions evaluates src using the requested syntax, execution and
// output configuration. A context cancellation interrupts the interpreter.
func ExecWithOptions(ctx context.Context, filename string, src any, predeclared starlark.StringDict, options ExecOptions) (starlark.StringDict, error) {
	if predeclared == nil {
		predeclared = starlark.StringDict{}
	}
	thread := NewThreadWithLoader(ctx, options.Loader)
	thread.Print = options.Print
	if options.MaxSteps > 0 {
		thread.SetMaxExecutionSteps(options.MaxSteps)
	}
	if ctx != nil {
		stop := context.AfterFunc(ctx, func() {
			thread.Cancel("context cancelled")
		})
		defer stop()
	}
	fileOptions := options.FileOptions
	return starlark.ExecFileOptions(&fileOptions, thread, filename, src, predeclared)
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
