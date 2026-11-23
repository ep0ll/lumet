package llb

import (
	"fmt"
	"net"
	"os"
	"time"

	buildkitllb "github.com/moby/buildkit/client/llb"
	"github.com/opencontainers/go-digest"
	"github.com/moby/buildkit/solver/pb"
	"go.starlark.net/starlark"
)

func optionBuiltins() map[string]starlark.Value {
	return map[string]starlark.Value{
		// Constraints and state metadata.
		"WithCustomName": optionBuiltin("WithCustomName", func(a starlark.Tuple) (any, error) {
			name, err := stringAt("WithCustomName", a, 0); if err != nil { return nil, err }
			return buildkitllb.WithCustomName(name), nil
		}, 1, 1, "ConstraintsOpt"),
		"WithExportCache": optionBuiltin("WithExportCache", func(a starlark.Tuple) (any, error) { return buildkitllb.WithExportCache(), nil }, 0, 0, "ConstraintsOpt"),
		"WithoutExportCache": optionBuiltin("WithoutExportCache", func(a starlark.Tuple) (any, error) { return buildkitllb.WithoutExportCache(), nil }, 0, 0, "ConstraintsOpt"),
		"WithoutDefaultExportCache": optionBuiltin("WithoutDefaultExportCache", func(a starlark.Tuple) (any, error) { return buildkitllb.WithoutDefaultExportCache(), nil }, 0, 0, "ConstraintsOpt"),
		"WithDescription": optionBuiltin("WithDescription", withDescription, 1, 1, "ConstraintsOpt"),
		"Network": optionBuiltin("Network", func(a starlark.Tuple) (any, error) {
			n, err := intAt("Network", a, 0); if err != nil { return nil, err }
			return buildkitllb.Network(pb.NetMode(n)), nil
		}, 1, 1, "StateOption"),
		"Security": optionBuiltin("Security", func(a starlark.Tuple) (any, error) {
			n, err := intAt("Security", a, 0); if err != nil { return nil, err }
			return buildkitllb.Security(pb.SecurityMode(n)), nil
		}, 1, 1, "StateOption"),
		"AddEnv": optionBuiltin("AddEnv", func(a starlark.Tuple) (any, error) {
			k, err := stringAt("AddEnv", a, 0); if err != nil { return nil, err }
			v, err := stringAt("AddEnv", a, 1); if err != nil { return nil, err }
			return buildkitllb.AddEnv(k, v), nil
		}, 2, 2, "StateOption"),
		"AddEnvf": optionBuiltin("AddEnvf", addEnvfOption, 2, -1, "StateOption"),
		"DelEnv": optionBuiltin("DelEnv", func(a starlark.Tuple) (any, error) {
			k, err := stringAt("DelEnv", a, 0); if err != nil { return nil, err }
			return buildkitllb.DelEnv(k), nil
		}, 1, 1, "StateOption"),
		"Dir": optionBuiltin("Dir", func(a starlark.Tuple) (any, error) {
			p, err := stringAt("Dir", a, 0); if err != nil { return nil, err }
			return buildkitllb.Dir(p), nil
		}, 1, 1, "StateOption"),
		"Dirf": optionBuiltin("Dirf", dirfOption, 1, -1, "StateOption"),
		"User": optionBuiltin("User", func(a starlark.Tuple) (any, error) {
			u, err := stringAt("User", a, 0); if err != nil { return nil, err }
			return buildkitllb.User(u), nil
		}, 1, 1, "StateOption"),
		"Hostname": optionBuiltin("Hostname", func(a starlark.Tuple) (any, error) {
			h, err := stringAt("Hostname", a, 0); if err != nil { return nil, err }
			return buildkitllb.Hostname(h), nil
		}, 1, 1, "StateOption"),

		// Image, Git, local, HTTP, and OCI-layout source options.
		"GitRef": stringOption("GitRef", "GitOption", buildkitllb.GitRef),
		"GitSubDir": stringOption("GitSubDir", "GitOption", buildkitllb.GitSubDir),
		"GitMTime": stringOption("GitMTime", "GitOption", buildkitllb.GitMTime),
		"GitMTimeCommit": optionBuiltin("GitMTimeCommit", func(starlark.Tuple) (any, error) { return buildkitllb.GitMTimeCommit(), nil }, 0, 0, "GitOption"),
		"GitSkipSubmodules": optionBuiltin("GitSkipSubmodules", func(starlark.Tuple) (any, error) { return buildkitllb.GitSkipSubmodules(), nil }, 0, 0, "GitOption"),
		"GitAdvice": boolOption("GitAdvice", "GitOption", buildkitllb.GitAdvice),
		"GitDebugCommands": boolOption("GitDebugCommands", "GitOption", buildkitllb.GitDebugCommands),
		"KeepGitDir": optionBuiltin("KeepGitDir", func(starlark.Tuple) (any, error) { return buildkitllb.KeepGitDir(), nil }, 0, 0, "GitOption"),
		"AuthTokenSecret": stringOption("AuthTokenSecret", "AuthOption", buildkitllb.AuthTokenSecret),
		"AuthHeaderSecret": stringOption("AuthHeaderSecret", "AuthOption", buildkitllb.AuthHeaderSecret),
		"KnownSSHHosts": stringOption("KnownSSHHosts", "GitOption", buildkitllb.KnownSSHHosts),
		"MountSSHSock": stringOption("MountSSHSock", "GitOption", buildkitllb.MountSSHSock),
		"GitChecksum": stringOption("GitChecksum", "GitOption", buildkitllb.GitChecksum),
		"GitFetchByCommit": optionBuiltin("GitFetchByCommit", func(starlark.Tuple) (any, error) { return buildkitllb.GitFetchByCommit(), nil }, 0, 0, "GitOption"),
		"GitBundleURL": optionBuiltin("GitBundleURL", func(a starlark.Tuple) (any, error) {
			locator, err := stringAt("GitBundleURL", a, 0); if err != nil { return nil, err }
			opts, err := optionsFromArgs[buildkitllb.GitBundleOption](a, 1, "GitBundleOption"); if err != nil { return nil, err }
			return buildkitllb.GitBundleURL(locator, opts...), nil
		}, 1, -1, "GitOption"),
		"GitBundleOCIStore": optionBuiltin("GitBundleOCIStore", func(a starlark.Tuple) (any, error) {
			session, err := stringAt("GitBundleOCIStore", a, 0); if err != nil { return nil, err }
			store, err := stringAt("GitBundleOCIStore", a, 1); if err != nil { return nil, err }
			return buildkitllb.GitBundleOCIStore(session, store), nil
		}, 2, 2, "GitBundleOption"),
		"GitCheckoutBundle": optionBuiltin("GitCheckoutBundle", func(starlark.Tuple) (any, error) { return buildkitllb.GitCheckoutBundle(), nil }, 0, 0, "GitOption"),
		"SessionID": stringOption("SessionID", "LocalOption", buildkitllb.SessionID),
		"IncludePatterns": stringListOption("IncludePatterns", "LocalOption", buildkitllb.IncludePatterns),
		"FollowPaths": stringListOption("FollowPaths", "LocalOption", buildkitllb.FollowPaths),
		"ExcludePatterns": stringListOption("ExcludePatterns", "LocalOption", buildkitllb.ExcludePatterns),
		"SharedKeyHint": stringOption("SharedKeyHint", "LocalOption", buildkitllb.SharedKeyHint),
		"MetadataOnlyTransfer": stringListOption("MetadataOnlyTransfer", "LocalOption", buildkitllb.MetadataOnlyTransfer),
		"Differ": optionBuiltin("Differ", func(a starlark.Tuple) (any, error) {
			diff, err := stringAt("Differ", a, 0); if err != nil { return nil, err }
			required, err := boolAt("Differ", a, 1); if err != nil { return nil, err }
			return buildkitllb.Differ(buildkitllb.DiffType(diff), required), nil
		}, 2, 2, "LocalOption"),
		"OCIStore": optionBuiltin("OCIStore", func(a starlark.Tuple) (any, error) {
			session, err := stringAt("OCIStore", a, 0); if err != nil { return nil, err }
			store, err := stringAt("OCIStore", a, 1); if err != nil { return nil, err }
			return buildkitllb.OCIStore(session, store), nil
		}, 2, 2, "OCILayoutOption"),
		"OCILayerLimit": intOption("OCILayerLimit", "OCILayoutOption", buildkitllb.OCILayerLimit),
		"ImageBlobOCIStore": optionBuiltin("ImageBlobOCIStore", func(a starlark.Tuple) (any, error) {
			session, err := stringAt("ImageBlobOCIStore", a, 0); if err != nil { return nil, err }
			store, err := stringAt("ImageBlobOCIStore", a, 1); if err != nil { return nil, err }
			return buildkitllb.ImageBlobOCIStore(session, store), nil
		}, 2, 2, "ImageBlobOption"),
		"Checksum": optionBuiltin("Checksum", checksumOption, 1, 1, "HTTPOption"),
		"HTTPChecksum": optionBuiltin("HTTPChecksum", checksumOption, 1, 1, "HTTPOption"),
		"OCIChecksum": optionBuiltin("OCIChecksum", ociChecksumOption, 1, 1, "OCILayoutOption"),
		"WithImageChecksum": optionBuiltin("WithImageChecksum", imageChecksumOption, 1, 1, "ImageOption"),
		"WithLayerLimit": intOption("WithLayerLimit", "ImageOption", buildkitllb.WithLayerLimit),
		"ResolveDigest": boolOption("ResolveDigest", "ImageOption", buildkitllb.ResolveDigest),
		"Filename": stringOption("Filename", "FileInfoOption", buildkitllb.Filename),
		"Chmod": intOption("Chmod", "FileInfoOption", func(n int) buildkitllb.FileInfoOption { return buildkitllb.Chmod(os.FileMode(n)) }),
		"Chown": optionBuiltin("Chown", func(a starlark.Tuple) (any, error) {
			uid, err := intAt("Chown", a, 0); if err != nil { return nil, err }
			gid, err := intAt("Chown", a, 1); if err != nil { return nil, err }
			return buildkitllb.Chown(uid, gid), nil
		}, 2, 2, "FileInfoOption"),

		// RUN and mount options.
		"Shlex": stringOption("Shlex", "RunOption", buildkitllb.Shlex),
		"Shlexf": optionBuiltin("Shlexf", shlexfOption, 1, -1, "RunOption"),
		"Args": optionBuiltin("Args", func(a starlark.Tuple) (any, error) {
			if len(a) == 1 {
				values, err := stringList(a[0]); if err != nil { return nil, err }
				return buildkitllb.Args(values), nil
			}
			values := make([]string, len(a))
			for i := range a { v, err := stringAt("Args", a, i); if err != nil { return nil, err }; values[i] = v }
			return buildkitllb.Args(values), nil
		}, 0, -1, "RunOption"),
		"ReadonlyRootFS": optionBuiltin("ReadonlyRootFS", func(starlark.Tuple) (any, error) { return buildkitllb.ReadonlyRootFS(), nil }, 0, 0, "RunOption"),
		"ValidExitCodes": optionBuiltin("ValidExitCodes", func(a starlark.Tuple) (any, error) {
			codes := make([]int, len(a))
			for i := range a { n, err := intAt("ValidExitCodes", a, i); if err != nil { return nil, err }; codes[i] = n }
			return buildkitllb.ValidExitCodes(codes...), nil
		}, 0, -1, "RunOption"),
		"WithCgroupParent": stringOption("WithCgroupParent", "RunOption", buildkitllb.WithCgroupParent),
		"SourcePath": stringOption("SourcePath", "MountOption", buildkitllb.SourcePath),
		"AsPersistentCacheDir": optionBuiltin("AsPersistentCacheDir", func(a starlark.Tuple) (any, error) {
			id, err := stringAt("AsPersistentCacheDir", a, 0); if err != nil { return nil, err }
			sharing, err := intAt("AsPersistentCacheDir", a, 1); if err != nil { return nil, err }
			return buildkitllb.AsPersistentCacheDir(id, buildkitllb.CacheMountSharingMode(sharing)), nil
		}, 2, 2, "MountOption"),
		"ContentCache": intOption("ContentCache", "MountOption", func(n int) buildkitllb.MountOption { return buildkitllb.ContentCache(buildkitllb.MountContentCache(n)) }),
		"TmpfsSize": int64Option("TmpfsSize", "TmpfsOption", buildkitllb.TmpfsSize),
		"Tmpfs": optionBuiltin("Tmpfs", func(a starlark.Tuple) (any, error) {
			opts, err := optionsFromArgs[buildkitllb.TmpfsOption](a, 0, "TmpfsOption"); if err != nil { return nil, err }
			return buildkitllb.Tmpfs(opts...), nil
		}, 0, -1, "MountOption"),
		"AddMount": optionBuiltin("AddMount", addMountOption, 2, -1, "RunOption"),
		"AddExtraHost": optionBuiltin("AddExtraHost", addExtraHostOption, 2, 2, "RunOption"),
		"AddCDIDevice": optionBuiltin("AddCDIDevice", addCDIDeviceOption, 0, -1, "RunOption"),
		"CDIDeviceName": stringOption("CDIDeviceName", "CDIDeviceOption", buildkitllb.CDIDeviceName),
		"AddSecret": optionBuiltin("AddSecret", addSecretOption, 1, -1, "RunOption"),
		"AddSecretWithDest": optionBuiltin("AddSecretWithDest", addSecretWithDestOption, 2, -1, "RunOption"),
		"AddSSHSocket": optionBuiltin("AddSSHSocket", func(a starlark.Tuple) (any, error) {
			opts, err := optionsFromArgs[buildkitllb.SSHOption](a, 0, "SSHOption"); if err != nil { return nil, err }
			return buildkitllb.AddSSHSocket(opts...), nil
		}, 0, -1, "RunOption"),
		"SSHID": stringOption("SSHID", "SSHOption", buildkitllb.SSHID),
		"SSHSocketTarget": stringOption("SSHSocketTarget", "SSHOption", buildkitllb.SSHSocketTarget),
		"SSHSocketOpt": optionBuiltin("SSHSocketOpt", func(a starlark.Tuple) (any, error) {
			target, err := stringAt("SSHSocketOpt", a, 0); if err != nil { return nil, err }
			uid, err := intAt("SSHSocketOpt", a, 1); if err != nil { return nil, err }
			gid, err := intAt("SSHSocketOpt", a, 2); if err != nil { return nil, err }
			mode, err := intAt("SSHSocketOpt", a, 3); if err != nil { return nil, err }
			return buildkitllb.SSHSocketOpt(target, uid, gid, mode), nil
		}, 4, 4, "SSHOption"),
		"SecretID": stringOption("SecretID", "SecretOption", buildkitllb.SecretID),
		"SecretAsEnv": boolOption("SecretAsEnv", "SecretOption", buildkitllb.SecretAsEnv),
		"SecretAsEnvName": stringOption("SecretAsEnvName", "SecretOption", buildkitllb.SecretAsEnvName),
		"SecretFileOpt": optionBuiltin("SecretFileOpt", func(a starlark.Tuple) (any, error) {
			uid, err := intAt("SecretFileOpt", a, 0); if err != nil { return nil, err }
			gid, err := intAt("SecretFileOpt", a, 1); if err != nil { return nil, err }
			mode, err := intAt("SecretFileOpt", a, 2); if err != nil { return nil, err }
			return buildkitllb.SecretFileOpt(uid, gid, mode), nil
		}, 3, 3, "SecretOption"),
		"ProxyEnv": builtin("ProxyEnv", proxyEnvBuiltin),
		"WithProxy": optionBuiltin("WithProxy", withProxyOption, 1, 1, "RunOption"),
		"With": optionBuiltin("With", func(a starlark.Tuple) (any, error) {
			opts, err := optionsFromArgs[buildkitllb.StateOption](a, 0, "StateOption"); if err != nil { return nil, err }
			return buildkitllb.With(opts...), nil
		}, 0, -1, "RunOption"),
		"AddUlimit": optionBuiltin("AddUlimit", func(a starlark.Tuple) (any, error) {
			name, err := stringAt("AddUlimit", a, 0); if err != nil { return nil, err }
			soft, err := int64At("AddUlimit", a, 1); if err != nil { return nil, err }
			hard, err := int64At("AddUlimit", a, 2); if err != nil { return nil, err }
			return buildkitllb.AddUlimit(buildkitllb.UlimitName(name), soft, hard), nil
		}, 3, 3, "RunOption"),

		// File operation options.
		"WithParents": boolOption("WithParents", "MkdirOption", buildkitllb.WithParents),
		"WithUser": stringOption("WithUser", "ChownOption", buildkitllb.WithUser),
		"WithUIDGID": optionBuiltin("WithUIDGID", func(a starlark.Tuple) (any, error) {
			uid, err := intAt("WithUIDGID", a, 0); if err != nil { return nil, err }
			gid, err := intAt("WithUIDGID", a, 1); if err != nil { return nil, err }
			return buildkitllb.WithUIDGID(uid, gid), nil
		}, 2, 2, "ChownOption"),
		"WithAllowNotFound": boolOption("WithAllowNotFound", "RmOption", buildkitllb.WithAllowNotFound),
		"WithAllowWildcard": boolOption("WithAllowWildcard", "RmOption", buildkitllb.WithAllowWildcard),
		"WithExcludePatterns": stringListOption("WithExcludePatterns", "CopyOption", buildkitllb.WithExcludePatterns),
		"ChmodMode": intOption("ChmodMode", "CopyOption", func(n int) buildkitllb.CopyOption { return buildkitllb.ChmodOpt{Mode: os.FileMode(n)} }),
		"ChmodSymbolic": stringOption("ChmodSymbolic", "CopyOption", func(s string) buildkitllb.CopyOption { return buildkitllb.ChmodOpt{ModeStr: s} }),
		"CreatedTime": optionBuiltin("WithCreatedTime", createdTimeOption, 1, 1, "CreatedTime"),
		"WithCreatedTime": optionBuiltin("WithCreatedTime", createdTimeOption, 1, 1, "CreatedTime"),
	}
}

