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

func (v *SourceOpValue) String() string        { return "<llb.SourceOp>" }
func (v *SourceOpValue) Type() string          { return "llb.SourceOp" }
func (v *SourceOpValue) Freeze()               {}
func (v *SourceOpValue) Truth() starlark.Bool  { return starlark.Bool(v != nil && v.source != nil) }
func (v *SourceOpValue) Hash() (uint32, error) { return 0, fmt.Errorf("unhashable: llb.SourceOp") }

func (v *SourceOpValue) Attr(name string) (starlark.Value, error) {
	if v == nil || v.source == nil { return nil, fmt.Errorf("llb.SourceOp: nil receiver") }
	switch name {
	case "Output", "output":
		return boundBuiltin("SourceOp.Output", func(_ *starlark.Thread, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
			if err := arity("SourceOp.Output", args, kwargs, 0, 0); err != nil { return nil, err }
			return WrapOutput(v.source.Output()), nil
		}), nil
	case "Inputs", "inputs":
		return boundBuiltin("SourceOp.Inputs", func(_ *starlark.Thread, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
			if err := arity("SourceOp.Inputs", args, kwargs, 0, 0); err != nil { return nil, err }
			return starlark.NewList(nil), nil
		}), nil
	case "Validate", "validate":
		return boundBuiltin("SourceOp.Validate", func(thread *starlark.Thread, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
			if err := arity("SourceOp.Validate", args, kwargs, 0, -1); err != nil { return nil, err }
			constraints, err := constraintsFromArgs(args, 0); if err != nil { return nil, err }
			if err := v.source.Validate(callContext(thread), constraints); err != nil { return nil, err }
			return starlark.None, nil
		}), nil
	case "Marshal", "marshal":
		return boundBuiltin("SourceOp.Marshal", func(thread *starlark.Thread, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
			if err := arity("SourceOp.Marshal", args, kwargs, 0, -1); err != nil { return nil, err }
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
	return []string{"Inputs", "Marshal", "Output", "Validate", "inputs", "marshal", "output", "validate"}
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
func (v *HTTPHeaderValue) String() string        { return "<llb.HTTPHeader>" }
func (v *HTTPHeaderValue) Type() string          { return "llb.HTTPHeader" }
func (v *HTTPHeaderValue) Freeze()               {}
func (v *HTTPHeaderValue) Truth() starlark.Bool  { return starlark.Bool(v != nil) }
func (v *HTTPHeaderValue) Hash() (uint32, error) { return 0, fmt.Errorf("unhashable: llb.HTTPHeader") }
func (v *HTTPHeaderValue) Attr(name string) (starlark.Value, error) {
	if v == nil { return nil, fmt.Errorf("llb.HTTPHeader: nil receiver") }
	switch name {
	case "accept", "Accept": return starlark.String(v.header.Accept), nil
	case "user_agent", "UserAgent": return starlark.String(v.header.UserAgent), nil
	default: return nil, nil
	}
}
func (v *HTTPHeaderValue) AttrNames() []string { return []string{"accept", "user_agent", "Accept", "UserAgent"} }

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
func (v *HTTPSignatureInfoValue) String() string        { return "<llb.HTTPSignatureInfo>" }
func (v *HTTPSignatureInfoValue) Type() string          { return "llb.HTTPSignatureInfo" }
func (v *HTTPSignatureInfoValue) Freeze()               {}
func (v *HTTPSignatureInfoValue) Truth() starlark.Bool  { return starlark.Bool(v != nil) }
func (v *HTTPSignatureInfoValue) Hash() (uint32, error) { return 0, fmt.Errorf("unhashable: llb.HTTPSignatureInfo") }
func (v *HTTPSignatureInfoValue) Attr(name string) (starlark.Value, error) {
	if v == nil { return nil, fmt.Errorf("llb.HTTPSignatureInfo: nil receiver") }
	switch name {
	case "pub_key", "PubKey": return starlark.Bytes(v.info.PubKey), nil
	case "signature", "Signature": return starlark.Bytes(v.info.Signature), nil
	default: return nil, nil
	}
}
func (v *HTTPSignatureInfoValue) AttrNames() []string { return []string{"pub_key", "signature", "PubKey", "Signature"} }

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
		"Scratch": builtin("Scratch", scratchSourceBuiltin),
		"Image": builtin("Image", imageSourceBuiltin),
		"ImageBlob": builtin("ImageBlob", imageBlobSourceBuiltin),
	}
}

func gitSourceBuiltins() map[string]starlark.Value {
	return map[string]starlark.Value{
		"Git": builtin("Git", gitSourceBuiltin),
		"GitRef": stringOptionBuiltin("GitRef", "GitOption", buildkitllb.GitRef),
		"GitSubDir": stringOptionBuiltin("GitSubDir", "GitOption", buildkitllb.GitSubDir),
		"GitSkipSubmodules": builtin("GitSkipSubmodules", gitSkipSubmodulesBuiltin),
		"GitMTimeCommit": builtin("GitMTimeCommit", gitMTimeCommitBuiltin),
		"GitMTime": stringOptionBuiltin("GitMTime", "GitOption", buildkitllb.GitMTime),
		"GitAdvice": boolOptionBuiltin("GitAdvice", "GitOption", buildkitllb.GitAdvice),
		"GitDebugCommands": boolOptionBuiltin("GitDebugCommands", "GitOption", buildkitllb.GitDebugCommands),
		"KeepGitDir": builtin("KeepGitDir", keepGitDirBuiltin),
		"AuthTokenSecret": stringOptionBuiltin("AuthTokenSecret", "GitOption", buildkitllb.AuthTokenSecret),
		"KnownSSHHosts": stringOptionBuiltin("KnownSSHHosts", "GitOption", buildkitllb.KnownSSHHosts),
		"MountSSHSock": stringOptionBuiltin("MountSSHSock", "GitOption", buildkitllb.MountSSHSock),
		"GitChecksum": stringOptionBuiltin("GitChecksum", "GitOption", buildkitllb.GitChecksum),
		"GitFetchByCommit": builtin("GitFetchByCommit", gitFetchByCommitBuiltin),
		"GitBundleURL": builtin("GitBundleURL", gitBundleURLBuiltin),
		"GitBundleOCIStore": builtin("GitBundleOCIStore", gitBundleOCIStoreBuiltin),
		"GitCheckoutBundle": builtin("GitCheckoutBundle", gitCheckoutBundleBuiltin),
		"AuthHeaderSecret": stringOptionBuiltin("AuthHeaderSecret", "AuthOption", buildkitllb.AuthHeaderSecret),
	}
}

func localSourceBuiltins() map[string]starlark.Value {
	return map[string]starlark.Value{
		"Local": builtin("Local", localSourceBuiltin),
		"SessionID": stringOptionBuiltin("SessionID", "LocalOption", buildkitllb.SessionID),
		"IncludePatterns": stringListOptionBuiltin("IncludePatterns", "LocalOption", buildkitllb.IncludePatterns),
		"FollowPaths": stringListOptionBuiltin("FollowPaths", "LocalOption", buildkitllb.FollowPaths),
		"ExcludePatterns": stringListOptionBuiltin("ExcludePatterns", "LocalOption", buildkitllb.ExcludePatterns),
		"SharedKeyHint": stringOptionBuiltin("SharedKeyHint", "LocalOption", buildkitllb.SharedKeyHint),
		"Differ": builtin("Differ", differBuiltin),
		"MetadataOnlyTransfer": stringListOptionBuiltin("MetadataOnlyTransfer", "LocalOption", buildkitllb.MetadataOnlyTransfer),
	}
}

func ociLayoutSourceBuiltins() map[string]starlark.Value {
	return map[string]starlark.Value{
		"OCILayout": builtin("OCILayout", ociLayoutSourceBuiltin),
		"OCILayoutBlob": builtin("OCILayoutBlob", ociLayoutBlobSourceBuiltin),
		"OCIStore": builtin("OCIStore", ociStoreBuiltin),
		"OCILayerLimit": intOptionBuiltin("OCILayerLimit", "OCILayoutOption", buildkitllb.OCILayerLimit),
		"OCIChecksum": builtin("OCIChecksum", ociChecksumBuiltin),
		"ImageBlobOCIStore": builtin("ImageBlobOCIStore", imageBlobOCIStoreBuiltin),
	}
}

func httpSourceBuiltins() map[string]starlark.Value {
	return map[string]starlark.Value{
		"HTTP": builtin("HTTP", httpSourceBuiltin),
		"Checksum": builtin("Checksum", httpChecksumBuiltin),
		"Header": builtin("Header", httpHeaderBuiltin),
		"VerifyPGPSignature": builtin("VerifyPGPSignature", verifyPGPSignatureBuiltin),
	}
}

func sourceOpBuiltins() map[string]starlark.Value {
	return map[string]starlark.Value{
		"NewSource": builtin("NewSource", newSourceBuiltin),
	}
}

func scratchSourceBuiltin(_ *starlark.Thread, _ *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	if err := arity("Scratch", args, kwargs, 0, 0); err != nil { return nil, err }
	return WrapState(buildkitllb.Scratch()), nil
}

func imageSourceBuiltin(_ *starlark.Thread, _ *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	if err := arity("Image", args, kwargs, 1, -1); err != nil { return nil, err }
	ref, err := stringAt("Image", args, 0); if err != nil { return nil, err }
	options, err := optionsFromArgs[buildkitllb.ImageOption](args, 1, "ImageOption")
	if err != nil { return nil, err }
	return WrapState(buildkitllb.Image(ref, options...)), nil
}

func imageBlobSourceBuiltin(_ *starlark.Thread, _ *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	if err := arity("ImageBlob", args, kwargs, 1, -1); err != nil { return nil, err }
	ref, err := stringAt("ImageBlob", args, 0); if err != nil { return nil, err }
	options, err := optionsFromArgs[buildkitllb.ImageBlobOption](args, 1, "ImageBlobOption")
	if err != nil { return nil, err }
	return WrapState(buildkitllb.ImageBlob(ref, options...)), nil
}

func gitSourceBuiltin(_ *starlark.Thread, _ *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	if len(kwargs) != 0 { return nil, fmt.Errorf("Git: use positional URL, optional fragment, then GitOption values") }
	if len(args) < 1 { return nil, fmt.Errorf("Git: missing repository URL") }
	url, err := stringAt("Git", args, 0); if err != nil { return nil, err }
	fragment := ""
	start := 1
	if len(args) > 1 {
		if value, ok := args[1].(starlark.String); ok {
			fragment = string(value)
			start = 2
		}
	}
	options, err := optionsFromArgs[buildkitllb.GitOption](args, start, "GitOption")
	if err != nil { return nil, err }
	return WrapState(buildkitllb.Git(url, fragment, options...)), nil
}

func localSourceBuiltin(_ *starlark.Thread, _ *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	if err := arity("Local", args, kwargs, 1, -1); err != nil { return nil, err }
	name, err := stringAt("Local", args, 0); if err != nil { return nil, err }
	options, err := optionsFromArgs[buildkitllb.LocalOption](args, 1, "LocalOption")
	if err != nil { return nil, err }
	return WrapState(buildkitllb.Local(name, options...)), nil
}

func ociLayoutSourceBuiltin(_ *starlark.Thread, _ *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	if err := arity("OCILayout", args, kwargs, 1, -1); err != nil { return nil, err }
	ref, err := stringAt("OCILayout", args, 0); if err != nil { return nil, err }
	options, err := optionsFromArgs[buildkitllb.OCILayoutOption](args, 1, "OCILayoutOption")
	if err != nil { return nil, err }
	return WrapState(buildkitllb.OCILayout(ref, options...)), nil
}

func ociLayoutBlobSourceBuiltin(_ *starlark.Thread, _ *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	if err := arity("OCILayoutBlob", args, kwargs, 1, -1); err != nil { return nil, err }
	ref, err := stringAt("OCILayoutBlob", args, 0); if err != nil { return nil, err }
	options, err := optionsFromArgs[buildkitllb.ImageBlobOption](args, 1, "ImageBlobOption")
	if err != nil { return nil, err }
	return WrapState(buildkitllb.OCILayoutBlob(ref, options...)), nil
}

func httpSourceBuiltin(_ *starlark.Thread, _ *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	if err := arity("HTTP", args, kwargs, 1, -1); err != nil { return nil, err }
	sourceURL, err := stringAt("HTTP", args, 0); if err != nil { return nil, err }
	parsed, err := url.Parse(sourceURL)
	if err != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") || parsed.Host == "" {
		return nil, fmt.Errorf("HTTP: expected an absolute http(s) URL")
	}
	options, err := optionsFromArgs[buildkitllb.HTTPOption](args, 1, "HTTPOption")
	if err != nil { return nil, err }
	return WrapState(buildkitllb.HTTP(sourceURL, options...)), nil
}

func newSourceBuiltin(_ *starlark.Thread, _ *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	if err := arity("NewSource", args, kwargs, 1, -1); err != nil { return nil, err }
	id, err := stringAt("NewSource", args, 0); if err != nil { return nil, err }
	var attrs map[string]string
	start := 1
	if len(args) > 1 {
		if _, ok := args[1].(*starlark.Dict); ok {
			attrs, err = stringDict(args[1], "NewSource attrs")
			if err != nil { return nil, err }
			start = 2
		}
	}
	constraints, err := constraintsFromArgs(args, start)
	if err != nil { return nil, err }
	return WrapSourceOp(buildkitllb.NewSource(id, attrs, *constraints)), nil
}

func gitSkipSubmodulesBuiltin(_ *starlark.Thread, _ *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	if err := arity("GitSkipSubmodules", args, kwargs, 0, 0); err != nil { return nil, err }
	return wrapOption("GitOption", buildkitllb.GitSkipSubmodules()), nil
}
func gitMTimeCommitBuiltin(_ *starlark.Thread, _ *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	if err := arity("GitMTimeCommit", args, kwargs, 0, 0); err != nil { return nil, err }
	return wrapOption("GitOption", buildkitllb.GitMTimeCommit()), nil
}
func keepGitDirBuiltin(_ *starlark.Thread, _ *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	if err := arity("KeepGitDir", args, kwargs, 0, 0); err != nil { return nil, err }
	return wrapOption("GitOption", buildkitllb.KeepGitDir()), nil
}
func gitFetchByCommitBuiltin(_ *starlark.Thread, _ *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	if err := arity("GitFetchByCommit", args, kwargs, 0, 0); err != nil { return nil, err }
	return wrapOption("GitOption", buildkitllb.GitFetchByCommit()), nil
}
func gitCheckoutBundleBuiltin(_ *starlark.Thread, _ *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	if err := arity("GitCheckoutBundle", args, kwargs, 0, 0); err != nil { return nil, err }
	return wrapOption("GitOption", buildkitllb.GitCheckoutBundle()), nil
}

func gitBundleURLBuiltin(_ *starlark.Thread, _ *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	if err := arity("GitBundleURL", args, kwargs, 1, -1); err != nil { return nil, err }
	locator, err := stringAt("GitBundleURL", args, 0); if err != nil { return nil, err }
	options, err := optionsFromArgs[buildkitllb.GitBundleOption](args, 1, "GitBundleOption")
	if err != nil { return nil, err }
	return wrapOption("GitOption", buildkitllb.GitBundleURL(locator, options...)), nil
}

func gitBundleOCIStoreBuiltin(_ *starlark.Thread, _ *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	if err := arity("GitBundleOCIStore", args, kwargs, 2, 2); err != nil { return nil, err }
	session, err := stringAt("GitBundleOCIStore", args, 0); if err != nil { return nil, err }
	store, err := stringAt("GitBundleOCIStore", args, 1); if err != nil { return nil, err }
	return wrapOption("GitBundleOption", buildkitllb.GitBundleOCIStore(session, store)), nil
}

func differBuiltin(_ *starlark.Thread, _ *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	if err := arity("Differ", args, kwargs, 2, 2); err != nil { return nil, err }
	kind, err := stringAt("Differ", args, 0); if err != nil { return nil, err }
	required, err := boolAt("Differ", args, 1); if err != nil { return nil, err }
	return wrapOption("LocalOption", buildkitllb.Differ(buildkitllb.DiffType(kind), required)), nil
}

func ociStoreBuiltin(_ *starlark.Thread, _ *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	if err := arity("OCIStore", args, kwargs, 2, 2); err != nil { return nil, err }
	session, err := stringAt("OCIStore", args, 0); if err != nil { return nil, err }
	store, err := stringAt("OCIStore", args, 1); if err != nil { return nil, err }
	return wrapOption("OCILayoutOption", buildkitllb.OCIStore(session, store)), nil
}

func ociChecksumBuiltin(_ *starlark.Thread, _ *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	if err := arity("OCIChecksum", args, kwargs, 1, 1); err != nil { return nil, err }
	raw, err := stringAt("OCIChecksum", args, 0); if err != nil { return nil, err }
	dgst, err := digest.Parse(raw); if err != nil { return nil, fmt.Errorf("OCIChecksum: %w", err) }
	return wrapOption("OCILayoutOption", buildkitllb.OCIChecksum(dgst)), nil
}

func imageBlobOCIStoreBuiltin(_ *starlark.Thread, _ *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	if err := arity("ImageBlobOCIStore", args, kwargs, 2, 2); err != nil { return nil, err }
	session, err := stringAt("ImageBlobOCIStore", args, 0); if err != nil { return nil, err }
	store, err := stringAt("ImageBlobOCIStore", args, 1); if err != nil { return nil, err }
	return wrapOption("ImageBlobOption", buildkitllb.ImageBlobOCIStore(session, store)), nil
}

func httpChecksumBuiltin(_ *starlark.Thread, _ *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	if err := arity("Checksum", args, kwargs, 1, 1); err != nil { return nil, err }
	raw, err := stringAt("Checksum", args, 0); if err != nil { return nil, err }
	dgst, err := digest.Parse(raw); if err != nil { return nil, fmt.Errorf("Checksum: %w", err) }
	return wrapOption("HTTPOption", buildkitllb.Checksum(dgst)), nil
}

func httpHeaderBuiltin(_ *starlark.Thread, _ *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	var header buildkitllb.HTTPHeader
	if err := starlark.UnpackArgs("Header", args, kwargs,
		"accept?", &header.Accept,
		"user_agent?", &header.UserAgent,
	); err != nil { return nil, err }
	return wrapOption("HTTPOption", buildkitllb.Header(header)), nil
}

func verifyPGPSignatureBuiltin(_ *starlark.Thread, _ *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	if err := arity("VerifyPGPSignature", args, kwargs, 2, 2); err != nil { return nil, err }
	pubKey, err := bytesAt("VerifyPGPSignature", args, 0); if err != nil { return nil, err }
	signature, err := bytesAt("VerifyPGPSignature", args, 1); if err != nil { return nil, err }
	return wrapOption("HTTPOption", buildkitllb.VerifyPGPSignature(buildkitllb.HTTPSignatureInfo{
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
		"DiffNone": starlark.String(string(buildkitllb.DiffNone)),
		"DiffMetadata": starlark.String(string(buildkitllb.DiffMetadata)),
	}
}

func gitAuthValues() map[string]starlark.Value {
	return map[string]starlark.Value{
		"GitAuthHeaderKey": starlark.String(buildkitllb.GitAuthHeaderKey),
		"GitAuthTokenKey": starlark.String(buildkitllb.GitAuthTokenKey),
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
		"MarkImageInternal": wrapOption("ImageOption", buildkitllb.MarkImageInternal),
	}
}
