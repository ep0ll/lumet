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
		return nil, nil, err
	}
	st, ok := inputs[inputRef]
	if !ok {
		return nil, nil, errors.Errorf("invalid input %q for %s", inputRef, req.NameWithPlatform)
	}

	raw, ok := req.Client.BuildOpts().Opts["input-metadata:"+inputRef]
	if !ok {
		return &st, nil, nil
	}

	meta := make(map[string][]byte)
	if err := json.Unmarshal([]byte(raw), &meta); err != nil {
		return nil, nil, errors.Wrap(err, "failed to parse input metadata")
	}

	cfg, ok := meta[exptypes.ExporterImageConfigKey]
	if !ok {
		return &st, nil, nil
	}

	st, err = st.WithImageConfig(cfg)
	if err != nil {
		return nil, nil, err
	}

	var img dockerspec.DockerOCIImage
	if err := json.Unmarshal(cfg, &img); err != nil {
		return nil, nil, errors.Wrapf(err, "failed to parse image config for %s", req.NameWithPlatform)
	}
	return &st, &img, nil
}
