package llb

import (
	"fmt"
	"net"
	"sort"

	buildkitllb "github.com/moby/buildkit/client/llb"
	"github.com/moby/buildkit/solver/pb"
	ocispecs "github.com/opencontainers/image-spec/specs-go/v1"
	"go.starlark.net/starlark"
)

func sourceBuiltins() map[string]starlark.Value {
	return map[string]starlark.Value{
		"Scratch": builtin("Scratch", scratchBuiltin),
		"Image": builtin("Image", imageBuiltin),
		"ImageBlob": builtin("ImageBlob", imageBlobBuiltin),
		"Git": builtin("Git", gitBuiltin),
		"Local": builtin("Local", localBuiltin),
		"HTTP": builtin("HTTP", httpBuiltin),
		"OCILayout": builtin("OCILayout", ociLayoutBuiltin),
		"OCILayoutBlob": builtin("OCILayoutBlob", ociLayoutBlobBuiltin),
		"Merge": builtin("Merge", mergeBuiltin),
		"Diff": builtin("Diff", diffBuiltin),
		"OCIPlatform": builtin("OCIPlatform", ociPlatformBuiltin),
	}
}

func scratchBuiltin(_ *starlark.Thread, _ *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	if err := arity("Scratch", args, kwargs, 0, 0); err != nil {
		return nil, err
	}
	return WrapState(buildkitllb.Scratch()), nil
}

func imageBuiltin(_ *starlark.Thread, _ *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	if err := arity("Image", args, kwargs, 1, -1); err != nil {
		return nil, err
	}
	ref, err := stringAt("Image", args, 0)
	if err != nil {
		return nil, err
	}
	opts, err := optionsFromArgs[buildkitllb.ImageOption](args, 1, "ImageOption")
	if err != nil {
		return nil, err
	}
	return WrapState(buildkitllb.Image(ref, opts...)), nil
}

func imageBlobBuiltin(_ *starlark.Thread, _ *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	if err := arity("ImageBlob", args, kwargs, 1, -1); err != nil {
		return nil, err
	}
	ref, err := stringAt("ImageBlob", args, 0)
	if err != nil {
		return nil, err
	}
	opts, err := optionsFromArgs[buildkitllb.ImageBlobOption](args, 1, "ImageBlobOption")
	if err != nil {
		return nil, err
	}
	return WrapState(buildkitllb.ImageBlob(ref, opts...)), nil
}

func gitBuiltin(_ *starlark.Thread, _ *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	if err := arity("Git", args, kwargs, 1, -1); err != nil {
		return nil, err
	}
	url, err := stringAt("Git", args, 0)
	if err != nil {
		return nil, err
	}
	fragment := ""
	start := 1
	if len(args) > 1 {
		if _, ok := args[1].(starlark.String); ok {
			fragment = string(args[1].(starlark.String))
			start = 2
		}
	}
	opts, err := optionsFromArgs[buildkitllb.GitOption](args, start, "GitOption")
	if err != nil {
		return nil, err
	}
	return WrapState(buildkitllb.Git(url, fragment, opts...)), nil
}

func localBuiltin(_ *starlark.Thread, _ *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	if err := arity("Local", args, kwargs, 1, -1); err != nil {
		return nil, err
	}
	name, err := stringAt("Local", args, 0)
	if err != nil {
		return nil, err
	}
	opts, err := optionsFromArgs[buildkitllb.LocalOption](args, 1, "LocalOption")
	if err != nil {
		return nil, err
	}
	return WrapState(buildkitllb.Local(name, opts...)), nil
}

func httpBuiltin(_ *starlark.Thread, _ *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	if err := arity("HTTP", args, kwargs, 1, -1); err != nil {
		return nil, err
	}
	url, err := stringAt("HTTP", args, 0)
	if err != nil {
		return nil, err
	}
	opts, err := optionsFromArgs[buildkitllb.HTTPOption](args, 1, "HTTPOption")
	if err != nil {
		return nil, err
	}
	return WrapState(buildkitllb.HTTP(url, opts...)), nil
}

