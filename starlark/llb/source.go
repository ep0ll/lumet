package llb

import (
	"fmt"
	"net/url"

	buildkitllb "github.com/moby/buildkit/client/llb"
	"go.starlark.net/starlark"
)

// SourceOpValue wraps BuildKit's source vertex, including source identifiers,
// attributes, validation and marshaling.
type SourceOpValue struct {
	source *buildkitllb.SourceOp
}

// WrapSourceOp adapts a native BuildKit SourceOp.
func WrapSourceOp(source *buildkitllb.SourceOp) *SourceOpValue {
	return &SourceOpValue{source: source}
}

// Unwrap returns the native SourceOp.
func (v *SourceOpValue) Unwrap() *buildkitllb.SourceOp {
	if v == nil { return nil }
	return v.source
}

func (v *SourceOpValue) String() string        { return "<llb.source_op>" }
func (v *SourceOpValue) Type() string          { return "llb.source_op" }
func (v *SourceOpValue) Freeze()               {}
func (v *SourceOpValue) Truth() starlark.Bool  { return starlark.Bool(v != nil && v.source != nil) }
func (v *SourceOpValue) Hash() (uint32, error) { return 0, fmt.Errorf("unhashable: llb.source_op") }

func (v *SourceOpValue) Attr(name string) (starlark.Value, error) {
		if v == nil || v.source == nil { return nil, fmt.Errorf("llb.source_op: nil receiver") }
	switch name {
	case "output":
		return boundBuiltin("source_op.output", func(_ *starlark.Thread, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
			if err := arity("source_op.output", args, kwargs, 0, 0); err != nil { return nil, err }
			return WrapOutput(v.source.Output()), nil
		}), nil
	case "inputs":
		return boundBuiltin("source_op.inputs", func(_ *starlark.Thread, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
			if err := arity("source_op.inputs", args, kwargs, 0, 0); err != nil { return nil, err }
			return starlark.NewList(nil), nil
		}), nil
	case "validate":
		return boundBuiltin("source_op.validate", func(thread *starlark.Thread, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
			if err := arity("source_op.validate", args, kwargs, 0, -1); err != nil { return nil, err }
			constraints, err := constraintsFromArgs(args, 0); if err != nil { return nil, err }
			if err := v.source.Validate(callContext(thread), constraints); err != nil { return nil, err }
			return starlark.None, nil
		}), nil
	case "marshal":
		return boundBuiltin("source_op.marshal", func(thread *starlark.Thread, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
			if err := arity("source_op.marshal", args, kwargs, 0, -1); err != nil { return nil, err }
			constraints, err := constraintsFromArgs(args, 0); if err != nil { return nil, err }
			dgst, data, metadata, sources, err := v.source.Marshal(callContext(thread), constraints)
			if err != nil { return nil, err }
			return marshalResult(dgst, data, metadata, sources), nil
		}), nil
	default:
		return nil, nil
	}
}

func (v *SourceOpValue) AttrNames() []string {
	return SnakeCaseNames([]string{"inputs", "marshal", "output", "validate", "inputs", "marshal", "output", "validate"})
}

// HTTPHeaderValue adapts the public HTTPHeader configuration type.
type HTTPHeaderValue struct { header buildkitllb.HTTPHeader }

// WrapHTTPHeader adapts a native HTTPHeader.
func WrapHTTPHeader(header buildkitllb.HTTPHeader) *HTTPHeaderValue { return &HTTPHeaderValue{header: header} }