func createdTimeOption(a starlark.Tuple) (any, error) {
	raw, err := stringAt("WithCreatedTime", a, 0)
	if err != nil { return nil, err }
	t, err := time.Parse(time.RFC3339Nano, raw)
	if err != nil { return nil, fmt.Errorf("WithCreatedTime: %w", err) }
	return buildkitllb.WithCreatedTime(t), nil
}


func optionBuiltin(name string, build func(starlark.Tuple) (any, error), min, max int, kind string) starlark.Value {
	return builtin(name, func(_ *starlark.Thread, _ *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
		if err := arity(name, args, kwargs, min, max); err != nil { return nil, err }
		value, err := build(args)
		if err != nil { return nil, err }
		return &optionValue{kind: kind, value: value}, nil
	})
}

func stringOption[T any](name, kind string, build func(string) T) starlark.Value {
	return optionBuiltin(name, func(a starlark.Tuple) (any, error) { s, err := stringAt(name, a, 0); if err != nil { return nil, err }; return build(s), nil }, 1, 1, kind)
}

func boolOption[T any](name, kind string, build func(bool) T) starlark.Value {
	return optionBuiltin(name, func(a starlark.Tuple) (any, error) { b, err := boolAt(name, a, 0); if err != nil { return nil, err }; return build(b), nil }, 1, 1, kind)
}

