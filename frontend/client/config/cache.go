package config

import (
	"encoding/json"

	controlapi "github.com/moby/buildkit/api/services/control"
	"github.com/moby/buildkit/frontend/gateway/client"
	"github.com/pkg/errors"
)

func ParseCacheImports(bopts client.BuildOpts, cfg *Config) error {
	v := bopts.Opts[keyCacheImports]
	if v == "" {
		return nil
	}
	var entries []*controlapi.CacheOptionsEntry
	if err := json.Unmarshal([]byte(v), &entries); err != nil {
		return errors.Wrapf(err, "failed to unmarshal %s (%q)", keyCacheImports, v)
	}
	imports := make([]client.CacheOptionsEntry, 0, len(entries))
	for i := 0; i < len(entries); i++ {
		imports = append(imports, client.CacheOptionsEntry{Type: entries[i].Type, Attrs: entries[i].Attrs})
	}
	cfg.CacheImports = imports
	return nil
}
