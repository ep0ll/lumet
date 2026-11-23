package llb

import (
	"context"
	"fmt"
	"sync"

	buildkitllb "github.com/moby/buildkit/client/llb"
	"github.com/moby/buildkit/client/llb/sourceresolver"
	digest "github.com/opencontainers/go-digest"
	"go.starlark.net/starlark"
	"google.golang.org/protobuf/proto"
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

func (v *ImageMetaResolverValue) String() string        { return "<llb.image_meta_resolver>" }
func (v *ImageMetaResolverValue) Type() string          { return "llb.image_meta_resolver" }
func (v *ImageMetaResolverValue) Freeze()               {}
func (v *ImageMetaResolverValue) Truth() starlark.Bool  { return starlark.Bool(v != nil && v.resolver != nil) }
func (v *ImageMetaResolverValue) Hash() (uint32, error) { return 0, fmt.Errorf("unhashable: llb.image_meta_resolver") }

func (v *ImageMetaResolverValue) Attr(name string) (starlark.Value, error) {
		if v == nil || v.resolver == nil {
		return nil, fmt.Errorf("llb.image_meta_resolver: nil receiver")
	}
	switch name {
	case "resolve_image_config":
		return boundBuiltin("image_meta_resolver.resolve_image_config", func(thread *starlark.Thread, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
			if err := arity("resolve_image_config", args, kwargs, 1, 2); err != nil {
				return nil, err
			}
			ref, err := stringAt("resolve_image_config", args, 0)
			if err != nil {
				return nil, err
			}
			var opts sourceresolver.Opt
			if len(args) == 2 {
				value, ok := args[1].(*SourceResolverOptValue)
				if !ok || value == nil {
					return nil, fmt.Errorf("resolve_image_config: got %s, want llb.source_resolver_opt", valueType(args[1]))
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
	return SnakeCaseNames([]string{"resolve_image_config", "resolve_image_config"})
}

// SourceResolverOptValue adapts sourceresolver.Opt for callbacks and Go globals.
type SourceResolverOptValue struct {
	opt sourceresolver.Opt
}

// WrapSourceResolverOpt adapts a native sourceresolver.Opt.
func WrapSourceResolverOpt(opt sourceresolver.Opt) *SourceResolverOptValue {
	return &SourceResolverOptValue{opt: opt}
}

// Unwrap returns the native source resolver options.
func (v *SourceResolverOptValue) Unwrap() sourceresolver.Opt {
	if v == nil { return sourceresolver.Opt{} }
	return v.opt
}

func (v *SourceResolverOptValue) String() string { return "<llb.source_resolver_opt>" }
func (v *SourceResolverOptValue) Type() string { return "llb.source_resolver_opt" }
func (v *SourceResolverOptValue) Freeze() {}
func (v *SourceResolverOptValue) Truth() starlark.Bool { return starlark.Bool(v != nil) }
func (v *SourceResolverOptValue) Hash() (uint32, error) { return 0, fmt.Errorf("unhashable: llb.source_resolver_opt") }

func (v *SourceResolverOptValue) Attr(name string) (starlark.Value, error) {
		if v == nil { return nil, fmt.Errorf("llb.source_resolver_opt: nil receiver") }
	switch name {
	case "log_name":
		return starlark.String(v.opt.LogName), nil
	case "image", "image_opt":
		if v.opt.ImageOpt == nil { return starlark.None, nil }
		return &resolveImageOptValue{opt: *v.opt.ImageOpt}, nil
	case "oci_layout", "oci_layout_opt":
		if v.opt.OCILayoutOpt == nil { return starlark.None, nil }
		return &resolveOCILayoutOptValue{opt: *v.opt.OCILayoutOpt}, nil
	case "git", "git_opt":
		if v.opt.GitOpt == nil { return starlark.None, nil }
		return &resolveGitOptValue{opt: *v.opt.GitOpt}, nil
	case "http", "http_opt":
		if v.opt.HTTPOpt == nil { return starlark.None, nil }
		return &resolveHTTPOptValue{opt: *v.opt.HTTPOpt}, nil
	case "source_policies":
		values := make([]starlark.Value, len(v.opt.SourcePolicies))
		for i, policy := range v.opt.SourcePolicies {
			if policy == nil { values[i] = starlark.None } else { values[i] = starlark.Bytes(mustProtoMarshal(policy)) }
		}
		return starlark.NewList(values), nil
	default:
		return nil, nil
	}
}

func (v *SourceResolverOptValue) AttrNames() []string {
	return SnakeCaseNames([]string{"log_name", "image", "oci_layout", "git", "http", "source_policies",
		"log_name", "image_opt", "oci_layout_opt", "git_opt", "http_opt", "source_policies"})
}

type resolveImageOptValue struct { opt sourceresolver.ResolveImageOpt }
func (v *resolveImageOptValue) String() string { return "<sourceresolver.resolve_image_opt>" }
func (v *resolveImageOptValue) Type() string { return "sourceresolver.resolve_image_opt" }
func (v *resolveImageOptValue) Freeze() {}
func (v *resolveImageOptValue) Truth() starlark.Bool { return starlark.Bool(v != nil) }
func (v *resolveImageOptValue) Hash() (uint32,error) { return 0, fmt.Errorf("unhashable: resolve_image_opt") }
func (v *resolveImageOptValue) Attr(name string) (starlark.Value,error) {
		if v == nil { return nil, fmt.Errorf("resolve_image_opt: nil receiver") }
	switch name {
	case "platform":
		if v.opt.Platform == nil { return starlark.None,nil }
		return WrapOCIPlatform(*v.opt.Platform),nil
	case "resolve_mode": return starlark.String(v.opt.ResolveMode),nil
	case "no_config": return starlark.Bool(v.opt.NoConfig),nil
	case "attestation_chain": return starlark.Bool(v.opt.AttestationChain),nil
	case "resolve_attestations": return stringSliceValue(v.opt.ResolveAttestations),nil
	default: return nil,nil
	}
}
func (v *resolveImageOptValue) AttrNames() []string { return SnakeCaseNames([]string{"platform","resolve_mode","no_config","attestation_chain","resolve_attestations","platform","resolve_mode","no_config","attestation_chain","resolve_attestations"}) }

type resolveGitOptValue struct { opt sourceresolver.ResolveGitOpt }
func (v *resolveGitOptValue) String() string { return "<sourceresolver.resolve_git_opt>" }
func (v *resolveGitOptValue) Type() string { return "sourceresolver.resolve_git_opt" }
func (v *resolveGitOptValue) Freeze() {}
func (v *resolveGitOptValue) Truth() starlark.Bool { return starlark.Bool(v != nil) }
func (v *resolveGitOptValue) Hash() (uint32,error) { return 0, fmt.Errorf("unhashable: resolve_git_opt") }
func (v *resolveGitOptValue) Attr(name string) (starlark.Value,error) {
		if name == "return_object" || name == "return_object" { return starlark.Bool(v.opt.ReturnObject),nil }
	return nil,nil
}
func (v *resolveGitOptValue) AttrNames() []string { return SnakeCaseNames([]string{"return_object","return_object"}) }

type resolveHTTPOptValue struct { opt sourceresolver.ResolveHTTPOpt }
func (v *resolveHTTPOptValue) String() string { return "<sourceresolver.resolve_http_opt>" }
func (v *resolveHTTPOptValue) Type() string { return "sourceresolver.resolve_http_opt" }
func (v *resolveHTTPOptValue) Freeze() {}
func (v *resolveHTTPOptValue) Truth() starlark.Bool { return starlark.Bool(v != nil) }
func (v *resolveHTTPOptValue) Hash() (uint32,error) { return 0, fmt.Errorf("unhashable: resolve_http_opt") }
func (v *resolveHTTPOptValue) Attr(name string) (starlark.Value,error) {
		if name == "checksum_req" || name == "checksum_req" {
		if v.opt.ChecksumReq == nil { return starlark.None,nil }
		return &resolveHTTPChecksumRequestValue{request:*v.opt.ChecksumReq},nil
	}
	return nil,nil
}
func (v *resolveHTTPOptValue) AttrNames() []string { return SnakeCaseNames([]string{"checksum_req","checksum_req"}) }

type resolveHTTPChecksumRequestValue struct { request sourceresolver.ResolveHTTPChecksumRequest }
func (v *resolveHTTPChecksumRequestValue) String() string { return "<sourceresolver.resolve_http_checksum_request>" }
func (v *resolveHTTPChecksumRequestValue) Type() string { return "sourceresolver.resolve_http_checksum_request" }
func (v *resolveHTTPChecksumRequestValue) Freeze() {}
func (v *resolveHTTPChecksumRequestValue) Truth() starlark.Bool { return starlark.Bool(v != nil) }
func (v *resolveHTTPChecksumRequestValue) Hash() (uint32,error) { return 0, fmt.Errorf("unhashable: resolve_http_checksum_request") }
func (v *resolveHTTPChecksumRequestValue) Attr(name string) (starlark.Value,error) {
		switch name {
	case "algo": return starlark.MakeInt(int(v.request.Algo)),nil
	case "suffix": return starlark.Bytes(v.request.Suffix),nil
	default: return nil,nil
	}
}
func (v *resolveHTTPChecksumRequestValue) AttrNames() []string { return SnakeCaseNames([]string{"algo","suffix","algo","suffix"}) }

type resolveOCILayoutOptValue struct { opt sourceresolver.ResolveOCILayoutOpt }
func (v *resolveOCILayoutOptValue) String() string { return "<sourceresolver.resolve_oci_layout_opt>" }
func (v *resolveOCILayoutOptValue) Type() string { return "sourceresolver.resolve_oci_layout_opt" }
func (v *resolveOCILayoutOptValue) Freeze() {}
func (v *resolveOCILayoutOptValue) Truth() starlark.Bool { return starlark.Bool(v != nil) }
func (v *resolveOCILayoutOptValue) Hash() (uint32,error) { return 0, fmt.Errorf("unhashable: resolve_oci_layout_opt") }
func (v *resolveOCILayoutOptValue) Attr(name string) (starlark.Value,error) {
		switch name {
	case "platform":
		if v.opt.Platform == nil { return starlark.None,nil }
		return WrapOCIPlatform(*v.opt.Platform),nil
	case "store": return &resolveImageConfigStoreValue{store:v.opt.Store},nil
	default: return nil,nil
	}
}
func (v *resolveOCILayoutOptValue) AttrNames() []string { return SnakeCaseNames([]string{"platform","store","platform","store"}) }

type resolveImageConfigStoreValue struct { store sourceresolver.ResolveImageConfigOptStore }
func (v *resolveImageConfigStoreValue) String() string { return "<sourceresolver.resolve_image_config_opt_store>" }
func (v *resolveImageConfigStoreValue) Type() string { return "sourceresolver.resolve_image_config_opt_store" }
func (v *resolveImageConfigStoreValue) Freeze() {}
func (v *resolveImageConfigStoreValue) Truth() starlark.Bool { return starlark.Bool(v != nil) }
func (v *resolveImageConfigStoreValue) Hash() (uint32,error) { return 0, fmt.Errorf("unhashable: resolve_image_config_opt_store") }
func (v *resolveImageConfigStoreValue) Attr(name string) (starlark.Value,error) {
		switch name {
	case "session_id": return starlark.String(v.store.SessionID),nil
	case "store_id": return starlark.String(v.store.StoreID),nil
	default: return nil,nil
	}
}
func (v *resolveImageConfigStoreValue) AttrNames() []string { return SnakeCaseNames([]string{"session_id","store_id","session_id","store_id"}) }

type protoMarshaler interface { Marshal() ([]byte,error) }
func mustProtoMarshal(m proto.Message) []byte {
	data, err := proto.Marshal(m)
	if err != nil { return nil }
	return data
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
		return "", "", nil, fmt.Errorf("image_meta_resolver callback must return (resolved_ref, digest, config_bytes)")
	}
	resolvedRef, ok := tuple[0].(starlark.String)
	if !ok {
		return "", "", nil, fmt.Errorf("image_meta_resolver callback result[0]: got %s, want string", valueType(tuple[0]))
	}
	rawDigest, ok := tuple[1].(starlark.String)
	if !ok {
		return "", "", nil, fmt.Errorf("image_meta_resolver callback result[1]: got %s, want digest string", valueType(tuple[1]))
	}
	parsedDigest, err := digest.Parse(string(rawDigest))
	if err != nil {
		return "", "", nil, fmt.Errorf("image_meta_resolver callback returned invalid digest: %w", err)
	}
	config, err := bytesFromValue(tuple[2], "image_meta_resolver callback config")
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
		"with_meta_resolver": builtin("with_meta_resolver", withMetaResolverBuiltin),
		"resolve_digest": boolOptionBuiltin("resolve_digest", "image_option", buildkitllb.ResolveDigest),
		"with_layer_limit": intOptionBuiltin("with_layer_limit", "image_option", buildkitllb.WithLayerLimit),
		"with_image_checksum": builtin("with_image_checksum", withImageChecksumBuiltin),
	}
}

func withMetaResolverBuiltin(thread *starlark.Thread, _ *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	if err := arity("with_meta_resolver", args, kwargs, 1, 1); err != nil {
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
		resolver = &imageMetaResolverCallback{thread: thread, callable: value}
	default:
		return nil, fmt.Errorf("with_meta_resolver: got %s, want llb.image_meta_resolver or callable", valueType(args[0]))
	}
	if resolver == nil {
		return nil, fmt.Errorf("with_meta_resolver: resolver must not be nil")
	}
	return wrapOption("image_option", buildkitllb.WithMetaResolver(resolver)), nil
}

func withImageChecksumBuiltin(_ *starlark.Thread, _ *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	if err := arity("with_image_checksum", args, kwargs, 1, 1); err != nil {
		return nil, err
	}
	raw, err := stringAt("with_image_checksum", args, 0)
	if err != nil {
		return nil, err
	}
	parsed, err := digest.Parse(raw)
	if err != nil {
		return nil, fmt.Errorf("with_image_checksum: %w", err)
	}
	return wrapOption("image_option", buildkitllb.WithImageChecksum(parsed)), nil
}

func resolveModeValues() map[string]starlark.Value {
	return map[string]starlark.Value{
		"resolve_mode_default":   wrapOption("image_option", buildkitllb.ResolveModeDefault),
		"resolve_mode_force_pull": wrapOption("image_option", buildkitllb.ResolveModeForcePull),
		"resolve_mode_prefer_local": wrapOption("image_option", buildkitllb.ResolveModePreferLocal),
	}
}

var (
	_ starlark.Value    = (*ImageMetaResolverValue)(nil)
	_ starlark.HasAttrs = (*ImageMetaResolverValue)(nil)
	_ buildkitllb.ImageMetaResolver = (*imageMetaResolverCallback)(nil)
)
