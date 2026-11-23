package llb

import (
	"fmt"
	"net"
	buildkitllb "github.com/moby/buildkit/client/llb"
	"github.com/moby/buildkit/solver/pb"
	"go.starlark.net/starlark"
)

// ExecStateValue wraps BuildKit's ExecState, which carries the RUN operation
// together with its root state and mount references.
type ExecStateValue struct {
	execState buildkitllb.ExecState
}

// WrapExecState adapts a native BuildKit ExecState.
func WrapExecState(execState buildkitllb.ExecState) *ExecStateValue {
	return &ExecStateValue{execState: execState}
}

// Unwrap returns the native execution state.
func (v *ExecStateValue) Unwrap() buildkitllb.ExecState {
	if v == nil {
		return buildkitllb.ExecState{}
	}
	return v.execState
}

func (v *ExecStateValue) String() string        { return "<llb.ExecState>" }
func (v *ExecStateValue) Type() string          { return "llb.ExecState" }
func (v *ExecStateValue) Freeze()               {}
func (v *ExecStateValue) Truth() starlark.Bool  { return starlark.Bool(v != nil) }
func (v *ExecStateValue) Hash() (uint32, error) { return 0, fmt.Errorf("unhashable: llb.ExecState") }

func (v *ExecStateValue) Attr(name string) (starlark.Value, error) {
	if v == nil {
		return nil, fmt.Errorf("llb.ExecState: nil receiver")
	}
	switch name {
	case "Root", "root":
		return boundBuiltin("ExecState.Root", func(_ *starlark.Thread, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
			if err := arity("ExecState.Root", args, kwargs, 0, 0); err != nil { return nil, err }
			return WrapState(v.execState.Root()), nil
		}), nil
	case "AddMount", "add_mount":
		return boundBuiltin("ExecState.AddMount", func(_ *starlark.Thread, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
			if err := arity("ExecState.AddMount", args, kwargs, 2, -1); err != nil { return nil, err }
			target, err := stringAt("ExecState.AddMount", args, 0); if err != nil { return nil, err }
			source, err := stateFromValue(args[1]); if err != nil { return nil, err }
			options, err := optionsFromArgs[buildkitllb.MountOption](args, 2, "MountOption")
			if err != nil { return nil, err }
			return WrapState(v.execState.AddMount(target, source, options...)), nil
		}), nil
	case "GetMount", "get_mount":
		return boundBuiltin("ExecState.GetMount", func(_ *starlark.Thread, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
			if err := arity("ExecState.GetMount", args, kwargs, 1, 1); err != nil { return nil, err }
			target, err := stringAt("ExecState.GetMount", args, 0); if err != nil { return nil, err }
			return WrapState(v.execState.GetMount(target)), nil
		}), nil
	default:
		return WrapState(v.execState.State).Attr(name)
	}
}

func (v *ExecStateValue) AttrNames() []string {
	names := []string{"AddMount", "GetMount", "Root", "add_mount", "get_mount", "root"}
	for _, group := range stateAttrNames {
		names = append(names, group...)
	}
	return compactStrings(names)
}

// ExecOpValue wraps BuildKit's ExecOp vertex and its mount operations.
type ExecOpValue struct {
	exec *buildkitllb.ExecOp
}

// WrapExecOp adapts a native BuildKit ExecOp.
func WrapExecOp(exec *buildkitllb.ExecOp) *ExecOpValue {
	return &ExecOpValue{exec: exec}
}

// Unwrap returns the native execution vertex.
func (v *ExecOpValue) Unwrap() *buildkitllb.ExecOp {
	if v == nil {
		return nil
	}
	return v.exec
}

func (v *ExecOpValue) String() string        { return "<llb.ExecOp>" }
func (v *ExecOpValue) Type() string          { return "llb.ExecOp" }
func (v *ExecOpValue) Freeze()               {}
func (v *ExecOpValue) Truth() starlark.Bool  { return starlark.Bool(v != nil && v.exec != nil) }
func (v *ExecOpValue) Hash() (uint32, error) { return 0, fmt.Errorf("unhashable: llb.ExecOp") }

