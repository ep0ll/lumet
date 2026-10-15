package config

import (
	"strconv"
	"strings"

	"github.com/containerd/platforms"
	"github.com/moby/buildkit/frontend/gateway/client"
	ocispecs "github.com/opencontainers/image-spec/specs-go/v1"
	"github.com/pkg/errors"
)

func ParseBuildPlatforms(bopts client.BuildOpts, cfg *Config) error {
	var p ocispecs.Platform
	if len(bopts.Workers) > 0 && len(bopts.Workers[0].Platforms) > 0 {
		p = bopts.Workers[0].Platforms[0]
	} else {
		p = platforms.Normalize(platforms.DefaultSpec())
	}
	cfg.BuildPlatforms = []ocispecs.Platform{p}
	return nil
}

func ParseTargetPlatforms(bopts client.BuildOpts, cfg *Config) error {
	v := bopts.Opts[keyTargetPlatform]
	if v == "" {
		return nil
	}
	pp, err := parsePlatforms(v)
	if err != nil {
		return err
	}
	cfg.TargetPlatforms = pp
	return nil
}

func parsePlatforms(v string) ([]ocispecs.Platform, error) {
	var pp []ocispecs.Platform
	for _, s := range strings.Split(v, ",") {
		p, err := platforms.Parse(s)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to parse target platform %s", s)
		}
		pp = append(pp, platforms.Normalize(p))
	}
	return pp, nil
}

// ParseMultiPlatform requires cfg.TargetPlatforms to be set already.
// The BUILDKIT_MULTI_PLATFORM build-arg takes precedence over the plain option.
func ParseMultiPlatform(bopts client.BuildOpts, cfg *Config) error {
	multi := len(cfg.TargetPlatforms) > 1

	v := bopts.Opts[keyMultiPlatform]
	if arg := bopts.Opts[keyMultiPlatformArg]; arg != "" {
		v = arg
	}
	if v != "" {
		b, err := strconv.ParseBool(v)
		if err != nil {
			return errors.Errorf("invalid boolean value for multi-platform: %s", v)
		}
		if !b && multi {
			return errors.Errorf("conflicting config: returning multiple target platforms is not allowed")
		}
		multi = b
	}
	cfg.MultiPlatformRequested = multi
	return nil
}
