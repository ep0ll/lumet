// Package result orchestrates concurrent multi-platform image builds for
// BuildKit frontends. Callers depend only on the Builder and Result interfaces.
package result

import (
	"context"
	"encoding/json"
	"fmt"
	"slices"

	"github.com/containerd/platforms"
	"github.com/ep0ll/lumet/frontend/client/config"
	"github.com/moby/buildkit/exporter/containerimage/exptypes"
	"github.com/moby/buildkit/frontend/gateway/client"
	"github.com/moby/buildkit/solver/result"
	dockerspec "github.com/moby/docker-image-spec/specs-go/v1"
	ocispecs "github.com/opencontainers/image-spec/specs-go/v1"
	"github.com/pkg/errors"
	"golang.org/x/sync/errgroup"
)

// TargetPlatform pairs an OCI platform with its original index so callers
// can correlate results when needed.
type TargetPlatform struct {
	Platform *ocispecs.Platform
	Index    int
}

// Requested returns the platform to build for, resolving nil to the host
// default.
func (t TargetPlatform) Requested() ocispecs.Platform {
	if t.Platform != nil {
		return *t.Platform
	}
	return platforms.DefaultSpec()
}

// BuildFunc is the per-platform build callback.
type BuildFunc func(ctx context.Context, target TargetPlatform) (
	ref client.Reference,
	img *dockerspec.DockerOCIImage,
	baseImg *dockerspec.DockerOCIImage,
	err error,
)

// Outcome is the output of one BuildFunc invocation.
type Outcome struct {
	Target    TargetPlatform
	Ref       client.Reference
	Image     *dockerspec.DockerOCIImage
	BaseImage *dockerspec.DockerOCIImage
}

// Builder is the entry point for multi-platform builds.
type Builder interface {
	Build(ctx context.Context, fn BuildFunc) (Result, error)
}

// Result is returned by Build and lets the caller add attestations
// and perform per-platform work before finalising.
type Result interface {
	Finalize() (*client.Result, error)
	AddAttestation(key string, att result.Attestation[client.Reference])
	EachPlatform(ctx context.Context, fn func(ctx context.Context, platform exptypes.Platform) error) error
}

// builder accumulates the final client.Result and platform index.
type builder struct {
	result *client.Result
	cfg    config.Config
	plats  *exptypes.Platforms
}

// New creates a ready-to-use Builder.
func New(cfg config.Config) Builder {
	return &builder{
		result: client.NewResult(),
		cfg:    cfg,
		plats: &exptypes.Platforms{
			Platforms: make([]exptypes.Platform, 0, len(cfg.TargetPlatforms)),
		},
	}
}

// Build runs fn concurrently for every target platform and returns a
// Result that can still be extended (attestations, extra meta, etc.).
func (b *builder) Build(ctx context.Context, fn BuildFunc) (Result, error) {
	targets := ResolveTargets(b.cfg.TargetPlatforms)

	outcomes, err := RunBuilds(ctx, targets, fn)
	if err != nil {
		return nil, err
	}

	b.plats.Platforms = make([]exptypes.Platform, len(targets))
	for _, o := range outcomes {
		expPlat, err := AddOutcome(b.result, b.cfg.MultiPlatformRequested, o)
		if err != nil {
			return nil, err
		}
		b.plats.Platforms[o.Target.Index] = expPlat
	}
	return b, nil
}

// ResolveTargets builds the target list, falling back to the default
// platform when none are requested.
func ResolveTargets(ps []ocispecs.Platform) []TargetPlatform {
	targets := make([]TargetPlatform, 0, len(ps))
	for i, p := range ps {
		targets = append(targets, TargetPlatform{
			Platform: &p,
			Index:    i,
		})
	}
	if len(targets) == 0 {
		targets = append(targets, TargetPlatform{
			Platform: nil,
			Index:    0,
		})
	}
	return targets
}