func (v *ExecOpValue) Attr(name string) (starlark.Value, error) {
	if v == nil || v.exec == nil {
		return nil, fmt.Errorf("llb.ExecOp: nil receiver")
	}
	switch name {
	case "AddMount", "add_mount":
		return boundBuiltin("ExecOp.AddMount", func(_ *starlark.Thread, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
			if err := arity("ExecOp.AddMount", args, kwargs, 2, -1); err != nil { return nil, err }
			target, err := stringAt("ExecOp.AddMount", args, 0); if err != nil { return nil, err }
			source, err := outputFromValue(args[1]); if err != nil { return nil, err }
			options, err := optionsFromArgs[buildkitllb.MountOption](args, 2, "MountOption")
			if err != nil { return nil, err }
			return WrapOutput(v.exec.AddMount(target, source, options...)), nil
		}), nil
	case "GetMount", "get_mount":
		return boundBuiltin("ExecOp.GetMount", func(_ *starlark.Thread, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
			if err := arity("ExecOp.GetMount", args, kwargs, 1, 1); err != nil { return nil, err }
			target, err := stringAt("ExecOp.GetMount", args, 0); if err != nil { return nil, err }
			return WrapOutput(v.exec.GetMount(target)), nil
		}), nil
	case "Output", "output":
		return boundBuiltin("ExecOp.Output", func(_ *starlark.Thread, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
			if err := arity("ExecOp.Output", args, kwargs, 0, 0); err != nil { return nil, err }
			return WrapOutput(v.exec.Output()), nil
		}), nil
	case "Inputs", "inputs":
		return boundBuiltin("ExecOp.Inputs", func(_ *starlark.Thread, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
			if err := arity("ExecOp.Inputs", args, kwargs, 0, 0); err != nil { return nil, err }
			inputs := v.exec.Inputs()
			values := make([]starlark.Value, len(inputs))
			for i, input := range inputs { values[i] = WrapOutput(input) }
			return starlark.NewList(values), nil
		}), nil
	case "Validate", "validate":
		return boundBuiltin("ExecOp.Validate", func(thread *starlark.Thread, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
			if err := arity("ExecOp.Validate", args, kwargs, 0, -1); err != nil { return nil, err }
			constraints, err := constraintsFromArgs(args, 0)
			if err != nil { return nil, err }
			if err := v.exec.Validate(callContext(thread), constraints); err != nil { return nil, err }
			return starlark.None, nil
		}), nil
	case "Marshal", "marshal":
		return boundBuiltin("ExecOp.Marshal", func(thread *starlark.Thread, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
			if err := arity("ExecOp.Marshal", args, kwargs, 0, -1); err != nil { return nil, err }
			constraints, err := constraintsFromArgs(args, 0)
			if err != nil { return nil, err }
			dgst, data, metadata, sources, err := v.exec.Marshal(callContext(thread), constraints)
			if err != nil { return nil, err }
			return marshalResult(dgst, data, metadata, sources), nil
		}), nil
	default:
		return nil, nil
	}
}

func (v *ExecOpValue) AttrNames() []string {
	return []string{"AddMount", "GetMount", "Inputs", "Marshal", "Output", "Validate",
		"add_mount", "get_mount", "inputs", "marshal", "output", "validate"}
}

// ProxyEnvValue adapts the proxy environment copied into an ExecOp.
type ProxyEnvValue struct {
	proxy buildkitllb.ProxyEnv
}

// WrapProxyEnv adapts a native BuildKit ProxyEnv.
func WrapProxyEnv(proxy buildkitllb.ProxyEnv) *ProxyEnvValue {
	return &ProxyEnvValue{proxy: proxy}
}

// Unwrap returns the native proxy environment value.
func (v *ProxyEnvValue) Unwrap() buildkitllb.ProxyEnv {
	if v == nil { return buildkitllb.ProxyEnv{} }
	return v.proxy
}

func (v *ProxyEnvValue) String() string        { return "<llb.ProxyEnv>" }
func (v *ProxyEnvValue) Type() string          { return "llb.ProxyEnv" }
func (v *ProxyEnvValue) Freeze()               {}
func (v *ProxyEnvValue) Truth() starlark.Bool  { return starlark.Bool(v != nil) }
func (v *ProxyEnvValue) Hash() (uint32, error) { return 0, fmt.Errorf("unhashable: llb.ProxyEnv") }

func (v *ProxyEnvValue) Attr(name string) (starlark.Value, error) {
	if v == nil { return nil, fmt.Errorf("llb.ProxyEnv: nil receiver") }
	switch name {
	case "HTTPProxy", "http_proxy": return starlark.String(v.proxy.HTTPProxy), nil
	case "HTTPSProxy", "https_proxy": return starlark.String(v.proxy.HTTPSProxy), nil
	case "FTPProxy", "ftp_proxy": return starlark.String(v.proxy.FTPProxy), nil
	case "NoProxy", "no_proxy": return starlark.String(v.proxy.NoProxy), nil
	case "AllProxy", "all_proxy": return starlark.String(v.proxy.AllProxy), nil
	default: return nil, nil
	}
}

func (v *ProxyEnvValue) AttrNames() []string {
	return []string{"AllProxy", "FTPProxy", "HTTPProxy", "HTTPSProxy", "NoProxy",
		"all_proxy", "ftp_proxy", "http_proxy", "https_proxy", "no_proxy"}
}

// ExecInfoValue exposes the resolved RunOption configuration for inspection.
type ExecInfoValue struct {
	info *buildkitllb.ExecInfo
}

// WrapExecInfo adapts a BuildKit ExecInfo.
func WrapExecInfo(info *buildkitllb.ExecInfo) *ExecInfoValue {
	return &ExecInfoValue{info: info}
}

