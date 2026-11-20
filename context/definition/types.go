// Package definition loads frontend build definitions and keeps their source
// maps, diagnostics, and associated ignore-file behavior together.
package definition

import (
	"context"
	"slices"

	"github.com/ep0ll/lumet/context/ignore"
	"github.com/ep0ll/lumet/context/source"
	"github.com/ep0ll/lumet/frontend"
	"github.com/moby/buildkit/client/llb"
	gwclient "github.com/moby/buildkit/frontend/gateway/client"
	"github.com/moby/buildkit/solver/pb"
)

// Source is a loaded build definition with source-map and optional ignore-file
// support.
type Source interface {
	Data() []byte
	Filename() string
	SourceMap() *llb.SourceMap
	Sources() *llb.SourceMap
	Location([]*pb.Range) llb.ConstraintsOpt
	WarnSources(context.Context, string, gwclient.WarnOpts)
	IgnorePatterns(context.Context) ([]string, error)
}

// Loader owns the strategy for loading one build definition.
type Loader interface {
	Load(context.Context, frontend.Client, Options) (Source, error)
}

// LoaderFunc adapts a function to Loader.
type LoaderFunc func(context.Context, frontend.Client, Options) (Source, error)

// Load implements Loader.
func (fn LoaderFunc) Load(ctx context.Context, client frontend.Client, opts Options) (Source, error) {
	return fn(ctx, client, opts)
}

// Parser parses one ignore file into exclusion patterns.
type Parser = ignore.Parser

// Options configure one BuildDefinition call. Source transport and BuildKit
// settings are shared with the other context loaders.
type Options struct {
	source.Options
	Source      source.Spec
	Filename    string
	Language    string
	Fallbacks   []string
	IgnoreFiles []string
	Parser      Parser
}

// Option configures one BuildDefinition call.
type Option func(*Options)

// WithSourceOptions applies common BuildKit source options.
func WithSourceOptions(options ...source.Option) Option {
	return func(opts *Options) { source.ApplyOptions(&opts.Options, options...) }
}

// WithSource selects the independent source that contains the definition.
func WithSource(spec source.Spec) Option {
	return func(opts *Options) { opts.Source = spec }
}

// WithFilename selects the primary definition filename. It may be omitted
// when a source (for example a raw HTTP file) supplies its own filename.
func WithFilename(filename string) Option {
	return func(opts *Options) { opts.Filename = filename }
}

// WithLanguage sets the source-map language identifier.
func WithLanguage(language string) Option {
	return func(opts *Options) { opts.Language = language }
}

// WithFallbacks adds alternative definition paths in priority order.
func WithFallbacks(paths ...string) Option {
	return func(opts *Options) { opts.Fallbacks = append(opts.Fallbacks, paths...) }
}

// WithIgnoreFiles associates ignore-file candidates with this definition.
// The first existing file is used.
func WithIgnoreFiles(paths ...string) Option {
	return func(opts *Options) { opts.IgnoreFiles = append(opts.IgnoreFiles, paths...) }
}

// WithParser replaces the standard Moby ignore-file parser.
func WithParser(parser Parser) Option {
	return func(opts *Options) { opts.Parser = parser }
}

// Apply builds independent options from option functions.
func Apply(options ...Option) Options {
	var opts Options
	for _, option := range options {
		if option != nil {
			option(&opts)
		}
	}
	opts.Options = opts.Options.Clone()
	opts.Fallbacks = slices.Clone(opts.Fallbacks)
	opts.IgnoreFiles = slices.Clone(opts.IgnoreFiles)
	return opts
}
