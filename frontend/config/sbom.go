package config

import (
	"github.com/dexnore/dexfile/sbom"
	"github.com/distribution/reference"
	"github.com/moby/buildkit/frontend/attestations"
	"github.com/moby/buildkit/frontend/gateway/client"
	"github.com/pkg/errors"
)

// ParseSBOM source: https://github.com/moby/buildkit/blob/cc0cb089361067b42c639fd85ba44e42fbba15d0/frontend/dockerui/config.go#L258-L284
func ParseSBOM(bopts client.BuildOpts, cfg *Config) error {
	attests, err := attestations.Parse(bopts.Opts)
	if err != nil {
		return err
	}
	attrs, ok := attests[attestations.KeyTypeSbom]
	if !ok {
		return nil
	}

	params := make(map[string]string)
	var ref reference.Named
	for k, v := range attrs {
		if k != "generator" {
			params[k] = v
			continue
		}
		ref, err = reference.ParseNormalizedNamed(v)
		if err != nil {
			return errors.Wrapf(err, "failed to parse sbom scanner %s", v)
		}
		ref = reference.TagNameOnly(ref)
	}
	if ref == nil {
		return errors.Errorf("sbom scanner cannot be empty")
	}

	cfg.SBOM = &sbom.SBOM{Generator: ref.String(), Parameters: params}
	return nil
}