func intOption[T any](name, kind string, build func(int) T) starlark.Value {
	return optionBuiltin(name, func(a starlark.Tuple) (any, error) { n, err := intAt(name, a, 0); if err != nil { return nil, err }; return build(n), nil }, 1, 1, kind)
}

func int64Option[T any](name, kind string, build func(int64) T) starlark.Value {
	return optionBuiltin(name, func(a starlark.Tuple) (any, error) { n, err := int64At(name, a, 0); if err != nil { return nil, err }; return build(n), nil }, 1, 1, kind)
}

func stringListOption[T any](name, kind string, build func([]string) T) starlark.Value {
	return optionBuiltin(name, func(a starlark.Tuple) (any, error) { values, err := stringList(a[0]); if err != nil { return nil, err }; return build(values), nil }, 1, 1, kind)
}

func withDescription(a starlark.Tuple) (any, error) {
	dict, ok := a[0].(*starlark.Dict)
	if !ok { return nil, fmt.Errorf("WithDescription: got %s, want dict", valueType(a[0])) }
	values := make(map[string]string, dict.Len())
	for _, item := range dict.Items() {
		key, ok := item[0].(starlark.String); if !ok { return nil, fmt.Errorf("WithDescription: keys must be strings") }
		value, ok := item[1].(starlark.String); if !ok { return nil, fmt.Errorf("WithDescription: values must be strings") }
		values[string(key)] = string(value)
	}
	return buildkitllb.WithDescription(values), nil
}