func ociLayoutBuiltin(_ *starlark.Thread, _ *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	if err := arity("OCILayout", args, kwargs, 1, -1); err != nil {
		return nil, err
	}
	ref, err := stringAt("OCILayout", args, 0)
	if err != nil {
		return nil, err
	}
	opts, err := optionsFromArgs[buildkitllb.OCILayoutOption](args, 1, "OCILayoutOption")
	if err != nil {
		return nil, err
	}
	return WrapState(buildkitllb.OCILayout(ref, opts...)), nil
}

func ociLayoutBlobBuiltin(_ *starlark.Thread, _ *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	if err := arity("OCILayoutBlob", args, kwargs, 1, -1); err != nil {
		return nil, err
	}
	ref, err := stringAt("OCILayoutBlob", args, 0)
	if err != nil {
		return nil, err
	}
	opts, err := optionsFromArgs[buildkitllb.ImageBlobOption](args, 1, "ImageBlobOption")
	if err != nil {
		return nil, err
	}
	return WrapState(buildkitllb.OCILayoutBlob(ref, opts...)), nil
}

func mergeBuiltin(_ *starlark.Thread, _ *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	if err := arity("Merge", args, kwargs, 1, -1); err != nil {
		return nil, err
	}
	values, ok := args[0].(starlark.Indexable)
	if !ok {
		return nil, fmt.Errorf("Merge: first argument must be a list or tuple of llb.State")
	}
	states := make([]buildkitllb.State, values.Len())
	for i := range values.Len() {
		state, err := stateFromValue(values.Index(i))
		if err != nil {
			return nil, fmt.Errorf("Merge: state %d: %w", i, err)
		}
		states[i] = state
	}
	opts, err := optionsFromArgs[buildkitllb.ConstraintsOpt](args, 1, "ConstraintsOpt")
	if err != nil {
		return nil, err
	}
	return WrapState(buildkitllb.Merge(states, opts...)), nil
}

func diffBuiltin(_ *starlark.Thread, _ *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	if err := arity("Diff", args, kwargs, 2, -1); err != nil {
		return nil, err
	}
	lower, err := stateFromValue(args[0])
	if err != nil {
		return nil, fmt.Errorf("Diff: lower: %w", err)
	}
	upper, err := stateFromValue(args[1])
	if err != nil {
		return nil, fmt.Errorf("Diff: upper: %w", err)
	}
	opts, err := optionsFromArgs[buildkitllb.ConstraintsOpt](args, 2, "ConstraintsOpt")
	if err != nil {
		return nil, err
	}
	return WrapState(buildkitllb.Diff(lower, upper, opts...)), nil
}

func ociPlatformBuiltin(_ *starlark.Thread, _ *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	var osName, architecture, variant, osVersion string
	if err := starlark.UnpackArgs("OCIPlatform", args, kwargs,
		"os", &osName,
		"architecture", &architecture,
		"variant?", &variant,
		"os_version?", &osVersion,
	); err != nil {
		return nil, err
	}
	return &platformValue{platform: ocispecs.Platform{
		OS: osName, Architecture: architecture, Variant: variant, OSVersion: osVersion,
	}}, nil
}

// platformValue is a value object for OCI platform metadata.
type platformValue struct{ platform ocispecs.Platform }

func (v *platformValue) String() string        { return fmt.Sprintf("<platform %s/%s>", v.platform.OS, v.platform.Architecture) }
func (v *platformValue) Type() string          { return "llb.Platform" }
func (v *platformValue) Freeze()               {}
func (v *platformValue) Truth() starlark.Bool  { return true }
func (v *platformValue) Hash() (uint32, error) { return 0, fmt.Errorf("unhashable: llb.Platform") }
func (v *platformValue) Attr(name string) (starlark.Value, error) {
	switch name {
	case "os":
		return starlark.String(v.platform.OS), nil
	case "architecture":
		return starlark.String(v.platform.Architecture), nil
	case "variant":
		return starlark.String(v.platform.Variant), nil
	case "os_version":
		return starlark.String(v.platform.OSVersion), nil
	default:
		return nil, nil
	}
}
func (v *platformValue) AttrNames() []string {
	return []string{"architecture", "os", "os_version", "variant"}
}

