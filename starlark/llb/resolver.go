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

func (v *SourceResolverOptValue) String() string { return "<llb.SourceResolverOpt>" }
func (v *SourceResolverOptValue) Type() string { return "llb.SourceResolverOpt" }
func (v *SourceResolverOptValue) Freeze() {}
func (v *SourceResolverOptValue) Truth() starlark.Bool { return starlark.Bool(v != nil) }
func (v *SourceResolverOptValue) Hash() (uint32, error) { return 0, fmt.Errorf("unhashable: llb.SourceResolverOpt") }

func (v *SourceResolverOptValue) Attr(name string) (starlark.Value, error) {
	if v == nil { return nil, fmt.Errorf("llb.SourceResolverOpt: nil receiver") }
	switch name {
	case "log_name", "LogName":
		return starlark.String(v.opt.LogName), nil
	case "image", "ImageOpt":
		if v.opt.ImageOpt == nil { return starlark.None, nil }
		return &resolveImageOptValue{opt: *v.opt.ImageOpt}, nil
	case "oci_layout", "OCILayoutOpt":
		if v.opt.OCILayoutOpt == nil { return starlark.None, nil }
		return &resolveOCILayoutOptValue{opt: *v.opt.OCILayoutOpt}, nil
	case "git", "GitOpt":
		if v.opt.GitOpt == nil { return starlark.None, nil }
		return &resolveGitOptValue{opt: *v.opt.GitOpt}, nil
	case "http", "HTTPOpt":
		if v.opt.HTTPOpt == nil { return starlark.None, nil }
		return &resolveHTTPOptValue{opt: *v.opt.HTTPOpt}, nil
	case "source_policies", "SourcePolicies":
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
	return []string{"log_name", "image", "oci_layout", "git", "http", "source_policies",
		"LogName", "ImageOpt", "OCILayoutOpt", "GitOpt", "HTTPOpt", "SourcePolicies"}
}

type resolveImageOptValue struct { opt sourceresolver.ResolveImageOpt }
func (v *resolveImageOptValue) String() string { return "<sourceresolver.ResolveImageOpt>" }
func (v *resolveImageOptValue) Type() string { return "sourceresolver.ResolveImageOpt" }
func (v *resolveImageOptValue) Freeze() {}
func (v *resolveImageOptValue) Truth() starlark.Bool { return starlark.Bool(v != nil) }
func (v *resolveImageOptValue) Hash() (uint32,error) { return 0, fmt.Errorf("unhashable: ResolveImageOpt") }
func (v *resolveImageOptValue) Attr(name string) (starlark.Value,error) {
	if v == nil { return nil, fmt.Errorf("ResolveImageOpt: nil receiver") }
	switch name {
	case "platform", "Platform":
		if v.opt.Platform == nil { return starlark.None,nil }
		return WrapOCIPlatform(*v.opt.Platform),nil
	case "resolve_mode", "ResolveMode": return starlark.String(v.opt.ResolveMode),nil
	case "no_config", "NoConfig": return starlark.Bool(v.opt.NoConfig),nil
	case "attestation_chain", "AttestationChain": return starlark.Bool(v.opt.AttestationChain),nil
	case "resolve_attestations", "ResolveAttestations": return stringSliceValue(v.opt.ResolveAttestations),nil
	default: return nil,nil
	}
}
func (v *resolveImageOptValue) AttrNames() []string { return []string{"platform","resolve_mode","no_config","attestation_chain","resolve_attestations","Platform","ResolveMode","NoConfig","AttestationChain","ResolveAttestations"} }

type resolveGitOptValue struct { opt sourceresolver.ResolveGitOpt }
func (v *resolveGitOptValue) String() string { return "<sourceresolver.ResolveGitOpt>" }
func (v *resolveGitOptValue) Type() string { return "sourceresolver.ResolveGitOpt" }
func (v *resolveGitOptValue) Freeze() {}
func (v *resolveGitOptValue) Truth() starlark.Bool { return starlark.Bool(v != nil) }
func (v *resolveGitOptValue) Hash() (uint32,error) { return 0, fmt.Errorf("unhashable: ResolveGitOpt") }
func (v *resolveGitOptValue) Attr(name string) (starlark.Value,error) {
	if name == "return_object" || name == "ReturnObject" { return starlark.Bool(v.opt.ReturnObject),nil }
	return nil,nil
}
func (v *resolveGitOptValue) AttrNames() []string { return []string{"return_object","ReturnObject"} }

type resolveHTTPOptValue struct { opt sourceresolver.ResolveHTTPOpt }
func (v *resolveHTTPOptValue) String() string { return "<sourceresolver.ResolveHTTPOpt>" }
func (v *resolveHTTPOptValue) Type() string { return "sourceresolver.ResolveHTTPOpt" }
func (v *resolveHTTPOptValue) Freeze() {}
func (v *resolveHTTPOptValue) Truth() starlark.Bool { return starlark.Bool(v != nil) }
func (v *resolveHTTPOptValue) Hash() (uint32,error) { return 0, fmt.Errorf("unhashable: ResolveHTTPOpt") }
func (v *resolveHTTPOptValue) Attr(name string) (starlark.Value,error) {
	if name == "checksum_req" || name == "ChecksumReq" {
		if v.opt.ChecksumReq == nil { return starlark.None,nil }
		return &resolveHTTPChecksumRequestValue{request:*v.opt.ChecksumReq},nil
	}
	return nil,nil
}
func (v *resolveHTTPOptValue) AttrNames() []string { return []string{"checksum_req","ChecksumReq"} }

type resolveHTTPChecksumRequestValue struct { request sourceresolver.ResolveHTTPChecksumRequest }
func (v *resolveHTTPChecksumRequestValue) String() string { return "<sourceresolver.ResolveHTTPChecksumRequest>" }
func (v *resolveHTTPChecksumRequestValue) Type() string { return "sourceresolver.ResolveHTTPChecksumRequest" }
func (v *resolveHTTPChecksumRequestValue) Freeze() {}
func (v *resolveHTTPChecksumRequestValue) Truth() starlark.Bool { return starlark.Bool(v != nil) }
func (v *resolveHTTPChecksumRequestValue) Hash() (uint32,error) { return 0, fmt.Errorf("unhashable: ResolveHTTPChecksumRequest") }
func (v *resolveHTTPChecksumRequestValue) Attr(name string) (starlark.Value,error) {
	switch name {
	case "algo", "Algo": return starlark.MakeInt(int(v.request.Algo)),nil
	case "suffix", "Suffix": return starlark.Bytes(v.request.Suffix),nil
	default: return nil,nil
	}
}
func (v *resolveHTTPChecksumRequestValue) AttrNames() []string { return []string{"algo","suffix","Algo","Suffix"} }

type resolveOCILayoutOptValue struct { opt sourceresolver.ResolveOCILayoutOpt }
func (v *resolveOCILayoutOptValue) String() string { return "<sourceresolver.ResolveOCILayoutOpt>" }
func (v *resolveOCILayoutOptValue) Type() string { return "sourceresolver.ResolveOCILayoutOpt" }
func (v *resolveOCILayoutOptValue) Freeze() {}
func (v *resolveOCILayoutOptValue) Truth() starlark.Bool { return starlark.Bool(v != nil) }
func (v *resolveOCILayoutOptValue) Hash() (uint32,error) { return 0, fmt.Errorf("unhashable: ResolveOCILayoutOpt") }
func (v *resolveOCILayoutOptValue) Attr(name string) (starlark.Value,error) {
	switch name {
	case "platform", "Platform":
		if v.opt.Platform == nil { return starlark.None,nil }
		return WrapOCIPlatform(*v.opt.Platform),nil
	case "store", "Store": return &resolveImageConfigStoreValue{store:v.opt.Store},nil
	default: return nil,nil
	}
}
func (v *resolveOCILayoutOptValue) AttrNames() []string { return []string{"platform","store","Platform","Store"} }

type resolveImageConfigStoreValue struct { store sourceresolver.ResolveImageConfigOptStore }
func (v *resolveImageConfigStoreValue) String() string { return "<sourceresolver.ResolveImageConfigOptStore>" }
func (v *resolveImageConfigStoreValue) Type() string { return "sourceresolver.ResolveImageConfigOptStore" }
func (v *resolveImageConfigStoreValue) Freeze() {}
func (v *resolveImageConfigStoreValue) Truth() starlark.Bool { return starlark.Bool(v != nil) }
func (v *resolveImageConfigStoreValue) Hash() (uint32,error) { return 0, fmt.Errorf("unhashable: ResolveImageConfigOptStore") }
func (v *resolveImageConfigStoreValue) Attr(name string) (starlark.Value,error) {
	switch name {
	case "session_id", "SessionID": return starlark.String(v.store.SessionID),nil
	case "store_id", "StoreID": return starlark.String(v.store.StoreID),nil
	default: return nil,nil
	}
}
func (v *resolveImageConfigStoreValue) AttrNames() []string { return []string{"session_id","store_id","SessionID","StoreID"} }

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

func withMetaResolverBuiltin(thread *starlark.Thread, _ *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
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
		resolver = &imageMetaResolverCallback{thread: thread, callable: value}
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
