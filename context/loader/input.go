package loader

import (
	"context"
	"encoding/json"

	"github.com/moby/buildkit/client/llb"
	"github.com/moby/buildkit/exporter/containerimage/exptypes"
	"github.com/pkg/errors"

	dockerspec "github.com/moby/docker-image-spec/specs-go/v1"
)

func loadInput(ctx context.Context, inputRef string, req Request) (*llb.State, *dockerspec.DockerOCIImage, error) {
	inputs, err := req.Client.Inputs(ctx)
	if err != nil {
		return nil, nil, errors.Wrap(err, "failed to load frontend inputs")
	}
	state, ok := inputs[inputRef]
	if !ok {
		return nil, nil, errors.Errorf("invalid input %q for %s", inputRef, req.NameWithPlatform)
	}

	raw, ok := req.Client.BuildOpts().Opts["input-metadata:"+inputRef]
	if !ok || raw == "" {
		return &state, nil, nil
	}

	metadata := make(map[string][]byte)
	if err := json.Unmarshal([]byte(raw), &metadata); err != nil {
		return nil, nil, errors.Wrapf(err, "failed to parse metadata for input %q", inputRef)
	}

	cfg, ok := metadata[exptypes.ExporterImageConfigKey]
	if !ok || len(cfg) == 0 {
		return &state, nil, nil
	}

	var img dockerspec.DockerOCIImage
	if err := json.Unmarshal(cfg, &img); err != nil {
		return nil, nil, errors.Wrapf(err, "failed to parse image config for %s", req.NameWithPlatform)
	}
	state, err = state.WithImageConfig(cfg)
	if err != nil {
		return nil, nil, errors.Wrapf(err, "failed to attach image config for %s", req.NameWithPlatform)
	}
	return &state, &img, nil
}