func constantValues() map[string]starlark.Value {
	return map[string]starlark.Value{
		"NetModeSandbox": starlark.MakeInt(int(buildkitllb.NetModeSandbox)),
		"NetModeHost": starlark.MakeInt(int(buildkitllb.NetModeHost)),
		"NetModeNone": starlark.MakeInt(int(buildkitllb.NetModeNone)),
		"SecurityModeInsecure": starlark.MakeInt(int(buildkitllb.SecurityModeInsecure)),
		"SecurityModeSandbox": starlark.MakeInt(int(buildkitllb.SecurityModeSandbox)),
		"CacheMountShared": starlark.MakeInt(int(buildkitllb.CacheMountShared)),
		"CacheMountPrivate": starlark.MakeInt(int(buildkitllb.CacheMountPrivate)),
		"CacheMountLocked": starlark.MakeInt(int(buildkitllb.CacheMountLocked)),
		"ResolveModeDefault": &optionValue{kind: "ResolveMode", value: buildkitllb.ResolveModeDefault},
		"ResolveModeForcePull": &optionValue{kind: "ResolveMode", value: buildkitllb.ResolveModeForcePull},
		"ResolveModePreferLocal": &optionValue{kind: "ResolveMode", value: buildkitllb.ResolveModePreferLocal},
		"MarkImageInternal": &optionValue{kind: "ImageOption", value: buildkitllb.MarkImageInternal},
		"IgnoreCache": &optionValue{kind: "ConstraintsOpt", value: buildkitllb.IgnoreCache},
		"GitAuthHeaderKey": starlark.String(buildkitllb.GitAuthHeaderKey),
		"GitAuthTokenKey": starlark.String(buildkitllb.GitAuthTokenKey),
		"DiffNone": starlark.String(string(buildkitllb.DiffNone)),
		"DiffMetadata": starlark.String(string(buildkitllb.DiffMetadata)),
		"LinuxAmd64": &optionValue{kind: "ConstraintsOpt", value: buildkitllb.LinuxAmd64},
		"LinuxArm64": &optionValue{kind: "ConstraintsOpt", value: buildkitllb.LinuxArm64},
		"LinuxArm": &optionValue{kind: "ConstraintsOpt", value: buildkitllb.LinuxArm},
		"LinuxS390x": &optionValue{kind: "ConstraintsOpt", value: buildkitllb.LinuxS390x},
		"LinuxPpc64le": &optionValue{kind: "ConstraintsOpt", value: buildkitllb.LinuxPpc64le},
		"LinuxArmhf": &optionValue{kind: "ConstraintsOpt", value: buildkitllb.LinuxArmhf},
		"LinuxArmel": &optionValue{kind: "ConstraintsOpt", value: buildkitllb.LinuxArmel},
		"LinuxPpc64": &optionValue{kind: "ConstraintsOpt", value: buildkitllb.LinuxPpc64},
		"LinuxRiscv64": &optionValue{kind: "ConstraintsOpt", value: buildkitllb.LinuxRiscv64},
		"Darwin": &optionValue{kind: "ConstraintsOpt", value: buildkitllb.Darwin},
		"Windows": &optionValue{kind: "ConstraintsOpt", value: buildkitllb.Windows},
		"CDIDeviceOptional": &optionValue{kind: "CDIDeviceOption", value: buildkitllb.CDIDeviceOptional},
		"SSHOptional": &optionValue{kind: "SSHOption", value: buildkitllb.SSHOptional},
		"SecretOptional": &optionValue{kind: "SecretOption", value: buildkitllb.SecretOptional},
		"MountContentCacheDefault": starlark.MakeInt(int(buildkitllb.MountContentCacheDefault)),
		"MountContentCacheOn": starlark.MakeInt(int(buildkitllb.MountContentCacheOn)),
		"MountContentCacheOff": starlark.MakeInt(int(buildkitllb.MountContentCacheOff)),
		"UlimitCore": starlark.String(buildkitllb.UlimitCore),
		"UlimitCPU": starlark.String(buildkitllb.UlimitCPU),
		"UlimitData": starlark.String(buildkitllb.UlimitData),
		"UlimitFsize": starlark.String(buildkitllb.UlimitFsize),
		"UlimitLocks": starlark.String(buildkitllb.UlimitLocks),
		"UlimitMemlock": starlark.String(buildkitllb.UlimitMemlock),
		"UlimitMsgqueue": starlark.String(buildkitllb.UlimitMsgqueue),
		"UlimitNice": starlark.String(buildkitllb.UlimitNice),
		"UlimitNofile": starlark.String(buildkitllb.UlimitNofile),
		"UlimitNproc": starlark.String(buildkitllb.UlimitNproc),
		"UlimitRss": starlark.String(buildkitllb.UlimitRss),
		"UlimitRtprio": starlark.String(buildkitllb.UlimitRtprio),
		"UlimitRttime": starlark.String(buildkitllb.UlimitRttime),
		"UlimitSigpending": starlark.String(buildkitllb.UlimitSigpending),
		"UlimitStack": starlark.String(buildkitllb.UlimitStack),
	}
}


