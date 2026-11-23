// Package modules owns construction, registration, and loading of every
// Starlark module exposed by Lumet.
package modules

import (
	"errors"
	"fmt"
	"sort"
	"strings"
	"sync"

	starlarkllb "github.com/ep0ll/lumet/starlark/llb"
	imagemetaresolver "github.com/ep0ll/lumet/starlark/llb/imagemetaresolver"
	llbbuild "github.com/ep0ll/lumet/starlark/llb/llbbuild"
	sourceresolver "github.com/ep0ll/lumet/starlark/llb/sourceresolver"
	"go.starlark.net/starlark"
	"go.starlark.net/starlarkstruct"
)

// ErrUnknownModule identifies an import that is not registered.
var ErrUnknownModule = errors.New("modules: unknown Starlark module")

var (
	registerOnce sync.Once
	registerErr  error
	registry     = struct {
		sync.RWMutex
		entries map[string]*starlarkstruct.Module
	}{entries: make(map[string]*starlarkstruct.Module)}
)

// Register constructs and registers every first-party module. It is safe to
// call repeatedly and is invoked automatically when this package is imported.
func Register() error {
	registerOnce.Do(func() {
		root, err := starlarkllb.BuiltinMembers()
		if err != nil {
			registerErr = err
			return
		}

		firstParty := []struct {
			name    string
			members starlark.StringDict
		}{
			{name: "llb", members: root},
			{name: "llb/imagemetaresolver", members: imagemetaresolver.Members()},
			{name: "llb/llbbuild", members: llbbuild.Members()},
			{name: "llb/sourceresolver", members: sourceresolver.Members()},
		}
		for _, module := range firstParty {
			if err := register(module.name, module.members); err != nil {
				registerErr = err
				return
			}
		}
	})
	return registerErr
}

// RegisterModule registers an application-owned module. Public members are
// normalized to snake_case and frozen before the module becomes visible.
// The llb namespace is reserved for Lumet's first-party modules.
func RegisterModule(name string, members starlark.StringDict) error {
	if err := Register(); err != nil {
		return err
	}
	name = normalizeModuleName(name)
	if name == "" || name == "llb" || strings.HasPrefix(name, "llb/") {
		return fmt.Errorf("modules: invalid or reserved module name %q", name)
	}
	return register(name, members)
}

func register(name string, members starlark.StringDict) error {
	name = normalizeModuleName(name)
	if name == "" {
		return fmt.Errorf("modules: module name must not be empty")
	}

	normalized, err := starlarkllb.NormalizeMembers(members)
	if err != nil {
		return fmt.Errorf("modules: %s: %w", name, err)
	}
	module := &starlarkstruct.Module{Name: name, Members: normalized}
	module.Freeze()

	registry.Lock()
	defer registry.Unlock()
	if _, exists := registry.entries[name]; exists {
		return fmt.Errorf("modules: module %q is already registered", name)
	}
	registry.entries[name] = module
	return nil
}

// Module returns a registered, frozen module by canonical name or alias.
func Module(name string) (*starlarkstruct.Module, bool) {
	if err := Register(); err != nil {
		return nil, false
	}
	name = normalizeModuleName(name)
	registry.RLock()
	module, ok := registry.entries[name]
	registry.RUnlock()
	return module, ok
}

// LLB returns the frozen root BuildKit LLB module.
func LLB() *starlarkstruct.Module {
	module, ok := Module("llb")
	if !ok {
		panic("modules: built-in llb module was not registered")
	}
	return module
}

// Predeclared returns a fresh globals dictionary with the root llb module.
func Predeclared() starlark.StringDict {
	return starlark.StringDict{"llb": LLB()}
}

// ModuleNames returns the sorted canonical names of all registered modules.
func ModuleNames() []string {
	if err := Register(); err != nil {
		return nil
	}
	registry.RLock()
	names := make([]string, 0, len(registry.entries))
	for name := range registry.entries {
		names = append(names, name)
	}
	registry.RUnlock()
	sort.Strings(names)
	return names
}

// Load implements starlark.Thread.Load for all registered modules.
func Load(_ *starlark.Thread, importName string) (starlark.StringDict, error) {
	module, ok := Module(importName)
	if !ok {
		return nil, fmt.Errorf("%w: %q", ErrUnknownModule, importName)
	}
	return module.Members, nil
}

func normalizeModuleName(name string) string {
	name = strings.TrimSpace(name)
	for {
		switch {
		case strings.HasSuffix(name, ".star"):
			name = strings.TrimSuffix(name, ".star")
		case strings.HasSuffix(name, ".bzl"):
			name = strings.TrimSuffix(name, ".bzl")
		default:
			goto aliases
		}
	}
aliases:
	switch name {
	case "lumet.llb":
		return "llb"
	case "imagemetaresolver":
		return "llb/imagemetaresolver"
	case "llbbuild":
		return "llb/llbbuild"
	case "sourceresolver":
		return "llb/sourceresolver"
	default:
		return name
	}
}

func init() {
	if err := Register(); err != nil {
		panic(err)
	}
}
