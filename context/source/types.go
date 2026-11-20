// Package source resolves BuildKit frontend inputs into reusable LLB states.
package source

import (
	"context"
	"fmt"
	"net/url"
	"slices"
	"strings"
	"sync"

	"github.com/ep0ll/lumet/frontend"
	"github.com/moby/buildkit/client/llb"
	dockerspec "github.com/moby/docker-image-spec/specs-go/v1"
	"github.com/opencontainers/go-digest"
	ocispecs "github.com/opencontainers/image-spec/specs-go/v1"
)

// Kind identifies a source transport. Custom kinds can be registered with
// Registry.Register.
type Kind string

const (
	KindLocal     Kind = "local"
	KindGit       Kind = "git"
	KindHTTP      Kind = "http"
	KindInput     Kind = "input"
	KindImage     Kind = "docker-image"
	KindOCILayout Kind = "oci-layout"
)

// Spec identifies one source independently from the build-definition,
// ignore-file, and main-context loaders.
type Spec struct {
	Kind      Kind
	Reference string
}

// String returns the canonical source specifier.
func (s Spec) String() string {
	if s.Kind == KindHTTP {
		return s.Reference
	}
	return string(s.Kind) + ":" + s.Reference
}

// Validate checks that the source has a kind and reference. The kind need not
// be built in; the selected Registry determines whether it has a handler.
func (s Spec) Validate() error {
	if strings.TrimSpace(string(s.Kind)) == "" {
		return fmt.Errorf("source kind is required")
	}
	if strings.TrimSpace(s.Reference) == "" {
		return fmt.Errorf("source reference is required")
	}
	return nil
}

// Local creates a local BuildKit session source.
func Local(name string) Spec { return Spec{Kind: KindLocal, Reference: name} }

// Git creates a Git source. The reference accepts BuildKit Git URL syntax,
// including refs, checksums, subdirectories, and query options.
func Git(reference string) Spec { return Spec{Kind: KindGit, Reference: reference} }

// HTTP creates an HTTP(S) source.
func HTTP(reference string) Spec { return Spec{Kind: KindHTTP, Reference: reference} }

// Input creates a BuildKit frontend input source.
func Input(name string) Spec { return Spec{Kind: KindInput, Reference: name} }

// Image creates a container image source.
func Image(reference string) Spec { return Spec{Kind: KindImage, Reference: reference} }

// OCI creates a source from a BuildKit client OCI layout store.
func OCI(reference string) Spec { return Spec{Kind: KindOCILayout, Reference: reference} }

func validatedSpec(spec Spec) (Spec, error) {
	if err := spec.Validate(); err != nil {
		return Spec{}, err
	}
	return spec, nil
}

// Parse parses explicit source specifiers such as local:context,
// input:dockerfile, docker-image:alpine, oci-layout:store@sha256:..., or an
// HTTP(S)/Git URL. Bare local names are intentionally not guessed.
func Parse(value string) (Spec, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return Spec{}, fmt.Errorf("source specifier is empty")
	}
	if strings.HasPrefix(value, "git@") || strings.HasPrefix(value, "github.com/") {
		return validatedSpec(Git(value))
	}

	parsed, err := url.Parse(value)
	if err == nil && parsed.Scheme != "" {
		switch strings.ToLower(parsed.Scheme) {
		case "http", "https":
			if parsed.Host == "" {
				return Spec{}, fmt.Errorf("invalid HTTP source %q: host is required", value)
			}
			if strings.HasSuffix(strings.ToLower(parsed.Path), ".git") {
				return Git(value), nil
			}
			return HTTP(value), nil
		case "git", "ssh":
			return validatedSpec(Git(value))
		case "local":
			return validatedSpec(Local(strings.TrimPrefix(strings.TrimPrefix(value, "local:"), "//")))
		case "input":
			return validatedSpec(Input(strings.TrimPrefix(strings.TrimPrefix(value, "input:"), "//")))
		case "docker-image":
			return validatedSpec(Image(strings.TrimPrefix(strings.TrimPrefix(value, "docker-image:"), "//")))
		case "oci-layout":
			return validatedSpec(OCI(strings.TrimPrefix(strings.TrimPrefix(value, "oci-layout:"), "//")))
		}
	}

	scheme, reference, ok := strings.Cut(value, ":")
	if !ok {
		return Spec{}, fmt.Errorf("source %q must use an explicit scheme", value)
	}
	spec := Spec{Kind: Kind(scheme), Reference: strings.TrimPrefix(reference, "//")}
	if err := spec.Validate(); err != nil {
		return Spec{}, err
	}
	return spec, nil
}

