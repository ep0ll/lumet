package llb

import (
	"context"
	"fmt"
	"sync"

	buildkitllb "github.com/moby/buildkit/client/llb"
	"github.com/moby/buildkit/client/llb/sourceresolver"
	digest "github.com/opencontainers/go-digest"
	"go.starlark.net/starlark"
)

// ImageMetaResolverValue wraps BuildKit's ImageMetaResolver interface.
type ImageMetaResolverValue struct {
	resolver buildkitllb.ImageMetaResolver
}

// WrapImageMetaResolver adapts a native BuildKit image metadata resolver.
func WrapImageMetaResolver(resolver buildkitllb.ImageMetaResolver) *ImageMetaResolverValue {
	return &ImageMetaResolverValue{resolver: resolver}
}

// Unwrap returns the native BuildKit metadata resolver.
func (v *ImageMetaResolverValue) Unwrap() buildkitllb.ImageMetaResolver {
	if v == nil {
		return nil
	}
	return v.resolver
}

func (v *ImageMetaResolverValue) String() string        { return "<llb.ImageMetaResolver>" }
func (v *ImageMetaResolverValue) Type() string          { return "llb.ImageMetaResolver" }
func (v *ImageMetaResolverValue) Freeze()               {}
func (v *ImageMetaResolverValue) Truth() starlark.Bool  { return starlark.Bool(v != nil && v.resolver != nil) }
func (v *ImageMetaResolverValue) Hash() (uint32, error) { return 0, fmt.Errorf("unhashable: llb.ImageMetaResolver") }

func (v *ImageMetaResolverValue) Attr(name string) (starlark.Value, error) {
	if v == nil || v.resolver == nil {
		return nil, fmt.Errorf("llb.ImageMetaResolver: nil receiver")
	}
	switch name {
	case "ResolveImageConfig", "resolve_image_config":
		return boundBuiltin("ImageMetaResolver.ResolveImageConfig", func(thread *starlark.Thread, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
			if err := arity("ResolveImageConfig", args, kwargs, 1, 2); err != nil {
				return nil, err
			}
			ref, err := stringAt("ResolveImageConfig", args, 0)
			if err != nil {
				return nil, err
			}
			var opts sourceresolver.Opt
			if len(args) == 2 {
				value, ok := args[1].(*SourceResolverOptValue)
				if !ok || value == nil {
					return nil, fmt.Errorf("ResolveImageConfig: got %s, want llb.SourceResolverOpt", valueType(args[1]))
				}
				opts = value.opt
			}
			resolved, dgst, config, err := v.resolver.ResolveImageConfig(callContext(thread), ref, opts)
			if err != nil {
				return nil, err
			}
			return starlark.Tuple{
				starlark.String(resolved),
				starlark.String(dgst.String()),
				starlark.Bytes(config),
			}, nil
		}), nil
	default:
		return nil, nil
	}
}

func (v *ImageMetaResolverValue) AttrNames() []string {
	return []string{"ResolveImageConfig", "resolve_image_config"}
}

type imageMetaResolverCallback struct {
	thread   *starlark.Thread
	callable starlark.Callable
	mu       sync.Mutex
}

