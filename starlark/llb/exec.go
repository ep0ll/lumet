package llb

import (
	"fmt"
	"net"
	buildkitllb "github.com/moby/buildkit/client/llb"
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

func (v *ExecStateValue) String() string        { return "<llb.exec_state>" }
func (v *ExecStateValue) Type() string          { return "llb.exec_state" }
func (v *ExecStateValue) Freeze()               {}
func (v *ExecStateValue) Truth() starlark.Bool  { return starlark.Bool(v != nil) }
func (v *ExecStateValue) Hash() (uint32, error) { return 0, fmt.Errorf("unhashable: llb.exec_state") }

func (v *ExecStateValue) Attr(name string) (starlark.Value, error) {
		if v == nil {
		return nil, fmt.Errorf("llb.exec_state: nil receiver")
	}
	switch name {
	case "root":
		return boundBuiltin("exec_state.root", func(_ *starlark.Thread, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
			if err := arity("exec_state.root", args, kwargs, 0, 0); err != nil { return nil, err }
			return WrapState(v.execState.Root()), nil
		}), nil
	case "add_mount":
		return boundBuiltin("exec_state.add_mount", func(_ *starlark.Thread, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
			if err := arity("exec_state.add_mount", args, kwargs, 2, -1); err != nil { return nil, err }
			target, err := stringAt("exec_state.add_mount", args, 0); if err != nil { return nil, err }
			source, err := stateFromValue(args[1]); if err != nil { return nil, err }
			options, err := optionsFromArgs[buildkitllb.MountOption](args, 2, "mount_option")
			if err != nil { return nil, err }
			return WrapState(v.execState.AddMount(target, source, options...)), nil
		}), nil
	case "get_mount":
		return boundBuiltin("exec_state.get_mount", func(_ *starlark.Thread, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
			if err := arity("exec_state.get_mount", args, kwargs, 1, 1); err != nil { return nil, err }
			target, err := stringAt("exec_state.get_mount", args, 0); if err != nil { return nil, err }
			return WrapState(v.execState.GetMount(target)), nil
		}), nil
	default:
		return WrapState(v.execState.State).Attr(name)
	}
}

func (v *ExecStateValue) AttrNames() []string {
	names := []string{"add_mount", "get_mount", "root", "add_mount", "get_mount", "root"}
	names = append(names, WrapState(v.execState.State).AttrNames()...)
	return SnakeCaseNames(compactStrings(names))
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

func (v *ExecOpValue) String() string        { return "<llb.exec_op>" }
func (v *ExecOpValue) Type() string          { return "llb.exec_op" }
func (v *ExecOpValue) Freeze()               {}
func (v *ExecOpValue) Truth() starlark.Bool  { return starlark.Bool(v != nil && v.exec != nil) }
func (v *ExecOpValue) Hash() (uint32, error) { return 0, fmt.Errorf("unhashable: llb.exec_op") }

func (v *ExecOpValue) Attr(name string) (starlark.Value, error) {
		if v == nil || v.exec == nil {
		return nil, fmt.Errorf("llb.exec_op: nil receiver")
	}
	switch name {
	case "add_mount":
		return boundBuiltin("exec_op.add_mount", func(_ *starlark.Thread, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
			if err := arity("exec_op.add_mount", args, kwargs, 2, -1); err != nil { return nil, err }
			target, err := stringAt("exec_op.add_mount", args, 0); if err != nil { return nil, err }
			source, err := outputFromValue(args[1]); if err != nil { return nil, err }
			options, err := optionsFromArgs[buildkitllb.MountOption](args, 2, "mount_option")
			if err != nil { return nil, err }
			return WrapOutput(v.exec.AddMount(target, source, options...)), nil
		}), nil
	case "get_mount":
		return boundBuiltin("exec_op.get_mount", func(_ *starlark.Thread, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
			if err := arity("exec_op.get_mount", args, kwargs, 1, 1); err != nil { return nil, err }
			target, err := stringAt("exec_op.get_mount", args, 0); if err != nil { return nil, err }
			return WrapOutput(v.exec.GetMount(target)), nil
		}), nil
	case "output":
		return boundBuiltin("exec_op.output", func(_ *starlark.Thread, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
			if err := arity("exec_op.output", args, kwargs, 0, 0); err != nil { return nil, err }
			return WrapOutput(v.exec.Output()), nil
		}), nil
	case "inputs":
		return boundBuiltin("exec_op.inputs", func(_ *starlark.Thread, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
			if err := arity("exec_op.inputs", args, kwargs, 0, 0); err != nil { return nil, err }
			inputs := v.exec.Inputs()
			values := make([]starlark.Value, len(inputs))
			for i, input := range inputs { values[i] = WrapOutput(input) }
			return starlark.NewList(values), nil
		}), nil
	case "validate":
		return boundBuiltin("exec_op.validate", func(thread *starlark.Thread, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
			if err := arity("exec_op.validate", args, kwargs, 0, -1); err != nil { return nil, err }
			constraints, err := constraintsFromArgs(args, 0)
			if err != nil { return nil, err }
			if err := v.exec.Validate(callContext(thread), constraints); err != nil { return nil, err }
			return starlark.None, nil
		}), nil
	case "marshal":
		return boundBuiltin("exec_op.marshal", func(thread *starlark.Thread, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
			if err := arity("exec_op.marshal", args, kwargs, 0, -1); err != nil { return nil, err }
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
	return SnakeCaseNames([]string{"add_mount", "get_mount", "inputs", "marshal", "output", "validate",
		"add_mount", "get_mount", "inputs", "marshal", "output", "validate"})
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

func (v *ProxyEnvValue) String() string        { return "<llb.proxy_env>" }
func (v *ProxyEnvValue) Type() string          { return "llb.proxy_env" }
func (v *ProxyEnvValue) Freeze()               {}
func (v *ProxyEnvValue) Truth() starlark.Bool  { return starlark.Bool(v != nil) }
func (v *ProxyEnvValue) Hash() (uint32, error) { return 0, fmt.Errorf("unhashable: llb.proxy_env") }

func (v *ProxyEnvValue) Attr(name string) (starlark.Value, error) {
		if v == nil { return nil, fmt.Errorf("llb.proxy_env: nil receiver") }
	switch name {
	case "http_proxy": return starlark.String(v.proxy.HTTPProxy), nil
	case "https_proxy": return starlark.String(v.proxy.HTTPSProxy), nil
	case "ftp_proxy": return starlark.String(v.proxy.FTPProxy), nil
	case "no_proxy": return starlark.String(v.proxy.NoProxy), nil
	case "all_proxy": return starlark.String(v.proxy.AllProxy), nil
	default: return nil, nil
	}
}

func (v *ProxyEnvValue) AttrNames() []string {
	return SnakeCaseNames([]string{"all_proxy", "ftp_proxy", "http_proxy", "https_proxy", "no_proxy",
		"all_proxy", "ftp_proxy", "http_proxy", "https_proxy", "no_proxy"})
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

func (v *ExecInfoValue) String() string        { return "<llb.exec_info>" }
func (v *ExecInfoValue) Type() string          { return "llb.exec_info" }
func (v *ExecInfoValue) Freeze()               {}
func (v *ExecInfoValue) Truth() starlark.Bool  { return starlark.Bool(v != nil && v.info != nil) }
func (v *ExecInfoValue) Hash() (uint32, error) { return 0, fmt.Errorf("unhashable: llb.exec_info") }

func (v *ExecInfoValue) Attr(name string) (starlark.Value, error) {
		if v == nil || v.info == nil { return nil, fmt.Errorf("llb.exec_info: nil receiver") }
	switch name {
	case "state":
		return WrapState(v.info.State), nil
	case "mounts":
		values := make([]starlark.Value, len(v.info.Mounts))
		for i, mount := range v.info.Mounts { values[i] = wrapMountInfo(mount) }
		return starlark.NewList(values), nil
	case "readonly_root_fs":
		return starlark.Bool(v.info.ReadonlyRootFS), nil
	case "proxy_env":
		if v.info.ProxyEnv == nil { return starlark.None, nil }
		return WrapProxyEnv(*v.info.ProxyEnv), nil
	case "secrets":
		values := make([]starlark.Value, len(v.info.Secrets))
		for i, secret := range v.info.Secrets { values[i] = wrapSecretInfo(secret) }
		return starlark.NewList(values), nil
	case "ssh":
		values := make([]starlark.Value, len(v.info.SSH))
		for i, item := range v.info.SSH { values[i] = wrapSSHInfo(item) }
		return starlark.NewList(values), nil
	case "cdi_devices":
		values := make([]starlark.Value, len(v.info.CDIDevices))
		for i, item := range v.info.CDIDevices { values[i] = wrapCDIDeviceInfo(item) }
		return starlark.NewList(values), nil
	case "constraints":
		return WrapConstraints(v.info.Constraints), nil
	default:
		return nil, nil
	}
}

func (v *ExecInfoValue) AttrNames() []string {
	return SnakeCaseNames([]string{"state", "mounts", "readonly_root_fs", "proxy_env", "secrets", "ssh", "cdi_devices", "constraints",
		"state", "mounts", "readonly_root_fs", "proxy_env", "secrets", "ssh", "cdi_devices", "constraints"})
}

type mountInfoValue struct{ info buildkitllb.MountInfo }

func (v *mountInfoValue) String() string        { return "<llb.mount_info>" }
func (v *mountInfoValue) Type() string          { return "llb.mount_info" }
func (v *mountInfoValue) Freeze()               {}
func (v *mountInfoValue) Truth() starlark.Bool  { return starlark.Bool(v != nil) }
func (v *mountInfoValue) Hash() (uint32, error) { return 0, fmt.Errorf("unhashable: llb.mount_info") }
func (v *mountInfoValue) Attr(name string) (starlark.Value, error) {
		if v == nil { return nil, fmt.Errorf("llb.mount_info: nil receiver") }
	switch name {
	case "target": return starlark.String(v.info.Target), nil
	case "source": return WrapOutput(v.info.Source), nil
	case "options", "opts": return optionSliceValue(v.info.Opts, "mount_option"), nil
	default: return nil, nil
	}
}
func (v *mountInfoValue) AttrNames() []string { return SnakeCaseNames([]string{"target", "source", "options", "target", "source", "opts"}) }

type cdiDeviceInfoValue struct{ info buildkitllb.CDIDeviceInfo }
func (v *cdiDeviceInfoValue) String() string        { return "<llb.cdi_device_info>" }
func (v *cdiDeviceInfoValue) Type() string          { return "llb.cdi_device_info" }
func (v *cdiDeviceInfoValue) Freeze()               {}
func (v *cdiDeviceInfoValue) Truth() starlark.Bool  { return starlark.Bool(v != nil) }
func (v *cdiDeviceInfoValue) Hash() (uint32, error) { return 0, fmt.Errorf("unhashable: llb.cdi_device_info") }
func (v *cdiDeviceInfoValue) Attr(name string) (starlark.Value, error) {
		if v == nil { return nil, fmt.Errorf("llb.cdi_device_info: nil receiver") }
	if name == "name" || name == "name" { return starlark.String(v.info.Name), nil }
	return nil, nil
}
func (v *cdiDeviceInfoValue) AttrNames() []string { return SnakeCaseNames([]string{"name", "name"}) }

type sshInfoValue struct{ info buildkitllb.SSHInfo }
func (v *sshInfoValue) String() string        { return "<llb.ssh_info>" }
func (v *sshInfoValue) Type() string          { return "llb.ssh_info" }
func (v *sshInfoValue) Freeze()               {}
func (v *sshInfoValue) Truth() starlark.Bool  { return starlark.Bool(v != nil) }
func (v *sshInfoValue) Hash() (uint32, error) { return 0, fmt.Errorf("unhashable: llb.ssh_info") }
func (v *sshInfoValue) Attr(name string) (starlark.Value, error) {
		if v == nil { return nil, fmt.Errorf("llb.ssh_info: nil receiver") }
	switch name {
	case "id": return starlark.String(v.info.ID), nil
	case "target": return starlark.String(v.info.Target), nil
	case "uid": return starlark.MakeInt(v.info.UID), nil
	case "gid": return starlark.MakeInt(v.info.GID), nil
	case "mode": return starlark.MakeInt(v.info.Mode), nil
	case "optional": return starlark.Bool(v.info.Optional), nil
	default: return nil, nil
	}
}
func (v *sshInfoValue) AttrNames() []string { return SnakeCaseNames([]string{"id", "target", "uid", "gid", "mode", "optional", "id", "target", "uid", "gid", "mode", "optional"}) }

type secretInfoValue struct{ info buildkitllb.SecretInfo }
func (v *secretInfoValue) String() string        { return "<llb.secret_info>" }
func (v *secretInfoValue) Type() string          { return "llb.secret_info" }
func (v *secretInfoValue) Freeze()               {}
func (v *secretInfoValue) Truth() starlark.Bool  { return starlark.Bool(v != nil) }
func (v *secretInfoValue) Hash() (uint32, error) { return 0, fmt.Errorf("unhashable: llb.secret_info") }
func (v *secretInfoValue) Attr(name string) (starlark.Value, error) {
		if v == nil { return nil, fmt.Errorf("llb.secret_info: nil receiver") }
	switch name {
	case "id": return starlark.String(v.info.ID), nil
	case "target":
		if v.info.Target == nil { return starlark.None, nil }
		return starlark.String(*v.info.Target), nil
	case "mode": return starlark.MakeInt(v.info.Mode), nil
	case "uid": return starlark.MakeInt(v.info.UID), nil
	case "gid": return starlark.MakeInt(v.info.GID), nil
	case "optional": return starlark.Bool(v.info.Optional), nil
	case "env": return starlark.Bool(v.info.Env), nil
	case "env_name": return starlark.String(v.info.EnvName), nil
	default: return nil, nil
	}
}
func (v *secretInfoValue) AttrNames() []string { return SnakeCaseNames([]string{"id", "target", "mode", "uid", "gid", "optional", "env", "env_name", "id", "target", "mode", "uid", "gid", "optional", "env", "env_name"}) }

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
	if name != "run" && name != "run" { return nil, false, nil }
	return boundBuiltin("state.run", func(_ *starlark.Thread, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
		if err := arity("state.run", args, kwargs, 0, -1); err != nil { return nil, err }
		options, err := optionsFromArgs[buildkitllb.RunOption](args, 0, "run_option")
		if err != nil { return nil, err }
		return WrapExecState(v.state.Run(options...)), nil
	}), true, nil
}

var stateExecAttrNames = []string{"run", "run"}

func execBuiltins() map[string]starlark.Value {
	return map[string]starlark.Value{
		"new_exec_op": builtin("new_exec_op", newExecOpBuiltin),
		"exec_info": builtin("exec_info", execInfoBuiltin),
		"proxy_env": builtin("proxy_env", proxyEnvBuiltin),

		"shlex": stringOptionBuiltin("shlex", "run_option", buildkitllb.Shlex),
		"shlexf": builtin("shlexf", shlexfBuiltin),
		"args": builtin("args", argsBuiltin),
		"readonly_root_fs": builtin("readonly_root_fs", readonlyRootFSBuiltin),
		"valid_exit_codes": builtin("valid_exit_codes", validExitCodesBuiltin),
		"with_cgroup_parent": stringOptionBuiltin("with_cgroup_parent", "run_option", buildkitllb.WithCgroupParent),
		"add_extra_host": builtin("add_extra_host", addExtraHostRunBuiltin),
		"add_ulimit": builtin("add_ulimit", addUlimitRunBuiltin),
		"add_cdi_device": builtin("add_cdi_device", addCDIDeviceBuiltin),
		"cdi_device_name": stringOptionBuiltin("cdi_device_name", "cdi_device_option", buildkitllb.CDIDeviceName),
		"add_mount": builtin("add_mount", addMountRunBuiltin),
		"add_ssh_socket": builtin("add_ssh_socket", addSSHSocketBuiltin),
		"sshid": stringOptionBuiltin("sshid", "ssh_option", buildkitllb.SSHID),
		"ssh_socket_target": stringOptionBuiltin("ssh_socket_target", "ssh_option", buildkitllb.SSHSocketTarget),
		"ssh_socket_opt": builtin("ssh_socket_opt", sshSocketOptionBuiltin),
		"add_secret": builtin("add_secret", addSecretBuiltin),
		"add_secret_with_dest": builtin("add_secret_with_dest", addSecretWithDestBuiltin),
		"secret_id": stringOptionBuiltin("secret_id", "secret_option", buildkitllb.SecretID),
		"secret_as_env": boolOptionBuiltin("secret_as_env", "secret_option", buildkitllb.SecretAsEnv),
		"secret_as_env_name": stringOptionBuiltin("secret_as_env_name", "secret_option", buildkitllb.SecretAsEnvName),
		"secret_file_opt": builtin("secret_file_opt", secretFileOptBuiltin),
		"source_path": stringOptionBuiltin("source_path", "mount_option", buildkitllb.SourcePath),
		"as_persistent_cache_dir": builtin("as_persistent_cache_dir", persistentCacheDirBuiltin),
		"content_cache": builtin("content_cache", contentCacheBuiltin),
		"tmpfs": builtin("tmpfs", tmpfsBuiltin),
		"tmpfs_size": int64OptionBuiltin("tmpfs_size", "tmpfs_option", buildkitllb.TmpfsSize),
		"with_proxy": optionBuiltin("with_proxy", "run_option", withProxyOption),
		"with": builtin("with", withRunBuiltin),
	}
}

func newExecOpBuiltin(_ *starlark.Thread, _ *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	if err := arity("new_exec_op", args, kwargs, 4, 4); err != nil { return nil, err }
	state, err := stateFromValue(args[0]); if err != nil { return nil, err }
	var proxy *buildkitllb.ProxyEnv
	if args[1] != starlark.None {
		value, ok := args[1].(*ProxyEnvValue)
		if !ok || value == nil { return nil, fmt.Errorf("new_exec_op: argument 2: got %s, want llb.proxy_env or none", valueType(args[1])) }
		p := value.proxy
		proxy = &p
	}
	readonly, ok := args[2].(starlark.Bool)
	if !ok { return nil, fmt.Errorf("new_exec_op: argument 3: got %s, want bool", valueType(args[2])) }
	constraints, ok := args[3].(*ConstraintsValue)
	if !ok || constraints == nil { return nil, fmt.Errorf("new_exec_op: argument 4: got %s, want llb.constraints", valueType(args[3])) }
	return WrapExecOp(buildkitllb.NewExecOp(state, proxy, bool(readonly), constraints.Unwrap())), nil
}

func execInfoBuiltin(_ *starlark.Thread, _ *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	if err := arity("exec_info", args, kwargs, 1, -1); err != nil { return nil, err }
	state, err := stateFromValue(args[0]); if err != nil { return nil, err }
	info := &buildkitllb.ExecInfo{State: state}
	options, err := optionsFromArgs[buildkitllb.RunOption](args, 1, "run_option")
	if err != nil { return nil, err }
	for _, option := range options { option.SetRunOption(info) }
	return WrapExecInfo(info), nil
}

func proxyEnvBuiltin(_ *starlark.Thread, _ *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	var proxy buildkitllb.ProxyEnv
	if err := UnpackArgs("proxy_env", args, kwargs,
		"http_proxy?", &proxy.HTTPProxy,
		"https_proxy?", &proxy.HTTPSProxy,
		"ftp_proxy?", &proxy.FTPProxy,
		"no_proxy?", &proxy.NoProxy,
		"all_proxy?", &proxy.AllProxy,
	); err != nil { return nil, err }
	return WrapProxyEnv(proxy), nil
}

func shlexfBuiltin(_ *starlark.Thread, _ *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	if err := arity("shlexf", args, kwargs, 1, -1); err != nil { return nil, err }
	format, err := stringAt("shlexf", args, 0); if err != nil { return nil, err }
	values := make([]any, 0, len(args)-1)
	for _, value := range args[1:] { values = append(values, goScalar(value)) }
	return wrapOption("run_option", buildkitllb.Shlexf(format, values...)), nil
}

func argsBuiltin(_ *starlark.Thread, _ *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	if err := arity("args", args, kwargs, 0, -1); err != nil { return nil, err }
	var values []string
	if len(args) == 1 {
		list, ok := args[0].(starlark.Indexable)
		if ok {
			var err error
			values, err = stringList(list)
			if err != nil { return nil, err }
		} else {
			value, err := stringAt("args", args, 0)
			if err != nil { return nil, err }
			values = []string{value}
		}
	} else {
		values = make([]string, len(args))
		for i := range args {
			value, err := stringAt("args", args, i)
			if err != nil { return nil, err }
			values[i] = value
		}
	}
	return wrapOption("run_option", buildkitllb.Args(values)), nil
}

func readonlyRootFSBuiltin(_ *starlark.Thread, _ *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	if err := arity("readonly_root_fs", args, kwargs, 0, 0); err != nil { return nil, err }
	return wrapOption("run_option", buildkitllb.ReadonlyRootFS()), nil
}

func validExitCodesBuiltin(_ *starlark.Thread, _ *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	if err := arity("valid_exit_codes", args, kwargs, 0, -1); err != nil { return nil, err }
	codes := make([]int, len(args))
	for i := range args {
		code, err := intAt("valid_exit_codes", args, i)
		if err != nil { return nil, err }
		codes[i] = code
	}
	return wrapOption("run_option", buildkitllb.ValidExitCodes(codes...)), nil
}

func addExtraHostRunBuiltin(_ *starlark.Thread, _ *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	if err := arity("add_extra_host", args, kwargs, 2, 2); err != nil { return nil, err }
	host, err := stringAt("add_extra_host", args, 0); if err != nil { return nil, err }
	rawIP, err := stringAt("add_extra_host", args, 1); if err != nil { return nil, err }
	ip := net.ParseIP(rawIP)
	if ip == nil { return nil, fmt.Errorf("add_extra_host: invalid ip address %q", rawIP) }
	return wrapOption("run_option", buildkitllb.AddExtraHost(host, ip)), nil
}

func addUlimitRunBuiltin(_ *starlark.Thread, _ *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	if err := arity("add_ulimit", args, kwargs, 3, 3); err != nil { return nil, err }
	name, err := stringAt("add_ulimit", args, 0); if err != nil { return nil, err }
	soft, err := int64At("add_ulimit", args, 1); if err != nil { return nil, err }
	hard, err := int64At("add_ulimit", args, 2); if err != nil { return nil, err }
	return wrapOption("run_option", buildkitllb.AddUlimit(buildkitllb.UlimitName(name), soft, hard)), nil
}

func addCDIDeviceBuiltin(_ *starlark.Thread, _ *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	if err := arity("add_cdi_device", args, kwargs, 0, -1); err != nil { return nil, err }
	options, err := optionsFromArgs[buildkitllb.CDIDeviceOption](args, 0, "cdi_device_option")
	if err != nil { return nil, err }
	return wrapOption("run_option", buildkitllb.AddCDIDevice(options...)), nil
}

func addMountRunBuiltin(_ *starlark.Thread, _ *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	if err := arity("add_mount", args, kwargs, 2, -1); err != nil { return nil, err }
	target, err := stringAt("add_mount", args, 0); if err != nil { return nil, err }
	state, err := stateFromValue(args[1]); if err != nil { return nil, err }
	options, err := optionsFromArgs[buildkitllb.MountOption](args, 2, "mount_option")
	if err != nil { return nil, err }
	return wrapOption("run_option", buildkitllb.AddMount(target, state, options...)), nil
}

func addSSHSocketBuiltin(_ *starlark.Thread, _ *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	if err := arity("add_ssh_socket", args, kwargs, 0, -1); err != nil { return nil, err }
	options, err := optionsFromArgs[buildkitllb.SSHOption](args, 0, "ssh_option")
	if err != nil { return nil, err }
	return wrapOption("run_option", buildkitllb.AddSSHSocket(options...)), nil
}

func sshSocketOptionBuiltin(_ *starlark.Thread, _ *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	if err := arity("ssh_socket_opt", args, kwargs, 4, 4); err != nil { return nil, err }
	target, err := stringAt("ssh_socket_opt", args, 0); if err != nil { return nil, err }
	uid, err := intAt("ssh_socket_opt", args, 1); if err != nil { return nil, err }
	gid, err := intAt("ssh_socket_opt", args, 2); if err != nil { return nil, err }
	mode, err := intAt("ssh_socket_opt", args, 3); if err != nil { return nil, err }
	return wrapOption("ssh_option", buildkitllb.SSHSocketOpt(target, uid, gid, mode)), nil
}

func addSecretBuiltin(_ *starlark.Thread, _ *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	if err := arity("add_secret", args, kwargs, 1, -1); err != nil { return nil, err }
	dest, err := stringAt("add_secret", args, 0); if err != nil { return nil, err }
	options, err := optionsFromArgs[buildkitllb.SecretOption](args, 1, "secret_option")
	if err != nil { return nil, err }
	return wrapOption("run_option", buildkitllb.AddSecret(dest, options...)), nil
}

func addSecretWithDestBuiltin(_ *starlark.Thread, _ *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	if err := arity("add_secret_with_dest", args, kwargs, 2, -1); err != nil { return nil, err }
	src, err := stringAt("add_secret_with_dest", args, 0); if err != nil { return nil, err }
	var dest *string
	if args[1] != starlark.None {
		value, ok := args[1].(starlark.String)
		if !ok { return nil, fmt.Errorf("add_secret_with_dest: destination must be string or none, got %s", valueType(args[1])) }
		target := string(value)
		dest = &target
	}
	options, err := optionsFromArgs[buildkitllb.SecretOption](args, 2, "secret_option")
	if err != nil { return nil, err }
	return wrapOption("run_option", buildkitllb.AddSecretWithDest(src, dest, options...)), nil
}

func secretFileOptBuiltin(_ *starlark.Thread, _ *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	if err := arity("secret_file_opt", args, kwargs, 3, 3); err != nil { return nil, err }
	uid, err := intAt("secret_file_opt", args, 0); if err != nil { return nil, err }
	gid, err := intAt("secret_file_opt", args, 1); if err != nil { return nil, err }
	mode, err := intAt("secret_file_opt", args, 2); if err != nil { return nil, err }
	return wrapOption("secret_option", buildkitllb.SecretFileOpt(uid, gid, mode)), nil
}

func persistentCacheDirBuiltin(_ *starlark.Thread, _ *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	if err := arity("as_persistent_cache_dir", args, kwargs, 2, 2); err != nil { return nil, err }
	id, err := stringAt("as_persistent_cache_dir", args, 0); if err != nil { return nil, err }
	sharing, err := intAt("as_persistent_cache_dir", args, 1); if err != nil { return nil, err }
	return wrapOption("mount_option", buildkitllb.AsPersistentCacheDir(id, buildkitllb.CacheMountSharingMode(sharing))), nil
}

func contentCacheBuiltin(_ *starlark.Thread, _ *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	if err := arity("content_cache", args, kwargs, 1, 1); err != nil { return nil, err }
	mode, err := intAt("content_cache", args, 0); if err != nil { return nil, err }
	return wrapOption("mount_option", buildkitllb.ContentCache(buildkitllb.MountContentCache(mode))), nil
}

func tmpfsBuiltin(_ *starlark.Thread, _ *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	if err := arity("tmpfs", args, kwargs, 0, -1); err != nil { return nil, err }
	options, err := optionsFromArgs[buildkitllb.TmpfsOption](args, 0, "tmpfs_option")
	if err != nil { return nil, err }
	return wrapOption("mount_option", buildkitllb.Tmpfs(options...)), nil
}

func withProxyOption(_ *starlark.Thread, args starlark.Tuple, kwargs []starlark.Tuple) (any, error) {
	if err := arity("with_proxy", args, kwargs, 1, 1); err != nil { return nil, err }
	proxy, ok := args[0].(*ProxyEnvValue)
	if !ok || proxy == nil { return nil, fmt.Errorf("with_proxy: got %s, want llb.proxy_env", valueType(args[0])) }
	return buildkitllb.WithProxy(proxy.proxy), nil
}

func withRunBuiltin(_ *starlark.Thread, _ *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	if err := arity("with", args, kwargs, 0, -1); err != nil { return nil, err }
	options, err := optionsFromArgs[buildkitllb.StateOption](args, 0, "state_option")
	if err != nil { return nil, err }
	return wrapOption("run_option", buildkitllb.With(options...)), nil
}

func execCacheMountSharingValues() map[string]starlark.Value {
	return map[string]starlark.Value{
		"cache_mount_shared":  starlark.MakeInt(int(buildkitllb.CacheMountShared)),
		"cache_mount_private": starlark.MakeInt(int(buildkitllb.CacheMountPrivate)),
		"cache_mount_locked":  starlark.MakeInt(int(buildkitllb.CacheMountLocked)),
	}
}

func cacheMountSharingValues() map[string]starlark.Value { return execCacheMountSharingValues() }

func mountContentCacheValues() map[string]starlark.Value {
	return map[string]starlark.Value{
		"mount_content_cache_default": starlark.MakeInt(int(buildkitllb.MountContentCacheDefault)),
		"mount_content_cache_on":      starlark.MakeInt(int(buildkitllb.MountContentCacheOn)),
		"mount_content_cache_off":     starlark.MakeInt(int(buildkitllb.MountContentCacheOff)),
	}
}

func ulimitValues() map[string]starlark.Value {
	names := map[string]buildkitllb.UlimitName{
		"ulimit_core": buildkitllb.UlimitCore, "ulimit_cpu": buildkitllb.UlimitCPU,
		"ulimit_data": buildkitllb.UlimitData, "ulimit_fsize": buildkitllb.UlimitFsize,
		"ulimit_locks": buildkitllb.UlimitLocks, "ulimit_memlock": buildkitllb.UlimitMemlock,
		"ulimit_msgqueue": buildkitllb.UlimitMsgqueue, "ulimit_nice": buildkitllb.UlimitNice,
		"ulimit_nofile": buildkitllb.UlimitNofile, "ulimit_nproc": buildkitllb.UlimitNproc,
		"ulimit_rss": buildkitllb.UlimitRss, "ulimit_rtprio": buildkitllb.UlimitRtprio,
		"ulimit_rttime": buildkitllb.UlimitRttime, "ulimit_sigpending": buildkitllb.UlimitSigpending,
		"ulimit_stack": buildkitllb.UlimitStack,
	}
	values := make(map[string]starlark.Value, len(names))
	for name, value := range names { values[name] = starlark.String(string(value)) }
	return values
}

func stateFromValue(value starlark.Value) (buildkitllb.State, error) {
	state, ok := value.(*StateValue)
	if !ok || state == nil { return buildkitllb.State{}, fmt.Errorf("got %s, want llb.state", valueType(value)) }
	return state.state, nil
}

func outputFromValue(value starlark.Value) (buildkitllb.Output, error) {
	switch v := value.(type) {
	case *StateValue:
		if v != nil { return v.state.Output(), nil }
	case *OutputValue:
		if v != nil { return v.output, nil }
	}
	return nil, fmt.Errorf("got %s, want llb.state or llb.output", valueType(value))
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
		"cdi_device_optional": wrapOption("cdi_device_option", buildkitllb.CDIDeviceOptional),
	}
}

// sshOptionValues groups SSH socket option flags.
func sshOptionValues() map[string]starlark.Value {
	return map[string]starlark.Value{
		"ssh_optional": wrapOption("ssh_option", buildkitllb.SSHOptional),
	}
}

// secretOptionValues groups secret option flags.
func secretOptionValues() map[string]starlark.Value {
	return map[string]starlark.Value{
		"secret_optional": wrapOption("secret_option", buildkitllb.SecretOptional),
	}
}
