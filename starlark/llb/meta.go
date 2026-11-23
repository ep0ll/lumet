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

func (v *EnvListValue) String() string        { return "<llb.EnvList>" }
func (v *EnvListValue) Type() string          { return "llb.EnvList" }
func (v *EnvListValue) Freeze()               {}
func (v *EnvListValue) Truth() starlark.Bool  { return starlark.Bool(v != nil && v.env != nil) }
func (v *EnvListValue) Hash() (uint32, error) { return 0, fmt.Errorf("unhashable: llb.EnvList") }

func (v *EnvListValue) Attr(name string) (starlark.Value, error) {
	if v == nil || v.env == nil {
		return nil, fmt.Errorf("llb.EnvList: nil receiver")
	}
	switch name {
	case "AddOrReplace", "add_or_replace":
		return boundBuiltin("EnvList.AddOrReplace", func(_ *starlark.Thread, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
			if err := arity("EnvList.AddOrReplace", args, kwargs, 2, 2); err != nil { return nil, err }
			key, err := stringAt("EnvList.AddOrReplace", args, 0); if err != nil { return nil, err }
			value, err := stringAt("EnvList.AddOrReplace", args, 1); if err != nil { return nil, err }
			return WrapEnvList(v.env.AddOrReplace(key, value)), nil
		}), nil
	case "SetDefault", "set_default":
		return boundBuiltin("EnvList.SetDefault", func(_ *starlark.Thread, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
			if err := arity("EnvList.SetDefault", args, kwargs, 2, 2); err != nil { return nil, err }
			key, err := stringAt("EnvList.SetDefault", args, 0); if err != nil { return nil, err }
			value, err := stringAt("EnvList.SetDefault", args, 1); if err != nil { return nil, err }
			return WrapEnvList(v.env.SetDefault(key, value)), nil
		}), nil
	case "Delete", "delete":
		return boundBuiltin("EnvList.Delete", func(_ *starlark.Thread, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
			if err := arity("EnvList.Delete", args, kwargs, 1, 1); err != nil { return nil, err }
			key, err := stringAt("EnvList.Delete", args, 0); if err != nil { return nil, err }
			env := v.env.Delete(key)
			return WrapEnvList(&env), nil
		}), nil
	case "Get", "get":
		return boundBuiltin("EnvList.Get", func(_ *starlark.Thread, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
			if err := arity("EnvList.Get", args, kwargs, 1, 1); err != nil { return nil, err }
			key, err := stringAt("EnvList.Get", args, 0); if err != nil { return nil, err }
			value, ok := v.env.Get(key)
			return starlark.Tuple{starlark.String(value), starlark.Bool(ok)}, nil
		}), nil
	case "Keys", "keys":
		return boundBuiltin("EnvList.Keys", func(_ *starlark.Thread, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
			if err := arity("EnvList.Keys", args, kwargs, 0, 0); err != nil { return nil, err }
			return stringSliceValue(v.env.Keys()), nil
		}), nil
	case "ToArray", "to_array":
		return boundBuiltin("EnvList.ToArray", func(_ *starlark.Thread, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
			if err := arity("EnvList.ToArray", args, kwargs, 0, 0); err != nil { return nil, err }
			return stringSliceValue(v.env.ToArray()), nil
		}), nil
	default:
		return nil, nil
	}
}

func (v *EnvListValue) AttrNames() []string {
	return []string{"AddOrReplace", "Delete", "Get", "Keys", "SetDefault", "ToArray",
		"add_or_replace", "delete", "get", "keys", "set_default", "to_array"}
}

// HostIPValue adapts BuildKit's host-to-IP mapping.
type HostIPValue struct {
	hostIP buildkitllb.HostIP
}

// WrapHostIP adapts a native HostIP value.
func WrapHostIP(hostIP buildkitllb.HostIP) *HostIPValue {
	return &HostIPValue{hostIP: hostIP}
}

func (v *HostIPValue) String() string        { return "<llb.HostIP>" }
func (v *HostIPValue) Type() string          { return "llb.HostIP" }
func (v *HostIPValue) Freeze()               {}
func (v *HostIPValue) Truth() starlark.Bool  { return starlark.Bool(v != nil) }
func (v *HostIPValue) Hash() (uint32, error) { return 0, fmt.Errorf("unhashable: llb.HostIP") }