type proxyEnvValue struct{ proxy buildkitllb.ProxyEnv }

func (v *proxyEnvValue) String() string        { return "<llb.ProxyEnv>" }
func (v *proxyEnvValue) Type() string          { return "llb.ProxyEnv" }
func (v *proxyEnvValue) Freeze()               {}
func (v *proxyEnvValue) Truth() starlark.Bool  { return true }
func (v *proxyEnvValue) Hash() (uint32, error) { return 0, fmt.Errorf("unhashable: llb.ProxyEnv") }
func (v *proxyEnvValue) Attr(name string) (starlark.Value, error) {
	switch name {
	case "HTTPProxy", "http_proxy": return starlark.String(v.proxy.HTTPProxy), nil
	case "HTTPSProxy", "https_proxy": return starlark.String(v.proxy.HTTPSProxy), nil
	case "FTPProxy", "ftp_proxy": return starlark.String(v.proxy.FTPProxy), nil
	case "NoProxy", "no_proxy": return starlark.String(v.proxy.NoProxy), nil
	case "AllProxy", "all_proxy": return starlark.String(v.proxy.AllProxy), nil
	default: return nil, nil
	}
}
func (v *proxyEnvValue) AttrNames() []string {
	return []string{"AllProxy", "FTPProxy", "HTTPProxy", "HTTPSProxy", "NoProxy", "all_proxy", "ftp_proxy", "http_proxy", "https_proxy", "no_proxy"}
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
	return &proxyEnvValue{proxy: proxy}, nil
}

