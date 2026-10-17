package frontend

import (
	"path"
	"strings"
)

// ContextOptKeys holds the frontend option key names used by the Dockerfile
// (dockerui) context loading code. These are the strings that appear in the
// BuildOpts.Opts map (or as --opt on the buildctl / buildx command line).
type ContextOptKeys struct {
	// Filename is the option that overrides the default Dockerfile name
	// (normally "Dockerfile").
	// Example: --opt filename=Dockerfile.dev
	Filename string // "filename"

	// ContextLocalName is the option that overrides the local name used for
	// the main build context (normally "context").
	// Example: --opt contextkey=mycontext
	ContextLocalName string // "contextkey"

	// DockerfileLocalNames lists the option keys that force a particular
	// local name for the Dockerfile source. The first matching key wins.
	// Common values: "dockerfilekey", or a custom frontend-specific key
	// such as "lumetkey".
	DockerfileLocalNames []string // "lumetkey", "dockerfilekey"

	// ContextSubDir is the option that selects a subdirectory inside the
	// main context to treat as the root of the build context.
	// Example: --opt contextsubdir=src
	ContextSubDir string // "contextsubdir"

	// DockerfileSubDir is the option that selects a subdirectory that
	// contains the Dockerfile (when the Dockerfile is not at the root of
	// its local source).
	// Example: --opt lumetsubdir=docker
	DockerfileSubDir string // "lumetsubdir"

	// KeepGitDir is the build-arg key that controls whether a remote Git
	// context retains its .git directory.
	// Example: --opt build-arg:BUILDKIT_CONTEXT_KEEP_GIT_DIR=1
	KeepGitDir string // "build-arg:BUILDKIT_CONTEXT_KEEP_GIT_DIR"
}

// Spec describes a frontend’s conventions for locating the build definition
// (entrypoint) and its associated build context.
//
// It is used by the shared context-loading helpers so that the same code can
// serve Dockerfile, Lumet, Earthfile, or any other similar frontend.
type Spec struct {
	// Name is a short identifier for the language / frontend.
	// It appears in source-map attributes and diagnostic messages.
	// Examples: "Dockerfile", "Lumet", "Earthfile".
	Name string

	// DefaultEntrypoint returns the canonical filename of the build definition
	// for the given configuration (or an empty config string for the default).
	// Examples: "Dockerfile", "Lumetfile", "Earthfile".
	DefaultEntrypoint func(config string) string

	// IgnoreFiles returns the candidate ignore-file names that should be
	// considered for the given entrypoint, in priority order.
	// The first existing file wins. May be nil when the frontend has no
	// ignore-file concept.
	// Example for Dockerfile: []string{".dockerignore"}
	IgnoreFiles func(entrypoint string) []string

	// DefaultContextLocalName is the local name used for the main build
	// context when no override is supplied via options.
	// Conventionally "context".
	DefaultContextLocalName string

	// EntrypointLocalNames lists the candidate local names that may contain
	// the build definition (the entrypoint file).
	// All names except the last are treated as optional overrides that are
	// used only when present in the frontend options; the final name is the
	// unconditional default.
	// Example for Dockerfile: []string{"dockerfile"}
	EntrypointLocalNames []string

	// OptKeys holds the exact option-key strings that the context-loading
	// code will look up in BuildOpts.Opts.
	OptKeys ContextOptKeys
}

// DefaultContextOptKeys returns the standard option key names expected by the
// Dockerfile frontend’s context-loading code.
func DefaultContextOptKeys() ContextOptKeys {
	return ContextOptKeys{
		Filename:             "Lumet",
		ContextLocalName:     "contextkey",
		DockerfileLocalNames: []string{"dockerfilekey"}, // add "lumetkey" (or any other aliases) if needed
		ContextSubDir:        "contextsubdir",
		DockerfileSubDir:     "entrypointsubdir", // or "lumetsubdir" / "dockerfilesubdir" depending on the frontend
		KeepGitDir:           "build-arg:BUILDKIT_CONTEXT_KEEP_GIT_DIR",
	}
}

// LowerBaseFallback is the common "Dockerfile -> dockerfile" fallback.
func LowerBaseFallback(def string) func(string) []string {
	return func(filename string) []string {
		if path.Base(filename) != def {
			return nil
		}
		return []string{path.Join(path.Dir(filename), strings.ToLower(def))}
	}
}

// SuffixIgnore yields "<entrypoint><suffix>", e.g. Dexfile.dexnore.
func SuffixIgnore(suffix string) func(string) []string {
	return func(filename string) []string { return []string{filename + suffix} }
}

// Validate fills sensible defaults and rejects unusable specs.
// It is safe to call multiple times; later calls are no-ops once the
// required fields are already populated.
func (s *Spec) Validate() error {
	if s.Name == "" {
		return invalidSpec("Name is required")
	}
	if s.DefaultEntrypoint == nil {
		return invalidSpec("DefaultEntrypoint is required")
	}

	if s.DefaultContextLocalName == "" {
		s.DefaultContextLocalName = "context"
	}

	if len(s.EntrypointLocalNames) == 0 {
		// Fallback that works for the classic Dockerfile frontend.
		// Frontends that use a different local name should set the field
		// explicitly before calling Validate.
		s.EntrypointLocalNames = []string{"dockerfile"}
	}

	// Ensure the option-key set is complete.  We only replace the whole
	// struct when the primary key is missing; callers that want a partial
	// override can populate the fields they care about.
	if s.OptKeys.Filename == "" {
		s.OptKeys = DefaultContextOptKeys()
	}

	return nil
}

// InvalidSpecError is returned by Spec.Validate when a required field is
// missing or the configuration is otherwise unusable.
type InvalidSpecError struct {
	Msg string
}

func (e *InvalidSpecError) Error() string {
	return "invalid frontend spec: " + e.Msg
}

func invalidSpec(msg string) error {
	return &InvalidSpecError{Msg: msg}
}

