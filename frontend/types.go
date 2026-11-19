package frontend

import (
	"github.com/ep0ll/lumet/frontend/config"
	gwclient "github.com/moby/buildkit/frontend/gateway/client"
)

// Client wraps a BuildKit gateway client with Lumet's parsed configuration
// and local-session/build-option helpers.
type Client interface {
	gwclient.Client
	Config() config.Config
	GetLocalSession(name string) (session string, found bool)
	SetOpt(key, value string) error
	Clone() Client
}