// Attr exposes Go-style and Python-style names for common State methods.
func (v *StateValue) Attr(name string) (starlark.Value, error) {
	if v == nil {
		return nil, fmt.Errorf("llb.State: nil receiver")
	}
	state := v.state
	switch name {
	case "SetMarshalDefaults", "set_marshal_defaults":
		return boundBuiltin("State.SetMarshalDefaults", func(_ *starlark.Thread, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
			if err := arity("State.SetMarshalDefaults", args, kwargs, 0, -1); err != nil { return nil, err }
			opts, err := optionsFromArgs[buildkitllb.ConstraintsOpt](args, 0, "ConstraintsOpt")
			if err != nil { return nil, err }
			return WrapState(state.SetMarshalDefaults(opts...)), nil
		}), nil
	case "WithImageConfig", "with_image_config":
		return boundBuiltin("State.WithImageConfig", func(_ *starlark.Thread, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
			if err := arity("State.WithImageConfig", args, kwargs, 1, 1); err != nil { return nil, err }
			data, err := bytesAt("State.WithImageConfig", args, 0)
			if err != nil { return nil, err }
			updated, err := state.WithImageConfig(data)
			if err != nil { return nil, err }
			return WrapState(updated), nil
		}), nil
	case "GetDir", "get_dir":
		return boundBuiltin("State.GetDir", func(thread *starlark.Thread, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
			if err := arity("State.GetDir", args, kwargs, 0, -1); err != nil { return nil, err }
			opts, err := optionsFromArgs[buildkitllb.ConstraintsOpt](args, 0, "ConstraintsOpt")
			if err != nil { return nil, err }
			dir, err := state.GetDir(callContext(thread), opts...)
			if err != nil { return nil, err }
			return starlark.String(dir), nil
		}), nil
	case "GetEnv", "get_env":
		return boundBuiltin("State.GetEnv", func(thread *starlark.Thread, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
			if err := arity("State.GetEnv", args, kwargs, 1, -1); err != nil { return nil, err }
			key, err := stringAt("State.GetEnv", args, 0)
			if err != nil { return nil, err }
			opts, err := optionsFromArgs[buildkitllb.ConstraintsOpt](args, 1, "ConstraintsOpt")
			if err != nil { return nil, err }
			value, found, err := state.GetEnv(callContext(thread), key, opts...)
			if err != nil { return nil, err }
			return starlark.Tuple{starlark.String(value), starlark.Bool(found)}, nil
		}), nil
	case "GetArgs", "get_args":
		return boundBuiltin("State.GetArgs", func(thread *starlark.Thread, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
			if err := arity("State.GetArgs", args, kwargs, 0, -1); err != nil { return nil, err }
			opts, err := optionsFromArgs[buildkitllb.ConstraintsOpt](args, 0, "ConstraintsOpt")
			if err != nil { return nil, err }
			values, err := state.GetArgs(callContext(thread), opts...)
			if err != nil { return nil, err }
			return stringSliceValue(values), nil
		}), nil
	case "GetHostname", "get_hostname":
		return boundBuiltin("State.GetHostname", func(thread *starlark.Thread, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
			if err := arity("State.GetHostname", args, kwargs, 0, -1); err != nil { return nil, err }
			opts, err := optionsFromArgs[buildkitllb.ConstraintsOpt](args, 0, "ConstraintsOpt")
			if err != nil { return nil, err }
			value, err := state.GetHostname(callContext(thread), opts...)
			if err != nil { return nil, err }
			return starlark.String(value), nil
		}), nil
	case "GetNetwork", "get_network":
		return boundBuiltin("State.GetNetwork", func(thread *starlark.Thread, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
			if err := arity("State.GetNetwork", args, kwargs, 0, -1); err != nil { return nil, err }
			opts, err := optionsFromArgs[buildkitllb.ConstraintsOpt](args, 0, "ConstraintsOpt")
			if err != nil { return nil, err }
			value, err := state.GetNetwork(callContext(thread), opts...)
			if err != nil { return nil, err }
			return starlark.MakeInt(int(value)), nil
		}), nil
	case "GetSecurity", "get_security":
		return boundBuiltin("State.GetSecurity", func(thread *starlark.Thread, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
			if err := arity("State.GetSecurity", args, kwargs, 0, -1); err != nil { return nil, err }
			opts, err := optionsFromArgs[buildkitllb.ConstraintsOpt](args, 0, "ConstraintsOpt")
			if err != nil { return nil, err }
			value, err := state.GetSecurity(callContext(thread), opts...)
			if err != nil { return nil, err }
			return starlark.MakeInt(int(value)), nil
		}), nil
	case "GetPlatform", "get_platform":
		return boundBuiltin("State.GetPlatform", func(thread *starlark.Thread, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
			if err := arity("State.GetPlatform", args, kwargs, 0, -1); err != nil { return nil, err }
			opts, err := optionsFromArgs[buildkitllb.ConstraintsOpt](args, 0, "ConstraintsOpt")
			if err != nil { return nil, err }
			value, err := state.GetPlatform(callContext(thread), opts...)
			if err != nil { return nil, err }
			if value == nil { return starlark.None, nil }
			return &platformValue{platform: *value}, nil
		}), nil
	case "Env", "env":
		return boundBuiltin("State.Env", func(thread *starlark.Thread, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
			if err := arity("State.Env", args, kwargs, 0, -1); err != nil { return nil, err }
			opts, err := optionsFromArgs[buildkitllb.ConstraintsOpt](args, 0, "ConstraintsOpt")
			if err != nil { return nil, err }
			env, err := state.Env(callContext(thread), opts...)
			if err != nil { return nil, err }
			return stringSliceValue(env.ToArray()), nil
		}), nil
	case "Marshal", "marshal":
		return boundBuiltin("State.Marshal", func(thread *starlark.Thread, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
			if err := arity("State.Marshal", args, kwargs, 0, -1); err != nil {
				return nil, err
			}
			opts, err := optionsFromArgs[buildkitllb.ConstraintsOpt](args, 0, "ConstraintsOpt")
			if err != nil {
				return nil, err
			}
			def, err := state.Marshal(callContext(thread), opts...)
			if err != nil {
				return nil, err
			}
			return &DefinitionValue{definition: def}, nil
		}), nil
	case "Run", "run":
		return boundBuiltin("State.Run", func(_ *starlark.Thread, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
			if err := arity("State.Run", args, kwargs, 0, -1); err != nil { return nil, err }
			opts, err := optionsFromArgs[buildkitllb.RunOption](args, 0, "RunOption")
			if err != nil { return nil, err }
			return &ExecStateValue{exec: state.Run(opts...)}, nil
		}), nil
	case "File", "file":
		return boundBuiltin("State.File", func(_ *starlark.Thread, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
			if err := arity("State.File", args, kwargs, 1, -1); err != nil { return nil, err }
			action, err := actionFromValue(args[0])
			if err != nil { return nil, err }
			opts, err := optionsFromArgs[buildkitllb.ConstraintsOpt](args, 1, "ConstraintsOpt")
			if err != nil { return nil, err }
			return WrapState(state.File(action, opts...)), nil
		}), nil
	case "With", "with":
		return boundBuiltin("State.With", func(_ *starlark.Thread, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
			if err := arity("State.With", args, kwargs, 0, -1); err != nil { return nil, err }
			opts, err := optionsFromArgs[buildkitllb.StateOption](args, 0, "StateOption")
			if err != nil { return nil, err }
			return WrapState(state.With(opts...)), nil
		}), nil
	case "AddEnv", "add_env":
		return stateStringPairMethod("State.AddEnv", state.AddEnv), nil
	case "DelEnv", "del_env":
		return stateStringMethod("State.DelEnv", state.DelEnv), nil
	case "Dir", "dir":
		return stateStringMethod("State.Dir", state.Dir), nil
	case "User", "user":
		return stateStringMethod("State.User", state.User), nil
	case "Hostname", "hostname":
		return stateStringMethod("State.Hostname", state.Hostname), nil
	case "WithCgroupParent", "with_cgroup_parent":
		return stateStringMethod("State.WithCgroupParent", state.WithCgroupParent), nil
	case "Network", "network":
		return boundBuiltin("State.Network", func(_ *starlark.Thread, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
			if err := arity("State.Network", args, kwargs, 1, 1); err != nil { return nil, err }
			n, err := intAt("State.Network", args, 0)
			if err != nil { return nil, err }
			return WrapState(state.Network(pb.NetMode(n))), nil
		}), nil
	case "Security", "security":
		return boundBuiltin("State.Security", func(_ *starlark.Thread, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
			if err := arity("State.Security", args, kwargs, 1, 1); err != nil { return nil, err }
			n, err := intAt("State.Security", args, 0)
			if err != nil { return nil, err }
			return WrapState(state.Security(pb.SecurityMode(n))), nil
		}), nil
	case "Platform", "platform":
		return boundBuiltin("State.Platform", func(_ *starlark.Thread, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
			if err := arity("State.Platform", args, kwargs, 1, 1); err != nil { return nil, err }
			p, ok := args[0].(*platformValue)
			if !ok { return nil, fmt.Errorf("State.Platform: got %s, want llb.OCIPlatform", valueType(args[0])) }
			return WrapState(state.Platform(p.platform)), nil
		}), nil
	case "Reset", "reset":
		return boundBuiltin("State.Reset", func(_ *starlark.Thread, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
			if err := arity("State.Reset", args, kwargs, 1, 1); err != nil { return nil, err }
			other, err := stateFromValue(args[0])
			if err != nil { return nil, err }
			return WrapState(state.Reset(other)), nil
		}), nil
	case "Requires", "requires":
		return boundBuiltin("State.Requires", func(_ *starlark.Thread, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
			if err := arity("State.Requires", args, kwargs, 1, -1); err != nil { return nil, err }
			id, err := stringAt("State.Requires", args, 0)
			if err != nil { return nil, err }
			deps := make([]buildkitllb.State, 0, len(args)-1)
			for i := 1; i < len(args); i++ {
				dep, err := stateFromValue(args[i])
				if err != nil { return nil, fmt.Errorf("dependency %d: %w", i, err) }
				deps = append(deps, dep)
			}
			return WrapState(state.Requires(id, deps...)), nil
		}), nil
	case "AddExtraHost", "add_extra_host":
		return boundBuiltin("State.AddExtraHost", func(_ *starlark.Thread, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
			if err := arity("State.AddExtraHost", args, kwargs, 2, 2); err != nil { return nil, err }
			host, err := stringAt("State.AddExtraHost", args, 0)
			if err != nil { return nil, err }
			ipString, err := stringAt("State.AddExtraHost", args, 1)
			if err != nil { return nil, err }
			ip := net.ParseIP(ipString)
			if ip == nil { return nil, fmt.Errorf("State.AddExtraHost: invalid IP address %q", ipString) }
			return WrapState(state.AddExtraHost(host, ip)), nil
		}), nil
	case "AddUlimit", "add_ulimit":
		return boundBuiltin("State.AddUlimit", func(_ *starlark.Thread, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
			if err := arity("State.AddUlimit", args, kwargs, 3, 3); err != nil { return nil, err }
			name, err := stringAt("State.AddUlimit", args, 0)
			if err != nil { return nil, err }
			soft, err := int64At("State.AddUlimit", args, 1)
			if err != nil { return nil, err }
			hard, err := int64At("State.AddUlimit", args, 2)
			if err != nil { return nil, err }
			return WrapState(state.AddUlimit(buildkitllb.UlimitName(name), soft, hard)), nil
		}), nil
	case "AddEnvf", "add_envf":
		return boundBuiltin("State.AddEnvf", func(_ *starlark.Thread, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
			if err := arity("State.AddEnvf", args, kwargs, 2, -1); err != nil { return nil, err }
			key, err := stringAt("State.AddEnvf", args, 0); if err != nil { return nil, err }
			format, err := stringAt("State.AddEnvf", args, 1); if err != nil { return nil, err }
			values := make([]any, 0, len(args)-2)
			for _, arg := range args[2:] { values = append(values, goScalar(arg)) }
			return WrapState(state.AddEnvf(key, format, values...)), nil
		}), nil
	case "Dirf", "dirf":
		return boundBuiltin("State.Dirf", func(_ *starlark.Thread, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
			if err := arity("State.Dirf", args, kwargs, 1, -1); err != nil { return nil, err }
			format, err := stringAt("State.Dirf", args, 0); if err != nil { return nil, err }
			values := make([]any, 0, len(args)-1)
			for _, arg := range args[1:] { values = append(values, goScalar(arg)) }
			return WrapState(state.Dirf(format, values...)), nil
		}), nil
	default:
		return nil, nil
	}
}

