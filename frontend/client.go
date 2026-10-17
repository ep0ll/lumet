// Package frontend wraps a BuildKit gateway client with parsed frontend
// configuration, per-stage cache control and local session lookup.
// It does not depend on the dexfile root package.
package frontend

import (
	"slices"

	"github.com/ep0ll/lumet/frontend/config"
	"github.com/ep0ll/lumet/frontend/resolver"
	gwclient "github.com/moby/buildkit/frontend/gateway/client"
)

// Client is not safe for concurrent mutation; Clone it per goroutine.
type client struct {
	gwclient.Client // gateway client (resolve-cached unless disabled)

	cfg       config.Config
	buildOpts gwclient.BuildOpts
	noCache   NoCache
	sessions  LocalSessions
	parsers   []config.Parser
	settings  settings
}

// New validates gw's capabilities and parses its build options.
func NewClient(gw gwclient.Client, opts ...Option) (Client, error) {
	var s settings
	for i := 0; i < len(opts); i++ {
		opts[i](&s)
	}

	c := &client{
		Client:    gw,
		buildOpts: cloneBuildOpts(gw.BuildOpts()),
		parsers:   s.parsers,
		settings:  s,
	}
	if !s.noResolveCache {
		c.Client = resolver.NewCache(gw)
	}

	if err := validateMinCaps(c.buildOpts.LLBCaps); err != nil {
		return nil, err
	}
	if err := c.InitConfig(); err != nil {
		return nil, err
	}
	return c, nil
}

// InitConfig re-parses the (possibly mutated) build options. On error the
// previous state is left untouched.
func (c *client) InitConfig() error {
	cfg, err := config.Parse(c.buildOpts, c.parsers...)
	if err != nil {
		return err
	}
	c.cfg = cfg
	c.noCache = ParseNoCache(c.buildOpts.Opts)
	c.sessions = ParseLocalSessions(c.buildOpts.Opts)
	return nil
}

// BuildOpts returns the client's own view of the build options.
func (c *client) BuildOpts() gwclient.BuildOpts { return c.buildOpts }

// Config returns the parsed configuration.
func (c *client) Config() config.Config { return c.cfg }

// IsNoCache reports whether the named stage must bypass the cache.
func (c *client) IsNoCache(stage string) bool { return c.noCache.Matches(stage) }

// GetLocalSession returns the session for the named local context, falling
// back to the build's own session (found == false) when none is registered.
func (c *client) GetLocalSession(name string) (session string, found bool) {
	if id, ok := c.sessions[name]; ok {
		return id, true
	}
	return c.buildOpts.SessionID, false
}

// Clone returns an independent copy sharing the underlying gateway
// connection (and its resolve cache).
func (c *client) Clone() Client {
	return &client{
		Client:    c.Client,
		cfg:       c.cfg.Clone(),
		buildOpts: cloneBuildOpts(c.buildOpts),
		noCache:   c.noCache.clone(),
		sessions:  c.sessions.clone(),
		parsers:   slices.Clone(c.parsers),
		settings:  c.settings,
	}
}