// Unwrap returns the underlying ExecInfo.
func (v *ExecInfoValue) Unwrap() *buildkitllb.ExecInfo {
	if v == nil { return nil }
	return v.info
}

func (v *ExecInfoValue) String() string        { return "<llb.ExecInfo>" }
func (v *ExecInfoValue) Type() string          { return "llb.ExecInfo" }
func (v *ExecInfoValue) Freeze()               {}
func (v *ExecInfoValue) Truth() starlark.Bool  { return starlark.Bool(v != nil && v.info != nil) }
func (v *ExecInfoValue) Hash() (uint32, error) { return 0, fmt.Errorf("unhashable: llb.ExecInfo") }

func (v *ExecInfoValue) Attr(name string) (starlark.Value, error) {
	if v == nil || v.info == nil { return nil, fmt.Errorf("llb.ExecInfo: nil receiver") }
	switch name {
	case "state", "State":
		return WrapState(v.info.State), nil
	case "mounts", "Mounts":
		values := make([]starlark.Value, len(v.info.Mounts))
		for i, mount := range v.info.Mounts { values[i] = wrapMountInfo(mount) }
		return starlark.NewList(values), nil
	case "readonly_root_fs", "ReadonlyRootFS":
		return starlark.Bool(v.info.ReadonlyRootFS), nil
	case "proxy_env", "ProxyEnv":
		if v.info.ProxyEnv == nil { return starlark.None, nil }
		return WrapProxyEnv(*v.info.ProxyEnv), nil
	case "secrets", "Secrets":
		values := make([]starlark.Value, len(v.info.Secrets))
		for i, secret := range v.info.Secrets { values[i] = wrapSecretInfo(secret) }
		return starlark.NewList(values), nil
	case "ssh", "SSH":
		values := make([]starlark.Value, len(v.info.SSH))
		for i, item := range v.info.SSH { values[i] = wrapSSHInfo(item) }
		return starlark.NewList(values), nil
	case "cdi_devices", "CDIDevices":
		values := make([]starlark.Value, len(v.info.CDIDevices))
		for i, item := range v.info.CDIDevices { values[i] = wrapCDIDeviceInfo(item) }
		return starlark.NewList(values), nil
	case "constraints", "Constraints":
		return WrapConstraints(v.info.Constraints), nil
	default:
		return nil, nil
	}
}

func (v *ExecInfoValue) AttrNames() []string {
	return []string{"state", "mounts", "readonly_root_fs", "proxy_env", "secrets", "ssh", "cdi_devices", "constraints",
		"State", "Mounts", "ReadonlyRootFS", "ProxyEnv", "Secrets", "SSH", "CDIDevices", "Constraints"}
}

type mountInfoValue struct{ info buildkitllb.MountInfo }

func (v *mountInfoValue) String() string        { return "<llb.MountInfo>" }
func (v *mountInfoValue) Type() string          { return "llb.MountInfo" }
func (v *mountInfoValue) Freeze()               {}
func (v *mountInfoValue) Truth() starlark.Bool  { return starlark.Bool(v != nil) }
func (v *mountInfoValue) Hash() (uint32, error) { return 0, fmt.Errorf("unhashable: llb.MountInfo") }
func (v *mountInfoValue) Attr(name string) (starlark.Value, error) {
	if v == nil { return nil, fmt.Errorf("llb.MountInfo: nil receiver") }
	switch name {
	case "target", "Target": return starlark.String(v.info.Target), nil
	case "source", "Source": return WrapOutput(v.info.Source), nil
	case "options", "Opts": return optionSliceValue(v.info.Opts, "MountOption"), nil
	default: return nil, nil
	}
}
func (v *mountInfoValue) AttrNames() []string { return []string{"target", "source", "options", "Target", "Source", "Opts"} }

type cdiDeviceInfoValue struct{ info buildkitllb.CDIDeviceInfo }
func (v *cdiDeviceInfoValue) String() string        { return "<llb.CDIDeviceInfo>" }
func (v *cdiDeviceInfoValue) Type() string          { return "llb.CDIDeviceInfo" }
func (v *cdiDeviceInfoValue) Freeze()               {}
func (v *cdiDeviceInfoValue) Truth() starlark.Bool  { return starlark.Bool(v != nil) }
func (v *cdiDeviceInfoValue) Hash() (uint32, error) { return 0, fmt.Errorf("unhashable: llb.CDIDeviceInfo") }
func (v *cdiDeviceInfoValue) Attr(name string) (starlark.Value, error) {
	if v == nil { return nil, fmt.Errorf("llb.CDIDeviceInfo: nil receiver") }
	if name == "name" || name == "Name" { return starlark.String(v.info.Name), nil }
	return nil, nil
}
func (v *cdiDeviceInfoValue) AttrNames() []string { return []string{"name", "Name"} }