// Unwrap returns the native HTTPHeader.
func (v *HTTPHeaderValue) Unwrap() buildkitllb.HTTPHeader {
	if v == nil { return buildkitllb.HTTPHeader{} }
	return v.header
}
func (v *HTTPHeaderValue) String() string        { return "<llb.http_header>" }
func (v *HTTPHeaderValue) Type() string          { return "llb.http_header" }
func (v *HTTPHeaderValue) Freeze()               {}
func (v *HTTPHeaderValue) Truth() starlark.Bool  { return starlark.Bool(v != nil) }
func (v *HTTPHeaderValue) Hash() (uint32, error) { return 0, fmt.Errorf("unhashable: llb.http_header") }
func (v *HTTPHeaderValue) Attr(name string) (starlark.Value, error) {
		if v == nil { return nil, fmt.Errorf("llb.http_header: nil receiver") }
	switch name {
	case "accept": return starlark.String(v.header.Accept), nil
	case "user_agent": return starlark.String(v.header.UserAgent), nil
	default: return nil, nil
	}
}
func (v *HTTPHeaderValue) AttrNames() []string { return SnakeCaseNames([]string{"accept", "user_agent", "accept", "user_agent"}) }

// HTTPSignatureInfoValue adapts BuildKit's detached HTTP signature metadata.
type HTTPSignatureInfoValue struct { info buildkitllb.HTTPSignatureInfo }

// WrapHTTPSignatureInfo adapts a native HTTPSignatureInfo.
func WrapHTTPSignatureInfo(info buildkitllb.HTTPSignatureInfo) *HTTPSignatureInfoValue {
	return &HTTPSignatureInfoValue{info: info}
}

// Unwrap returns a copy of the native signature information.
func (v *HTTPSignatureInfoValue) Unwrap() buildkitllb.HTTPSignatureInfo {
	if v == nil { return buildkitllb.HTTPSignatureInfo{} }
	return buildkitllb.HTTPSignatureInfo{PubKey: append([]byte(nil), v.info.PubKey...), Signature: append([]byte(nil), v.info.Signature...)}
}
func (v *HTTPSignatureInfoValue) String() string        { return "<llb.http_signature_info>" }
func (v *HTTPSignatureInfoValue) Type() string          { return "llb.http_signature_info" }
func (v *HTTPSignatureInfoValue) Freeze()               {}
func (v *HTTPSignatureInfoValue) Truth() starlark.Bool  { return starlark.Bool(v != nil) }
func (v *HTTPSignatureInfoValue) Hash() (uint32, error) { return 0, fmt.Errorf("unhashable: llb.http_signature_info") }
func (v *HTTPSignatureInfoValue) Attr(name string) (starlark.Value, error) {
		if v == nil { return nil, fmt.Errorf("llb.http_signature_info: nil receiver") }
	switch name {
	case "pub_key": return starlark.Bytes(v.info.PubKey), nil
	case "signature": return starlark.Bytes(v.info.Signature), nil
	default: return nil, nil
	}
}
func (v *HTTPSignatureInfoValue) AttrNames() []string { return SnakeCaseNames([]string{"pub_key", "signature", "pub_key", "signature"}) }

func sourceBuiltins() map[string]starlark.Value {
	members := make(map[string]starlark.Value)
	groups := []map[string]starlark.Value{
		imageSourceBuiltins(),
		gitSourceBuiltins(),
		localSourceBuiltins(),
		ociLayoutSourceBuiltins(),
		httpSourceBuiltins(),
		sourceOpBuiltins(),
	}
	for _, group := range groups {
		for name, value := range group {
			if _, exists := members[name]; exists { panic("starlark/llb: duplicate source builtin " + name) }
			members[name] = value
		}
	}
	return members
}

func imageSourceBuiltins() map[string]starlark.Value {
	return map[string]starlark.Value{
		"scratch": builtin("scratch", scratchSourceBuiltin),
		"image": builtin("image", imageSourceBuiltin),
		"image_blob": builtin("image_blob", imageBlobSourceBuiltin),
	}
}

