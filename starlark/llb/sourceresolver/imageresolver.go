package sourceresolver

import (
	"context"
	"fmt"
	"sync"
	"time"

	starlarkllb "github.com/ep0ll/lumet/starlark/llb"
	buildkitsourceresolver "github.com/moby/buildkit/client/llb/sourceresolver"
	"github.com/moby/buildkit/solver/pb"
	digest "github.com/opencontainers/go-digest"
	ocispecs "github.com/opencontainers/image-spec/specs-go/v1"
	"go.starlark.net/starlark"
)

// metaResolverCallback implements BuildKit's MetaResolver interface by invoking
// a Starlark callable. The callable receives (SourceOp, Opt) and returns a
// MetaResponse value.
type metaResolverCallback struct {
	thread   *starlark.Thread
	callable starlark.Callable
	mu       sync.Mutex
}

func (r *metaResolverCallback) ResolveSourceMetadata(ctx context.Context, op *pb.SourceOp, opt buildkitsourceresolver.Opt) (*buildkitsourceresolver.MetaResponse, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	oldContext := r.thread.Local("context")
	r.thread.SetLocal("context", ctx)
	defer r.thread.SetLocal("context", oldContext)

	result, err := starlark.Call(r.thread, r.callable, starlark.Tuple{
		&SourceOpValue{op: op},
		starlarkllb.WrapSourceResolverOpt(opt),
	}, nil)
	if err != nil {
		return nil, err
	}
	response, err := recordFrom[*buildkitsourceresolver.MetaResponse](result, "meta_response")
	if err != nil {
		return nil, err
	}
	return response, nil
}

func newImageMetaResolverBuiltin(thread *starlark.Thread, _ *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	var callback starlark.Callable
	if err := starlarkllb.UnpackArgs("new_image_meta_resolver", args, kwargs, "resolver", &callback); err != nil {
		return nil, err
	}
	resolver := buildkitsourceresolver.NewImageMetaResolver(&metaResolverCallback{
		thread: thread, callable: callback,
	})
	return starlarkllb.WrapImageMetaResolver(resolver), nil
}

func metaResponseBuiltin(_ *starlark.Thread, _ *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	var opValue starlark.Value
	var imageValue, gitValue, httpValue starlark.Value = starlark.None, starlark.None, starlark.None
	if err := starlarkllb.UnpackArgs("meta_response", args, kwargs,
		"op", &opValue,
		"image?", &imageValue,
		"git?", &gitValue,
		"http?", &httpValue,
	); err != nil {
		return nil, err
	}
	op, ok := opValue.(*SourceOpValue)
	if !ok || op == nil || op.op == nil {
		return nil, fmt.Errorf("meta_response.op: got %s, want sourceresolver.source_op", starlarkType(opValue))
	}
	response := &buildkitsourceresolver.MetaResponse{Op: op.op}
	fields := starlark.StringDict{"op": op}
	if imageValue != starlark.None {
		image, err := recordFrom[*buildkitsourceresolver.ResolveImageResponse](imageValue, "resolve_image_response")
		if err != nil { return nil, err }
		response.Image = image
		fields["image"] = imageValue
	} else {
		fields["image"] = starlark.None
	}
	if gitValue != starlark.None {
		git, err := recordFrom[*buildkitsourceresolver.ResolveGitResponse](gitValue, "resolve_git_response")
		if err != nil { return nil, err }
		response.Git = git
		fields["git"] = gitValue
	} else {
		fields["git"] = starlark.None
	}
	if httpValue != starlark.None {
		httpResponse, err := recordFrom[*buildkitsourceresolver.ResolveHTTPResponse](httpValue, "resolve_http_response")
		if err != nil { return nil, err }
		response.HTTP = httpResponse
		fields["http"] = httpValue
	} else {
		fields["http"] = starlark.None
	}
	return newRecord("meta_response", response, fields), nil
}

func resolveImageResponseBuiltin(_ *starlark.Thread, _ *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	var rawDigest string
	var configValue starlark.Value
	var chainValue starlark.Value = starlark.None
	if err := starlarkllb.UnpackArgs("resolve_image_response", args, kwargs,
		"digest", &rawDigest,
		"config", &configValue,
		"attestation_chain?", &chainValue,
	); err != nil {
		return nil, err
	}
	config, err := bytesArg("resolve_image_response.config", configValue)
	if err != nil { return nil, err }
	dgst, err := digest.Parse(rawDigest)
	if err != nil { return nil, fmt.Errorf("resolve_image_response.digest: %w", err) }
	response := &buildkitsourceresolver.ResolveImageResponse{Digest: dgst, Config: config}
	fields := starlark.StringDict{"digest": digestField(dgst), "config": bytesField(config), "attestation_chain": starlark.None}
	if chainValue != starlark.None {
		chain, err := recordFrom[*buildkitsourceresolver.AttestationChain](chainValue, "attestation_chain")
		if err != nil { return nil, err }
		response.AttestationChain = chain
		fields["attestation_chain"] = chainValue
	}
	return newRecord("resolve_image_response", response, fields), nil
}