func addEnvfOption(a starlark.Tuple) (any, error) {
	key, err := stringAt("AddEnvf", a, 0); if err != nil { return nil, err }
	format, err := stringAt("AddEnvf", a, 1); if err != nil { return nil, err }
	values := make([]any, 0, len(a)-2)
	for _, arg := range a[2:] { values = append(values, goScalar(arg)) }
	return buildkitllb.AddEnvf(key, format, values...), nil
}

func dirfOption(a starlark.Tuple) (any, error) {
	format, err := stringAt("Dirf", a, 0); if err != nil { return nil, err }
	values := make([]any, 0, len(a)-1)
	for _, arg := range a[1:] { values = append(values, goScalar(arg)) }
	return buildkitllb.Dirf(format, values...), nil
}

func shlexfOption(a starlark.Tuple) (any, error) {
	format, err := stringAt("Shlexf", a, 0); if err != nil { return nil, err }
	values := make([]any, 0, len(a)-1)
	for _, arg := range a[1:] { values = append(values, goScalar(arg)) }
	return buildkitllb.Shlexf(format, values...), nil
}

func addMountOption(a starlark.Tuple) (any, error) {
	target, err := stringAt("AddMount", a, 0); if err != nil { return nil, err }
	state, err := stateFromValue(a[1]); if err != nil { return nil, err }
	opts, err := optionsFromArgs[buildkitllb.MountOption](a, 2, "MountOption"); if err != nil { return nil, err }
	return buildkitllb.AddMount(target, state, opts...), nil
}