func (v *StateValue) AttrNames() []string {
	names := []string{
		"AddEnv", "AddEnvf", "AddExtraHost", "AddUlimit", "DelEnv", "Env", "GetArgs", "GetDir", "GetEnv", "GetHostname", "GetNetwork", "GetPlatform", "GetSecurity", "Marshal",
		"Dir", "Dirf", "File", "Hostname", "Network", "Platform",
		"Requires", "Reset", "Run", "Security", "User", "With",
		"WithCgroupParent", "SetMarshalDefaults", "WithImageConfig", "add_env", "add_envf", "add_extra_host",
		"add_ulimit", "del_env", "dir", "dirf", "file", "hostname",
		"env", "get_args", "get_dir", "get_env", "get_hostname", "get_network", "get_platform", "get_security", "marshal", "network", "platform", "requires", "reset", "run", "security", "set_marshal_defaults", "with_image_config",
		"user", "with", "with_cgroup_parent",
	}
	sort.Strings(names)
	return names
}

func (v *ExecStateValue) Attr(name string) (starlark.Value, error) {
	if v == nil {
		return nil, fmt.Errorf("llb.ExecState: nil receiver")
	}
	execState := v.exec
	switch name {
	case "Root", "root":
		return boundBuiltin("ExecState.Root", func(_ *starlark.Thread, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
			if err := arity("ExecState.Root", args, kwargs, 0, 0); err != nil { return nil, err }
			return WrapState(execState.Root()), nil
		}), nil
	case "AddMount", "add_mount":
		return boundBuiltin("ExecState.AddMount", func(_ *starlark.Thread, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
			if err := arity("ExecState.AddMount", args, kwargs, 2, -1); err != nil { return nil, err }
			target, err := stringAt("ExecState.AddMount", args, 0)
			if err != nil { return nil, err }
			source, err := stateFromValue(args[1])
			if err != nil { return nil, err }
			opts, err := optionsFromArgs[buildkitllb.MountOption](args, 2, "MountOption")
			if err != nil { return nil, err }
			return WrapState(execState.AddMount(target, source, opts...)), nil
		}), nil
	case "GetMount", "get_mount":
		return boundBuiltin("ExecState.GetMount", func(_ *starlark.Thread, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
			if err := arity("ExecState.GetMount", args, kwargs, 1, 1); err != nil { return nil, err }
			target, err := stringAt("ExecState.GetMount", args, 0)
			if err != nil { return nil, err }
			return WrapState(execState.GetMount(target)), nil
		}), nil
	default:
		return nil, nil
	}
}

