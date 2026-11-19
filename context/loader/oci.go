package loader

import (
	"context"
	"strings"

	"github.com/distribution/reference"
	"github.com/moby/buildkit/client/llb"
	"github.com/moby/buildkit/client/llb/sourceresolver"
	dockerspec "github.com/moby/docker-image-spec/specs-go/v1"
	"github.com/pkg/errors"
)

func loadOCI(ctx context.Context, refSpec string, req Request) (*llb.State, *dockerspec.DockerOCIImage, error) {
	refSpec = strings.TrimPrefix(refSpec, "//")

	ref, err := reference.Parse(refSpec)
	if err != nil {
		return nil, nil, errors.Wrapf(err, "invalid oci-layout reference %q", refSpec)
	}
	named, ok := ref.(reference.Named)
	if !ok {
		return nil, nil, errors.Errorf("oci-layout reference %q has no name", ref)
	}
	digested, ok := named.(reference.Digested)
	if !ok {
		return nil, nil, errors.Errorf("oci-layout reference %q has no digest", named)
	}

	dummy, err := reference.ParseNormalizedNamed(req.Name)
	if err != nil {
		return nil, nil, errors.Wrapf(err, "could not build dummy ref from %q", req.Name)
	}
	dummy, err = reference.WithDigest(dummy, digested.Digest())
	if err != nil {
		return nil, nil, err
	}

	_, dgst, cfg, err := req.Client.ResolveImageConfig(ctx, dummy.String(), sourceresolver.Opt{
		LogName:  label(req.NameWithPlatform, "load metadata for "+dummy.String()),
		OCILayoutOpt: &sourceresolver.ResolveOCILayoutOpt{
			Store: sourceresolver.ResolveImageConfigOptStore{
				SessionID: req.Client.BuildOpts().SessionID,
				StoreID:   named.Name(),
			},
			Platform: req.Opt.Platform,
		},
	})
	if err != nil {
		return nil, nil, err
	}

	opts := []llb.OCILayoutOption{
		llb.WithCustomName(label(req.NameWithPlatform, "OCI load from client")),
		llb.OCIStore(req.Client.BuildOpts().SessionID, named.Name()),
	}
	if req.Opt.Platform != nil {
		opts = append(opts, llb.Platform(*req.Opt.Platform))
	}

	st := llb.OCILayout(dummy.String(), opts...)
	return attachConfig(st, cfg, dgst, req)
}
