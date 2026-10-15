package config

import (
	"github.com/moby/buildkit/frontend/dockerfile/linter"
	"github.com/moby/buildkit/frontend/gateway/client"
	"github.com/pkg/errors"
)

func ParseLinter(bopts client.BuildOpts, cfg *Config) error {
	v, ok := bopts.Opts[keyFrontendLintArg]
	if !ok {
		return nil
	}
	lc, err := linter.ParseLintOptions(v)
	if err != nil {
		return errors.Wrapf(err, "failed to parse %s", keyFrontendLintArg)
	}
	cfg.LinterConfig = lc
	return nil
}