func (v *ExecStateValue) AttrNames() []string {
	return []string{"AddMount", "GetMount", "Root", "add_mount", "get_mount", "root"}
}

func stringSliceValue(values []string) starlark.Value {
	items := make([]starlark.Value, len(values))
	for i, value := range values {
		items[i] = starlark.String(value)
	}
	return starlark.NewList(items)
}

func stateStringMethod(name string, fn func(string) buildkitllb.State) starlark.Value {
	return boundBuiltin(name, func(_ *starlark.Thread, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
		if err := arity(name, args, kwargs, 1, 1); err != nil { return nil, err }
		value, err := stringAt(name, args, 0)
		if err != nil { return nil, err }
		return WrapState(fn(value)), nil
	})
}

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

// ---- Starlark argument conversion helpers ----

func arity(name string, args starlark.Tuple, kwargs []starlark.Tuple, min, max int) error {
	if len(kwargs) != 0 {
		return fmt.Errorf("%s: keyword arguments are not supported by this binding; pass options positionally", name)
	}
	if len(args) < min || (max >= 0 && len(args) > max) {
		if max < 0 { return fmt.Errorf("%s: got %d arguments, want at least %d", name, len(args), min) }
		if min == max { return fmt.Errorf("%s: got %d arguments, want %d", name, len(args), min) }
		return fmt.Errorf("%s: got %d arguments, want %d..%d", name, len(args), min, max)
	}
	return nil
}