type sshInfoValue struct{ info buildkitllb.SSHInfo }
func (v *sshInfoValue) String() string        { return "<llb.SSHInfo>" }
func (v *sshInfoValue) Type() string          { return "llb.SSHInfo" }
func (v *sshInfoValue) Freeze()               {}
func (v *sshInfoValue) Truth() starlark.Bool  { return starlark.Bool(v != nil) }
func (v *sshInfoValue) Hash() (uint32, error) { return 0, fmt.Errorf("unhashable: llb.SSHInfo") }
func (v *sshInfoValue) Attr(name string) (starlark.Value, error) {
	if v == nil { return nil, fmt.Errorf("llb.SSHInfo: nil receiver") }
	switch name {
	case "id", "ID": return starlark.String(v.info.ID), nil
	case "target", "Target": return starlark.String(v.info.Target), nil
	case "uid", "UID": return starlark.MakeInt(v.info.UID), nil
	case "gid", "GID": return starlark.MakeInt(v.info.GID), nil
	case "mode", "Mode": return starlark.MakeInt(v.info.Mode), nil
	case "optional", "Optional": return starlark.Bool(v.info.Optional), nil
	default: return nil, nil
	}
}
func (v *sshInfoValue) AttrNames() []string { return []string{"id", "target", "uid", "gid", "mode", "optional", "ID", "Target", "UID", "GID", "Mode", "Optional"} }

type secretInfoValue struct{ info buildkitllb.SecretInfo }
func (v *secretInfoValue) String() string        { return "<llb.SecretInfo>" }
func (v *secretInfoValue) Type() string          { return "llb.SecretInfo" }
func (v *secretInfoValue) Freeze()               {}
func (v *secretInfoValue) Truth() starlark.Bool  { return starlark.Bool(v != nil) }
func (v *secretInfoValue) Hash() (uint32, error) { return 0, fmt.Errorf("unhashable: llb.SecretInfo") }
func (v *secretInfoValue) Attr(name string) (starlark.Value, error) {
	if v == nil { return nil, fmt.Errorf("llb.SecretInfo: nil receiver") }
	switch name {
	case "id", "ID": return starlark.String(v.info.ID), nil
	case "target", "Target":
		if v.info.Target == nil { return starlark.None, nil }
		return starlark.String(*v.info.Target), nil
	case "mode", "Mode": return starlark.MakeInt(v.info.Mode), nil
	case "uid", "UID": return starlark.MakeInt(v.info.UID), nil
	case "gid", "GID": return starlark.MakeInt(v.info.GID), nil
	case "optional", "Optional": return starlark.Bool(v.info.Optional), nil
	case "env", "Env": return starlark.Bool(v.info.Env), nil
	case "env_name", "EnvName": return starlark.String(v.info.EnvName), nil
	default: return nil, nil
	}
}
func (v *secretInfoValue) AttrNames() []string { return []string{"id", "target", "mode", "uid", "gid", "optional", "env", "env_name", "ID", "Target", "Mode", "UID", "GID", "Optional", "Env", "EnvName"} }

// WrapMountInfo adapts a native mount description.
func WrapMountInfo(info buildkitllb.MountInfo) starlark.Value { return &mountInfoValue{info: info} }

// WrapCDIDeviceInfo adapts a native CDI device description.
func WrapCDIDeviceInfo(info buildkitllb.CDIDeviceInfo) starlark.Value { return &cdiDeviceInfoValue{info: info} }

// WrapSSHInfo adapts a native SSH socket description.
func WrapSSHInfo(info buildkitllb.SSHInfo) starlark.Value { return &sshInfoValue{info: info} }

// WrapSecretInfo adapts a native secret description.
func WrapSecretInfo(info buildkitllb.SecretInfo) starlark.Value { return &secretInfoValue{info: info} }

func wrapMountInfo(info buildkitllb.MountInfo) starlark.Value { return WrapMountInfo(info) }
func wrapCDIDeviceInfo(info buildkitllb.CDIDeviceInfo) starlark.Value { return WrapCDIDeviceInfo(info) }
func wrapSSHInfo(info buildkitllb.SSHInfo) starlark.Value { return WrapSSHInfo(info) }
func wrapSecretInfo(info buildkitllb.SecretInfo) starlark.Value { return WrapSecretInfo(info) }

func optionSliceValue[T any](values []T, kind string) starlark.Value {
	items := make([]starlark.Value, len(values))
	for i, value := range values { items[i] = wrapOption(kind, value) }
	return starlark.NewList(items)
}

func stateExecAttr(v *StateValue, name string) (starlark.Value, bool, error) {
	if name != "Run" && name != "run" { return nil, false, nil }
	return boundBuiltin("State.Run", func(_ *starlark.Thread, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
		if err := arity("State.Run", args, kwargs, 0, -1); err != nil { return nil, err }
		options, err := optionsFromArgs[buildkitllb.RunOption](args, 0, "RunOption")
		if err != nil { return nil, err }
		return WrapExecState(v.state.Run(options...)), nil
	}), true, nil
}