func gitSourceBuiltins() map[string]starlark.Value {
	return map[string]starlark.Value{
		"git": builtin("git", gitSourceBuiltin),
		"git_ref": stringOptionBuiltin("git_ref", "git_option", buildkitllb.GitRef),
		"git_sub_dir": stringOptionBuiltin("git_sub_dir", "git_option", buildkitllb.GitSubDir),
		"git_skip_submodules": builtin("git_skip_submodules", gitSkipSubmodulesBuiltin),
		"git_m_time_commit": builtin("git_m_time_commit", gitMTimeCommitBuiltin),
		"git_m_time": stringOptionBuiltin("git_m_time", "git_option", buildkitllb.GitMTime),
		"git_advice": boolOptionBuiltin("git_advice", "git_option", buildkitllb.GitAdvice),
		"git_debug_commands": boolOptionBuiltin("git_debug_commands", "git_option", buildkitllb.GitDebugCommands),
		"keep_git_dir": builtin("keep_git_dir", keepGitDirBuiltin),
		"auth_token_secret": stringOptionBuiltin("auth_token_secret", "git_option", buildkitllb.AuthTokenSecret),
		"known_ssh_hosts": stringOptionBuiltin("known_ssh_hosts", "git_option", buildkitllb.KnownSSHHosts),
		"mount_ssh_sock": stringOptionBuiltin("mount_ssh_sock", "git_option", buildkitllb.MountSSHSock),
		"git_checksum": stringOptionBuiltin("git_checksum", "git_option", buildkitllb.GitChecksum),
		"git_fetch_by_commit": builtin("git_fetch_by_commit", gitFetchByCommitBuiltin),
		"git_bundle_url": builtin("git_bundle_url", gitBundleURLBuiltin),
		"git_bundle_oci_store": builtin("git_bundle_oci_store", gitBundleOCIStoreBuiltin),
		"git_checkout_bundle": builtin("git_checkout_bundle", gitCheckoutBundleBuiltin),
		"auth_header_secret": stringOptionBuiltin("auth_header_secret", "auth_option", buildkitllb.AuthHeaderSecret),
	}
}

func localSourceBuiltins() map[string]starlark.Value {
	return map[string]starlark.Value{
		"local": builtin("local", localSourceBuiltin),
		"session_id": stringOptionBuiltin("session_id", "local_option", buildkitllb.SessionID),
		"include_patterns": stringListOptionBuiltin("include_patterns", "local_option", buildkitllb.IncludePatterns),
		"follow_paths": stringListOptionBuiltin("follow_paths", "local_option", buildkitllb.FollowPaths),
		"exclude_patterns": stringListOptionBuiltin("exclude_patterns", "local_option", buildkitllb.ExcludePatterns),
		"shared_key_hint": stringOptionBuiltin("shared_key_hint", "local_option", buildkitllb.SharedKeyHint),
		"differ": builtin("differ", differBuiltin),
		"metadata_only_transfer": stringListOptionBuiltin("metadata_only_transfer", "local_option", buildkitllb.MetadataOnlyTransfer),
	}
}

func ociLayoutSourceBuiltins() map[string]starlark.Value {
	return map[string]starlark.Value{
		"oci_layout": builtin("oci_layout", ociLayoutSourceBuiltin),
		"oci_layout_blob": builtin("oci_layout_blob", ociLayoutBlobSourceBuiltin),
		"oci_store": builtin("oci_store", ociStoreBuiltin),
		"oci_layer_limit": intOptionBuiltin("oci_layer_limit", "oci_layout_option", buildkitllb.OCILayerLimit),
		"oci_checksum": builtin("oci_checksum", ociChecksumBuiltin),
		"image_blob_oci_store": builtin("image_blob_oci_store", imageBlobOCIStoreBuiltin),
	}
}

func httpSourceBuiltins() map[string]starlark.Value {
	return map[string]starlark.Value{
		"http": builtin("http", httpSourceBuiltin),
		"checksum": builtin("checksum", httpChecksumBuiltin),
		"header": builtin("header", httpHeaderBuiltin),
		"verify_pgp_signature": builtin("verify_pgp_signature", verifyPGPSignatureBuiltin),
	}
}

func sourceOpBuiltins() map[string]starlark.Value {
	return map[string]starlark.Value{
		"new_source": builtin("new_source", newSourceBuiltin),
	}
}

