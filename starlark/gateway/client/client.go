// Package client exposes BuildKit's frontend/gateway/client API to Starlark.
// A ClientValue is bound to one BuildKit solve invocation; it is deliberately
// supplied by the embedding application rather than registered globally.
package client

import (
	"context"
	"fmt"
	"time"

	buildkitsourceresolver "github.com/moby/buildkit/client/llb/sourceresolver"
	gwclient "github.com/moby/buildkit/frontend/gateway/client"
	"github.com/moby/buildkit/solver/pb"
	digest "github.com/opencontainers/go-digest"
	ocispecs "github.com/opencontainers/image-spec/specs-go/v1"
	starlarkllb "github.com/ep0ll/lumet/starlark/llb"
	starlarksourceresolver "github.com/ep0ll/lumet/starlark/llb/sourceresolver"
	"go.starlark.net/starlark"
	"go.starlark.net/starlarkstruct"
)

// Aliases retain BuildKit's native gateway API types for Go callers while
// the Starlark globals below expose the corresponding snake_case values.
type (
	Client = gwclient.Client
	Result = gwclient.Result
	Attestation = gwclient.Attestation
	BuildFunc = gwclient.BuildFunc
	NewContainerRequest = gwclient.NewContainerRequest
	Mount = gwclient.Mount
	Container = gwclient.Container
	StartRequest = gwclient.StartRequest
	WinSize = gwclient.WinSize
	ContainerProcess = gwclient.ContainerProcess
	Reference = gwclient.Reference
	ReadRequest = gwclient.ReadRequest
	ReadContainerRequest = gwclient.ReadContainerRequest
	FileRange = gwclient.FileRange
	ReadDirRequest = gwclient.ReadDirRequest
	ReadDirContainerRequest = gwclient.ReadDirContainerRequest
	StatRequest = gwclient.StatRequest
	StatContainerRequest = gwclient.StatContainerRequest
	SolveRequest = gwclient.SolveRequest
	CacheOptionsEntry = gwclient.CacheOptionsEntry
	WorkerInfo = gwclient.WorkerInfo
	BuildOpts = gwclient.BuildOpts
	WarnOpts = gwclient.WarnOpts
	MetaResolver = buildkitsourceresolver.MetaResolver
	ResolverOpt = buildkitsourceresolver.Opt
	MetaResponse = buildkitsourceresolver.MetaResponse
)

type ClientValue struct { client gwclient.Client }

func WrapClient(client gwclient.Client) *ClientValue { return &ClientValue{client: client} }
func (v *ClientValue) Unwrap() gwclient.Client { if v == nil { return nil }; return v.client }
func (v *ClientValue) String() string { return "<gateway.client.client>" }
func (v *ClientValue) Type() string { return "gateway.client.client" }
func (v *ClientValue) Freeze() {}
func (v *ClientValue) Truth() starlark.Bool { return starlark.Bool(v != nil && v.client != nil) }
func (v *ClientValue) Hash() (uint32, error) { return 0, fmt.Errorf("unhashable: gateway.client.client") }

