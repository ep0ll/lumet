package llb

import (
	"fmt"
	"net"
	"sort"

	buildkitllb "github.com/moby/buildkit/client/llb"
	"github.com/moby/buildkit/solver/pb"
	"go.starlark.net/starlark"
)

// EnvListValue adapts BuildKit's lazily materialized environment list.
type EnvListValue struct {
	env *buildkitllb.EnvList
}

// WrapEnvList adapts a native EnvList.
func WrapEnvList(env *buildkitllb.EnvList) *EnvListValue {
	return &EnvListValue{env: env}
}

func (v *EnvListValue) String() string        { return "<llb.env_list>" }
func (v *EnvListValue) Type() string          { return "llb.env_list" }
func (v *EnvListValue) Freeze()               {}
func (v *EnvListValue) Truth() starlark.Bool  { return starlark.Bool(v != nil && v.env != nil) }
func (v *EnvListValue) Hash() (uint32, error) { return 0, fmt.Errorf("unhashable: llb.env_list") }

func (v *EnvListValue) Attr(name string) (starlark.Value, error) {
		if v == nil || v.env == nil {
		return nil, fmt.Errorf("llb.env_list: nil receiver")
	}
	switch name {
	case "add_or_replace":
		return boundBuiltin("env_list.add_or_replace", func(_ *starlark.Thread, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
			if err := arity("env_list.add_or_replace", args, kwargs, 2, 2); err != nil { return nil, err }
			key, err := stringAt("env_list.add_or_replace", args, 0); if err != nil { return nil, err }
			value, err := stringAt("env_list.add_or_replace", args, 1); if err != nil { return nil, err }
			return WrapEnvList(v.env.AddOrReplace(key, value)), nil
		}), nil
	case "set_default":
		return boundBuiltin("env_list.set_default", func(_ *starlark.Thread, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
			if err := arity("env_list.set_default", args, kwargs, 2, 2); err != nil { return nil, err }
			key, err := stringAt("env_list.set_default", args, 0); if err != nil { return nil, err }
			value, err := stringAt("env_list.set_default", args, 1); if err != nil { return nil, err }
			return WrapEnvList(v.env.SetDefault(key, value)), nil
		}), nil
	case "delete":
		return boundBuiltin("env_list.delete", func(_ *starlark.Thread, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
			if err := arity("env_list.delete", args, kwargs, 1, 1); err != nil { return nil, err }
			key, err := stringAt("env_list.delete", args, 0); if err != nil { return nil, err }
			env := v.env.Delete(key)
			return WrapEnvList(&env), nil
		}), nil
	case "get":
		return boundBuiltin("env_list.get", func(_ *starlark.Thread, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
			if err := arity("env_list.get", args, kwargs, 1, 1); err != nil { return nil, err }
			key, err := stringAt("env_list.get", args, 0); if err != nil { return nil, err }
			value, ok := v.env.Get(key)
			return starlark.Tuple{starlark.String(value), starlark.Bool(ok)}, nil
		}), nil
	case "keys":
		return boundBuiltin("env_list.keys", func(_ *starlark.Thread, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
			if err := arity("env_list.keys", args, kwargs, 0, 0); err != nil { return nil, err }
			return stringSliceValue(v.env.Keys()), nil
		}), nil
	case "to_array":
		return boundBuiltin("env_list.to_array", func(_ *starlark.Thread, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
			if err := arity("env_list.to_array", args, kwargs, 0, 0); err != nil { return nil, err }
			return stringSliceValue(v.env.ToArray()), nil
		}), nil
	default:
		return nil, nil
	}
}

func (v *EnvListValue) AttrNames() []string {
	return SnakeCaseNames([]string{"add_or_replace", "delete", "get", "keys", "set_default", "to_array",
		"add_or_replace", "delete", "get", "keys", "set_default", "to_array"})
}

// HostIPValue adapts BuildKit's host-to-IP mapping.
type HostIPValue struct {
	hostIP buildkitllb.HostIP
}

// WrapHostIP adapts a native HostIP value.
func WrapHostIP(hostIP buildkitllb.HostIP) *HostIPValue {
	return &HostIPValue{hostIP: hostIP}
}

func (v *HostIPValue) String() string        { return "<llb.host_ip>" }
func (v *HostIPValue) Type() string          { return "llb.host_ip" }
func (v *HostIPValue) Freeze()               {}
func (v *HostIPValue) Truth() starlark.Bool  { return starlark.Bool(v != nil) }
func (v *HostIPValue) Hash() (uint32, error) { return 0, fmt.Errorf("unhashable: llb.host_ip") }

