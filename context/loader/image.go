package loader

import (
	"context"
	"encoding/json"
	"strings"

	"github.com/dexnore/dexfile"
	"github.com/distribution/reference"
	"github.com/moby/buildkit/client/llb"
	"github.com/moby/buildkit/client/llb/sourceresolver"
	"github.com/moby/buildkit/util/imageutil"
	"github.com/opencontainers/go-digest"
	"github.com/pkg/errors"

	dockerspec "github.com/moby/docker-image-spec/specs-go/v1"
)

func loadImage(ctx context.Context, ref string, req Request) (*llb.State, *dockerspec.DockerOCIImage, error) {
	ref = strings.TrimPrefix(ref, "//")
	if ref == dexfile.EmptyImageName {
		state := llb.Scratch()
		return &state, nil, nil
	}

	named, err := reference.ParseNormalizedNamed(ref)
	if err != nil {
		return nil, nil, errors.Wrapf(err, "invalid image reference %q", ref)
	}
	named = reference.TagNameOnly(named)

	resolved, dgst, cfg, err := req.Client.ResolveImageConfig(ctx, named.String(), sourceresolver.Opt{
		LogName: label(req.NameWithPlatform, "load metadata for "+ref),
		ImageOpt: &sourceresolver.ResolveImageOpt{
			ResolveMode: req.Opt.ResolveMode,
			Platform:    req.Opt.Platform,
		},
	})
	if err != nil {
		return redirectIfNonImage(err, req)
	}

	return attachConfig(llb.Image(resolved, imageOptions(req, ref)...), cfg, dgst, req)
}

func redirectIfNonImage(err error, req Request) (*llb.State, *dockerspec.DockerOCIImage, error) {
	var redirect *imageutil.ResolveToNonImageError
	if !errors.As(err, &redirect) {
		return nil, nil, err
	}
	if req.Depth >= maxContextDepth {
		return nil, nil, errors.Errorf("context redirect limit (%d) exceeded for %s", maxContextDepth, req.NameWithPlatform)
	}

	before, after, ok := strings.Cut(redirect.Updated, "://")
	if !ok || before == "" || after == "" {
		return nil, nil, errors.Errorf("could not parse redirected context reference %q", redirect.Updated)
	}
	newInput := before + ":" + after

	if err := req.Client.SetOpt("context:"+req.NameWithPlatform, newInput); err != nil {
		return nil, nil, errors.Wrap(err, "failed to update redirected context option")
	}
	if req.Retry == nil {
		return nil, nil, errors.New("context redirect requires a retry callback")
	}
	return req.Retry(newInput)
}

func imageOptions(req Request, ref string) []llb.ImageOption {
	opts := []llb.ImageOption{llb.WithCustomName(label(req.NameWithPlatform, ref))}
	if req.Opt.Platform != nil {
		opts = append(opts, llb.Platform(*req.Opt.Platform))
	}
	return opts
}

// attachConfig validates the image configuration before attaching it to the
// LLB state. It is shared by image and OCI-layout loaders.
func attachConfig(state llb.State, cfg []byte, dgst digest.Digest, req Request) (*llb.State, *dockerspec.DockerOCIImage, error) {
	if len(cfg) == 0 {
		return nil, nil, errors.New("resolved image has an empty image configuration")
	}
	var img dockerspec.DockerOCIImage
	if err := json.Unmarshal(cfg, &img); err != nil {
		return nil, nil, errors.Wrap(err, "failed to parse image configuration")
	}
	img.Created = nil

	state, err := state.WithImageConfig(cfg)
	if err != nil {
		return nil, nil, errors.Wrap(err, "failed to attach image configuration")
	}
	if req.Opt.CaptureDigest != nil {
		*req.Opt.CaptureDigest = dgst
	}
	return &state, &img, nil
}
