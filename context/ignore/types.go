// Package ignore loads and parses frontend exclusion patterns independently
// from build-definition and main-context loading.
package ignore

import (
	"context"
	"slices"

	"github.com/ep0ll/lumet/context/llbutil"
	"github.com/ep0ll/lumet/context/source"
	"github.com/ep0ll/lumet/frontend"
)

// Definition is the part of a build definition needed by the default provider.
type Definition interface {
	IgnorePatterns(context.Context) ([]string, error)
}

// Loader owns the strategy for obtaining ignore patterns.
type Loader interface {
	Load(context.Context, frontend.Client, Options) ([]string, error)
}

// LoaderFunc adapts a function to Loader.
type LoaderFunc func(context.Context, frontend.Client, Options) ([]string, error)

// Load implements Loader.
func (fn LoaderFunc) Load(ctx context.Context, client frontend.Client, opts Options) ([]string, error) {
	return fn(ctx, client, opts)
}

// Parser parses the contents of one ignore file.
type Parser = llbutil.IgnoreParser

// Options describe an independent ignore-pattern request. Definition can be
// set to read patterns associated with an already-loaded build definition;
// Source and Files select a wholly independent ignore-file source instead.
type Options struct {
	source.Options
	Definition Definition
	Source     source.Spec
	Files      []string
	Parser     Parser
}

// Option configures one IgnorePatterns call.
type Option func(*Options)

// WithSourceOptions applies common BuildKit source options.
func WithSourceOptions(options ...source.Option) Option {
	return func(opts *Options) { source.ApplyOptions(&opts.Options, options...) }
}

// WithDefinition obtains patterns from a previously loaded definition.
func WithDefinition(definition Definition) Option {
	return func(opts *Options) { opts.Definition = definition }
}

// WithSource selects an independent source containing ignore files.
func WithSource(spec source.Spec) Option {
	return func(opts *Options) { opts.Source = spec }
}

// WithFiles sets candidate ignore-file paths in priority order.
func WithFiles(paths ...string) Option {
	return func(opts *Options) { opts.Files = append(opts.Files, paths...) }
}

// WithParser replaces the standard Moby ignore-file parser.
func WithParser(parser Parser) Option {
	return func(opts *Options) { opts.Parser = parser }
}

// Apply builds an independent Options value.
func Apply(options ...Option) Options {
	var opts Options
	for _, option := range options {
		if option != nil {
			option(&opts)
		}
	}
	opts.Options = opts.Options.Clone()
	opts.Files = slices.Clone(opts.Files)
	return opts
}