func (v *HostIPValue) Attr(name string) (starlark.Value, error) {
		if v == nil { return nil, fmt.Errorf("llb.host_ip: nil receiver") }
	switch name {
	case "host":
		return starlark.String(v.hostIP.Host), nil
	case "ip":
		return starlark.String(v.hostIP.IP.String()), nil
	default:
		return nil, nil
	}
}

func (v *HostIPValue) AttrNames() []string { return SnakeCaseNames([]string{"host", "ip", "host", "ip"}) }

// LinuxResourcesValue adapts BuildKit's container resource limits.
type LinuxResourcesValue struct {
	resources buildkitllb.LinuxResources
}

// WrapLinuxResources adapts native LinuxResources.
func WrapLinuxResources(resources buildkitllb.LinuxResources) *LinuxResourcesValue {
	return &LinuxResourcesValue{resources: resources}
}

func (v *LinuxResourcesValue) String() string        { return "<llb.linux_resources>" }
func (v *LinuxResourcesValue) Type() string          { return "llb.linux_resources" }
func (v *LinuxResourcesValue) Freeze()               {}
func (v *LinuxResourcesValue) Truth() starlark.Bool  { return starlark.Bool(v != nil) }
func (v *LinuxResourcesValue) Hash() (uint32, error) { return 0, fmt.Errorf("unhashable: llb.linux_resources") }

func (v *LinuxResourcesValue) Attr(name string) (starlark.Value, error) {
		if v == nil { return nil, fmt.Errorf("llb.linux_resources: nil receiver") }
	switch name {
	case "memory":
		return starlark.MakeInt64(v.resources.Memory), nil
	case "memory_swap":
		return starlark.MakeInt64(v.resources.MemorySwap), nil
	case "cpu_shares":
		return starlark.MakeUint64(v.resources.CPUShares), nil
	case "cpu_period":
		return starlark.MakeUint64(v.resources.CPUPeriod), nil
	case "cpu_quota":
		return starlark.MakeInt64(v.resources.CPUQuota), nil
	case "cpuset_cpus":
		return starlark.String(v.resources.CpusetCpus), nil
	case "cpuset_mems":
		return starlark.String(v.resources.CpusetMems), nil
	default:
		return nil, nil
	}
}

func (v *LinuxResourcesValue) AttrNames() []string {
	return SnakeCaseNames([]string{"memory", "memory_swap", "cpu_shares", "cpu_period", "cpu_quota", "cpuset_cpus", "cpuset_mems",
		"memory", "memory_swap", "cpu_shares", "cpu_period", "cpu_quota", "cpuset_cpus", "cpuset_mems"})
}