var stateExecAttrNames = []string{"Run", "run"}

func execBuiltins() map[string]starlark.Value {
	return map[string]starlark.Value{
		"NewExecOp": builtin("NewExecOp", newExecOpBuiltin),
		"ExecInfo": builtin("ExecInfo", execInfoBuiltin),
		"ProxyEnv": builtin("ProxyEnv", proxyEnvBuiltin),

		"Shlex": stringOptionBuiltin("Shlex", "RunOption", buildkitllb.Shlex),
		"Shlexf": builtin("Shlexf", shlexfBuiltin),
		"Args": builtin("Args", argsBuiltin),
		"ReadonlyRootFS": builtin("ReadonlyRootFS", readonlyRootFSBuiltin),
		"ValidExitCodes": builtin("ValidExitCodes", validExitCodesBuiltin),
		"WithCgroupParent": stringOptionBuiltin("WithCgroupParent", "RunOption", buildkitllb.WithCgroupParent),
		"AddExtraHost": builtin("AddExtraHost", addExtraHostRunBuiltin),
		"AddUlimit": builtin("AddUlimit", addUlimitRunBuiltin),
		"AddCDIDevice": builtin("AddCDIDevice", addCDIDeviceBuiltin),
		"CDIDeviceName": stringOptionBuiltin("CDIDeviceName", "CDIDeviceOption", buildkitllb.CDIDeviceName),
		"AddMount": builtin("AddMount", addMountRunBuiltin),
		"AddSSHSocket": builtin("AddSSHSocket", addSSHSocketBuiltin),
		"SSHID": stringOptionBuiltin("SSHID", "SSHOption", buildkitllb.SSHID),
		"SSHSocketTarget": stringOptionBuiltin("SSHSocketTarget", "SSHOption", buildkitllb.SSHSocketTarget),
		"SSHSocketOpt": builtin("SSHSocketOpt", sshSocketOptionBuiltin),
		"AddSecret": builtin("AddSecret", addSecretBuiltin),
		"AddSecretWithDest": builtin("AddSecretWithDest", addSecretWithDestBuiltin),
		"SecretID": stringOptionBuiltin("SecretID", "SecretOption", buildkitllb.SecretID),
		"SecretAsEnv": boolOptionBuiltin("SecretAsEnv", "SecretOption", buildkitllb.SecretAsEnv),
		"SecretAsEnvName": stringOptionBuiltin("SecretAsEnvName", "SecretOption", buildkitllb.SecretAsEnvName),
		"SecretFileOpt": builtin("SecretFileOpt", secretFileOptBuiltin),
		"SourcePath": stringOptionBuiltin("SourcePath", "MountOption", buildkitllb.SourcePath),
		"AsPersistentCacheDir": builtin("AsPersistentCacheDir", persistentCacheDirBuiltin),
		"ContentCache": builtin("ContentCache", contentCacheBuiltin),
		"Tmpfs": builtin("Tmpfs", tmpfsBuiltin),
		"TmpfsSize": int64OptionBuiltin("TmpfsSize", "TmpfsOption", buildkitllb.TmpfsSize),
		"WithProxy": optionBuiltin("WithProxy", "RunOption", withProxyOption),
		"With": builtin("With", withRunBuiltin),
	}
}

func newExecOpBuiltin(_ *starlark.Thread, _ *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	if err := arity("NewExecOp", args, kwargs, 4, 4); err != nil { return nil, err }
	state, err := stateFromValue(args[0]); if err != nil { return nil, err }
	var proxy *buildkitllb.ProxyEnv
	if args[1] != starlark.None {
		value, ok := args[1].(*ProxyEnvValue)
		if !ok || value == nil { return nil, fmt.Errorf("NewExecOp: argument 2: got %s, want llb.ProxyEnv or None", valueType(args[1])) }
		p := value.proxy
		proxy = &p
	}
	readonly, ok := args[2].(starlark.Bool)
	if !ok { return nil, fmt.Errorf("NewExecOp: argument 3: got %s, want bool", valueType(args[2])) }
	constraints, ok := args[3].(*ConstraintsValue)
	if !ok || constraints == nil { return nil, fmt.Errorf("NewExecOp: argument 4: got %s, want llb.Constraints", valueType(args[3])) }
	return WrapExecOp(buildkitllb.NewExecOp(state, proxy, bool(readonly), constraints.Unwrap())), nil
}

func execInfoBuiltin(_ *starlark.Thread, _ *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	if err := arity("ExecInfo", args, kwargs, 1, -1); err != nil { return nil, err }
	state, err := stateFromValue(args[0]); if err != nil { return nil, err }
	info := &buildkitllb.ExecInfo{State: state}
	options, err := optionsFromArgs[buildkitllb.RunOption](args, 1, "RunOption")
	if err != nil { return nil, err }
	for _, option := range options { option.SetRunOption(info) }
	return WrapExecInfo(info), nil
}