func addSecretOption(a starlark.Tuple) (any, error) {
	id, err := stringAt("AddSecret", a, 0); if err != nil { return nil, err }
	opts, err := optionsFromArgs[buildkitllb.SecretOption](a, 1, "SecretOption"); if err != nil { return nil, err }
	return buildkitllb.AddSecret(id, opts...), nil
}

var _ = digest.Digest("")


func checksumOption(a starlark.Tuple) (any, error) {
	raw, err := stringAt("Checksum", a, 0)
	if err != nil { return nil, err }
	d, err := digest.Parse(raw)
	if err != nil { return nil, fmt.Errorf("Checksum: %w", err) }
	return buildkitllb.Checksum(d), nil
}

func imageChecksumOption(a starlark.Tuple) (any, error) {
	raw, err := stringAt("WithImageChecksum", a, 0)
	if err != nil { return nil, err }
	d, err := digest.Parse(raw)
	if err != nil { return nil, fmt.Errorf("WithImageChecksum: %w", err) }
	return buildkitllb.WithImageChecksum(d), nil
}

func addExtraHostOption(a starlark.Tuple) (any, error) {
	host, err := stringAt("AddExtraHost", a, 0)
	if err != nil { return nil, err }
	ipText, err := stringAt("AddExtraHost", a, 1)
	if err != nil { return nil, err }
	ip := net.ParseIP(ipText)
	if ip == nil { return nil, fmt.Errorf("AddExtraHost: invalid IP address %q", ipText) }
	return buildkitllb.AddExtraHost(host, ip), nil
}

