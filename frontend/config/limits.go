package config

import (
	"strconv"

	"github.com/docker/go-units"
	"github.com/moby/buildkit/frontend/gateway/client"
	"github.com/moby/buildkit/solver/pb"
)

func ParseShmSize(bopts client.BuildOpts, cfg *Config) error {
	v := bopts.Opts[keyShmSize]
	if v == "" {
		return nil
	}
	size, err := strconv.ParseInt(v, 10, 64)
	if err != nil {
		return err
	}
	cfg.ShmSize = size
	return nil
}

func ParseUlimits(bopts client.BuildOpts, cfg *Config) error {
	var limits []*pb.Ulimit
	err := eachCSVField(bopts.Opts[keyUlimit], func(field string) error {
		u, err := units.ParseUlimit(field)
		if err != nil {
			return err
		}
		limits = append(limits, &pb.Ulimit{Name: u.Name, Soft: u.Soft, Hard: u.Hard})
		return nil
	})
	if err != nil {
		return err
	}
	cfg.Ulimits = limits
	return nil
}

func ParseCgroupParent(bopts client.BuildOpts, cfg *Config) error {
	cfg.CgroupParent = bopts.Opts[keyCgroupParent]
	return nil
}
