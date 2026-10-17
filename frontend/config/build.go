package config

import (
	"strconv"
	"time"

	"github.com/moby/buildkit/client/llb"
	"github.com/moby/buildkit/frontend/gateway/client"
	"github.com/moby/buildkit/solver/pb"
	"github.com/pkg/errors"
)

func ParseTarget(bopts client.BuildOpts, cfg *Config) error {
	cfg.Target = bopts.Opts[KeyTarget]
	return nil
}

func ParseBuildArgs(bopts client.BuildOpts, cfg *Config) error {
	cfg.BuildArgs = filterPrefix(bopts.Opts, buildArgPrefix)
	return nil
}

func ParseLabels(bopts client.BuildOpts, cfg *Config) error {
	cfg.Labels = filterPrefix(bopts.Opts, labelPrefix)
	return nil
}

func ParseCacheMountNS(bopts client.BuildOpts, cfg *Config) error {
	cfg.CacheIDNamespace = bopts.Opts[keyCacheNSArg]
	return nil
}

func ParseResolveMode(bopts client.BuildOpts, cfg *Config) error {
	switch v := bopts.Opts[keyImageResolveMode]; v {
	case pb.AttrImageResolveModeDefault, "":
		cfg.ImageResolveMode = llb.ResolveModeDefault
	case pb.AttrImageResolveModeForcePull:
		cfg.ImageResolveMode = llb.ResolveModeForcePull
	case pb.AttrImageResolveModePreferLocal:
		cfg.ImageResolveMode = llb.ResolveModePreferLocal
	default:
		return errors.Errorf("invalid image-resolve-mode: %s", v)
	}
	return nil
}

func ParseSourceDateEpoch(bopts client.BuildOpts, cfg *Config) error {
	v := bopts.Opts[keySourceDateEpoch]
	if v == "" {
		return nil
	}
	sde, err := strconv.ParseInt(v, 10, 64)
	if err != nil {
		return errors.Wrapf(err, "invalid SOURCE_DATE_EPOCH: %s", v)
	}
	tm := time.Unix(sde, 0).UTC()
	cfg.Epoch = &tm
	return nil
}
