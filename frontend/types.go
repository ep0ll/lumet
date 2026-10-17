package frontend

import (
	"github.com/ep0ll/lumet/frontend/config"
	gwclient "github.com/moby/buildkit/frontend/gateway/client"
)

type Client interface {
	gwclient.Client
	Config() config.Config
}
