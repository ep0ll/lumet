package context

import (
	"context"
	"fmt"

	"github.com/ep0ll/lumet/frontend"
	"github.com/moby/buildkit/client/llb"
	"github.com/moby/buildkit/util/flightcontrol"
)

// SourceResolver translates a source identifier into an LLB state. Returning
// a nil ResolvedSource means that the identifier should be treated as a local
// BuildKit session name. Implementations can add frontend-specific transports
// without changing the context orchestration code.
type SourceResolver interface {
	Resolve(context.Context, string, ResolveOptions, frontend.Client) (*ResolvedSource, error)
}

// ResolveOptions carries source-resolution policy shared by the default and
// custom source resolvers.
type ResolveOptions struct {
	KeepGitDir bool
}

// ResolvedSource is a non-local source returned by a SourceResolver.
type ResolvedSource struct {
	State    *llb.State
	Filename string
}

// Option configures the context client.
type Option func(*options)

type options struct {
	resolver SourceResolver
}

// WithSourceResolver replaces the built-in Git/HTTP source resolver.
func WithSourceResolver(resolver SourceResolver) Option {
	return func(o *options) {
		o.resolver = resolver
	}
}

type buildContext struct {
	group    flightcontrol.CachedGroup[*buildContextState]
	spec     frontend.Spec
	client   frontend.Client
	resolver SourceResolver
	initErr  error
}

// New creates a context client using the default BuildKit Git/HTTP resolver.
func New(spec frontend.Spec, client frontend.Client) BuildClient {
	return NewWithOptions(spec, client)
}

// NewWithOptions creates a context client with an optional source resolver.
// The specification is cloned and validated so callers can safely reuse it.
func NewWithOptions(spec frontend.Spec, client frontend.Client, opts ...Option) BuildClient {
	spec = spec.Clone()
	err := spec.Validate()
	if client == nil && err == nil {
		err = fmt.Errorf("frontend client is nil")
	}

	var configured options
	for _, opt := range opts {
		if opt != nil {
			opt(&configured)
		}
	}
	if configured.resolver == nil {
		configured.resolver = BuildKitSourceResolver{}
	}

	return &buildContext{
		spec:     spec,
		client:   client,
		resolver: configured.resolver,
		initErr:  err,
	}
}

// buildContext caches the resolved source names and LLB states for the
// lifetime of this frontend invocation.
func (b *buildContext) buildContext(ctx context.Context) (*buildContextState, error) {
	if b.initErr != nil {
		return nil, b.initErr
	}
	return b.group.Do(ctx, "initcontext", func(ctx context.Context) (*buildContextState, error) {
		return b.initContext(ctx)
	})
}

// DefaultMainContext builds the conventional local context source without
// applying frontend ignore patterns. It is useful for frontends that do not
// use the higher-level BuildClient.
func DefaultMainContext(opts ...llb.LocalOption) *llb.State {
	options := []llb.LocalOption{
		llb.SharedKeyHint("context"),
		llb.WithCustomName("[internal] load build context"),
	}
	options = append(options, opts...)
	state := llb.Local("context", options...)
	return &state
}