func (v *HostIPValue) Attr(name string) (starlark.Value, error) {
	if v == nil { return nil, fmt.Errorf("llb.HostIP: nil receiver") }
	switch name {
	case "host", "Host":
		return starlark.String(v.hostIP.Host), nil
	case "ip", "IP":
		return starlark.String(v.hostIP.IP.String()), nil
	default:
		return nil, nil
	}
}

func (v *HostIPValue) AttrNames() []string { return []string{"host", "ip", "Host", "IP"} }

// LinuxResourcesValue adapts BuildKit's container resource limits.
type LinuxResourcesValue struct {
	resources buildkitllb.LinuxResources
}

// WrapLinuxResources adapts native LinuxResources.
func WrapLinuxResources(resources buildkitllb.LinuxResources) *LinuxResourcesValue {
	return &LinuxResourcesValue{resources: resources}
}

func (v *LinuxResourcesValue) String() string        { return "<llb.LinuxResources>" }
func (v *LinuxResourcesValue) Type() string          { return "llb.LinuxResources" }
func (v *LinuxResourcesValue) Freeze()               {}
func (v *LinuxResourcesValue) Truth() starlark.Bool  { return starlark.Bool(v != nil) }
func (v *LinuxResourcesValue) Hash() (uint32, error) { return 0, fmt.Errorf("unhashable: llb.LinuxResources") }

func (v *LinuxResourcesValue) Attr(name string) (starlark.Value, error) {
	if v == nil { return nil, fmt.Errorf("llb.LinuxResources: nil receiver") }
	switch name {
	case "memory", "Memory":
		return starlark.MakeInt64(v.resources.Memory), nil
	case "memory_swap", "MemorySwap":
		return starlark.MakeInt64(v.resources.MemorySwap), nil
	case "cpu_shares", "CPUShares":
		return starlark.MakeUint64(v.resources.CPUShares), nil
	case "cpu_period", "CPUPeriod":
		return starlark.MakeUint64(v.resources.CPUPeriod), nil
	case "cpu_quota", "CPUQuota":
		return starlark.MakeInt64(v.resources.CPUQuota), nil
	case "cpuset_cpus", "CpusetCpus":
		return starlark.String(v.resources.CpusetCpus), nil
	case "cpuset_mems", "CpusetMems":
		return starlark.String(v.resources.CpusetMems), nil
	default:
		return nil, nil
	}
}

func (v *LinuxResourcesValue) AttrNames() []string {
	return []string{"memory", "memory_swap", "cpu_shares", "cpu_period", "cpu_quota", "cpuset_cpus", "cpuset_mems",
		"Memory", "MemorySwap", "CPUShares", "CPUPeriod", "CPUQuota", "CpusetCpus", "CpusetMems"}
}

