package loader

import (
	"context"
	"encoding/json"
	"strings"

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
		st := llb.Scratch()
		return &st, nil, nil
	}

	named, err := reference.ParseNormalizedNamed(ref)
	if err != nil {
		return nil, nil, err
	}
	named = reference.TagNameOnly(named)

	resolved, dgst, cfg, err := req.Client.ResolveImageConfig(ctx, named.String(), sourceresolver.Opt{
		LogName:  label(req.NameWithPlatform, "load metadata for "+ref),
		ImageOpt: &sourceresolver.ResolveImageOpt{ResolveMode: req.Opt.ResolveMode, Platform: req.Opt.Platform},
	})
	if err != nil {
		return redirectIfNonImage(err, req)
	}

	return attachConfig(llb.Image(resolved, imageOptions(req, ref)...), cfg, dgst, req)
}

func redirectIfNonImage(err error, req Request) (*llb.State, *dockerspec.DockerOCIImage, error) {
	var e *imageutil.ResolveToNonImageError
	if !errors.As(err, &e) {
		return nil, nil, err
	}

	before, after, ok := strings.Cut(e.Updated, "://")
	if !ok {
		return nil, nil, errors.Errorf("could not parse redirected ref %q", e.Updated)
	}

	newInput := before + ":" + after
	req.Client.SetOpt("context:"+req.NameWithPlatform, newInput)

	if req.Retry == nil {
		return nil, nil, errors.New("retry callback required")
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

// attachConfig is shared by image and oci loaders.
func attachConfig(st llb.State, cfg []byte, dgst digest.Digest, req Request) (*llb.State, *dockerspec.DockerOCIImage, error) {
	var img dockerspec.DockerOCIImage
	if err := json.Unmarshal(cfg, &img); err != nil {
		return nil, nil, err
	}
	img.Created = nil

	st, err := st.WithImageConfig(cfg)
	if err != nil {
		return nil, nil, err
	}
	if req.Opt.CaptureDigest != nil {
		*req.Opt.CaptureDigest = dgst
	}
	return &st, &img, nil
}
