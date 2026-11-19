package frontend

import (
	"path"
	"slices"
	"strings"
)

// ContextOptKeys holds the frontend option key names used by shared context
// loading. These are the keys in BuildOpts.Opts (or --opt buildctl/buildx).
type ContextOptKeys struct {
	Filename             string
	ContextLocalName     string
	EntrypointLocalNames []string
	ContextSubDir        string
	EntrypointSubDir     string
	KeepGitDir            string
}

// Spec describes a frontend's conventions for locating its build definition
// and associated build context.
type Spec struct {
	Name                    string
	DefaultEntrypoint       func(config string) string
	IgnoreFiles             func(entrypoint string) []string
	DefaultContextLocalName string
	EntrypointLocalNames    []string
	OptKeys                 ContextOptKeys
}

// DefaultContextOptKeys returns the conventional Dockerfile frontend option
// names. Other frontends can override individual keys in their Spec.
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

// LowerBaseFallback is the common "Dockerfile" to "dockerfile" fallback.
func LowerBaseFallback(def string) func(string) []string {
	return func(filename string) []string {
		if path.Base(filename) != def {
			return nil
		}
		fallback := path.Join(path.Dir(filename), strings.ToLower(def))
		if fallback == filename {
			return nil
		}
		return []string{fallback}
	}
}

// SuffixIgnore yields "<entrypoint><suffix>", e.g. "Lumetfile.dexnore".
func SuffixIgnore(suffix string) func(string) []string {
	return func(filename string) []string { return []string{filename + suffix} }
}

// Validate fills missing defaults and rejects unusable specifications. It is
// safe to call repeatedly and does not replace caller-provided option keys.
func (s *Spec) Validate() error {
	if s == nil {
		return invalidSpec("spec is nil")
	}
	s.Name = strings.TrimSpace(s.Name)
	if s.Name == "" {
		return invalidSpec("Name is required")
	}
	if s.DefaultEntrypoint == nil {
		return invalidSpec("DefaultEntrypoint is required")
	}
	if filename := strings.TrimSpace(s.DefaultEntrypoint("")); filename == "" {
		return invalidSpec("DefaultEntrypoint returned an empty filename")
	}

	s.DefaultContextLocalName = strings.TrimSpace(s.DefaultContextLocalName)
	if s.DefaultContextLocalName == "" {
		s.DefaultContextLocalName = "context"
	}
	if len(s.EntrypointLocalNames) == 0 {
		s.EntrypointLocalNames = []string{"dockerfile"}
	}
	s.EntrypointLocalNames = slices.Clone(s.EntrypointLocalNames)
	for i, name := range s.EntrypointLocalNames {
		if strings.TrimSpace(name) == "" {
			return invalidSpec("EntrypointLocalNames cannot contain empty names")
		}
		s.EntrypointLocalNames[i] = strings.TrimSpace(name)
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
	} else {
		s.OptKeys.EntrypointLocalNames = slices.Clone(s.OptKeys.EntrypointLocalNames)
		for i, name := range s.OptKeys.EntrypointLocalNames {
			if strings.TrimSpace(name) == "" {
				return invalidSpec("OptKeys.EntrypointLocalNames cannot contain empty names")
			}
			s.OptKeys.EntrypointLocalNames[i] = strings.TrimSpace(name)
		}
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

	return nil
}

// InvalidSpecError is returned by Spec.Validate when a required field is
// missing or the specification is otherwise unusable.
type InvalidSpecError struct {
	Msg string
}

func (e *InvalidSpecError) Error() string {
	return "invalid frontend spec: " + e.Msg
}

func invalidSpec(msg string) error {
	return &InvalidSpecError{Msg: msg}
}