func scratchSourceBuiltin(_ *starlark.Thread, _ *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	if err := arity("scratch", args, kwargs, 0, 0); err != nil { return nil, err }
	return WrapState(buildkitllb.Scratch()), nil
}

func imageSourceBuiltin(_ *starlark.Thread, _ *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	if err := arity("image", args, kwargs, 1, -1); err != nil { return nil, err }
	ref, err := stringAt("image", args, 0); if err != nil { return nil, err }
	options, err := optionsFromArgs[buildkitllb.ImageOption](args, 1, "image_option")
	if err != nil { return nil, err }
	return WrapState(buildkitllb.Image(ref, options...)), nil
}

func imageBlobSourceBuiltin(_ *starlark.Thread, _ *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	if err := arity("image_blob", args, kwargs, 1, -1); err != nil { return nil, err }
	ref, err := stringAt("image_blob", args, 0); if err != nil { return nil, err }
	options, err := optionsFromArgs[buildkitllb.ImageBlobOption](args, 1, "image_blob_option")
	if err != nil { return nil, err }
	return WrapState(buildkitllb.ImageBlob(ref, options...)), nil
}

func gitSourceBuiltin(_ *starlark.Thread, _ *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	if len(kwargs) != 0 { return nil, fmt.Errorf("git: use positional url, optional fragment, then git_option values") }
	if len(args) < 1 { return nil, fmt.Errorf("git: missing repository url") }
	url, err := stringAt("git", args, 0); if err != nil { return nil, err }
	fragment := ""
	start := 1
	if len(args) > 1 {
		if value, ok := args[1].(starlark.String); ok {
			fragment = string(value)
			start = 2
		}
	}
	options, err := optionsFromArgs[buildkitllb.GitOption](args, start, "git_option")
	if err != nil { return nil, err }
	return WrapState(buildkitllb.Git(url, fragment, options...)), nil
}

func localSourceBuiltin(_ *starlark.Thread, _ *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	if err := arity("local", args, kwargs, 1, -1); err != nil { return nil, err }
	name, err := stringAt("local", args, 0); if err != nil { return nil, err }
	options, err := optionsFromArgs[buildkitllb.LocalOption](args, 1, "local_option")
	if err != nil { return nil, err }
	return WrapState(buildkitllb.Local(name, options...)), nil
}

func ociLayoutSourceBuiltin(_ *starlark.Thread, _ *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	if err := arity("oci_layout", args, kwargs, 1, -1); err != nil { return nil, err }
	ref, err := stringAt("oci_layout", args, 0); if err != nil { return nil, err }
	options, err := optionsFromArgs[buildkitllb.OCILayoutOption](args, 1, "oci_layout_option")
	if err != nil { return nil, err }
	return WrapState(buildkitllb.OCILayout(ref, options...)), nil
}

func ociLayoutBlobSourceBuiltin(_ *starlark.Thread, _ *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	if err := arity("oci_layout_blob", args, kwargs, 1, -1); err != nil { return nil, err }
	ref, err := stringAt("oci_layout_blob", args, 0); if err != nil { return nil, err }
	options, err := optionsFromArgs[buildkitllb.ImageBlobOption](args, 1, "image_blob_option")
	if err != nil { return nil, err }
	return WrapState(buildkitllb.OCILayoutBlob(ref, options...)), nil
}

func httpSourceBuiltin(_ *starlark.Thread, _ *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	if err := arity("http", args, kwargs, 1, -1); err != nil { return nil, err }
	sourceURL, err := stringAt("http", args, 0); if err != nil { return nil, err }
	parsed, err := url.Parse(sourceURL)
	if err != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") || parsed.Host == "" {
		return nil, fmt.Errorf("http: expected an absolute http(s) url")
	}
	options, err := optionsFromArgs[buildkitllb.HTTPOption](args, 1, "http_option")
	if err != nil { return nil, err }
	return WrapState(buildkitllb.HTTP(sourceURL, options...)), nil
}