func stringAt(name string, args starlark.Tuple, index int) (string, error) {
	value, ok := args[index].(starlark.String)
	if !ok { return "", fmt.Errorf("%s: argument %d: got %s, want string", name, index+1, valueType(args[index])) }
	return string(value), nil
}

func boolAt(name string, args starlark.Tuple, index int) (bool, error) {
	value, ok := args[index].(starlark.Bool)
	if !ok { return false, fmt.Errorf("%s: argument %d: got %s, want bool", name, index+1, valueType(args[index])) }
	return bool(value), nil
}

func intAt(name string, args starlark.Tuple, index int) (int, error) {
	value, ok := args[index].(starlark.Int)
	if !ok { return 0, fmt.Errorf("%s: argument %d: got %s, want int", name, index+1, valueType(args[index])) }
	result, ok := value.Int64()
	if !ok || int64(int(result)) != result { return 0, fmt.Errorf("%s: argument %d: integer out of range", name, index+1) }
	return int(result), nil
}

func int64At(name string, args starlark.Tuple, index int) (int64, error) {
	value, ok := args[index].(starlark.Int)
	if !ok { return 0, fmt.Errorf("%s: argument %d: got %s, want int", name, index+1, valueType(args[index])) }
	result, ok := value.Int64()
	if !ok { return 0, fmt.Errorf("%s: argument %d: integer out of range", name, index+1) }
	return result, nil
}

func stringList(value starlark.Value) ([]string, error) {
	sequence, ok := value.(starlark.Indexable)
	if !ok { return nil, fmt.Errorf("got %s, want list or tuple of strings", valueType(value)) }
	result := make([]string, sequence.Len())
	for i := range result {
		item, ok := sequence.Index(i).(starlark.String)
		if !ok { return nil, fmt.Errorf("element %d: got %s, want string", i, valueType(sequence.Index(i))) }
		result[i] = string(item)
	}
	return result, nil
}

func goScalar(value starlark.Value) any {
	switch v := value.(type) {
	case starlark.String:
		return string(v)
	case starlark.Int:
		if n, ok := v.Int64(); ok { return n }
		return v.BigInt()
	case starlark.Float:
		return float64(v)
	case starlark.Bool:
		return bool(v)
	case starlark.NoneType:
		return nil
	default:
		return value.String()
	}
}

var _ starlark.HasAttrs = (*StateValue)(nil)
var _ starlark.HasAttrs = (*ExecStateValue)(nil)