// RunBuilds runs fn concurrently for every target and returns the outcomes
// in target order.
func RunBuilds(ctx context.Context, targets []TargetPlatform, fn BuildFunc) ([]Outcome, error) {
	outcomes := make([]Outcome, len(targets))

	eg, ctx := errgroup.WithContext(ctx)
	for i, target := range targets {
		eg.Go(func() error {
			ref, img, baseImg, err := fn(ctx, target)
			if err != nil {
				return err
			}
			outcomes[i] = Outcome{Target: target, Ref: ref, Image: img, BaseImage: baseImg}
			return nil
		})
	}

	if err := eg.Wait(); err != nil {
		return nil, err
	}
	return outcomes, nil
}

// AddOutcome records one build outcome into res and returns its exporter
// platform. With multi it keys the ref and metadata by platform ID.
func AddOutcome(res *client.Result, multi bool, o Outcome) (exptypes.Platform, error) {
	imgConfig, err := json.Marshal(o.Image)
	if err != nil {
		return exptypes.Platform{}, errors.Wrap(err, "marshal image config")
	}

	var baseConfig []byte
	if o.BaseImage != nil {
		baseConfig, err = json.Marshal(o.BaseImage)
		if err != nil {
			return exptypes.Platform{}, errors.Wrap(err, "marshal base image config")
		}
	}


	expPlat := ExportPlatform(o.Target.Requested(), o.Image.Platform)

	if multi {
		res.AddRef(expPlat.ID, o.Ref)
		res.AddMeta(fmt.Sprintf("%s/%s", exptypes.ExporterImageConfigKey, expPlat.ID), imgConfig)
		if len(baseConfig) > 0 {
			res.AddMeta(fmt.Sprintf("%s/%s", exptypes.ExporterImageBaseConfigKey, expPlat.ID), baseConfig)
		}
	} else {
		res.SetRef(o.Ref)
		res.AddMeta(exptypes.ExporterImageConfigKey, imgConfig)
		if len(baseConfig) > 0 {
			res.AddMeta(exptypes.ExporterImageBaseConfigKey, baseConfig)
		}
	}

	return expPlat, nil
}

// AddAttestation implements Result.
func (b *builder) AddAttestation(key string, att result.Attestation[client.Reference]) {
	b.result.AddAttestation(key, att)
}

// EachPlatform implements Result.
func (b *builder) EachPlatform(ctx context.Context, fn func(ctx context.Context, platform exptypes.Platform) error) error {
	eg, ctx := errgroup.WithContext(ctx)
	for _, p := range b.plats.Platforms {
		eg.Go(func() error {
			return fn(ctx, p)
		})
	}
	return eg.Wait()
}

// Finalize implements Result. It writes the platforms index and returns
// the finished client.Result.
func (b *builder) Finalize() (*client.Result, error) {
	err := FinalizePlatforms(b.plats, b.result.AddMeta)
	return b.result, err
}

func FinalizePlatforms(platforms *exptypes.Platforms, attest func(k string, v []byte)) error {
	data, err := json.Marshal(platforms)
	if err != nil {
		return err
	}

	attest(exptypes.ExporterPlatformsKey, data)

	return nil
}

// Compile-time interface checks.
var (
	_ Builder = (*builder)(nil)
	_ Result  = (*builder)(nil)
)

// ExportPlatform builds the exporter platform descriptor.
// For Windows it inherits OSVersion / OSFeatures from the image platform
// when the requested platform left them empty, then re-normalises so the
// ID reflects the final values.
func ExportPlatform(requested, image ocispecs.Platform) exptypes.Platform {
	p := platforms.Normalize(requested)

	if p.OS == "windows" && image.OS == "windows" {
		if p.OSVersion == "" && image.OSVersion != "" {
			p.OSVersion = image.OSVersion
		}
		if p.OSFeatures == nil && len(image.OSFeatures) > 0 {
			p.OSFeatures = slices.Clone(image.OSFeatures)
		}
		p = platforms.Normalize(p)
	}

	return exptypes.Platform{
		ID:       platforms.FormatAll(p),
		Platform: p,
	}
}