func newSourceBuiltin(_ *starlark.Thread, _ *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	if err := arity("new_source", args, kwargs, 1, -1); err != nil { return nil, err }
	id, err := stringAt("new_source", args, 0); if err != nil { return nil, err }
	var attrs map[string]string
	start := 1
	if len(args) > 1 {
		if _, ok := args[1].(*starlark.Dict); ok {
			attrs, err = stringDict(args[1], "new_source attrs")
			if err != nil { return nil, err }
			start = 2
		}
	}
	constraints, err := constraintsFromArgs(args, start)
	if err != nil { return nil, err }
	return WrapSourceOp(buildkitllb.NewSource(id, attrs, *constraints)), nil
}

func gitSkipSubmodulesBuiltin(_ *starlark.Thread, _ *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	if err := arity("git_skip_submodules", args, kwargs, 0, 0); err != nil { return nil, err }
	return wrapOption("git_option", buildkitllb.GitSkipSubmodules()), nil
}
func gitMTimeCommitBuiltin(_ *starlark.Thread, _ *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	if err := arity("git_m_time_commit", args, kwargs, 0, 0); err != nil { return nil, err }
	return wrapOption("git_option", buildkitllb.GitMTimeCommit()), nil
}
func keepGitDirBuiltin(_ *starlark.Thread, _ *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	if err := arity("keep_git_dir", args, kwargs, 0, 0); err != nil { return nil, err }
	return wrapOption("git_option", buildkitllb.KeepGitDir()), nil
}
func gitFetchByCommitBuiltin(_ *starlark.Thread, _ *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	if err := arity("git_fetch_by_commit", args, kwargs, 0, 0); err != nil { return nil, err }
	return wrapOption("git_option", buildkitllb.GitFetchByCommit()), nil
}
func gitCheckoutBundleBuiltin(_ *starlark.Thread, _ *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	if err := arity("git_checkout_bundle", args, kwargs, 0, 0); err != nil { return nil, err }
	return wrapOption("git_option", buildkitllb.GitCheckoutBundle()), nil
}

func gitBundleURLBuiltin(_ *starlark.Thread, _ *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	if err := arity("git_bundle_url", args, kwargs, 1, -1); err != nil { return nil, err }
	locator, err := stringAt("git_bundle_url", args, 0); if err != nil { return nil, err }
	options, err := optionsFromArgs[buildkitllb.GitBundleOption](args, 1, "git_bundle_option")
	if err != nil { return nil, err }
	return wrapOption("git_option", buildkitllb.GitBundleURL(locator, options...)), nil
}

func gitBundleOCIStoreBuiltin(_ *starlark.Thread, _ *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	if err := arity("git_bundle_oci_store", args, kwargs, 2, 2); err != nil { return nil, err }
	session, err := stringAt("git_bundle_oci_store", args, 0); if err != nil { return nil, err }
	store, err := stringAt("git_bundle_oci_store", args, 1); if err != nil { return nil, err }
	return wrapOption("git_bundle_option", buildkitllb.GitBundleOCIStore(session, store)), nil
}

func differBuiltin(_ *starlark.Thread, _ *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	if err := arity("differ", args, kwargs, 2, 2); err != nil { return nil, err }
	kind, err := stringAt("differ", args, 0); if err != nil { return nil, err }
	required, err := boolAt("differ", args, 1); if err != nil { return nil, err }
	return wrapOption("local_option", buildkitllb.Differ(buildkitllb.DiffType(kind), required)), nil
}

func ociStoreBuiltin(_ *starlark.Thread, _ *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	if err := arity("oci_store", args, kwargs, 2, 2); err != nil { return nil, err }
	session, err := stringAt("oci_store", args, 0); if err != nil { return nil, err }
	store, err := stringAt("oci_store", args, 1); if err != nil { return nil, err }
	return wrapOption("oci_layout_option", buildkitllb.OCIStore(session, store)), nil
}

