package config

import "github.com/moby/buildkit/frontend/gateway/client"

// Parser fills one aspect of cfg from the build options.
// Parsers must not mutate bopts.
type Parser func(bopts client.BuildOpts, cfg *Config) error

// Defaults returns a fresh copy of the default parser list.
// Order matters: ParseMultiPlatform needs ParseTargetPlatforms first.
func Defaults() []Parser {
	return []Parser{
		ParseBuildPlatforms,
		ParseTargetPlatforms,
		ParseMultiPlatform,
		ParseResolveMode,
		ParseExtraHosts,
		ParseShmSize,
		ParseUlimits,
		ParseNetMode,
		ParseSourceDateEpoch,
		ParseCacheImports,
		ParseSBOM,
		ParseBuildArgs,
		ParseLabels,
		ParseCacheMountNS,
		ParseCgroupParent,
		ParseTarget,
		ParseHostname,
		ParseLinter,
	}
}

// Parse runs parsers (Defaults when none are given) over bopts and returns
// the resulting Config. Nothing is returned on error.
func Parse(bopts client.BuildOpts, parsers ...Parser) (Config, error) {
	if len(parsers) == 0 {
		parsers = Defaults()
	}
	var cfg Config
	for i := 0; i < len(parsers); i++ {
		if err := parsers[i](bopts, &cfg); err != nil {
			return Config{}, err
		}
	}
	return cfg, nil
}
