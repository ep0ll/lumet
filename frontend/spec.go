package frontend

import (
	"path"
	"slices"
	"strings"
)

// ContextOptKeys names the BuildKit frontend options used to select the
// build-definition source and the main build context. These are option keys,
// not option values.
type ContextOptKeys struct {
	// Filename overrides the entrypoint filename inside its source.
	Filename string

	// ContextLocalName overrides the local name used for the main context.
	ContextLocalName string

	// EntrypointLocalNames contains option keys whose values select the
	// build-definition local name. The first non-empty option wins.
	EntrypointLocalNames []string

	// ContextSubDir scopes the main context to a directory inside its source.
	ContextSubDir string

	// EntrypointSubDir scopes the build-definition source to a directory.
	EntrypointSubDir string

	// KeepGitDir controls whether Git sources retain their .git directory.
	KeepGitDir string

	// DockerfileLocalNames is retained for source compatibility. New specs
	// should use EntrypointLocalNames.
	//
	// Deprecated: use EntrypointLocalNames.
	DockerfileLocalNames []string

	// DockerfileSubDir is retained for source compatibility. New specs should
	// use EntrypointSubDir.
	//
	// Deprecated: use EntrypointSubDir.
	DockerfileSubDir string
}

// Spec describes the naming and ignore-file conventions of one BuildKit
// frontend. It deliberately contains no parser, solver, or frontend-specific
// implementation; those remain injectable at the consumer boundary.
type Spec struct {
	// Name identifies the language in source maps and diagnostics.
	Name string

	// DefaultEntrypoint returns the default build-definition filename for a
	// frontend configuration. The empty configuration is used when the
	// frontend has no configuration-specific entrypoint.
	DefaultEntrypoint func(config string) string

	// EntrypointFallback returns alternative entrypoint paths, in priority
	// order. It is useful for conventions such as Dockerfile/dockerfile.
	EntrypointFallback func(entrypoint string) []string

	// IgnoreFiles returns ignore files associated with the main build-context
	// root, in priority order. The first existing file is used. A nil callback
	// means that this frontend has no context ignore-file convention.
	IgnoreFiles func(entrypoint string) []string

	// DefinitionIgnoreFiles returns entrypoint-specific ignore files that
	// override IgnoreFiles when present. For example, Dockerfile frontends can
	// return Dockerfile.dockerignore here and .dockerignore from IgnoreFiles.
	DefinitionIgnoreFiles func(entrypoint string) []string

	// IgnoreParser optionally parses an ignore file. When nil, the standard
	// Moby ignorefile parser is used by the context package.
	IgnoreParser func(filename string, data []byte) ([]string, error)

	// DefaultContextLocalName is the BuildKit local name used when no
	// context-local-name option is supplied. The conventional value is
	// context.
	DefaultContextLocalName string

	// EntrypointLocalNames contains candidate BuildKit input/local names.
	// The final element is the unconditional default; earlier elements may
	// be selected by the option keys in OptKeys.EntrypointLocalNames.
	EntrypointLocalNames []string

	// OptKeys contains the exact option keys used to read source overrides.
	OptKeys ContextOptKeys
}

// DefaultContextOptKeys returns the conventional BuildKit option names. The
// filename option is frontend-neutral; frontend identity belongs in Name and
// DefaultEntrypoint.
func DefaultContextOptKeys() ContextOptKeys {
	return ContextOptKeys{
		Filename:             "filename",
		ContextLocalName:     "contextkey",
		EntrypointLocalNames: []string{"dockerfilekey"},
		ContextSubDir:        "contextsubdir",
		EntrypointSubDir:     "entrypointsubdir",
		KeepGitDir:           "build-arg:BUILDKIT_CONTEXT_KEEP_GIT_DIR",
	}
}

// Clone returns a copy whose mutable slices are independent of the original.
func (s Spec) Clone() Spec {
	s.EntrypointLocalNames = slices.Clone(s.EntrypointLocalNames)
	s.OptKeys.EntrypointLocalNames = slices.Clone(s.OptKeys.EntrypointLocalNames)
	s.OptKeys.DockerfileLocalNames = slices.Clone(s.OptKeys.DockerfileLocalNames)
	return s
}