// Options configure the BuildKit source resolution shared by definition,
// ignore-file, and main-context loading.
type Options struct {
	Name             string
	NameWithPlatform string
	FollowPaths      []string
	ExcludePatterns  []string
	LocalOptions     []llb.LocalOption
	Resolver         Resolver
	KeepGitDir       bool
	Platform         *ocispecs.Platform
	ResolveMode      string
	CaptureDigest    *digest.Digest
	SharedKey        string
}

// Option configures common source-resolution options.
type Option func(*Options)

// ApplyOptions applies source options to an existing options value.
func ApplyOptions(target *Options, options ...Option) {
	for _, option := range options {
		if option != nil {
			option(target)
		}
	}
}

// WithName sets the BuildKit diagnostic name.
func WithName(name string) Option {
	return func(opts *Options) { opts.Name = name }
}

// WithNameWithPlatform sets the diagnostic name for a platform-scoped source.
func WithNameWithPlatform(name string) Option {
	return func(opts *Options) { opts.NameWithPlatform = name }
}

// WithFollowPaths limits a local source to the supplied source-relative paths.
func WithFollowPaths(paths ...string) Option {
	return func(opts *Options) { opts.FollowPaths = append(opts.FollowPaths, paths...) }
}

// WithExcludePatterns excludes source-relative paths without loading an
// ignore-file source.
func WithExcludePatterns(patterns ...string) Option {
	return func(opts *Options) { opts.ExcludePatterns = append(opts.ExcludePatterns, patterns...) }
}

// WithLocalOptions appends BuildKit local-source options.
func WithLocalOptions(localOpts ...llb.LocalOption) Option {
	return func(opts *Options) { opts.LocalOptions = append(opts.LocalOptions, localOpts...) }
}

// WithResolver replaces the source resolver for this operation.
func WithResolver(resolver Resolver) Option {
	return func(opts *Options) { opts.Resolver = resolver }
}

// WithKeepGitDir retains .git for Git-backed sources.
func WithKeepGitDir(keep bool) Option {
	return func(opts *Options) { opts.KeepGitDir = keep }
}

// WithPlatform selects the target platform for image and OCI sources.
func WithPlatform(platform ocispecs.Platform) Option {
	return func(opts *Options) { opts.Platform = ClonePlatform(platform) }
}

// WithResolveMode sets the image resolve mode.
func WithResolveMode(mode string) Option {
	return func(opts *Options) { opts.ResolveMode = mode }
}

// WithCaptureDigest stores the resolved image digest in the provided pointer.
func WithCaptureDigest(digest *digest.Digest) Option {
	return func(opts *Options) { opts.CaptureDigest = digest }
}

// WithSharedKey differentiates otherwise identical local source operations.
func WithSharedKey(key string) Option {
	return func(opts *Options) { opts.SharedKey = key }
}

// ClonePlatform copies a platform and its mutable OSFeatures slice.
func ClonePlatform(platform ocispecs.Platform) *ocispecs.Platform {
	p := platform
	p.OSFeatures = slices.Clone(platform.OSFeatures)
	return &p
}

// Clone returns a copy whose mutable slices and platform data do not alias
// the original options.
func (o Options) Clone() Options {
	o.FollowPaths = slices.Clone(o.FollowPaths)
	o.ExcludePatterns = slices.Clone(o.ExcludePatterns)
	o.LocalOptions = slices.Clone(o.LocalOptions)
	if o.Platform != nil {
		o.Platform = ClonePlatform(*o.Platform)
	}
	return o
}

// Resolved is the state and metadata returned by a Resolver.
type Resolved struct {
	Spec     Spec
	State    *llb.State
	Filename string
	Image    *dockerspec.DockerOCIImage

	mu     sync.Mutex
	solved *Solved
}

// Resolver resolves a source independently of its eventual use.
type Resolver interface {
	Resolve(context.Context, frontend.Client, Spec, Options) (*Resolved, error)
}

// ResolverFunc adapts a function to Resolver.
type ResolverFunc func(context.Context, frontend.Client, Spec, Options) (*Resolved, error)

// Resolve implements Resolver.
func (fn ResolverFunc) Resolve(ctx context.Context, client frontend.Client, spec Spec, opts Options) (*Resolved, error) {
	return fn(ctx, client, spec, opts)
}

// Handler resolves one source kind. Handlers can be registered on a Registry
// without replacing the default handlers for other source kinds.
type Handler interface {
	Resolve(context.Context, frontend.Client, Spec, Options) (*Resolved, error)
}

// HandlerFunc adapts a function to Handler.
type HandlerFunc func(context.Context, frontend.Client, Spec, Options) (*Resolved, error)

// Resolve implements Handler.
func (fn HandlerFunc) Resolve(ctx context.Context, client frontend.Client, spec Spec, opts Options) (*Resolved, error) {
	return fn(ctx, client, spec, opts)
}
