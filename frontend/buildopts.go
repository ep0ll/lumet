package frontend

import (
	"fmt"
	"maps"
	"slices"
	"strings"

	gwclient "github.com/moby/buildkit/frontend/gateway/client"
	"github.com/moby/buildkit/util/apicaps"
	"github.com/pkg/errors"
)

// cloneBuildOpts copies the mutable parts (Opts, Workers) so the Client can
// edit its options without touching the gateway's originals.
func cloneBuildOpts(src gwclient.BuildOpts) gwclient.BuildOpts {
	dst := src
	dst.Opts = maps.Clone(src.Opts)
	if dst.Opts == nil {
		dst.Opts = map[string]string{}
	}
	dst.Workers = slices.Clone(src.Workers)
	return dst
}

// SetOpt sets (or overwrites) a build option. Call InitConfig to re-parse.
func (c *client) SetOpt(key, value string) error {
	c.buildOpts.Opts[key] = value
	return nil
}

// DelOpt deletes the given options; keys that were not set are reported
// after all deletions have been applied.
func (c *client) DelOpt(keys ...string) error {
	var missing []string
	for i := 0; i < len(keys); i++ {
		if _, ok := c.buildOpts.Opts[keys[i]]; !ok {
			missing = append(missing, keys[i])
		}
		delete(c.buildOpts.Opts, keys[i])
	}
	if len(missing) > 0 {
		return errors.Errorf("build options not set: %s", strings.Join(missing, ", "))
	}
	return nil
}

func (c *client) WithSession(id string)   { c.buildOpts.SessionID = id }
func (c *client) Caps() apicaps.CapSet    { return c.buildOpts.Caps }
func (c *client) LLBCaps() apicaps.CapSet { return c.buildOpts.LLBCaps }

// SetOpt returns a new Client that contains the given key/value pair.
// The original client is never mutated.
func SetOpt(c Client, key, value string) Client {
	cl := mustClone(c)
	if cl.buildOpts.Opts == nil {
		cl.buildOpts.Opts = make(map[string]string)
	}
	cl.buildOpts.Opts[key] = value
	return cl
}

// DelOpt returns a new Client with the given keys removed.
// Missing keys are reported after all deletions have been applied.
func DelOpt(c Client, keys ...string) (Client, error) {
	cl := mustClone(c)
	var missing []string
	for _, k := range keys {
		if _, ok := cl.buildOpts.Opts[k]; !ok {
			missing = append(missing, k)
		}
		delete(cl.buildOpts.Opts, k)
	}
	if len(missing) > 0 {
		return cl, fmt.Errorf("build options not set: %v", missing)
	}
	return cl, nil
}

func mustClone(c Client) *client {
	if cl, ok := c.(*client); ok {
		return cl.Clone().(*client)
	}
	// Should never happen with the constructors in this package.
	panic("frontend: unexpected Client implementation")
}
