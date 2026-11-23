package llb

import (
	"fmt"

	"go.starlark.net/starlark"
)

// BuiltinMembers returns the global members exported by the high-level
// BuildKit client/llb facade. It does not construct or register a module;
// module lifecycle and registration are owned by starlark/modules.
func BuiltinMembers() (starlark.StringDict, error) {
	members := make(starlark.StringDict)
	origins := make(map[string]string)

	add := func(group map[string]starlark.Value, kind string) error {
		for name, value := range group {
			if name == "" || value == nil {
				return fmt.Errorf("starlark/llb: invalid %s %q", kind, name)
			}
			if previous, exists := origins[name]; exists {
				return fmt.Errorf("starlark/llb: duplicate %s %q (already registered as %s)", kind, name, previous)
			}
			origins[name] = kind
			members[name] = value
		}
		return nil
	}

	builtinGroups := []func() map[string]starlark.Value{
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
	for _, group := range builtinGroups {
		if err := add(group(), "builtin"); err != nil {
			return nil, err
		}
	}

	valueGroups := []map[string]starlark.Value{
		platformConstraintValues(),
		stateConstraintValues(),
		imageOptionValues(),
		networkModeValues(),
		securityModeValues(),
		cacheMountSharingValues(),
		mountContentCacheValues(),
		ulimitValues(),
		cdiDeviceOptionValues(),
		sshOptionValues(),
		secretOptionValues(),
		diffTypeValues(),
		gitAuthValues(),
		resolveModeValues(),
	}
	for _, group := range valueGroups {
		if err := add(group, "constant"); err != nil {
			return nil, err
		}
	}

	normalized, err := NormalizeMembers(members)
	if err != nil {
		return nil, fmt.Errorf("starlark/llb: invalid builtin registry: %w", err)
	}
	return normalized, nil
}
