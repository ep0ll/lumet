// Package context composes independent BuildKit build-definition,
// ignore-pattern, main-context, and named-context loaders.
package context

import (
	"context"
	"fmt"

	"github.com/ep0ll/lumet/context/definition"
	"github.com/ep0ll/lumet/context/ignore"
	"github.com/ep0ll/lumet/context/loader"
	"github.com/ep0ll/lumet/context/maincontext"
	"github.com/ep0ll/lumet/context/namedcontext"
	"github.com/ep0ll/lumet/context/source"
	"github.com/ep0ll/lumet/frontend"
	"github.com/moby/buildkit/client/llb"
)

// Client is a composition root for independent context-loading services.
// It does not maintain a shared BuildContext or force definition, ignore, and
// main-context sources to share a lifecycle.
type Client struct {
	client frontend.Client

	resolver         source.Resolver
	definitionLoader definition.Loader
	ignoreLoader     ignore.Loader
	mainLoader       maincontext.Loader
}

// Option configures the independent loading strategies used by Client.
type Option func(*settings)

type settings struct {
	resolver         source.Resolver
	definitionLoader definition.Loader
	ignoreLoader     ignore.Loader
	mainLoader       maincontext.Loader
}

// WithSourceResolver replaces the default resolver used by built-in loaders.
// A custom Resolver can also be selected per operation.
func WithSourceResolver(resolver source.Resolver) Option {
	return func(opts *settings) { opts.resolver = resolver }
}

// WithDefinitionLoader replaces the BuildDefinition strategy. This is the
// intended extension point for a frontend that already has a parser or a
// custom BuildDefinition function.
func WithDefinitionLoader(loader definition.Loader) Option {
	return func(opts *settings) { opts.definitionLoader = loader }
}

// WithIgnoreLoader replaces the IgnorePatterns strategy independently from
// build-definition and main-context loading.
func WithIgnoreLoader(loader ignore.Loader) Option {
	return func(opts *settings) { opts.ignoreLoader = loader }
}

// WithMainContextLoader replaces the MainContext strategy independently from
// build-definition and ignore-pattern loading.
func WithMainContextLoader(loader maincontext.Loader) Option {
	return func(opts *settings) { opts.mainLoader = loader }
}

// New creates a context client with independent default BuildKit loaders.
func New(client frontend.Client, opts ...Option) (*Client, error) {
	if client == nil {
		return nil, fmt.Errorf("frontend client is nil")
	}

	var configured settings
	for _, opt := range opts {
		if opt != nil {
			opt(&configured)
		}
	}
	if configured.resolver == nil {
		configured.resolver = source.NewRegistry()
	}
	if configured.definitionLoader == nil {
		configured.definitionLoader = definition.BuildKitLoader{Resolver: configured.resolver}
	}
	if configured.ignoreLoader == nil {
		configured.ignoreLoader = ignore.BuildKitLoader{Resolver: configured.resolver}
	}
	if configured.mainLoader == nil {
		configured.mainLoader = maincontext.BuildKitLoader{Resolver: configured.resolver}
	}

	return &Client{
		client:           client,
		resolver:         configured.resolver,
		definitionLoader: configured.definitionLoader,
		ignoreLoader:     configured.ignoreLoader,
		mainLoader:       configured.mainLoader,
	}, nil
}

// BuildDefinition loads a frontend definition from the source and filename
// supplied for this call. It is independent from MainContext.
func (c *Client) BuildDefinition(ctx context.Context, opts ...definition.Option) (definition.Source, error) {
	if c == nil || c.client == nil {
		return nil, fmt.Errorf("context client is nil")
	}
	options := definition.Apply(opts...)
	if options.Resolver == nil {
		options.Resolver = c.resolver
	}
	return c.definitionLoader.Load(ctx, c.client, options)
}

// IgnorePatterns loads patterns either from an existing Definition source or
// from an entirely independent source selected by ignore options.
func (c *Client) IgnorePatterns(ctx context.Context, opts ...ignore.Option) ([]string, error) {
	if c == nil || c.client == nil {
		return nil, fmt.Errorf("context client is nil")
	}
	options := ignore.Apply(opts...)
	if options.Resolver == nil {
		options.Resolver = c.resolver
	}
	return c.ignoreLoader.Load(ctx, c.client, options)
}

// MainContext loads the main filesystem context. It does not load ignore
// patterns; callers opt into filtering by passing maincontext.WithIgnorePatterns.
func (c *Client) MainContext(ctx context.Context, opts ...maincontext.Option) (*llb.State, error) {
	if c == nil || c.client == nil {
		return nil, fmt.Errorf("context client is nil")
	}
	options := maincontext.Apply(opts...)
	if options.Resolver == nil {
		options.Resolver = c.resolver
	}
	return c.mainLoader.Load(ctx, c.client, options)
}

// NamedContext resolves a context registered with BuildKit's named-context
// options. This preserves the existing image, OCI, Git, HTTP, input and local
// named-context loaders.
func (c *Client) NamedContext(name string, opts loader.ContextOpt) (namedcontext.NamedContext, error) {
	if c == nil || c.client == nil {
		return nil, fmt.Errorf("context client is nil")
	}
	return namedcontext.Resolve(name, c.client, opts)
}

// BaseContext resolves a literal base source using named-context semantics.
func (c *Client) BaseContext(name string, opts loader.ContextOpt) (namedcontext.NamedContext, error) {
	if c == nil || c.client == nil {
		return nil, fmt.Errorf("context client is nil")
	}
	return namedcontext.ResolveBase(name, c.client, opts)
}