func proxyEnvBuiltin(_ *starlark.Thread, _ *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	var proxy buildkitllb.ProxyEnv
	if err := starlark.UnpackArgs("ProxyEnv", args, kwargs,
		"http_proxy?", &proxy.HTTPProxy,
		"https_proxy?", &proxy.HTTPSProxy,
		"ftp_proxy?", &proxy.FTPProxy,
		"no_proxy?", &proxy.NoProxy,
		"all_proxy?", &proxy.AllProxy,
	); err != nil { return nil, err }
	return WrapProxyEnv(proxy), nil
}

func shlexfBuiltin(_ *starlark.Thread, _ *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	if err := arity("Shlexf", args, kwargs, 1, -1); err != nil { return nil, err }
	format, err := stringAt("Shlexf", args, 0); if err != nil { return nil, err }
	values := make([]any, 0, len(args)-1)
	for _, value := range args[1:] { values = append(values, goScalar(value)) }
	return wrapOption("RunOption", buildkitllb.Shlexf(format, values...)), nil
}

func argsBuiltin(_ *starlark.Thread, _ *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	if err := arity("Args", args, kwargs, 0, -1); err != nil { return nil, err }
	var values []string
	if len(args) == 1 {
		list, ok := args[0].(starlark.Indexable)
		if ok {
			var err error
			values, err = stringList(list)
			if err != nil { return nil, err }
		} else {
			value, err := stringAt("Args", args, 0)
			if err != nil { return nil, err }
			values = []string{value}
		}
	} else {
		values = make([]string, len(args))
		for i := range args {
			value, err := stringAt("Args", args, i)
			if err != nil { return nil, err }
			values[i] = value
		}
	}
	return wrapOption("RunOption", buildkitllb.Args(values)), nil
}

func readonlyRootFSBuiltin(_ *starlark.Thread, _ *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	if err := arity("ReadonlyRootFS", args, kwargs, 0, 0); err != nil { return nil, err }
	return wrapOption("RunOption", buildkitllb.ReadonlyRootFS()), nil
}

func validExitCodesBuiltin(_ *starlark.Thread, _ *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	if err := arity("ValidExitCodes", args, kwargs, 0, -1); err != nil { return nil, err }
	codes := make([]int, len(args))
	for i := range args {
		code, err := intAt("ValidExitCodes", args, i)
		if err != nil { return nil, err }
		codes[i] = code
	}
	return wrapOption("RunOption", buildkitllb.ValidExitCodes(codes...)), nil
}

func addExtraHostRunBuiltin(_ *starlark.Thread, _ *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	if err := arity("AddExtraHost", args, kwargs, 2, 2); err != nil { return nil, err }
	host, err := stringAt("AddExtraHost", args, 0); if err != nil { return nil, err }
	rawIP, err := stringAt("AddExtraHost", args, 1); if err != nil { return nil, err }
	ip := net.ParseIP(rawIP)
	if ip == nil { return nil, fmt.Errorf("AddExtraHost: invalid IP address %q", rawIP) }
	return wrapOption("RunOption", buildkitllb.AddExtraHost(host, ip)), nil
}

func addUlimitRunBuiltin(_ *starlark.Thread, _ *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	if err := arity("AddUlimit", args, kwargs, 3, 3); err != nil { return nil, err }
	name, err := stringAt("AddUlimit", args, 0); if err != nil { return nil, err }
	soft, err := int64At("AddUlimit", args, 1); if err != nil { return nil, err }
	hard, err := int64At("AddUlimit", args, 2); if err != nil { return nil, err }
	return wrapOption("RunOption", buildkitllb.AddUlimit(buildkitllb.UlimitName(name), soft, hard)), nil
}

func addCDIDeviceBuiltin(_ *starlark.Thread, _ *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	if err := arity("AddCDIDevice", args, kwargs, 0, -1); err != nil { return nil, err }
	options, err := optionsFromArgs[buildkitllb.CDIDeviceOption](args, 0, "CDIDeviceOption")
	if err != nil { return nil, err }
	return wrapOption("RunOption", buildkitllb.AddCDIDevice(options...)), nil
}

func addMountRunBuiltin(_ *starlark.Thread, _ *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	if err := arity("AddMount", args, kwargs, 2, -1); err != nil { return nil, err }
	target, err := stringAt("AddMount", args, 0); if err != nil { return nil, err }
	state, err := stateFromValue(args[1]); if err != nil { return nil, err }
	options, err := optionsFromArgs[buildkitllb.MountOption](args, 2, "MountOption")
	if err != nil { return nil, err }
	return wrapOption("RunOption", buildkitllb.AddMount(target, state, options...)), nil
}

