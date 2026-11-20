// Package maincontext resolves the primary filesystem context for a build.
// Ignore patterns are opt-in and are never loaded implicitly.
package maincontext

import (
	"context"
	"slices"

	"github.com/ep0ll/lumet/context/source"
	"github.com/ep0ll/lumet/context/optionutil"
	"github.com/ep0ll/lumet/frontend"
	"github.com/moby/buildkit/client/llb"
)

// Loader owns the strategy for creating a main build context.
type Loader interface {
	Load(context.Context, frontend.Client, Options) (*llb.State, error)
}

// LoaderFunc adapts a function to Loader.
type LoaderFunc func(context.Context, frontend.Client, Options) (*llb.State, error)

// Load implements Loader.
func (fn LoaderFunc) Load(ctx context.Context, client frontend.Client, opts Options) (*llb.State, error) {
	return fn(ctx, client, opts)
}

// Options configure one MainContext call.
type Options struct {
	source.Options
	Source              source.Spec
	SubDir              string
	IgnorePatterns      []string
	ApplyIgnorePatterns bool
}

// Option configures one MainContext call.
type Option func(*Options)

// WithSourceOptions applies common BuildKit source options.
func WithSourceOptions(options ...source.Option) Option {
	return func(opts *Options) { source.ApplyOptions(&opts.Options, options...) }
}

// WithSource selects the main context source.
func WithSource(spec source.Spec) Option {
	return func(opts *Options) { opts.Source = spec }
}

// WithSubdirectory scopes the resulting context to a directory.
func WithSubdirectory(directory string) Option {
	return func(opts *Options) { opts.SubDir = directory }
}

// WithIgnorePatterns applies the supplied patterns to the main context.
func WithIgnorePatterns(patterns ...string) Option {
	return func(opts *Options) {
		opts.IgnorePatterns = append(opts.IgnorePatterns, patterns...)
		opts.ApplyIgnorePatterns = true
	}
}

// WithoutIgnorePatterns explicitly disables ignore-pattern loading and
// filtering. This is also the default behavior.
func WithoutIgnorePatterns() Option {
	return func(opts *Options) {
		opts.IgnorePatterns = nil
		opts.ApplyIgnorePatterns = false
	}
}

// Apply builds an independent Options value.
func Apply(options ...Option) Options {
	var opts Options
	optionutil.Apply(&opts, options)
	opts.Options = opts.Options.Clone()
	opts.IgnorePatterns = slices.Clone(opts.IgnorePatterns)
	return opts
}