func (v *StateValue) stateMetaAttr(name string) (starlark.Value, bool, error) {
	state := v.state
	switch name {
	case "add_env":
		return stateStringPairMethod("state.add_env", state.AddEnv), true, nil
	case "add_envf":
		return boundBuiltin("state.add_envf", func(_ *starlark.Thread, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
			if err := arity("state.add_envf", args, kwargs, 2, -1); err != nil { return nil, err }
			key, err := stringAt("state.add_envf", args, 0); if err != nil { return nil, err }
			format, err := stringAt("state.add_envf", args, 1); if err != nil { return nil, err }
			values := make([]any, 0, len(args)-2)
			for _, arg := range args[2:] { values = append(values, goScalar(arg)) }
			return WrapState(state.AddEnvf(key, format, values...)), nil
		}), true, nil
	case "del_env":
		return stateStringMethod("state.del_env", state.DelEnv), true, nil
	case "dir":
		return stateStringMethod("state.dir", state.Dir), true, nil
	case "dirf":
		return boundBuiltin("state.dirf", func(_ *starlark.Thread, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
			if err := arity("state.dirf", args, kwargs, 1, -1); err != nil { return nil, err }
			format, err := stringAt("state.dirf", args, 0); if err != nil { return nil, err }
			values := make([]any, 0, len(args)-1)
			for _, arg := range args[1:] { values = append(values, goScalar(arg)) }
			return WrapState(state.Dirf(format, values...)), nil
		}), true, nil
	case "user":
		return stateStringMethod("state.user", state.User), true, nil
	case "hostname":
		return stateStringMethod("state.hostname", state.Hostname), true, nil
	case "with_cgroup_parent":
		return stateStringMethod("state.with_cgroup_parent", state.WithCgroupParent), true, nil
	case "with":
		return boundBuiltin("state.with", func(_ *starlark.Thread, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
			if err := arity("state.with", args, kwargs, 0, -1); err != nil { return nil, err }
			options, err := optionsFromArgs[buildkitllb.StateOption](args, 0, "state_option")
			if err != nil { return nil, err }
			return WrapState(state.With(options...)), nil
		}), true, nil
	case "reset":
		return boundBuiltin("state.reset", func(_ *starlark.Thread, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
			if err := arity("state.reset", args, kwargs, 1, 1); err != nil { return nil, err }
			other, err := stateFromValue(args[0])
			if err != nil { return nil, err }
			return WrapState(state.Reset(other)), nil
		}), true, nil
	case "network":
		return boundBuiltin("state.network", func(_ *starlark.Thread, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
			if err := arity("state.network", args, kwargs, 1, 1); err != nil { return nil, err }
			mode, err := intAt("state.network", args, 0)
			if err != nil { return nil, err }
			return WrapState(state.Network(pb.NetMode(mode))), nil
		}), true, nil
	case "security":
		return boundBuiltin("state.security", func(_ *starlark.Thread, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
			if err := arity("state.security", args, kwargs, 1, 1); err != nil { return nil, err }
			mode, err := intAt("state.security", args, 0)
			if err != nil { return nil, err }
			return WrapState(state.Security(pb.SecurityMode(mode))), nil
		}), true, nil
	case "platform":
		return boundBuiltin("state.platform", func(_ *starlark.Thread, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
			if err := arity("state.platform", args, kwargs, 1, 1); err != nil { return nil, err }
			platform, ok := args[0].(*OCIPlatformValue)
			if !ok || platform == nil { return nil, fmt.Errorf("state.platform: got %s, want llb.oci_platform", valueType(args[0])) }
			return WrapState(state.Platform(platform.platform)), nil
		}), true, nil
	case "add_extra_host":
		return boundBuiltin("state.add_extra_host", func(_ *starlark.Thread, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
			if err := arity("state.add_extra_host", args, kwargs, 2, 2); err != nil { return nil, err }
			host, err := stringAt("state.add_extra_host", args, 0); if err != nil { return nil, err }
			rawIP, err := stringAt("state.add_extra_host", args, 1); if err != nil { return nil, err }
			ip := net.ParseIP(rawIP)
			if ip == nil { return nil, fmt.Errorf("state.add_extra_host: invalid ip address %q", rawIP) }
			return WrapState(state.AddExtraHost(host, ip)), nil
		}), true, nil
	case "requires":
		return boundBuiltin("state.requires", func(_ *starlark.Thread, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
			if err := arity("state.requires", args, kwargs, 1, -1); err != nil { return nil, err }
			id, err := stringAt("state.requires", args, 0)
			if err != nil { return nil, err }
			dependencies := make([]buildkitllb.State, 0, len(args)-1)
			for i := 1; i < len(args); i++ {
				dependency, err := stateFromValue(args[i])
				if err != nil { return nil, fmt.Errorf("state.requires dependency %d: %w", i, err) }
				dependencies = append(dependencies, dependency)
			}
			return WrapState(state.Requires(id, dependencies...)), nil
		}), true, nil
	case "add_ulimit":
		return boundBuiltin("state.add_ulimit", func(_ *starlark.Thread, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
			if err := arity("state.add_ulimit", args, kwargs, 3, 3); err != nil { return nil, err }
			name, err := stringAt("state.add_ulimit", args, 0); if err != nil { return nil, err }
			soft, err := int64At("state.add_ulimit", args, 1); if err != nil { return nil, err }
			hard, err := int64At("state.add_ulimit", args, 2); if err != nil { return nil, err }
			return WrapState(state.AddUlimit(buildkitllb.UlimitName(name), soft, hard)), nil
		}), true, nil
	case "get_env":
		return boundBuiltin("state.get_env", func(thread *starlark.Thread, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
			if err := arity("state.get_env", args, kwargs, 1, -1); err != nil { return nil, err }
			key, err := stringAt("state.get_env", args, 0); if err != nil { return nil, err }
			options, err := optionsFromArgs[buildkitllb.ConstraintsOpt](args, 1, "constraints_opt")
			if err != nil { return nil, err }
			value, ok, err := state.GetEnv(callContext(thread), key, options...)
			if err != nil { return nil, err }
			return starlark.Tuple{starlark.String(value), starlark.Bool(ok)}, nil
		}), true, nil
	case "env":
		return boundBuiltin("state.env", func(thread *starlark.Thread, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
			if err := arity("state.env", args, kwargs, 0, -1); err != nil { return nil, err }
			options, err := optionsFromArgs[buildkitllb.ConstraintsOpt](args, 0, "constraints_opt")
			if err != nil { return nil, err }
			env, err := state.Env(callContext(thread), options...)
			if err != nil { return nil, err }
			return WrapEnvList(env), nil
		}), true, nil
	case "get_dir":
		return boundBuiltin("state.get_dir", func(thread *starlark.Thread, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
			if err := arity("state.get_dir", args, kwargs, 0, -1); err != nil { return nil, err }
			options, err := optionsFromArgs[buildkitllb.ConstraintsOpt](args, 0, "constraints_opt")
			if err != nil { return nil, err }
			value, err := state.GetDir(callContext(thread), options...)
			if err != nil { return nil, err }
			return starlark.String(value), nil
		}), true, nil
	case "get_args":
		return boundBuiltin("state.get_args", func(thread *starlark.Thread, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
			if err := arity("state.get_args", args, kwargs, 0, -1); err != nil { return nil, err }
			options, err := optionsFromArgs[buildkitllb.ConstraintsOpt](args, 0, "constraints_opt")
			if err != nil { return nil, err }
			values, err := state.GetArgs(callContext(thread), options...)
			if err != nil { return nil, err }
			return stringSliceValue(values), nil
		}), true, nil
	case "get_hostname":
		return boundBuiltin("state.get_hostname", func(thread *starlark.Thread, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
			if err := arity("state.get_hostname", args, kwargs, 0, -1); err != nil { return nil, err }
			options, err := optionsFromArgs[buildkitllb.ConstraintsOpt](args, 0, "constraints_opt")
			if err != nil { return nil, err }
			value, err := state.GetHostname(callContext(thread), options...)
			if err != nil { return nil, err }
			return starlark.String(value), nil
		}), true, nil
	case "get_platform":
		return boundBuiltin("state.get_platform", func(thread *starlark.Thread, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
			if err := arity("state.get_platform", args, kwargs, 0, -1); err != nil { return nil, err }
			options, err := optionsFromArgs[buildkitllb.ConstraintsOpt](args, 0, "constraints_opt")
			if err != nil { return nil, err }
			value, err := state.GetPlatform(callContext(thread), options...)
			if err != nil { return nil, err }
			if value == nil { return starlark.None, nil }
			return WrapOCIPlatform(*value), nil
		}), true, nil
	case "get_network":
		return boundBuiltin("state.get_network", func(thread *starlark.Thread, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
			if err := arity("state.get_network", args, kwargs, 0, -1); err != nil { return nil, err }
			options, err := optionsFromArgs[buildkitllb.ConstraintsOpt](args, 0, "constraints_opt")
			if err != nil { return nil, err }
			value, err := state.GetNetwork(callContext(thread), options...)
			if err != nil { return nil, err }
			return starlark.MakeInt(int(value)), nil
		}), true, nil
	case "get_security":
		return boundBuiltin("state.get_security", func(thread *starlark.Thread, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
			if err := arity("state.get_security", args, kwargs, 0, -1); err != nil { return nil, err }
			options, err := optionsFromArgs[buildkitllb.ConstraintsOpt](args, 0, "constraints_opt")
			if err != nil { return nil, err }
			value, err := state.GetSecurity(callContext(thread), options...)
			if err != nil { return nil, err }
			return starlark.MakeInt(int(value)), nil
		}), true, nil
	default:
		return nil, false, nil
	}
}