func resolveGitResponseBuiltin(_ *starlark.Thread, _ *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	var checksum, ref, commitChecksum string
	var commitObjectValue, tagObjectValue starlark.Value = starlark.None, starlark.None
	if err := starlarkllb.UnpackArgs("resolve_git_response", args, kwargs,
		"checksum?", &checksum, "ref?", &ref, "commit_checksum?", &commitChecksum,
		"commit_object?", &commitObjectValue, "tag_object?", &tagObjectValue,
	); err != nil { return nil, err }
	var commitObject, tagObject []byte
	var err error
	if commitObjectValue != starlark.None {
		commitObject, err = bytesArg("resolve_git_response.commit_object", commitObjectValue)
		if err != nil { return nil, err }
	}
	if tagObjectValue != starlark.None {
		tagObject, err = bytesArg("resolve_git_response.tag_object", tagObjectValue)
		if err != nil { return nil, err }
	}
	response := &buildkitsourceresolver.ResolveGitResponse{
		Checksum: checksum, Ref: ref, CommitChecksum: commitChecksum,
		CommitObject: commitObject, TagObject: tagObject,
	}
	return newRecord("resolve_git_response", response, starlark.StringDict{
		"checksum": stringField("checksum", checksum), "ref": stringField("ref", ref),
		"commit_checksum": stringField("commit_checksum", commitChecksum),
		"commit_object": bytesField(commitObject), "tag_object": bytesField(tagObject),
	}), nil
}

func resolveHTTPResponseBuiltin(_ *starlark.Thread, _ *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	var rawDigest, filename, modified string
	var checksumValue starlark.Value = starlark.None
	if err := starlarkllb.UnpackArgs("resolve_http_response", args, kwargs,
		"digest", &rawDigest, "filename?", &filename, "last_modified?", &modified,
		"checksum_response?", &checksumValue,
	); err != nil { return nil, err }
	dgst, err := digest.Parse(rawDigest)
	if err != nil { return nil, fmt.Errorf("resolve_http_response.digest: %w", err) }
	response := &buildkitsourceresolver.ResolveHTTPResponse{Digest: dgst, Filename: filename}
	fields := starlark.StringDict{"digest": digestField(dgst), "filename": stringField("filename", filename), "last_modified": starlark.None, "checksum_response": starlark.None}
	if modified != "" {
		value, err := time.Parse(time.RFC3339Nano, modified)
		if err != nil { return nil, fmt.Errorf("resolve_http_response.last_modified: %w", err) }
		response.LastModified = &value
		fields["last_modified"] = starlark.String(value.Format(time.RFC3339Nano))
	}
	if checksumValue != starlark.None {
		checksum, err := recordFrom[*buildkitsourceresolver.ResolveHTTPChecksumResponse](checksumValue, "resolve_http_checksum_response")
		if err != nil { return nil, err }
		response.ChecksumResponse = checksum
		fields["checksum_response"] = checksumValue
	}
	return newRecord("resolve_http_response", response, fields), nil
}

func resolveHTTPChecksumResponseBuiltin(_ *starlark.Thread, _ *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	var value string
	var suffixValue starlark.Value = starlark.None
	if err := starlarkllb.UnpackArgs("resolve_http_checksum_response", args, kwargs, "digest", &value, "suffix?",&suffixValue); err != nil { return nil, err }
	var suffix []byte
	if suffixValue != starlark.None {
		var err error
		suffix, err = bytesArg("resolve_http_checksum_response.suffix", suffixValue)
		if err != nil { return nil, err }
	}
	response := &buildkitsourceresolver.ResolveHTTPChecksumResponse{Digest:value,Suffix:suffix}
	return newRecord("resolve_http_checksum_response",response,starlark.StringDict{"digest":starlark.String(value),"suffix":bytesField(suffix)}),nil
}

