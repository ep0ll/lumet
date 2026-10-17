// Package config turns BuildKit frontend options into a typed Config.
// It has no dependency on the dexfile root package.
package config

import (
	"maps"
	"slices"
	"time"

	"github.com/dexnore/dexfile/sbom"
	"github.com/moby/buildkit/client/llb"
	"github.com/moby/buildkit/frontend/dockerfile/linter"
	"github.com/moby/buildkit/frontend/gateway/client"
	"github.com/moby/buildkit/solver/pb"
	ocispecs "github.com/opencontainers/image-spec/specs-go/v1"
)

// Config is the parsed, immutable-by-convention view of the build options.
type Config struct {
	Frontend         string
	Target           string
	Hostname         string
	CgroupParent     string
	CacheIDNamespace string
	Epoch            *time.Time

	BuildArgs map[string]string
	Labels    map[string]string

	ExtraHosts       []llb.HostIP
	NetworkMode      pb.NetMode
	ImageResolveMode llb.ResolveMode
	ShmSize          int64
	Ulimits          []*pb.Ulimit
	Devices          []*pb.CDIDevice

	LinterConfig *linter.Config
	SBOM         *sbom.SBOM

	CacheImports           []client.CacheOptionsEntry
	TargetPlatforms        []ocispecs.Platform
	BuildPlatforms         []ocispecs.Platform
	MultiPlatformRequested bool
}

// Clone returns a copy that shares no mutable slices, maps or pointers
// with c (pb.Ulimit / pb.CDIDevice elements are shared, they are read-only).
func (c Config) Clone() Config {
	ret := c
	if c.Epoch != nil {
		epoch := *c.Epoch
		ret.Epoch = &epoch
	}
	if c.SBOM != nil {
		s := *c.SBOM
		s.Parameters = maps.Clone(c.SBOM.Parameters)
		ret.SBOM = &s
	}
	if c.LinterConfig != nil {
		l := *c.LinterConfig
		ret.LinterConfig = &l
	}
	ret.BuildArgs = maps.Clone(c.BuildArgs)
	ret.Labels = maps.Clone(c.Labels)
	ret.ExtraHosts = slices.Clone(c.ExtraHosts)
	ret.Ulimits = slices.Clone(c.Ulimits)
	ret.Devices = slices.Clone(c.Devices)
	ret.CacheImports = slices.Clone(c.CacheImports)
	ret.TargetPlatforms = slices.Clone(c.TargetPlatforms)
	ret.BuildPlatforms = slices.Clone(c.BuildPlatforms)
	return ret
}