func (v *ClientValue) Attr(name string) (starlark.Value, error) {
	if v == nil || v.client == nil { return nil, fmt.Errorf("gateway.client.client: nil receiver") }
	switch name {
	case "solve":
		return bind("client.solve", func(thread *starlark.Thread, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
			var request starlark.Value
			if err := starlark.UnpackArgs("client.solve", args, kwargs, "request", &request); err != nil { return nil, err }
			req, err := recordFrom[gwclient.SolveRequest](request, "solve_request")
			if err != nil { return nil, err }
			result, err := v.client.Solve(threadContext(thread), req)
			if err != nil { return nil, err }
			return WrapResult(result), nil
		}), nil
	case "resolve_image_config":
		return bind("client.resolve_image_config", func(thread *starlark.Thread, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
			var ref string
			var optionValue starlark.Value = starlark.None
			if err := starlark.UnpackArgs("client.resolve_image_config", args, kwargs, "ref", &ref, "opt?", &optionValue); err != nil { return nil, err }
			var opt buildkitsourceresolver.Opt
			if optionValue != starlark.None {
				var err error
				opt, err = starlarksourceresolver.UnwrapOpt(optionValue)
				if err != nil { return nil, err }
			}
			resolved, dgst, config, err := v.client.ResolveImageConfig(threadContext(thread), ref, opt)
			if err != nil { return nil, err }
			return starlark.Tuple{starlark.String(resolved), starlark.String(dgst.String()), starlark.Bytes(config)}, nil
		}), nil
	case "resolve_source_metadata":
		return bind("client.resolve_source_metadata", func(thread *starlark.Thread, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
			var opValue starlark.Value
			var optionValue starlark.Value = starlark.None
			if err := starlark.UnpackArgs("client.resolve_source_metadata", args, kwargs, "op", &opValue, "opt?", &optionValue); err != nil { return nil, err }
			op, ok := opValue.(*starlarksourceresolver.SourceOpValue)
			if !ok || op == nil || op.Unwrap() == nil { return nil, fmt.Errorf("client.resolve_source_metadata.op: got %s, want sourceresolver.source_op", valueType(opValue)) }
			var opt buildkitsourceresolver.Opt
			if optionValue != starlark.None {
				var err error
				opt, err = starlarksourceresolver.UnwrapOpt(optionValue)
				if err != nil { return nil, err }
			}
			response, err := v.client.ResolveSourceMetadata(threadContext(thread), op.Unwrap(), opt)
			if err != nil { return nil, err }
			return metaResponseValue(response), nil
		}), nil
	case "build_opts":
		return bind("client.build_opts", func(_ *starlark.Thread, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
			if err := starlark.UnpackArgs("client.build_opts", args, kwargs); err != nil { return nil, err }
			return buildOptsValue(v.client.BuildOpts()), nil
		}), nil
	case "inputs":
		return bind("client.inputs", func(thread *starlark.Thread, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
			if err := starlark.UnpackArgs("client.inputs", args, kwargs); err != nil { return nil, err }
			inputs, err := v.client.Inputs(threadContext(thread))
			if err != nil { return nil, err }
			dict := starlark.NewDict(len(inputs))
			for name, state := range inputs {
				if err := dict.SetKey(starlark.String(name), starlarkllb.WrapState(state)); err != nil { return nil, err }
			}
			return dict, nil
		}), nil
	case "new_container":
		return bind("client.new_container", func(thread *starlark.Thread, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
			var request starlark.Value
			if err := starlark.UnpackArgs("client.new_container", args, kwargs, "request", &request); err != nil { return nil, err }
			req, err := recordFrom[gwclient.NewContainerRequest](request, "new_container_request")
			if err != nil { return nil, err }
			container, err := v.client.NewContainer(threadContext(thread), req)
			if err != nil { return nil, err }
			return WrapContainer(container), nil
		}), nil
	case "warn":
		return bind("client.warn", func(thread *starlark.Thread, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
			var dgst, message string
			var optionsValue starlark.Value = starlark.None
			if err := starlark.UnpackArgs("client.warn", args, kwargs, "digest", &dgst, "message", &message, "opts?", &optionsValue); err != nil { return nil, err }
			var opts gwclient.WarnOpts
			if optionsValue != starlark.None {
				var err error
				opts, err = recordFrom[gwclient.WarnOpts](optionsValue, "warn_opts")
				if err != nil { return nil, err }
			}
			if err := v.client.Warn(threadContext(thread), digest.Digest(dgst), message, opts); err != nil { return nil, err }
			return starlark.None, nil
		}), nil
	default:
		return nil, nil
	}
}

func (v *ClientValue) AttrNames() []string {
	return []string{"build_opts", "inputs", "new_container", "resolve_image_config", "resolve_source_metadata", "solve", "warn"}
}