var stateMetaAttrNames = []string{
	"add_env", "add_envf", "add_extra_host", "add_ulimit", "del_env", "dir", "dirf",
	"env", "get_args", "get_dir", "get_env", "get_hostname", "get_network", "get_platform", "get_security",
	"hostname", "network", "platform", "reset", "requires", "security", "user", "with", "with_cgroup_parent",
	"add_env", "add_envf", "add_extra_host", "add_ulimit", "del_env", "dir", "dirf", "env",
	"get_args", "get_dir", "get_env", "get_hostname", "get_network", "get_platform", "get_security",
	"hostname", "network", "platform", "reset", "requires", "security", "user", "with", "with_cgroup_parent",
}

func stateMetaAttr(v *StateValue, name string) (starlark.Value, bool, error) {
	return v.stateMetaAttr(name)
}

func metaBuiltins() map[string]starlark.Value {
	return map[string]starlark.Value{
		"add_env": optionBuiltin("add_env", "state_option", func(_ *starlark.Thread, args starlark.Tuple, kwargs []starlark.Tuple) (any, error) {
			if err := arity("add_env", args, kwargs, 2, 2); err != nil { return nil, err }
			key, err := stringAt("add_env", args, 0); if err != nil { return nil, err }
			value, err := stringAt("add_env", args, 1); if err != nil { return nil, err }
			return buildkitllb.AddEnv(key, value), nil
		}),
		"add_envf": optionBuiltin("add_envf", "state_option", func(_ *starlark.Thread, args starlark.Tuple, kwargs []starlark.Tuple) (any, error) {
			if err := arity("add_envf", args, kwargs, 2, -1); err != nil { return nil, err }
			key, err := stringAt("add_envf", args, 0); if err != nil { return nil, err }
			format, err := stringAt("add_envf", args, 1); if err != nil { return nil, err }
			values := make([]any, 0, len(args)-2)
			for _, arg := range args[2:] { values = append(values, goScalar(arg)) }
			return buildkitllb.AddEnvf(key, format, values...), nil
		}),
		"del_env": optionBuiltin("del_env", "state_option", func(_ *starlark.Thread, args starlark.Tuple, kwargs []starlark.Tuple) (any, error) {
			if err := arity("del_env", args, kwargs, 1, 1); err != nil { return nil, err }
			key, err := stringAt("del_env", args, 0); if err != nil { return nil, err }
			return buildkitllb.DelEnv(key), nil
		}),
		"dir": optionBuiltin("dir", "state_option", func(_ *starlark.Thread, args starlark.Tuple, kwargs []starlark.Tuple) (any, error) {
			if err := arity("dir", args, kwargs, 1, 1); err != nil { return nil, err }
			dir, err := stringAt("dir", args, 0); if err != nil { return nil, err }
			return buildkitllb.Dir(dir), nil
		}),
		"dirf": optionBuiltin("dirf", "state_option", func(_ *starlark.Thread, args starlark.Tuple, kwargs []starlark.Tuple) (any, error) {
			if err := arity("dirf", args, kwargs, 1, -1); err != nil { return nil, err }
			format, err := stringAt("dirf", args, 0); if err != nil { return nil, err }
			values := make([]any, 0, len(args)-1)
			for _, arg := range args[1:] { values = append(values, goScalar(arg)) }
			return buildkitllb.Dirf(format, values...), nil
		}),
		"user": optionBuiltin("user", "state_option", func(_ *starlark.Thread, args starlark.Tuple, kwargs []starlark.Tuple) (any, error) {
			if err := arity("user", args, kwargs, 1, 1); err != nil { return nil, err }
			user, err := stringAt("user", args, 0); if err != nil { return nil, err }
			return buildkitllb.User(user), nil
		}),
		"reset": optionBuiltin("reset", "state_option", func(_ *starlark.Thread, args starlark.Tuple, kwargs []starlark.Tuple) (any, error) {
			if err := arity("reset", args, kwargs, 1, 1); err != nil { return nil, err }
			state, err := stateFromValue(args[0]); if err != nil { return nil, err }
			return buildkitllb.Reset(state), nil
		}),
		"hostname": optionBuiltin("hostname", "state_option", func(_ *starlark.Thread, args starlark.Tuple, kwargs []starlark.Tuple) (any, error) {
			if err := arity("hostname", args, kwargs, 1, 1); err != nil { return nil, err }
			hostname, err := stringAt("hostname", args, 0); if err != nil { return nil, err }
			return buildkitllb.Hostname(hostname), nil
		}),
		"network": optionBuiltin("network", "state_option", func(_ *starlark.Thread, args starlark.Tuple, kwargs []starlark.Tuple) (any, error) {
			if err := arity("network", args, kwargs, 1, 1); err != nil { return nil, err }
			mode, err := intAt("network", args, 0); if err != nil { return nil, err }
			return buildkitllb.Network(pb.NetMode(mode)), nil
		}),
		"security": optionBuiltin("security", "state_option", func(_ *starlark.Thread, args starlark.Tuple, kwargs []starlark.Tuple) (any, error) {
			if err := arity("security", args, kwargs, 1, 1); err != nil { return nil, err }
			mode, err := intAt("security", args, 0); if err != nil { return nil, err }
			return buildkitllb.Security(pb.SecurityMode(mode)), nil
		}),
		"env_list": builtin("env_list", func(_ *starlark.Thread, _ *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
			if err := arity("env_list", args, kwargs, 0, 0); err != nil { return nil, err }
			return WrapEnvList(&buildkitllb.EnvList{}), nil
		}),
		"linux_resources": builtin("linux_resources", linuxResourcesBuiltin),
		"host_ip": builtin("host_ip", hostIPBuiltin),
	}
}

