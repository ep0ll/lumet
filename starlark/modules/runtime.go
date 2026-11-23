package modules

import (
	"context"
	"errors"
	"fmt"
	"sync"

	starlarkllb "github.com/ep0ll/lumet/starlark/llb"
	"go.starlark.net/starlark"
	"go.starlark.net/syntax"
)

// ModuleLoader resolves application-owned Starlark source modules.
type ModuleLoader func(thread *starlark.Thread, module string) (starlark.StringDict, error)

// ExecOptions configures one Starlark evaluation.
type ExecOptions struct {
	// FileOptions controls language features enabled for the source file.
	FileOptions syntax.FileOptions

	// MaxSteps limits interpreter execution steps. Zero disables the limit.
	MaxSteps uint64

	// Print receives messages emitted by Starlark's print() builtin.
	Print func(thread *starlark.Thread, message string)

	// Loader resolves application modules after the built-in registry.
	Loader ModuleLoader
}

// NewThread creates a thread with Lumet's registered module loader.
func NewThread(ctx context.Context) *starlark.Thread {
	return NewThreadWithLoader(ctx, nil)
}

// NewThreadWithLoader creates a thread with registered first-party modules
// and an optional application loader. Loaded globals are frozen and cached
// per thread. Registered modules take precedence over the fallback loader.
func NewThreadWithLoader(ctx context.Context, loader ModuleLoader) *starlark.Thread {
	thread := &starlark.Thread{Name: "lumet", Load: Load}
	if ctx != nil {
		thread.SetLocal("context", ctx)
	}

	var mu sync.Mutex
	cache := make(map[string]starlark.StringDict)
	loading := make(map[string]bool)

	thread.Load = func(thread *starlark.Thread, importName string) (starlark.StringDict, error) {
		name := normalizeModuleName(importName)
		if name == "" {
			return nil, fmt.Errorf("modules: module name must not be empty")
		}

		mu.Lock()
		if globals, ok := cache[importName]; ok {
			mu.Unlock()
			return globals, nil
		}
		if globals, ok := cache[name]; ok {
			cache[importName] = globals
			mu.Unlock()
			return globals, nil
		}
		if loading[name] {
			mu.Unlock()
			return nil, fmt.Errorf("starlark: cyclic load of module %q", importName)
		}
		loading[name] = true
		mu.Unlock()

		globals, err := Load(thread, importName)
		if errors.Is(err, ErrUnknownModule) && loader != nil {
			globals, err = loader(thread, importName)
			if err == nil {
				globals, err = starlarkllb.NormalizeMembers(globals)
			}
		}

		mu.Lock()
		delete(loading, name)
		if err == nil {
			if globals == nil {
				err = fmt.Errorf("starlark: module loader returned nil globals for %q", importName)
			} else {
				globals.Freeze()
				cache[name] = globals
				cache[importName] = globals
			}
		}
		mu.Unlock()
		return globals, err
	}
	return thread
}

// Exec evaluates src as a Starlark file. A nil predeclared dictionary
// automatically exposes the built-in llb module as a global named llb.
func Exec(ctx context.Context, filename string, src any, predeclared starlark.StringDict) (starlark.StringDict, error) {
	return ExecWithOptions(ctx, filename, src, predeclared, ExecOptions{})
}

// ExecWithOptions evaluates src using the supplied syntax, execution, output,
// and module-loading configuration. Context cancellation interrupts execution.
func ExecWithOptions(ctx context.Context, filename string, src any, predeclared starlark.StringDict, options ExecOptions) (starlark.StringDict, error) {
	if predeclared == nil {
		predeclared = Predeclared()
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