func ociChecksumBuiltin(_ *starlark.Thread, _ *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	if err := arity("oci_checksum", args, kwargs, 1, 1); err != nil { return nil, err }
	raw, err := stringAt("oci_checksum", args, 0); if err != nil { return nil, err }
	dgst, err := digest.Parse(raw); if err != nil { return nil, fmt.Errorf("oci_checksum: %w", err) }
	return wrapOption("oci_layout_option", buildkitllb.OCIChecksum(dgst)), nil
}

func imageBlobOCIStoreBuiltin(_ *starlark.Thread, _ *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	if err := arity("image_blob_oci_store", args, kwargs, 2, 2); err != nil { return nil, err }
	session, err := stringAt("image_blob_oci_store", args, 0); if err != nil { return nil, err }
	store, err := stringAt("image_blob_oci_store", args, 1); if err != nil { return nil, err }
	return wrapOption("image_blob_option", buildkitllb.ImageBlobOCIStore(session, store)), nil
}

func httpChecksumBuiltin(_ *starlark.Thread, _ *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	if err := arity("checksum", args, kwargs, 1, 1); err != nil { return nil, err }
	raw, err := stringAt("checksum", args, 0); if err != nil { return nil, err }
	dgst, err := digest.Parse(raw); if err != nil { return nil, fmt.Errorf("checksum: %w", err) }
	return wrapOption("http_option", buildkitllb.Checksum(dgst)), nil
}

func httpHeaderBuiltin(_ *starlark.Thread, _ *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	var header buildkitllb.HTTPHeader
	if err := UnpackArgs("header", args, kwargs,
		"accept?", &header.Accept,
		"user_agent?", &header.UserAgent,
	); err != nil { return nil, err }
	return wrapOption("http_option", buildkitllb.Header(header)), nil
}

func verifyPGPSignatureBuiltin(_ *starlark.Thread, _ *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	if err := arity("verify_pgp_signature", args, kwargs, 2, 2); err != nil { return nil, err }
	pubKey, err := bytesAt("verify_pgp_signature", args, 0); if err != nil { return nil, err }
	signature, err := bytesAt("verify_pgp_signature", args, 1); if err != nil { return nil, err }
	return wrapOption("http_option", buildkitllb.VerifyPGPSignature(buildkitllb.HTTPSignatureInfo{
		PubKey: pubKey, Signature: signature,
	})), nil
}

func stringListOptionBuiltin[T any](name, kind string, build func([]string) T) *starlark.Builtin {
	return optionBuiltin(name, kind, func(_ *starlark.Thread, args starlark.Tuple, kwargs []starlark.Tuple) (any, error) {
		if err := arity(name, args, kwargs, 1, 1); err != nil { return nil, err }
		values, err := stringList(args[0])
		if err != nil { return nil, err }
		return build(values), nil
	})
}

func diffTypeValues() map[string]starlark.Value {
	return map[string]starlark.Value{
		"diff_none": starlark.String(string(buildkitllb.DiffNone)),
		"diff_metadata": starlark.String(string(buildkitllb.DiffMetadata)),
	}
}

func gitAuthValues() map[string]starlark.Value {
	return map[string]starlark.Value{
		"git_auth_header_key": starlark.String(buildkitllb.GitAuthHeaderKey),
		"git_auth_token_key": starlark.String(buildkitllb.GitAuthTokenKey),
	}
}

var (
	_ starlark.Value    = (*SourceOpValue)(nil)
	_ starlark.HasAttrs = (*SourceOpValue)(nil)
	_ starlark.Value    = (*HTTPHeaderValue)(nil)
	_ starlark.HasAttrs = (*HTTPHeaderValue)(nil)
	_ starlark.Value    = (*HTTPSignatureInfoValue)(nil)
	_ starlark.HasAttrs = (*HTTPSignatureInfoValue)(nil)
)


// imageOptionValues contains ImageOption values rather than source constructors.
func imageOptionValues() map[string]starlark.Value {
	return map[string]starlark.Value{
		"mark_image_internal": wrapOption("image_option", buildkitllb.MarkImageInternal),
	}
}