func (r *imageMetaResolverCallback) ResolveImageConfig(ctx context.Context, ref string, opt sourceresolver.Opt) (string, digest.Digest, []byte, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	oldContext := callContext(r.thread)
	r.thread.SetLocal("context", ctx)
	defer r.thread.SetLocal("context", oldContext)

	result, err := starlark.Call(r.thread, r.callable, starlark.Tuple{
		starlark.String(ref),
		WrapSourceResolverOpt(opt),
	}, nil)
	if err != nil {
		return "", "", nil, err
	}

	tuple, ok := result.(starlark.Tuple)
	if !ok || len(tuple) != 3 {
		return "", "", nil, fmt.Errorf("ImageMetaResolver callback must return (resolved_ref, digest, config_bytes)")
	}
	resolvedRef, ok := tuple[0].(starlark.String)
	if !ok {
		return "", "", nil, fmt.Errorf("ImageMetaResolver callback result[0]: got %s, want string", valueType(tuple[0]))
	}
	rawDigest, ok := tuple[1].(starlark.String)
	if !ok {
		return "", "", nil, fmt.Errorf("ImageMetaResolver callback result[1]: got %s, want digest string", valueType(tuple[1]))
	}
	parsedDigest, err := digest.Parse(string(rawDigest))
	if err != nil {
		return "", "", nil, fmt.Errorf("ImageMetaResolver callback returned invalid digest: %w", err)
	}
	config, err := bytesFromValue(tuple[2], "ImageMetaResolver callback config")
	if err != nil {
		return "", "", nil, err
	}
	return string(resolvedRef), parsedDigest, config, nil
}

func bytesFromValue(value starlark.Value, name string) ([]byte, error) {
	switch v := value.(type) {
	case starlark.Bytes:
		return append([]byte(nil), []byte(v)...), nil
	case starlark.String:
		return []byte(string(v)), nil
	default:
		return nil, fmt.Errorf("%s: got %s, want bytes or string", name, valueType(value))
	}
}

func resolverBuiltins() map[string]starlark.Value {
	return map[string]starlark.Value{
		"WithMetaResolver": builtin("WithMetaResolver", withMetaResolverBuiltin),
		"ResolveDigest": boolOptionBuiltin("ResolveDigest", "ImageOption", buildkitllb.ResolveDigest),
		"WithLayerLimit": intOptionBuiltin("WithLayerLimit", "ImageOption", buildkitllb.WithLayerLimit),
		"WithImageChecksum": builtin("WithImageChecksum", withImageChecksumBuiltin),
	}
}

func withMetaResolverBuiltin(_ *starlark.Thread, _ *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	if err := arity("WithMetaResolver", args, kwargs, 1, 1); err != nil {
		return nil, err
	}
	var resolver buildkitllb.ImageMetaResolver
	switch value := args[0].(type) {
	case *ImageMetaResolverValue:
		if value != nil {
			resolver = value.resolver
		}
	case starlark.Callable:
		// A callback accepts (ref, SourceResolverOpt) and returns
		// (resolved_ref, digest_string, config_bytes).
		resolver = &imageMetaResolverCallback{callable: value}
	default:
		return nil, fmt.Errorf("WithMetaResolver: got %s, want llb.ImageMetaResolver or callable", valueType(args[0]))
	}
	if resolver == nil {
		return nil, fmt.Errorf("WithMetaResolver: resolver must not be nil")
	}
	return wrapOption("ImageOption", buildkitllb.WithMetaResolver(resolver)), nil
}

func withImageChecksumBuiltin(_ *starlark.Thread, _ *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	if err := arity("WithImageChecksum", args, kwargs, 1, 1); err != nil {
		return nil, err
	}
	raw, err := stringAt("WithImageChecksum", args, 0)
	if err != nil {
		return nil, err
	}
	parsed, err := digest.Parse(raw)
	if err != nil {
		return nil, fmt.Errorf("WithImageChecksum: %w", err)
	}
	return wrapOption("ImageOption", buildkitllb.WithImageChecksum(parsed)), nil
}

func resolveModeValues() map[string]starlark.Value {
	return map[string]starlark.Value{
		"ResolveModeDefault":   wrapOption("ImageOption", buildkitllb.ResolveModeDefault),
		"ResolveModeForcePull": wrapOption("ImageOption", buildkitllb.ResolveModeForcePull),
		"ResolveModePreferLocal": wrapOption("ImageOption", buildkitllb.ResolveModePreferLocal),
	}
}

var (
	_ starlark.Value    = (*ImageMetaResolverValue)(nil)
	_ starlark.HasAttrs = (*ImageMetaResolverValue)(nil)
	_ buildkitllb.ImageMetaResolver = (*imageMetaResolverCallback)(nil)
)
