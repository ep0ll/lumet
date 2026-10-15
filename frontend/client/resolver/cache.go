// Package resolver provides a caching wrapper around ResolveImageConfig.
//
// source: https://github.com/moby/buildkit/blob/10a81797a96c5a0d23d35c3561e897cd8d69fc3d/frontend/dockerfile/builder/resolvecache.go
package resolver

import (
	"context"
	"fmt"

	"github.com/mitchellh/hashstructure/v2"
	"github.com/moby/buildkit/client/llb/sourceresolver"
	"github.com/moby/buildkit/frontend/gateway/client"
	"github.com/moby/buildkit/util/flightcontrol"
	digest "github.com/opencontainers/go-digest"
)

// ResolveCache wraps a client.Client so ResolveImageConfig calls are
// deduplicated and cached (errors included) via flightcontrol.CachedGroup.
type ResolveCache struct {
	client.Client
	group flightcontrol.CachedGroup[*resolveResult]
}

type resolveResult struct {
	ref  string
	dgst digest.Digest
	cfg  []byte
}

var _ client.Client = (*ResolveCache)(nil)

// NewCache must be used instead of a struct literal so the group is configured once, race-free.
func NewCache(c client.Client) *ResolveCache {
	rc := &ResolveCache{Client: c}
	rc.group.CacheError = true
	return rc
}

func (c *ResolveCache) ResolveImageConfig(ctx context.Context, ref string, opt sourceresolver.Opt) (string, digest.Digest, []byte, error) {
	optHash, err := hashstructure.Hash(opt, hashstructure.FormatV2, nil)
	if err != nil {
		return "", "", nil, err
	}
	key := fmt.Sprintf("%s,%d", ref, optHash)
	res, err := c.group.Do(ctx, key, func(ctx context.Context) (*resolveResult, error) {
		ref, dgst, cfg, err := c.Client.ResolveImageConfig(ctx, ref, opt)
		if err != nil {
			return nil, err
		}
		return &resolveResult{ref: ref, dgst: dgst, cfg: cfg}, nil
	})
	if err != nil {
		return "", "", nil, err
	}
	return res.ref, res.dgst, res.cfg, nil
}
