package frontend

import gwclient "github.com/moby/buildkit/frontend/gateway/client"

// New is retained as the short constructor name from the original client
// package API. NewClient is preferred in new code.
//
// Deprecated: use NewClient.
func New(gw gwclient.Client, opts ...Option) (Client, error) {
	return NewClient(gw, opts...)
}