// Members returns the per-build gateway/client module globals. It is not
// registered in the process-wide module registry because each BuildKit
// invocation has a different gateway client instance.
func Members(client gwclient.Client) starlark.StringDict {
	return starlark.StringDict{
		"client": WrapClient(client),
		"new_result": builtin("new_result", newResultBuiltin),
		"attestation": builtin("attestation", attestationBuiltin),
		"attestation_from_pb": builtin("attestation_from_pb", attestationFromPBBuiltin),
		"attestation_to_pb": builtin("attestation_to_pb", attestationToPBBuiltin),
		"solve_request": builtin("solve_request", solveRequestBuiltin),
		"cache_options_entry": builtin("cache_options_entry", cacheOptionsEntryBuiltin),
		"mount": builtin("mount", mountBuiltin),
		"new_container_request": builtin("new_container_request", newContainerRequestBuiltin),
		"start_request": builtin("start_request", startRequestBuiltin),
		"file_range": builtin("file_range", fileRangeBuiltin),
		"read_request": builtin("read_request", readRequestBuiltin),
		"read_container_request": builtin("read_container_request", readContainerRequestBuiltin),
		"read_dir_request": builtin("read_dir_request", readDirRequestBuiltin),
		"read_dir_container_request": builtin("read_dir_container_request", readDirContainerRequestBuiltin),
		"stat_request": builtin("stat_request", statRequestBuiltin),
		"stat_container_request": builtin("stat_container_request", statContainerRequestBuiltin),
		"win_size": builtin("win_size", winSizeBuiltin),
		"warn_opts": builtin("warn_opts", warnOptsBuiltin),
		"net_mode_unset": starlark.MakeInt(int(pb.NetMode_UNSET)),
		"net_mode_host": starlark.MakeInt(int(pb.NetMode_HOST)),
		"net_mode_none": starlark.MakeInt(int(pb.NetMode_NONE)),
		"mount_type_bind": starlark.MakeInt(int(pb.MountType_BIND)),
		"mount_type_secret": starlark.MakeInt(int(pb.MountType_SECRET)),
		"mount_type_ssh": starlark.MakeInt(int(pb.MountType_SSH)),
		"mount_type_cache": starlark.MakeInt(int(pb.MountType_CACHE)),
		"mount_type_tmpfs": starlark.MakeInt(int(pb.MountType_TMPFS)),
		"security_mode_sandbox": starlark.MakeInt(int(pb.SecurityMode_SANDBOX)),
		"security_mode_insecure": starlark.MakeInt(int(pb.SecurityMode_INSECURE)),
	}
}

func NewModule(client gwclient.Client) *starlarkstruct.Module {
	members := Members(client)
	normalized, err := starlarkllb.NormalizeMembers(members)
	if err != nil { panic(err) }
	module := &starlarkstruct.Module{Name: "gateway/client", Members: normalized}
	module.Freeze()
	return module
}

func buildOptsValue(opts gwclient.BuildOpts) starlark.Value {
	workers := make([]starlark.Value, len(opts.Workers))
	for i, worker := range opts.Workers {
		platforms := make([]starlark.Value, len(worker.Platforms))
		for j, platform := range worker.Platforms { platforms[j] = ociPlatformValue(platform) }
		workers[i] = structValue("worker_info", starlark.StringDict{
			"id": starlark.String(worker.ID),
			"labels": stringMapValue(worker.Labels),
			"platforms": starlark.NewList(platforms),
		})
	}
	return structValue("build_opts", starlark.StringDict{
		"opts": stringMapValue(opts.Opts),
		"session_id": starlark.String(opts.SessionID),
		"workers": starlark.NewList(workers),
		"product": starlark.String(opts.Product),
		"llb_caps": starlarkllb.WrapCapSet(opts.LLBCaps),
		"caps": starlarkllb.WrapCapSet(opts.Caps),
	})
}

func ociPlatformValue(platform ocispecs.Platform) starlark.Value {
	return starlarkllb.WrapOCIPlatform(platform)
}

func metaResponseValue(response *buildkitsourceresolver.MetaResponse) starlark.Value {
	if response == nil { return starlark.None }
	fields := starlark.StringDict{"op": starlark.None, "image": starlark.None, "git": starlark.None, "http": starlark.None}
	if response.Op != nil { fields["op"] = starlarksourceresolver.WrapSourceOp(response.Op) }
	// Keep response conversion explicit so the Starlark schema is stable and
	// independent of Go field-name reflection.
	if response.Image != nil {
		image := response.Image
		fields["image"] = structValue("resolve_image_response", starlark.StringDict{
			"digest": starlark.String(image.Digest.String()),
			"config": starlark.Bytes(append([]byte(nil), image.Config...)),
			"attestation_chain": attestationChainValue(image.AttestationChain),
		})
	}
	if response.Git != nil {
		git := response.Git
		fields["git"] = structValue("resolve_git_response", starlark.StringDict{
			"checksum": starlark.String(git.Checksum), "ref": starlark.String(git.Ref),
			"commit_checksum": starlark.String(git.CommitChecksum),
			"commit_object": starlark.Bytes(git.CommitObject), "tag_object": starlark.Bytes(git.TagObject),
		})
	}
	if response.HTTP != nil {
		http := response.HTTP
		lastModified := starlark.Value(starlark.None)
		if http.LastModified != nil { lastModified = starlark.String(http.LastModified.Format(time.RFC3339Nano)) }
		checksum := starlark.Value(starlark.None)
		if http.ChecksumResponse != nil {
			checksum = structValue("resolve_http_checksum_response", starlark.StringDict{
				"digest": starlark.String(http.ChecksumResponse.Digest),
				"suffix": starlark.Bytes(http.ChecksumResponse.Suffix),
			})
		}
		fields["http"] = structValue("resolve_http_response", starlark.StringDict{
			"digest": starlark.String(http.Digest.String()), "filename": starlark.String(http.Filename),
			"last_modified": lastModified, "checksum_response": checksum,
		})
	}
	return structValue("meta_response", fields)
}