func linuxResourcesBuiltin(_ *starlark.Thread, _ *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	var resources buildkitllb.LinuxResources
	if err := UnpackArgs("linux_resources", args, kwargs,
		"memory?", &resources.Memory,
		"memory_swap?", &resources.MemorySwap,
		"cpu_shares?", &resources.CPUShares,
		"cpu_period?", &resources.CPUPeriod,
		"cpu_quota?", &resources.CPUQuota,
		"cpuset_cpus?", &resources.CpusetCpus,
		"cpuset_mems?", &resources.CpusetMems,
	); err != nil { return nil, err }
	return WrapLinuxResources(resources), nil
}

func hostIPBuiltin(_ *starlark.Thread, _ *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	if err := arity("host_ip", args, kwargs, 2, 2); err != nil { return nil, err }
	host, err := stringAt("host_ip", args, 0); if err != nil { return nil, err }
	rawIP, err := stringAt("host_ip", args, 1); if err != nil { return nil, err }
	ip := net.ParseIP(rawIP)
	if ip == nil { return nil, fmt.Errorf("host_ip: invalid ip address %q", rawIP) }
	return WrapHostIP(buildkitllb.HostIP{Host: host, IP: ip}), nil
}

func networkModeValues() map[string]starlark.Value {
	return map[string]starlark.Value{
		"net_mode_sandbox": starlark.MakeInt(int(buildkitllb.NetModeSandbox)),
		"net_mode_host":    starlark.MakeInt(int(buildkitllb.NetModeHost)),
		"net_mode_none":    starlark.MakeInt(int(buildkitllb.NetModeNone)),
	}
}