func addCDIDeviceOption(a starlark.Tuple) (any, error) {
	opts, err := optionsFromArgs[buildkitllb.CDIDeviceOption](a, 0, "CDIDeviceOption")
	if err != nil { return nil, err }
	return buildkitllb.AddCDIDevice(opts...), nil
}

func addSecretWithDestOption(a starlark.Tuple) (any, error) {
	src, err := stringAt("AddSecretWithDest", a, 0)
	if err != nil { return nil, err }
	target, err := stringAt("AddSecretWithDest", a, 1)
	if err != nil { return nil, err }
	opts, err := optionsFromArgs[buildkitllb.SecretOption](a, 2, "SecretOption")
	if err != nil { return nil, err }
	return buildkitllb.AddSecretWithDest(src, &target, opts...), nil
}

func withProxyOption(a starlark.Tuple) (any, error) {
	proxy, ok := a[0].(*proxyEnvValue)
	if !ok || proxy == nil { return nil, fmt.Errorf("WithProxy: got %s, want llb.ProxyEnv", valueType(a[0])) }
	return buildkitllb.WithProxy(proxy.proxy), nil
}


func ociChecksumOption(a starlark.Tuple) (any, error) {
	raw, err := stringAt("OCIChecksum", a, 0)
	if err != nil { return nil, err }
	d, err := digest.Parse(raw)
	if err != nil { return nil, fmt.Errorf("OCIChecksum: %w", err) }
	return buildkitllb.OCIChecksum(d), nil
}
