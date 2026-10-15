package config

import (
	"net"
	"strings"

	"github.com/moby/buildkit/client/llb"
	"github.com/moby/buildkit/frontend/gateway/client"
	"github.com/pkg/errors"
)

func ParseExtraHosts(bopts client.BuildOpts, cfg *Config) error {
	hosts, err := parseExtraHosts(bopts.Opts[keyGlobalAddHosts])
	if err != nil {
		return err
	}
	cfg.ExtraHosts = hosts
	return nil
}

func parseExtraHosts(v string) ([]llb.HostIP, error) {
	var hosts []llb.HostIP
	err := eachCSVField(v, func(field string) error {
		key, val, ok := strings.Cut(strings.ToLower(field), "=")
		if !ok {
			return errors.Errorf("invalid key-value pair %s", field)
		}
		ip := net.ParseIP(val)
		if ip == nil {
			return errors.Errorf("failed to parse IP %s", val)
		}
		hosts = append(hosts, llb.HostIP{Host: key, IP: ip})
		return nil
	})
	return hosts, err
}

func ParseNetMode(bopts client.BuildOpts, cfg *Config) error {
	switch v := bopts.Opts[keyForceNetwork]; v {
	case "none":
		cfg.NetworkMode = llb.NetModeNone
	case "host":
		cfg.NetworkMode = llb.NetModeHost
	case "sandbox", "":
		cfg.NetworkMode = llb.NetModeSandbox
	default:
		return errors.Errorf("invalid netmode %s", v)
	}
	return nil
}

// ParseHostname prefers the BUILDKIT_SANDBOX_HOSTNAME build-arg over the plain option.
func ParseHostname(bopts client.BuildOpts, cfg *Config) error {
	cfg.Hostname = bopts.Opts[keyHostname]
	if arg := bopts.Opts[keyHostnameArg]; arg != "" {
		cfg.Hostname = arg
	}
	return nil
}