// LowerBaseFallback returns the lower-case basename alternative for a
// canonical entrypoint, preserving its parent directory.
func LowerBaseFallback(defaultEntrypoint string) func(string) []string {
	return func(filename string) []string {
		if path.Base(filename) != defaultEntrypoint {
			return nil
		}
		fallback := path.Join(path.Dir(filename), strings.ToLower(defaultEntrypoint))
		if fallback == filename {
			return nil
		}
		return []string{fallback}
	}
}

// SuffixIgnore returns the conventional entrypoint-plus-suffix ignore path.
func SuffixIgnore(suffix string) func(string) []string {
	return func(entrypoint string) []string {
		if suffix == "" || entrypoint == "" {
			return nil
		}
		return []string{entrypoint + suffix}
	}
}

// Validate fills standard option-key defaults and rejects unusable specs.
// It does not mutate caller-owned slices.
func (s *Spec) Validate() error {
	if s == nil {
		return invalidSpec("spec is nil")
	}
	if strings.TrimSpace(s.Name) == "" {
		return invalidSpec("Name is required")
	}
	if s.DefaultEntrypoint == nil {
		return invalidSpec("DefaultEntrypoint is required")
	}

	s.EntrypointLocalNames = slices.Clone(s.EntrypointLocalNames)
	s.OptKeys.EntrypointLocalNames = slices.Clone(s.OptKeys.EntrypointLocalNames)
	s.OptKeys.DockerfileLocalNames = slices.Clone(s.OptKeys.DockerfileLocalNames)

	if s.DefaultContextLocalName == "" {
		s.DefaultContextLocalName = "context"
	}
	if len(s.EntrypointLocalNames) == 0 {
		s.EntrypointLocalNames = []string{"dockerfile"}
	}
	for i, name := range s.EntrypointLocalNames {
		if strings.TrimSpace(name) == "" {
			return invalidSpec("EntrypointLocalNames contains an empty name")
		}
		s.EntrypointLocalNames[i] = strings.TrimSpace(name)
	}

	// Migrate the Dockerfile-specific option fields without removing support
	// for existing callers that initialized the earlier spec shape.
	if len(s.OptKeys.EntrypointLocalNames) == 0 && len(s.OptKeys.DockerfileLocalNames) != 0 {
		s.OptKeys.EntrypointLocalNames = slices.Clone(s.OptKeys.DockerfileLocalNames)
	}
	if s.OptKeys.EntrypointSubDir == "" {
		s.OptKeys.EntrypointSubDir = s.OptKeys.DockerfileSubDir
	}

	defaults := DefaultContextOptKeys()
	if s.OptKeys.Filename == "" {
		s.OptKeys.Filename = defaults.Filename
	}
	if s.OptKeys.ContextLocalName == "" {
		s.OptKeys.ContextLocalName = defaults.ContextLocalName
	}
	if len(s.OptKeys.EntrypointLocalNames) == 0 {
		s.OptKeys.EntrypointLocalNames = slices.Clone(defaults.EntrypointLocalNames)
	}
	if s.OptKeys.ContextSubDir == "" {
		s.OptKeys.ContextSubDir = defaults.ContextSubDir
	}
	if s.OptKeys.EntrypointSubDir == "" {
		s.OptKeys.EntrypointSubDir = defaults.EntrypointSubDir
	}
	if s.OptKeys.KeepGitDir == "" {
		s.OptKeys.KeepGitDir = defaults.KeepGitDir
	}
	for i, key := range s.OptKeys.EntrypointLocalNames {
		if strings.TrimSpace(key) == "" {
			return invalidSpec("OptKeys.EntrypointLocalNames contains an empty key")
		}
		s.OptKeys.EntrypointLocalNames[i] = strings.TrimSpace(key)
	}
	return nil
}

// InvalidSpecError is returned when a frontend specification is incomplete.
type InvalidSpecError struct {
	Msg string
}

func (e *InvalidSpecError) Error() string {
	return "invalid frontend spec: " + e.Msg
}

func invalidSpec(msg string) error {
	return &InvalidSpecError{Msg: msg}
}
