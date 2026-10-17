package frontend

import "github.com/ep0ll/lumet/frontend/config"

type settings struct {
	parsers       []config.Parser
	noResolveCache bool
}

// Option customises New.
type Option func(*settings)

// WithParsers replaces the default config parsers.
func WithParsers(parsers ...config.Parser) Option {
	return func(s *settings) { s.parsers = parsers }
}

// WithoutResolveCache disables ResolveImageConfig caching.
func WithoutResolveCache() Option {
	return func(s *settings) { s.noResolveCache = true }
}
