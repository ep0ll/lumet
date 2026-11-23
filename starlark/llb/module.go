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

// Module is the frozen Starlark module for BuildKit's high-level LLB API.
var Module = newModule()

func newModule() *starlarkstruct.Module {
	members := make(starlark.StringDict)
	groups := []func() map[string]starlark.Value{
		stateBuiltins,
		metaBuiltins,
		execBuiltins,
		fileOpBuiltins,
		sourceBuiltins,
		resolverBuiltins,
		diffBuiltins,
		mergeBuiltins,
		passthroughBuiltins,
		definitionBuiltins,
		marshalBuiltins,
		sourceMapBuiltins,
	}
	for _, group := range groups {
		for name, value := range group() {
			if _, exists := members[name]; exists {
				panic("starlark/llb: duplicate builtin " + name)
			}
			members[name] = value
		}
	}
	for _, group := range [][]map[string]starlark.Value{
		{
			platformConstraintValues(),
			stateConstraintValues(),
			networkModeValues(),
			securityModeValues(),
			cacheMountSharingValues(),
			mountContentCacheValues(),
			ulimitValues(),
			diffTypeValues(),
			gitAuthValues(),
			resolveModeValues(),
		},
	} {
		for _, values := range group {
			for name, value := range values {
				if _, exists := members[name]; exists {
					panic("starlark/llb: duplicate constant " + name)
				}
				members[name] = value
			}
		}
	}
	module := &starlarkstruct.Module{Name: "llb", Members: members}
	module.Freeze()
	return module
}

// ModuleLoader resolves an application-owned Starlark module. Evaluate the
// module using the supplied thread so nested loads share its cache and cycle
// detection.
type ModuleLoader func(thread *starlark.Thread, module string) (starlark.StringDict, error)

var registeredModules = struct {
	sync.RWMutex
	members map[string]starlark.StringDict
}{
	members: make(map[string]starlark.StringDict),
}

// RegisterModule registers a frozen module for use with Load and NewThread.
// Subpackages use this hook to register their own Starlark facade without
// introducing an import cycle from the root llb package.
func RegisterModule(name string, members starlark.StringDict) error {
	name = normalizeModuleName(name)
	if name == "" || name == "llb" || name == "lumet.llb" {
		return fmt.Errorf("starlark/llb: invalid or reserved module name %q", name)
	}
	if members == nil {
		return fmt.Errorf("starlark/llb: module %q has nil globals", name)
	}

	copyMembers := make(starlark.StringDict, len(members))
	for key, value := range members {
		if key == "" || value == nil {
			return fmt.Errorf("starlark/llb: module %q has an invalid global %q", name, key)
		}
		copyMembers[key] = value
	}
	copyMembers.Freeze()

	registeredModules.Lock()
	defer registeredModules.Unlock()
	if _, exists := registeredModules.members[name]; exists {
		return fmt.Errorf("starlark/llb: module %q is already registered", name)
	}
	registeredModules.members[name] = copyMembers
	return nil
}

// Load implements starlark.Thread.Load for the built-in llb module and any
// modules registered with RegisterModule.
func Load(_ *starlark.Thread, module string) (starlark.StringDict, error) {
	name := normalizeModuleName(module)
	switch name {
	case "llb", "lumet.llb":
		return Module.Members, nil
	}

	registeredModules.RLock()
	members, ok := registeredModules.members[name]
	registeredModules.RUnlock()
	if ok {
		return members, nil
	}
	return nil, fmt.Errorf("llb: unknown Starlark module %q", module)
}

func normalizeModuleName(module string) string {
	module = strings.TrimSuffix(module, ".star")
	module = strings.TrimSuffix(module, ".bzl")
	return strings.TrimSpace(module)
}

// NewThread creates a Starlark thread configured with the built-in LLB module
// and modules registered with RegisterModule.
func NewThread(ctx context.Context) *starlark.Thread {
	return NewThreadWithLoader(ctx, nil)
}

// NewThreadWithLoader creates a Starlark thread with the built-in LLB module,
// registered modules, and an optional application loader. Successfully loaded
// module globals are frozen and cached for the lifetime of the thread.
func NewThreadWithLoader(ctx context.Context, loader ModuleLoader) *starlark.Thread {
	thread := &starlark.Thread{Name: "lumet llb", Load: Load}
	if ctx != nil {
		thread.SetLocal("context", ctx)
	}

	var mu sync.Mutex
	cache := map[string]starlark.StringDict{
		"llb":          Module.Members,
		"lumet.llb":    Module.Members,
		"llb.star":     Module.Members,
		"lumet.llb.star": Module.Members,
	}
	loading := make(map[string]bool)

	thread.Load = func(thread *starlark.Thread, module string) (starlark.StringDict, error) {
		name := normalizeModuleName(module)

		mu.Lock()
		if globals, ok := cache[module]; ok {
			mu.Unlock()
			return globals, nil
		}
		if globals, ok := cache[name]; ok {
			cache[module] = globals
			mu.Unlock()
			return globals, nil
		}
		if loading[name] {
			mu.Unlock()
			return nil, fmt.Errorf("starlark: cyclic load of module %q", module)
		}
		loading[name] = true
		mu.Unlock()

		var globals starlark.StringDict
		var err error
		if loader != nil {
			globals, err = loader(thread, module)
		} else {
			globals, err = Load(thread, module)
		}

		mu.Lock()
		delete(loading, name)
		if err == nil {
			if globals == nil {
				err = fmt.Errorf("starlark: module loader returned nil globals for %q", module)
			} else {
				globals.Freeze()
				cache[name] = globals
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
	// FileOptions controls the Starlark language features enabled for this file.
	FileOptions syntax.FileOptions

	// MaxSteps limits interpreter execution steps. Zero disables the limit.
	MaxSteps uint64

	// Print receives messages emitted by the Starlark print() builtin.
	Print func(thread *starlark.Thread, message string)

	// Loader resolves application modules in addition to llb and registered modules.
	Loader ModuleLoader
}

// Exec evaluates src as a Starlark file using the built-in LLB module loader.
// predeclared may be nil. The returned globals are frozen by Starlark.
func Exec(ctx context.Context, filename string, src any, predeclared starlark.StringDict) (starlark.StringDict, error) {
	return ExecWithOptions(ctx, filename, src, predeclared, ExecOptions{})
}

// ExecWithOptions evaluates src using the requested syntax, execution and
// output configuration. Context cancellation interrupts the interpreter.
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

// Predeclared returns a fresh global dictionary containing the "llb" module.
func Predeclared() starlark.StringDict {
	return starlark.StringDict{"llb": Module}
}

// callContext retrieves the context supplied to NewThread. Starlark programs
// executed through the standard interpreter always receive a non-nil context
// unless the caller explicitly uses NewThread(nil).
func callContext(thread *starlark.Thread) context.Context {
	if thread != nil {
		if ctx, ok := thread.Local("context").(context.Context); ok && ctx != nil {
			return ctx
		}
	}
	return context.Background()
}