func addSSHSocketBuiltin(_ *starlark.Thread, _ *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	if err := arity("AddSSHSocket", args, kwargs, 0, -1); err != nil { return nil, err }
	options, err := optionsFromArgs[buildkitllb.SSHOption](args, 0, "SSHOption")
	if err != nil { return nil, err }
	return wrapOption("RunOption", buildkitllb.AddSSHSocket(options...)), nil
}

func sshSocketOptionBuiltin(_ *starlark.Thread, _ *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	if err := arity("SSHSocketOpt", args, kwargs, 4, 4); err != nil { return nil, err }
	target, err := stringAt("SSHSocketOpt", args, 0); if err != nil { return nil, err }
	uid, err := intAt("SSHSocketOpt", args, 1); if err != nil { return nil, err }
	gid, err := intAt("SSHSocketOpt", args, 2); if err != nil { return nil, err }
	mode, err := intAt("SSHSocketOpt", args, 3); if err != nil { return nil, err }
	return wrapOption("SSHOption", buildkitllb.SSHSocketOpt(target, uid, gid, mode)), nil
}

func addSecretBuiltin(_ *starlark.Thread, _ *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	if err := arity("AddSecret", args, kwargs, 1, -1); err != nil { return nil, err }
	dest, err := stringAt("AddSecret", args, 0); if err != nil { return nil, err }
	options, err := optionsFromArgs[buildkitllb.SecretOption](args, 1, "SecretOption")
	if err != nil { return nil, err }
	return wrapOption("RunOption", buildkitllb.AddSecret(dest, options...)), nil
}

func addSecretWithDestBuiltin(_ *starlark.Thread, _ *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	if err := arity("AddSecretWithDest", args, kwargs, 2, -1); err != nil { return nil, err }
	src, err := stringAt("AddSecretWithDest", args, 0); if err != nil { return nil, err }
	var dest *string
	if args[1] != starlark.None {
		value, ok := args[1].(starlark.String)
		if !ok { return nil, fmt.Errorf("AddSecretWithDest: destination must be string or None, got %s", valueType(args[1])) }
		target := string(value)
		dest = &target
	}
	options, err := optionsFromArgs[buildkitllb.SecretOption](args, 2, "SecretOption")
	if err != nil { return nil, err }
	return wrapOption("RunOption", buildkitllb.AddSecretWithDest(src, dest, options...)), nil
}

func secretFileOptBuiltin(_ *starlark.Thread, _ *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	if err := arity("SecretFileOpt", args, kwargs, 3, 3); err != nil { return nil, err }
	uid, err := intAt("SecretFileOpt", args, 0); if err != nil { return nil, err }
	gid, err := intAt("SecretFileOpt", args, 1); if err != nil { return nil, err }
	mode, err := intAt("SecretFileOpt", args, 2); if err != nil { return nil, err }
	return wrapOption("SecretOption", buildkitllb.SecretFileOpt(uid, gid, mode)), nil
}

func persistentCacheDirBuiltin(_ *starlark.Thread, _ *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	if err := arity("AsPersistentCacheDir", args, kwargs, 2, 2); err != nil { return nil, err }
	id, err := stringAt("AsPersistentCacheDir", args, 0); if err != nil { return nil, err }
	sharing, err := intAt("AsPersistentCacheDir", args, 1); if err != nil { return nil, err }
	return wrapOption("MountOption", buildkitllb.AsPersistentCacheDir(id, buildkitllb.CacheMountSharingMode(sharing))), nil
}

func contentCacheBuiltin(_ *starlark.Thread, _ *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	if err := arity("ContentCache", args, kwargs, 1, 1); err != nil { return nil, err }
	mode, err := intAt("ContentCache", args, 0); if err != nil { return nil, err }
	return wrapOption("MountOption", buildkitllb.ContentCache(buildkitllb.MountContentCache(mode))), nil
}

func tmpfsBuiltin(_ *starlark.Thread, _ *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	if err := arity("Tmpfs", args, kwargs, 0, -1); err != nil { return nil, err }
	options, err := optionsFromArgs[buildkitllb.TmpfsOption](args, 0, "TmpfsOption")
	if err != nil { return nil, err }
	return wrapOption("MountOption", buildkitllb.Tmpfs(options...)), nil
}

func withProxyOption(_ *starlark.Thread, args starlark.Tuple, kwargs []starlark.Tuple) (any, error) {
	if err := arity("WithProxy", args, kwargs, 1, 1); err != nil { return nil, err }
	proxy, ok := args[0].(*ProxyEnvValue)
	if !ok || proxy == nil { return nil, fmt.Errorf("WithProxy: got %s, want llb.ProxyEnv", valueType(args[0])) }
	return buildkitllb.WithProxy(proxy.proxy), nil
}

func withRunBuiltin(_ *starlark.Thread, _ *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	if err := arity("With", args, kwargs, 0, -1); err != nil { return nil, err }
	options, err := optionsFromArgs[buildkitllb.StateOption](args, 0, "StateOption")
	if err != nil { return nil, err }
	return wrapOption("RunOption", buildkitllb.With(options...)), nil
}

