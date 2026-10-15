package client

import (
	"maps"
	"slices"
	"strings"

	"github.com/moby/buildkit/frontend/gateway/client"
	"github.com/moby/buildkit/util/apicaps"
	"github.com/pkg/errors"
)

// cloneBuildOpts copies the mutable parts (Opts, Workers) so the Client can
// edit its options without touching the gateway's originals.
func cloneBuildOpts(src client.BuildOpts) client.BuildOpts {
	dst := src
	dst.Opts = maps.Clone(src.Opts)
	if dst.Opts == nil {
		dst.Opts = map[string]string{}
	}
	dst.Workers = slices.Clone(src.Workers)
	return dst
}

// SetOpt sets (or overwrites) a build option. Call InitConfig to re-parse.
func (c *Client) SetOpt(key, value string) error {
	c.buildOpts.Opts[key] = value
	return nil
}

// DelOpt deletes the given options; keys that were not set are reported
// after all deletions have been applied.
func (c *Client) DelOpt(keys ...string) error {
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

func (c *Client) WithSession(id string)   { c.buildOpts.SessionID = id }
func (c *Client) Caps() apicaps.CapSet    { return c.buildOpts.Caps }
func (c *Client) LLBCaps() apicaps.CapSet { return c.buildOpts.LLBCaps }