func (v *StateValue) stateMetaAttr(name string) (starlark.Value, bool, error) {
	state := v.state
	switch name {
	case "AddEnv", "add_env":
		return stateStringPairMethod("State.AddEnv", state.AddEnv), true, nil
	case "AddEnvf", "add_envf":
		return boundBuiltin("State.AddEnvf", func(_ *starlark.Thread, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
			if err := arity("State.AddEnvf", args, kwargs, 2, -1); err != nil { return nil, err }
			key, err := stringAt("State.AddEnvf", args, 0); if err != nil { return nil, err }
			format, err := stringAt("State.AddEnvf", args, 1); if err != nil { return nil, err }
			values := make([]any, 0, len(args)-2)
			for _, arg := range args[2:] { values = append(values, goScalar(arg)) }
			return WrapState(state.AddEnvf(key, format, values...)), nil
		}), true, nil
	case "DelEnv", "del_env":
		return stateStringMethod("State.DelEnv", state.DelEnv), true, nil
	case "Dir", "dir":
		return stateStringMethod("State.Dir", state.Dir), true, nil
	case "Dirf", "dirf":
		return boundBuiltin("State.Dirf", func(_ *starlark.Thread, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
			if err := arity("State.Dirf", args, kwargs, 1, -1); err != nil { return nil, err }
			format, err := stringAt("State.Dirf", args, 0); if err != nil { return nil, err }
			values := make([]any, 0, len(args)-1)
			for _, arg := range args[1:] { values = append(values, goScalar(arg)) }
			return WrapState(state.Dirf(format, values...)), nil
		}), true, nil
	case "User", "user":
		return stateStringMethod("State.User", state.User), true, nil
	case "Hostname", "hostname":
		return stateStringMethod("State.Hostname", state.Hostname), true, nil
	case "WithCgroupParent", "with_cgroup_parent":
		return stateStringMethod("State.WithCgroupParent", state.WithCgroupParent), true, nil
	case "With", "with":
		return boundBuiltin("State.With", func(_ *starlark.Thread, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
			if err := arity("State.With", args, kwargs, 0, -1); err != nil { return nil, err }
			options, err := optionsFromArgs[buildkitllb.StateOption](args, 0, "StateOption")
			if err != nil { return nil, err }
			return WrapState(state.With(options...)), nil
		}), true, nil
	case "Reset", "reset":
		return boundBuiltin("State.Reset", func(_ *starlark.Thread, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
			if err := arity("State.Reset", args, kwargs, 1, 1); err != nil { return nil, err }
			other, err := stateFromValue(args[0])
			if err != nil { return nil, err }
			return WrapState(state.Reset(other)), nil
		}), true, nil
	case "Network", "network":
		return boundBuiltin("State.Network", func(_ *starlark.Thread, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
			if err := arity("State.Network", args, kwargs, 1, 1); err != nil { return nil, err }
			mode, err := intAt("State.Network", args, 0)
			if err != nil { return nil, err }
			return WrapState(state.Network(pb.NetMode(mode))), nil
		}), true, nil
	case "Security", "security":
		return boundBuiltin("State.Security", func(_ *starlark.Thread, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
			if err := arity("State.Security", args, kwargs, 1, 1); err != nil { return nil, err }
			mode, err := intAt("State.Security", args, 0)
			if err != nil { return nil, err }
			return WrapState(state.Security(pb.SecurityMode(mode))), nil
		}), true, nil
	case "Platform", "platform":
		return boundBuiltin("State.Platform", func(_ *starlark.Thread, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
			if err := arity("State.Platform", args, kwargs, 1, 1); err != nil { return nil, err }
			platform, ok := args[0].(*OCIPlatformValue)
			if !ok || platform == nil { return nil, fmt.Errorf("State.Platform: got %s, want llb.OCIPlatform", valueType(args[0])) }
			return WrapState(state.Platform(platform.platform)), nil
		}), true, nil
	case "AddExtraHost", "add_extra_host":
		return boundBuiltin("State.AddExtraHost", func(_ *starlark.Thread, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
			if err := arity("State.AddExtraHost", args, kwargs, 2, 2); err != nil { return nil, err }
			host, err := stringAt("State.AddExtraHost", args, 0); if err != nil { return nil, err }
			rawIP, err := stringAt("State.AddExtraHost", args, 1); if err != nil { return nil, err }
			ip := net.ParseIP(rawIP)
			if ip == nil { return nil, fmt.Errorf("State.AddExtraHost: invalid IP address %q", rawIP) }
			return WrapState(state.AddExtraHost(host, ip)), nil
		}), true, nil
	case "Requires", "requires":
		return boundBuiltin("State.Requires", func(_ *starlark.Thread, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
			if err := arity("State.Requires", args, kwargs, 1, -1); err != nil { return nil, err }
			id, err := stringAt("State.Requires", args, 0)
			if err != nil { return nil, err }
			dependencies := make([]buildkitllb.State, 0, len(args)-1)
			for i := 1; i < len(args); i++ {
				dependency, err := stateFromValue(args[i])
				if err != nil { return nil, fmt.Errorf("State.Requires dependency %d: %w", i, err) }
				dependencies = append(dependencies, dependency)
			}
			return WrapState(state.Requires(id, dependencies...)), nil
		}), true, nil
	case "AddUlimit", "add_ulimit":
		return boundBuiltin("State.AddUlimit", func(_ *starlark.Thread, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
			if err := arity("State.AddUlimit", args, kwargs, 3, 3); err != nil { return nil, err }
			name, err := stringAt("State.AddUlimit", args, 0); if err != nil { return nil, err }
			soft, err := int64At("State.AddUlimit", args, 1); if err != nil { return nil, err }
			hard, err := int64At("State.AddUlimit", args, 2); if err != nil { return nil, err }
			return WrapState(state.AddUlimit(buildkitllb.UlimitName(name), soft, hard)), nil
		}), true, nil
	case "GetEnv", "get_env":
		return boundBuiltin("State.GetEnv", func(thread *starlark.Thread, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
			if err := arity("State.GetEnv", args, kwargs, 1, -1); err != nil { return nil, err }
			key, err := stringAt("State.GetEnv", args, 0); if err != nil { return nil, err }
			options, err := optionsFromArgs[buildkitllb.ConstraintsOpt](args, 1, "ConstraintsOpt")
			if err != nil { return nil, err }
			value, ok, err := state.GetEnv(callContext(thread), key, options...)
			if err != nil { return nil, err }
			return starlark.Tuple{starlark.String(value), starlark.Bool(ok)}, nil
		}), true, nil
	case "Env", "env":
		return boundBuiltin("State.Env", func(thread *starlark.Thread, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
			if err := arity("State.Env", args, kwargs, 0, -1); err != nil { return nil, err }
			options, err := optionsFromArgs[buildkitllb.ConstraintsOpt](args, 0, "ConstraintsOpt")
			if err != nil { return nil, err }
			env, err := state.Env(callContext(thread), options...)
			if err != nil { return nil, err }
			return WrapEnvList(env), nil
		}), true, nil
	case "GetDir", "get_dir":
		return boundBuiltin("State.GetDir", func(thread *starlark.Thread, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
			if err := arity("State.GetDir", args, kwargs, 0, -1); err != nil { return nil, err }
			options, err := optionsFromArgs[buildkitllb.ConstraintsOpt](args, 0, "ConstraintsOpt")
			if err != nil { return nil, err }
			value, err := state.GetDir(callContext(thread), options...)
			if err != nil { return nil, err }
			return starlark.String(value), nil
		}), true, nil
	case "GetArgs", "get_args":
		return boundBuiltin("State.GetArgs", func(thread *starlark.Thread, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
			if err := arity("State.GetArgs", args, kwargs, 0, -1); err != nil { return nil, err }
			options, err := optionsFromArgs[buildkitllb.ConstraintsOpt](args, 0, "ConstraintsOpt")
			if err != nil { return nil, err }
			values, err := state.GetArgs(callContext(thread), options...)
			if err != nil { return nil, err }
			return stringSliceValue(values), nil
		}), true, nil
	case "GetHostname", "get_hostname":
		return boundBuiltin("State.GetHostname", func(thread *starlark.Thread, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
			if err := arity("State.GetHostname", args, kwargs, 0, -1); err != nil { return nil, err }
			options, err := optionsFromArgs[buildkitllb.ConstraintsOpt](args, 0, "ConstraintsOpt")
			if err != nil { return nil, err }
			value, err := state.GetHostname(callContext(thread), options...)
			if err != nil { return nil, err }
			return starlark.String(value), nil
		}), true, nil
	case "GetPlatform", "get_platform":
		return boundBuiltin("State.GetPlatform", func(thread *starlark.Thread, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
			if err := arity("State.GetPlatform", args, kwargs, 0, -1); err != nil { return nil, err }
			options, err := optionsFromArgs[buildkitllb.ConstraintsOpt](args, 0, "ConstraintsOpt")
			if err != nil { return nil, err }
			value, err := state.GetPlatform(callContext(thread), options...)
			if err != nil { return nil, err }
			if value == nil { return starlark.None, nil }
			return WrapOCIPlatform(*value), nil
		}), true, nil
	case "GetNetwork", "get_network":
		return boundBuiltin("State.GetNetwork", func(thread *starlark.Thread, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
			if err := arity("State.GetNetwork", args, kwargs, 0, -1); err != nil { return nil, err }
			options, err := optionsFromArgs[buildkitllb.ConstraintsOpt](args, 0, "ConstraintsOpt")
			if err != nil { return nil, err }
			value, err := state.GetNetwork(callContext(thread), options...)
			if err != nil { return nil, err }
			return starlark.MakeInt(int(value)), nil
		}), true, nil
	case "GetSecurity", "get_security":
		return boundBuiltin("State.GetSecurity", func(thread *starlark.Thread, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
			if err := arity("State.GetSecurity", args, kwargs, 0, -1); err != nil { return nil, err }
			options, err := optionsFromArgs[buildkitllb.ConstraintsOpt](args, 0, "ConstraintsOpt")
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
	"AddEnv", "AddEnvf", "AddExtraHost", "AddUlimit", "DelEnv", "Dir", "Dirf",
	"Env", "GetArgs", "GetDir", "GetEnv", "GetHostname", "GetNetwork", "GetPlatform", "GetSecurity",
	"Hostname", "Network", "Platform", "Reset", "Requires", "Security", "User", "With", "WithCgroupParent",
	"add_env", "add_envf", "add_extra_host", "add_ulimit", "del_env", "dir", "dirf", "env",
	"get_args", "get_dir", "get_env", "get_hostname", "get_network", "get_platform", "get_security",
	"hostname", "network", "platform", "reset", "requires", "security", "user", "with", "with_cgroup_parent",
}

func stateMetaAttr(v *StateValue, name string) (starlark.Value, bool, error) {
	return v.stateMetaAttr(name)
}

func metaBuiltins() map[string]starlark.Value {
	return map[string]starlark.Value{
		"AddEnv": optionBuiltin("AddEnv", "StateOption", func(_ *starlark.Thread, args starlark.Tuple, kwargs []starlark.Tuple) (any, error) {
			if err := arity("AddEnv", args, kwargs, 2, 2); err != nil { return nil, err }
			key, err := stringAt("AddEnv", args, 0); if err != nil { return nil, err }
			value, err := stringAt("AddEnv", args, 1); if err != nil { return nil, err }
			return buildkitllb.AddEnv(key, value), nil
		}),
		"AddEnvf": optionBuiltin("AddEnvf", "StateOption", func(_ *starlark.Thread, args starlark.Tuple, kwargs []starlark.Tuple) (any, error) {
			if err := arity("AddEnvf", args, kwargs, 2, -1); err != nil { return nil, err }
			key, err := stringAt("AddEnvf", args, 0); if err != nil { return nil, err }
			format, err := stringAt("AddEnvf", args, 1); if err != nil { return nil, err }
			values := make([]any, 0, len(args)-2)
			for _, arg := range args[2:] { values = append(values, goScalar(arg)) }
			return buildkitllb.AddEnvf(key, format, values...), nil
		}),
		"DelEnv": optionBuiltin("DelEnv", "StateOption", func(_ *starlark.Thread, args starlark.Tuple, kwargs []starlark.Tuple) (any, error) {
			if err := arity("DelEnv", args, kwargs, 1, 1); err != nil { return nil, err }
			key, err := stringAt("DelEnv", args, 0); if err != nil { return nil, err }
			return buildkitllb.DelEnv(key), nil
		}),
		"Dir": optionBuiltin("Dir", "StateOption", func(_ *starlark.Thread, args starlark.Tuple, kwargs []starlark.Tuple) (any, error) {
			if err := arity("Dir", args, kwargs, 1, 1); err != nil { return nil, err }
			dir, err := stringAt("Dir", args, 0); if err != nil { return nil, err }
			return buildkitllb.Dir(dir), nil
		}),
		"Dirf": optionBuiltin("Dirf", "StateOption", func(_ *starlark.Thread, args starlark.Tuple, kwargs []starlark.Tuple) (any, error) {
			if err := arity("Dirf", args, kwargs, 1, -1); err != nil { return nil, err }
			format, err := stringAt("Dirf", args, 0); if err != nil { return nil, err }
			values := make([]any, 0, len(args)-1)
			for _, arg := range args[1:] { values = append(values, goScalar(arg)) }
			return buildkitllb.Dirf(format, values...), nil
		}),
		"User": optionBuiltin("User", "StateOption", func(_ *starlark.Thread, args starlark.Tuple, kwargs []starlark.Tuple) (any, error) {
			if err := arity("User", args, kwargs, 1, 1); err != nil { return nil, err }
			user, err := stringAt("User", args, 0); if err != nil { return nil, err }
			return buildkitllb.User(user), nil
		}),
		"Reset": optionBuiltin("Reset", "StateOption", func(_ *starlark.Thread, args starlark.Tuple, kwargs []starlark.Tuple) (any, error) {
			if err := arity("Reset", args, kwargs, 1, 1); err != nil { return nil, err }
			state, err := stateFromValue(args[0]); if err != nil { return nil, err }
			return buildkitllb.Reset(state), nil
		}),
		"Hostname": optionBuiltin("Hostname", "StateOption", func(_ *starlark.Thread, args starlark.Tuple, kwargs []starlark.Tuple) (any, error) {
			if err := arity("Hostname", args, kwargs, 1, 1); err != nil { return nil, err }
			hostname, err := stringAt("Hostname", args, 0); if err != nil { return nil, err }
			return buildkitllb.Hostname(hostname), nil
		}),
		"Network": optionBuiltin("Network", "StateOption", func(_ *starlark.Thread, args starlark.Tuple, kwargs []starlark.Tuple) (any, error) {
			if err := arity("Network", args, kwargs, 1, 1); err != nil { return nil, err }
			mode, err := intAt("Network", args, 0); if err != nil { return nil, err }
			return buildkitllb.Network(pb.NetMode(mode)), nil
		}),
		"Security": optionBuiltin("Security", "StateOption", func(_ *starlark.Thread, args starlark.Tuple, kwargs []starlark.Tuple) (any, error) {
			if err := arity("Security", args, kwargs, 1, 1); err != nil { return nil, err }
			mode, err := intAt("Security", args, 0); if err != nil { return nil, err }
			return buildkitllb.Security(pb.SecurityMode(mode)), nil
		}),
		"EnvList": builtin("EnvList", func(_ *starlark.Thread, _ *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
			if err := arity("EnvList", args, kwargs, 0, 0); err != nil { return nil, err }
			return WrapEnvList(&buildkitllb.EnvList{}), nil
		}),
		"LinuxResources": builtin("LinuxResources", linuxResourcesBuiltin),
		"HostIP": builtin("HostIP", hostIPBuiltin),
	}
}

func linuxResourcesBuiltin(_ *starlark.Thread, _ *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	var resources buildkitllb.LinuxResources
	if err := starlark.UnpackArgs("LinuxResources", args, kwargs,
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
	if err := arity("HostIP", args, kwargs, 2, 2); err != nil { return nil, err }
	host, err := stringAt("HostIP", args, 0); if err != nil { return nil, err }
	rawIP, err := stringAt("HostIP", args, 1); if err != nil { return nil, err }
	ip := net.ParseIP(rawIP)
	if ip == nil { return nil, fmt.Errorf("HostIP: invalid IP address %q", rawIP) }
	return WrapHostIP(buildkitllb.HostIP{Host: host, IP: ip}), nil
}

func networkModeValues() map[string]starlark.Value {
	return map[string]starlark.Value{
		"NetModeSandbox": starlark.MakeInt(int(buildkitllb.NetModeSandbox)),
		"NetModeHost":    starlark.MakeInt(int(buildkitllb.NetModeHost)),
		"NetModeNone":    starlark.MakeInt(int(buildkitllb.NetModeNone)),
	}
}

func securityModeValues() map[string]starlark.Value {
	return map[string]starlark.Value{
		"SecurityModeInsecure": starlark.MakeInt(int(buildkitllb.SecurityModeInsecure)),
		"SecurityModeSandbox":  starlark.MakeInt(int(buildkitllb.SecurityModeSandbox)),
	}
}

func (v *StateValue) stateMetaAttrNames() []string {
	names := append([]string(nil), stateMetaAttrNames...)
	sort.Strings(names)
	return compactStrings(names)
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
		return nil, fmt.Errorf("NewOpMetadata: expected zero or one llb.OpMetadataPB argument")
	}
	var native *pb.OpMetadata
	if len(args) == 1 && args[0] != starlark.None {
		value, ok := args[0].(*OpMetadataProtoValue)
		if !ok || value == nil {
			return nil, fmt.Errorf("NewOpMetadata: got %s, want llb.OpMetadataPB or None", valueType(args[0]))
		}
		native = value.metadata
	}
	return WrapOpMetadata(buildkitllb.NewOpMetadata(native)), nil
}