func execCacheMountSharingValues() map[string]starlark.Value {
	return map[string]starlark.Value{
		"CacheMountShared":  starlark.MakeInt(int(buildkitllb.CacheMountShared)),
		"CacheMountPrivate": starlark.MakeInt(int(buildkitllb.CacheMountPrivate)),
		"CacheMountLocked":  starlark.MakeInt(int(buildkitllb.CacheMountLocked)),
	}
}

func cacheMountSharingValues() map[string]starlark.Value { return execCacheMountSharingValues() }

func mountContentCacheValues() map[string]starlark.Value {
	return map[string]starlark.Value{
		"MountContentCacheDefault": starlark.MakeInt(int(buildkitllb.MountContentCacheDefault)),
		"MountContentCacheOn":      starlark.MakeInt(int(buildkitllb.MountContentCacheOn)),
		"MountContentCacheOff":     starlark.MakeInt(int(buildkitllb.MountContentCacheOff)),
	}
}

func ulimitValues() map[string]starlark.Value {
	names := map[string]buildkitllb.UlimitName{
		"UlimitCore": buildkitllb.UlimitCore, "UlimitCPU": buildkitllb.UlimitCPU,
		"UlimitData": buildkitllb.UlimitData, "UlimitFsize": buildkitllb.UlimitFsize,
		"UlimitLocks": buildkitllb.UlimitLocks, "UlimitMemlock": buildkitllb.UlimitMemlock,
		"UlimitMsgqueue": buildkitllb.UlimitMsgqueue, "UlimitNice": buildkitllb.UlimitNice,
		"UlimitNofile": buildkitllb.UlimitNofile, "UlimitNproc": buildkitllb.UlimitNproc,
		"UlimitRss": buildkitllb.UlimitRss, "UlimitRtprio": buildkitllb.UlimitRtprio,
		"UlimitRttime": buildkitllb.UlimitRttime, "UlimitSigpending": buildkitllb.UlimitSigpending,
		"UlimitStack": buildkitllb.UlimitStack,
	}
	values := make(map[string]starlark.Value, len(names))
	for name, value := range names { values[name] = starlark.String(string(value)) }
	return values
}

func stateFromValue(value starlark.Value) (buildkitllb.State, error) {
	state, ok := value.(*StateValue)
	if !ok || state == nil { return buildkitllb.State{}, fmt.Errorf("got %s, want llb.State", valueType(value)) }
	return state.state, nil
}

func outputFromValue(value starlark.Value) (buildkitllb.Output, error) {
	switch v := value.(type) {
	case *StateValue:
		if v != nil { return v.state.Output(), nil }
	case *OutputValue:
		if v != nil { return v.output, nil }
	}
	return nil, fmt.Errorf("got %s, want llb.State or llb.Output", valueType(value))
}

func (v *ExecStateValue) AttrNamesForState() []string { return stateAttrNames }

var (
	_ starlark.Value    = (*ExecStateValue)(nil)
	_ starlark.HasAttrs = (*ExecStateValue)(nil)
	_ starlark.Value    = (*ExecOpValue)(nil)
	_ starlark.HasAttrs = (*ExecOpValue)(nil)
	_ starlark.Value    = (*ProxyEnvValue)(nil)
	_ starlark.HasAttrs = (*ProxyEnvValue)(nil)
	_ starlark.Value    = (*ExecInfoValue)(nil)
	_ starlark.HasAttrs = (*ExecInfoValue)(nil)
	_ starlark.Value    = (*mountInfoValue)(nil)
	_ starlark.HasAttrs = (*mountInfoValue)(nil)
	_ starlark.Value    = (*cdiDeviceInfoValue)(nil)
	_ starlark.HasAttrs = (*cdiDeviceInfoValue)(nil)
	_ starlark.Value    = (*sshInfoValue)(nil)
	_ starlark.HasAttrs = (*sshInfoValue)(nil)
	_ starlark.Value    = (*secretInfoValue)(nil)
	_ starlark.HasAttrs = (*secretInfoValue)(nil)
)


// cdiDeviceOptionValues groups CDI device option values separately from RUN
// options and the other optional socket/secret flags.
func cdiDeviceOptionValues() map[string]starlark.Value {
	return map[string]starlark.Value{
		"CDIDeviceOptional": wrapOption("CDIDeviceOption", buildkitllb.CDIDeviceOptional),
	}
}

// sshOptionValues groups SSH socket option flags.
func sshOptionValues() map[string]starlark.Value {
	return map[string]starlark.Value{
		"SSHOptional": wrapOption("SSHOption", buildkitllb.SSHOptional),
	}
}

// secretOptionValues groups secret option flags.
func secretOptionValues() map[string]starlark.Value {
	return map[string]starlark.Value{
		"SecretOptional": wrapOption("SecretOption", buildkitllb.SecretOptional),
	}
}