func attestationChainValue(chain *buildkitsourceresolver.AttestationChain) starlark.Value {
	if chain == nil { return starlark.None }
	blobs := starlark.NewDict(len(chain.Blobs))
	for dgst, blob := range chain.Blobs {
		urls := make([]starlark.Value, len(blob.Descriptor.URLs))
		for i, url := range blob.Descriptor.URLs {
			urls[i] = starlark.String(url)
		}
		var platform starlark.Value = starlark.None
		if blob.Descriptor.Platform != nil {
			platform = starlarkllb.WrapOCIPlatform(*blob.Descriptor.Platform)
		}
		descriptor := structValue("descriptor", starlark.StringDict{
			"media_type": starlark.String(string(blob.Descriptor.MediaType)),
			"digest": starlark.String(blob.Descriptor.Digest.String()),
			"size": starlark.MakeInt64(blob.Descriptor.Size),
			"urls": starlark.NewList(urls),
			"annotations": stringMapValue(blob.Descriptor.Annotations),
			"data": starlark.Bytes(append([]byte(nil), blob.Descriptor.Data...)),
			"artifact_type": starlark.String(blob.Descriptor.ArtifactType),
			"platform": platform,
		})
		value := structValue("blob", starlark.StringDict{
			"descriptor": descriptor,
			"data": starlark.Bytes(append([]byte(nil), blob.Data...)),
		})
		_ = blobs.SetKey(starlark.String(dgst.String()), value)
	}
	signatures := make([]starlark.Value, len(chain.SignatureManifests))
	for i, dgst := range chain.SignatureManifests { signatures[i] = starlark.String(dgst.String()) }
	return structValue("attestation_chain", starlark.StringDict{
		"root": starlark.String(chain.Root.String()),
		"image_manifest": starlark.String(chain.ImageManifest.String()),
		"attestation_manifest": starlark.String(chain.AttestationManifest.String()),
		"signature_manifests": starlark.NewList(signatures),
		"blobs": blobs,
	})
}

func threadContext(thread *starlark.Thread) context.Context {
	if thread != nil {
		if ctx, ok := thread.Local("context").(context.Context); ok && ctx != nil { return ctx }
	}
	return context.Background()
}

func bind(name string, fn func(*starlark.Thread, starlark.Tuple, []starlark.Tuple) (starlark.Value, error)) starlark.Value {
	return starlark.NewBuiltin(name, func(thread *starlark.Thread, _ *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
		return fn(thread, args, kwargs)
	})
}

func builtin(name string, fn func(*starlark.Thread, *starlark.Builtin, starlark.Tuple, []starlark.Tuple) (starlark.Value, error)) starlark.Value {
	return starlark.NewBuiltin(name, fn)
}

// BuildCallback adapts a Starlark callable to BuildKit's gateway BuildFunc. The
// callable receives the invocation-scoped ClientValue and must return a
// gateway.client.result created by client.solve() or new_result().
func BuildCallback(build starlark.Callable) gwclient.BuildFunc {
	return func(ctx context.Context, client gwclient.Client) (*gwclient.Result, error) {
		thread := &starlark.Thread{Name: "lumet gateway build"}
		if ctx != nil {
			thread.SetLocal("context", ctx)
			stop := context.AfterFunc(ctx, func() { thread.Cancel("context cancelled") })
			defer stop()
		}
		value, err := starlark.Call(thread, build, starlark.Tuple{WrapClient(client)}, nil)
		if err != nil { return nil, err }
		result, ok := value.(*ResultValue)
		if !ok || result == nil || result.result == nil {
			return nil, fmt.Errorf("gateway build callback returned %s, want gateway.client.result", valueType(value))
		}
		return result.result, nil
	}
}