func blobBuiltin(_ *starlark.Thread, _ *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	var mediaType, rawDigest string
	var size int64
	var dataValue starlark.Value
	if err := starlarkllb.UnpackArgs("blob", args, kwargs,
		"media_type", &mediaType, "digest", &rawDigest, "size", &size, "data", &dataValue,
	); err != nil { return nil, err }
	data, err := bytesArg("blob.data", dataValue)
	if err != nil { return nil, err }
	dgst, err := digest.Parse(rawDigest)
	if err != nil { return nil, err }
	blob := buildkitsourceresolver.Blob{
		Descriptor: ocispecs.Descriptor{MediaType:mediaType,Digest:dgst,Size:size},
		Data:data,
	}
	return newRecord("blob",blob,starlark.StringDict{
		"media_type":starlark.String(mediaType),"digest":digestField(dgst),
		"size":starlark.MakeInt64(size),"data":starlark.Bytes(data),
	}),nil
}

func attestationChainBuiltin(_ *starlark.Thread,_ *starlark.Builtin,args starlark.Tuple,kwargs []starlark.Tuple)(starlark.Value,error){
	var rootRaw,imageRaw,attestationRaw string
	var signatureValues starlark.Value=starlark.NewList(nil)
	var blobValues starlark.Value=starlark.None
	if err:=starlarkllb.UnpackArgs("attestation_chain",args,kwargs,
		"root",&rootRaw,"image_manifest",&imageRaw,"attestation_manifest",&attestationRaw,
		"signature_manifests?",&signatureValues,"blobs?",&blobValues);err!=nil{return nil,err}
	root,err:=digest.Parse(rootRaw);if err!=nil{return nil,err}
	image,err:=digest.Parse(imageRaw);if err!=nil{return nil,err}
	attestation,err:=digest.Parse(attestationRaw);if err!=nil{return nil,err}
	signatures,err:=parseDigestList(signatureValues);if err!=nil{return nil,err}
	blobs:=map[digest.Digest]buildkitsourceresolver.Blob{}
	if blobValues!=starlark.None{
		dict,ok:=blobValues.(*starlark.Dict);if !ok{return nil,fmt.Errorf("attestation_chain.blobs: want dict")}
		for _,item:=range dict.Items(){
			key,err:=stringArg("attestation_chain.blobs key",item[0]);if err!=nil{return nil,err}
			dgst,err:=digest.Parse(key);if err!=nil{return nil,err}
			value,err:=recordFrom[buildkitsourceresolver.Blob](item[1],"blob");if err!=nil{return nil,err}
			blobs[dgst]=value
		}
	}
	chain:=buildkitsourceresolver.AttestationChain{Root:root,ImageManifest:image,AttestationManifest:attestation,SignatureManifests:signatures,Blobs:blobs}
	return newRecord("attestation_chain",chain,starlark.StringDict{
		"root":digestField(root),"image_manifest":digestField(image),
		"attestation_manifest":digestField(attestation),"signature_manifests":digestList(signatures),
		"blobs":starlark.None,
	}),nil
}
func parseDigestList(value starlark.Value)([]digest.Digest,error){
	list,ok:=value.(starlark.Indexable);if !ok{return nil,fmt.Errorf("expected list of digests")}
	out:=make([]digest.Digest,list.Len())
	for i:=range out{raw,err:=stringArg("digest list",list.Index(i));if err!=nil{return nil,err};d,err:=digest.Parse(raw);if err!=nil{return nil,err};out[i]=d}
	return out,nil
}
func digestList(values []digest.Digest)starlark.Value{
	out:=make([]starlark.Value,len(values));for i,v:=range values{out[i]=starlark.String(v.String())};return starlark.NewList(out)
}
func builtins()map[string]starlark.Value{
	members:=map[string]starlark.Value{
		"new_image_meta_resolver":builtin("new_image_meta_resolver",newImageMetaResolverBuiltin),
		"meta_response":builtin("meta_response",metaResponseBuiltin),
		"resolve_image_response":builtin("resolve_image_response",resolveImageResponseBuiltin),
		"resolve_git_response":builtin("resolve_git_response",resolveGitResponseBuiltin),
		"resolve_http_response":builtin("resolve_http_response",resolveHTTPResponseBuiltin),
		"resolve_http_checksum_response":builtin("resolve_http_checksum_response",resolveHTTPChecksumResponseBuiltin),
		"attestation_chain":builtin("attestation_chain",attestationChainBuiltin),
		"blob":builtin("blob",blobBuiltin),
	}
	for name,value:=range sourceResolverTypesBuiltins(){members[name]=value}
	return members
}

var (
	_ buildkitsourceresolver.MetaResolver = (*metaResolverCallback)(nil)
	_ starlark.Value = (*SourceOpValue)(nil)
	_ starlark.HasAttrs = (*SourceOpValue)(nil)
)