func securityModeValues() map[string]starlark.Value {
	return map[string]starlark.Value{
		"security_mode_insecure": starlark.MakeInt(int(buildkitllb.SecurityModeInsecure)),
		"security_mode_sandbox":  starlark.MakeInt(int(buildkitllb.SecurityModeSandbox)),
	}
}

func (v *StateValue) stateMetaAttrNames() []string {
	names := append([]string(nil), stateMetaAttrNames...)
	sort.Strings(names)
	return SnakeCaseNames(compactStrings(names))
}

var (
	_ starlark.Value    = (*EnvListValue)(nil)
	_ starlark.HasAttrs = (*EnvListValue)(nil)
	_ starlark.Value    = (*HostIPValue)(nil)
	_ starlark.HasAttrs = (*HostIPValue)(nil)
	_ starlark.Value    = (*LinuxResourcesValue)(nil)
	_ starlark.HasAttrs = (*LinuxResourcesValue)(nil)
)


// stateStringMethod adapts a State method that accepts a single string.
func stateStringMethod(name string, fn func(string) buildkitllb.State) starlark.Value {
	return boundBuiltin(name, func(_ *starlark.Thread, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
		if err := arity(name, args, kwargs, 1, 1); err != nil { return nil, err }
		value, err := stringAt(name, args, 0)
		if err != nil { return nil, err }
		return WrapState(fn(value)), nil
	})
}

// stateStringPairMethod adapts a State method that accepts two strings.
func stateStringPairMethod(name string, fn func(string, string) buildkitllb.State) starlark.Value {
	return boundBuiltin(name, func(_ *starlark.Thread, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
		if err := arity(name, args, kwargs, 2, 2); err != nil { return nil, err }
		key, err := stringAt(name, args, 0)
		if err != nil { return nil, err }
		value, err := stringAt(name, args, 1)
		if err != nil { return nil, err }
		return WrapState(fn(key, value)), nil
	})
}


func newOpMetadataBuiltin(_ *starlark.Thread, _ *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	if len(kwargs) != 0 || len(args) > 1 {
		return nil, fmt.Errorf("new_op_metadata: expected zero or one llb.op_metadata_pb argument")
	}
	var native *pb.OpMetadata
	if len(args) == 1 && args[0] != starlark.None {
		value, ok := args[0].(*OpMetadataProtoValue)
		if !ok || value == nil {
			return nil, fmt.Errorf("new_op_metadata: got %s, want llb.op_metadata_pb or none", valueType(args[0]))
		}
		native = value.metadata
	}
	return WrapOpMetadata(buildkitllb.NewOpMetadata(native)), nil
}
