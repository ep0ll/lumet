package dockerfile

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"path"
	"strconv"
	"strings"
	"time"

	"github.com/moby/buildkit/client/llb"
	"github.com/moby/buildkit/solver/pb"
	"github.com/moby/buildkit/util/system"
	dockerspec "github.com/moby/docker-image-spec/specs-go/v1"
)

// NewLLBHandlers returns the standard instruction handlers backed by BuildKit's
// client/llb package. Applications can replace a binding or register new kinds.
func NewLLBHandlers() *Handlers {
	h := NewHandlers()
	register := func(err error) {
		if err != nil { panic(err) }
	}
	register(RegisterHandler(h, KindFrom, handleFrom))
	register(RegisterHandler(h, KindResetStage, handleResetStage))
	register(RegisterHandler(h, KindArg, handleArg))
	register(RegisterHandler(h, KindEnv, handleEnv))
	register(RegisterHandler(h, KindLabel, handleLabel))
	register(RegisterHandler(h, KindMaintainer, handleMaintainer))
	register(RegisterHandler(h, KindRun, handleRun))
	register(RegisterHandler(h, KindCmd, handleCmd))
	register(RegisterHandler(h, KindEntrypoint, handleEntrypoint))
	register(RegisterHandler(h, KindShell, handleShell))
	register(RegisterHandler(h, KindExpose, handleExpose))
	register(RegisterHandler(h, KindVolume, handleVolume))
	register(RegisterHandler(h, KindUser, handleUser))
	register(RegisterHandler(h, KindWorkdir, handleWorkdir))
	register(RegisterHandler(h, KindCopy, handleCopy))
	register(RegisterHandler(h, KindAdd, handleAdd))
	register(RegisterHandler(h, KindStopSignal, handleStopSignal))
	register(RegisterHandler(h, KindHealthcheck, handleHealthcheck))
	register(RegisterHandler(h, KindOnBuild, handleOnBuild))
	return h
}

func handleFrom(ctx context.Context, call Call[FromSpec]) (Stage, error) {
	return startStage(ctx, call, call.Spec, false)
}
func handleResetStage(ctx context.Context, call Call[ResetSpec]) (Stage, error) {
	converted := Call[FromSpec]{
		Kind: call.Kind, Stage: call.Stage, Spec: call.Spec.FromSpec,
		Handlers: call.Handlers, Runtime: call.Runtime,
	}
	return startStage(ctx, converted, call.Spec.FromSpec, true)
}

func startStage(ctx context.Context, call Call[FromSpec], spec FromSpec, reset bool) (Stage, error) {
	if call.Runtime == nil { return Stage{}, errors.New("FROM requires a build runtime") }
	if spec.Base == nil { return Stage{}, errors.New("FROM base is required") }
	platform := call.Runtime.Platform
	if spec.Platform != nil { platform = *spec.Platform }

	var (
		state llb.State
		image = emptyImage(platform)
		baseRef string
		inheritedArgs = map[string]string{}
		inheritedTriggers []Trigger
		rawOnBuild []string
	)
	switch base := spec.Base.(type) {
	case ScratchBase:
		state = llb.Scratch()
		baseRef = "scratch"
	case ImageRef:
		ref, err := call.Runtime.Expand(base.Ref, Stage{})
		if err != nil { return Stage{}, fmt.Errorf("expand FROM reference: %w", err) }
		if strings.TrimSpace(ref) == "" { return Stage{}, errors.New("FROM image reference cannot be empty") }
		baseRef = ref
		imageOptions := append([]llb.ImageOption(nil), base.Options...)
		imageOptions = append(imageOptions, spec.Options...)
		imageStage := Stage{Name: spec.Name, Platform: platform}
		for _, constraint := range operationConstraints(call.Runtime, imageStage, spec.Common, "FROM "+ref) {
			imageOptions = append(imageOptions, constraint)
		}
		state = llb.Image(ref, imageOptions...)
		if call.Runtime.ImageResolver != nil {
			resolved, err := call.Runtime.ImageResolver(ctx, ref, platform)
			if err != nil { return Stage{}, fmt.Errorf("resolve image metadata: %w", err) }
			image = cloneImage(resolved)
			rawOnBuild = append([]string(nil), image.Config.OnBuild...)
			config, err := json.Marshal(image)
			if err != nil { return Stage{}, fmt.Errorf("encode image config: %w", err) }
			state, err = state.WithImageConfig(config)
			if err != nil { return Stage{}, fmt.Errorf("apply image config: %w", err) }
		}
	case StageRef:
		if call.Runtime.ResolveStage == nil { return Stage{}, errors.New("FROM stage reference requires a stage resolver") }
		inherited, err := call.Runtime.ResolveStage(base.Name)
		if err != nil { return Stage{}, err }
		state, image = inherited.State, cloneImage(inherited.Image)
		if spec.Platform == nil { platform = inherited.Platform }
		baseRef = string(base.Name)
		inheritedArgs = cloneStringMap(inherited.Args)
		inheritedTriggers = append([]Trigger(nil), inherited.Triggers...)
		rawOnBuild = append([]string(nil), inherited.RawOnBuild...)
	case ContextRef:
		named, ok := call.Runtime.NamedContexts[base.Name]
		if !ok { return Stage{}, fmt.Errorf("named context %q not found", base.Name) }
		state, image = named.State, cloneImage(named.Image)
		rawOnBuild = append([]string(nil), image.Config.OnBuild...)
		if platform.OS == "" && named.Platform.OS != "" { platform = named.Platform }
		baseRef = "context:" + base.Name
	default:
		return Stage{}, fmt.Errorf("unsupported FROM base %T", spec.Base)
	}

	image.OS, image.Architecture, image.Variant = platform.OS, platform.Architecture, platform.Variant
	image.OSVersion = platform.OSVersion
	image.OSFeatures = append([]string(nil), platform.OSFeatures...)
	state = state.Platform(platform)

	env := environmentFromConfig(image.Config.Env)
	if image.Config.WorkingDir != "" { state = state.Dir(image.Config.WorkingDir) }
	if image.Config.User != "" { state = state.User(image.Config.User) }
	for _, pair := range image.Config.Env {
		key, value, ok := strings.Cut(pair, "=")
		if ok { state = state.AddEnv(key, value) }
	}
	if image.Config.WorkingDir == "" { image.Config.WorkingDir = "/" }

	stage := Stage{
		Name: spec.Name,
		Base: BaseImage{Ref: baseRef, State: state, Image: cloneImage(image), Platform: platform},
		State: state, Image: image, Platform: platform,
		Args: cloneStringMap(inheritedArgs), Env: env, Triggers: inheritedTriggers, RawOnBuild: rawOnBuild,
	}
	if reset {
		stage.Args = map[string]string{}
		stage.Triggers = nil
		stage.RawOnBuild = nil
		stage.Image.Config.OnBuild = nil
	}
	if stage.Name == "" { stage.Name = spec.Name }
	return stage, nil
}

func handleArg(_ context.Context, call Call[ArgSpec]) (Stage, error) {
	if call.Runtime == nil { return call.Stage, errors.New("ARG requires a build runtime") }
	spec := call.Spec
	if strings.TrimSpace(spec.Name) == "" { return call.Stage, errors.New("ARG name cannot be empty") }
	stage := call.Stage
	var (
		value string
		found bool
	)
	if value, found = call.Runtime.BuildArgs[spec.Name]; !found && spec.Default != nil {
		expanded, err := call.Runtime.Expand(*spec.Default, stage)
		if err != nil { return stage, fmt.Errorf("expand ARG default: %w", err) }
		value, found = expanded, true
	}
	if !found && stage.Name != "" {
		value, found = stage.Args[spec.Name]
	}
	if !found {
		value, found = call.Runtime.GlobalArgs[spec.Name]
	}
	if stage.Name == "" {
		if found { call.Runtime.GlobalArgs[spec.Name] = value }
		return stage, nil
	}
	if found {
		stage.Args = ensureStringMap(stage.Args)
		stage.Args[spec.Name] = value
		stage.State = stage.State.AddEnv(spec.Name, value)
	}
	return stage, nil
}

func handleEnv(_ context.Context, call Call[EnvSpec]) (Stage, error) {
	stage := call.Stage
	if len(call.Spec.Vars) == 0 { return stage, errors.New("ENV requires at least one key/value pair") }
	previous := cloneStringMap(stage.Env)
	for _, pair := range call.Spec.Vars {
		if pair.Key == "" { return call.Stage, errors.New("ENV key cannot be empty") }
		value, err := call.Runtime.Expand(pair.Value, Stage{Args: stage.Args, Env: previous})
		if err != nil { return call.Stage, fmt.Errorf("expand ENV %s: %w", pair.Key, err) }
		stage.State = stage.State.AddEnv(pair.Key, value)
		stage.Image.Config.Env = setEnv(stage.Image.Config.Env, pair.Key, value)
		if stage.Env == nil { stage.Env = make(map[string]string) }
		stage.Env[pair.Key] = value
	}
	return stage, nil
}

func handleLabel(_ context.Context, call Call[LabelSpec]) (Stage, error) {
	stage := call.Stage
	if len(call.Spec.Labels) == 0 { return stage, errors.New("LABEL requires at least one key/value pair") }
	if stage.Image.Config.Labels == nil { stage.Image.Config.Labels = make(map[string]string, len(call.Spec.Labels)) }
	for _, pair := range call.Spec.Labels {
		if pair.Key == "" { return call.Stage, errors.New("LABEL key cannot be empty") }
		value, err := call.Runtime.Expand(pair.Value, stage)
		if err != nil { return call.Stage, fmt.Errorf("expand LABEL %s: %w", pair.Key, err) }
		stage.Image.Config.Labels[pair.Key] = value
	}
	return stage, nil
}

func handleMaintainer(_ context.Context, call Call[MaintainerSpec]) (Stage, error) {
	stage := call.Stage
	stage.Image.Author = call.Spec.Name
	return stage, nil
}

func handleRun(_ context.Context, call Call[RunSpec]) (Stage, error) {
	stage := call.Stage
	if len(call.Spec.Mounts) > 0 && call.Runtime == nil { return stage, errors.New("RUN mounts require a build runtime") }
	args, err := commandArgs(call.Spec.Command, stage.Image.Config.Shell, stage.Platform.OS)
	if err != nil { return stage, err }

	options := make([]llb.RunOption, 0, 8+len(call.Spec.Options)+len(call.Spec.Mounts))
	options = append(options, llb.Args(args))
	name := call.Spec.CustomName
	if name == "" { name = call.Spec.ProgressTitle }
	if name == "" { name = "RUN " + formatCommand(call.Spec.Command) }
	options = append(options, llb.WithCustomName(buildkitCustomName(call.Runtime, stage, name)))

	for _, mount := range call.Spec.Mounts {
		mountOptions, err := runMountOptions(call.Runtime, stage, mount)
		if err != nil { return call.Stage, err }
		options = append(options, mountOptions...)
	}
	switch call.Spec.Network {
	case NetworkDefault:
	case NetworkNone:
		options = append(options, llb.Network(pb.NetMode_NONE))
	case NetworkHost:
		options = append(options, llb.Network(pb.NetMode_HOST))
	default:
		return call.Stage, fmt.Errorf("unsupported RUN network mode %d", call.Spec.Network)
	}
	switch call.Spec.Security {
	case SecuritySandbox:
		// Sandbox is BuildKit's default execution mode.
	case SecurityInsecure:
		options = append(options, llb.Security(pb.SecurityMode_INSECURE))
	default:
		return call.Stage, fmt.Errorf("unsupported RUN security mode %d", call.Spec.Security)
	}
	for _, device := range call.Spec.Devices {
		name, err := call.Runtime.Expand(device.Name, stage)
		if err != nil { return call.Stage, fmt.Errorf("expand RUN device name: %w", err) }
		if name == "" { return call.Stage, errors.New("RUN device name cannot be empty") }
		deviceOptions := []llb.CDIDeviceOption{llb.CDIDeviceName(name)}
		if device.Optional { deviceOptions = append(deviceOptions, llb.CDIDeviceOptional) }
		options = append(options, llb.AddCDIDevice(deviceOptions...))
	}
	for _, limit := range call.Spec.Ulimits {
		if limit.Name == "" || limit.Soft < 0 || limit.Hard < 0 || limit.Soft > limit.Hard {
			return call.Stage, fmt.Errorf("invalid ulimit %q: soft=%d hard=%d", limit.Name, limit.Soft, limit.Hard)
		}
		options = append(options, llb.AddUlimit(limit.Name, limit.Soft, limit.Hard))
	}
	if call.Spec.ReadOnlyRoot { options = append(options, llb.ReadonlyRootFS()) }
	options = append(options, call.Spec.Options...)
	for _, constraint := range combinedConstraints(call.Runtime, call.Spec.Constraints...) {
		options = append(options, constraint)
	}
	stage.State = stage.State.Run(options...).Root()
	return stage, nil
}

func runMountOptions(runtime *Runtime, stage Stage, mount Mount) ([]llb.RunOption, error) {
	if runtime == nil { return nil, errors.New("mount requires a build runtime") }
	switch value := mount.(type) {
	case BindMount:
		target, err := normalizeMountTarget(runtime, stage, value.Target)
		if err != nil { return nil, fmt.Errorf("bind mount target: %w", err) }
		source, err := resolveSource(runtime, stage, value.From)
		if err != nil { return nil, fmt.Errorf("bind mount source: %w", err) }
		opts := []llb.MountOption{}
		if value.Source != "" {
			sourcePath, err := runtime.Expand(value.Source, stage)
			if err != nil { return nil, err }
			opts = append(opts, llb.SourcePath(sourcePath))
		}
		if !value.ReadWrite {
			opts = append(opts, llb.Readonly)
		} else {
			opts = append(opts, llb.ForceNoOutput)
		}
		return []llb.RunOption{llb.AddMount(target, source, opts...)}, nil
	case CacheMount:
		target, err := normalizeMountTarget(runtime, stage, value.Target)
		if err != nil { return nil, fmt.Errorf("cache mount target: %w", err) }
		source := llb.Scratch()
		if value.From != nil {
			var err error
			source, err = resolveSource(runtime, stage, value.From)
			if err != nil { return nil, fmt.Errorf("cache mount source: %w", err) }
		}
		sourcePath := value.Source
		if sourcePath != "" {
			sourcePath, err = runtime.Expand(sourcePath, stage)
			if err != nil { return nil, err }
		}
		if value.UID != 0 || value.GID != 0 || value.Mode != 0 {
			mode := value.Mode
			if mode == 0 { mode = 0o755 }
			owner := fmt.Sprintf("%d:%d", value.UID, value.GID)
			if sourcePath == "" { sourcePath = "/cache" }
			source = source.File(llb.Mkdir(sourcePath, mode, llb.WithParents(true), llb.WithUser(owner)), combinedConstraints(runtime)...)
		}
		id := value.ID
		if id != "" {
			id, err = runtime.Expand(id, stage)
			if err != nil { return nil, err }
		}
		if id == "" { id = strings.TrimPrefix(path.Clean(target), "/") }
		if id == "" || id == "." { return nil, errors.New("cache mount ID cannot be derived from target") }
		sharing := value.Sharing
		mountOptions := []llb.MountOption{llb.AsPersistentCacheDir(id, sharing)}
		if sourcePath != "" { mountOptions = append(mountOptions, llb.SourcePath(sourcePath)) }
		if value.ReadOnly { mountOptions = append(mountOptions, llb.Readonly) }
		return []llb.RunOption{llb.AddMount(target, source, mountOptions...)}, nil
	case TmpfsMount:
		target, err := normalizeMountTarget(runtime, stage, value.Target)
		if err != nil { return nil, fmt.Errorf("tmpfs mount target: %w", err) }
		if value.Size < 0 { return nil, errors.New("tmpfs mount size cannot be negative") }
		return []llb.RunOption{llb.AddMount(target, llb.Scratch(), llb.Tmpfs(llb.TmpfsSize(value.Size)))}, nil
	case SecretMount:
		var err error
		id := value.ID
		if id != "" {
			id, err := runtime.Expand(id, stage)
			if err != nil { return nil, err }
		}
		target := value.Target
		if target != "" {
			target, err = normalizeMountTarget(runtime, stage, target)
			if err != nil { return nil, fmt.Errorf("secret mount target: %w", err) }
		}
		if id == "" && target != "" { id = path.Base(target) }
		if id == "" { return nil, errors.New("secret mount ID or target is required") }
		var destination *string
		if target != "" {
			destination = &target
		} else if value.Env == "" {
			target = "/run/secrets/" + id
			destination = &target
		}
		options := []llb.SecretOption{}
		if !value.Required { options = append(options, llb.SecretOptional) }
		mode := value.Mode
		if mode == 0 { mode = 0o400 }
		options = append(options, llb.SecretFileOpt(value.UID, value.GID, int(mode)))
		if value.Env != "" {
			envName, err := runtime.Expand(value.Env, stage)
			if err != nil { return nil, err }
			destination = nil
			options = append(options, llb.SecretAsEnvName(envName))
		}
		return []llb.RunOption{llb.AddSecretWithDest(id, destination, options...)}, nil
	case SSHMount:
		var err error
		id := value.ID
		if id != "" {
			id, err = runtime.Expand(id, stage)
			if err != nil { return nil, err }
		}
		if id == "" { id = "default" }
		target := value.Target
		if target != "" {
			target, err = normalizeMountTarget(runtime, stage, target)
			if err != nil { return nil, fmt.Errorf("SSH mount target: %w", err) }
		}
		options := []llb.SSHOption{llb.SSHID(id)}
		if target != "" { options = append(options, llb.SSHSocketTarget(target)) }
		mode := value.Mode
		if mode == 0 { mode = 0o600 }
		options = append(options, llb.SSHSocketOpt(target, value.UID, value.GID, int(mode)))
		if !value.Required { options = append(options, llb.SSHOptional) }
		return []llb.RunOption{llb.AddSSHSocket(options...)}, nil
	default:
		return nil, fmt.Errorf("unsupported RUN mount type %T", mount)
	}
}

func handleCmd(_ context.Context, call Call[CmdSpec]) (Stage, error) {
	stage := call.Stage
	args, err := commandArgs(call.Spec.Command, stage.Image.Config.Shell, stage.Platform.OS)
	if err != nil { return call.Stage, err }
	stage.Image.Config.Cmd = args
	stage.Image.Config.ArgsEscaped = true
	stage.CmdSet = true
	return stage, nil
}
func handleEntrypoint(_ context.Context, call Call[EntrypointSpec]) (Stage, error) {
	stage := call.Stage
	args, err := commandArgs(call.Spec.Command, stage.Image.Config.Shell, stage.Platform.OS)
	if err != nil { return call.Stage, err }
	stage.Image.Config.Entrypoint = args
	if !stage.CmdSet { stage.Image.Config.Cmd = nil }
	return stage, nil
}
func handleShell(_ context.Context, call Call[ShellSpec]) (Stage, error) {
	stage := call.Stage
	if len(call.Spec.Shell) == 0 || strings.TrimSpace(call.Spec.Shell[0]) == "" {
		return call.Stage, errors.New("SHELL requires a non-empty executable")
	}
	stage.Image.Config.Shell = append([]string(nil), call.Spec.Shell...)
	return stage, nil
}

func handleExpose(_ context.Context, call Call[ExposeSpec]) (Stage, error) {
	stage := call.Stage
	if len(call.Spec.Ports) == 0 { return stage, errors.New("EXPOSE requires at least one port") }
	if stage.Image.Config.ExposedPorts == nil { stage.Image.Config.ExposedPorts = make(map[string]struct{}) }
	for _, exposed := range call.Spec.Ports {
		switch port := exposed.(type) {
		case Port:
			if port.Number == 0 { return call.Stage, errors.New("EXPOSE port must be in range 1..65535") }
			stage.Image.Config.ExposedPorts[portKey(port.Number, port.Proto)] = struct{}{}
		case PortRange:
			if port.From == 0 || port.To < port.From { return call.Stage, fmt.Errorf("invalid EXPOSE port range %d-%d", port.From, port.To) }
			for number := uint32(port.From); number <= uint32(port.To); number++ {
				stage.Image.Config.ExposedPorts[portKey(uint16(number), port.Proto)] = struct{}{}
			}
		default:
			return call.Stage, fmt.Errorf("unsupported exposed port type %T", exposed)
		}
	}
	return stage, nil
}
func portKey(number uint16, protocol Protocol) string {
	suffix := "tcp"
	if protocol == ProtoUDP { suffix = "udp" }
	return strconv.Itoa(int(number)) + "/" + suffix
}

func handleVolume(_ context.Context, call Call[VolumeSpec]) (Stage, error) {
	stage := call.Stage
	if len(call.Spec.Paths) == 0 { return stage, errors.New("VOLUME requires at least one path") }
	if stage.Image.Config.Volumes == nil { stage.Image.Config.Volumes = make(map[string]struct{}) }
	for _, item := range call.Spec.Paths {
		value, err := call.Runtime.Expand(item, stage)
		if err != nil { return call.Stage, err }
		if value == "" { return call.Stage, errors.New("VOLUME path cannot be empty") }
		normalized, err := system.NormalizeWorkdir(stage.Image.Config.WorkingDir, value, stage.Platform.OS)
		if err != nil { return call.Stage, fmt.Errorf("normalize VOLUME path: %w", err) }
		stage.Image.Config.Volumes[system.ToSlash(normalized, stage.Platform.OS)] = struct{}{}
	}
	return stage, nil
}

func handleUser(_ context.Context, call Call[UserSpec]) (Stage, error) {
	stage := call.Stage
	user := principalString(call.Spec.User)
	if call.Spec.Group != nil { user += ":" + principalString(*call.Spec.Group) }
	if user == "" { return call.Stage, errors.New("USER cannot be empty") }
	expanded, err := call.Runtime.Expand(user, stage)
	if err != nil { return call.Stage, err }
	stage.State = stage.State.User(expanded)
	stage.Image.Config.User = expanded
	return stage, nil
}

func handleWorkdir(_ context.Context, call Call[WorkdirSpec]) (Stage, error) {
	stage := call.Stage
	value, err := call.Runtime.Expand(call.Spec.Path, stage)
	if err != nil { return call.Stage, err }
	if strings.TrimSpace(value) == "" { return call.Stage, errors.New("WORKDIR path cannot be empty") }
	normalized, err := system.NormalizeWorkdir(stage.Image.Config.WorkingDir, value, stage.Platform.OS)
	if err != nil { return call.Stage, fmt.Errorf("normalize WORKDIR: %w", err) }
	stage.Image.Config.WorkingDir = normalized
	slashPath := system.ToSlash(normalized, stage.Platform.OS)
	stage.State = stage.State.Dir(slashPath)
	if slashPath != "/" {
		options := []llb.MkdirOption{llb.WithParents(true)}
		if stage.Image.Config.User != "" { options = append(options, llb.WithUser(stage.Image.Config.User)) }
		if call.Runtime != nil && call.Runtime.CreatedAt != nil { options = append(options, llb.WithCreatedTime(*call.Runtime.CreatedAt)) }
		constraints := operationConstraints(call.Runtime, stage, call.Spec.Common, "WORKDIR "+slashPath)
		stage.State = stage.State.File(llb.Mkdir(slashPath, 0o755, options...), constraints...)
	}
	return stage, nil
}

func handleCopy(_ context.Context, call Call[CopySpec]) (Stage, error) {
	stage := call.Stage
	if len(call.Spec.Src) == 0 { return stage, errors.New("COPY requires at least one source") }
	if call.Spec.Dest == "" { return stage, errors.New("COPY destination cannot be empty") }
	destination, err := call.Runtime.Expand(call.Spec.Dest, stage)
	if err != nil { return stage, err }
	destination, err = system.NormalizePath(stage.Image.Config.WorkingDir, destination, stage.Platform.OS, true)
	if err != nil { return stage, fmt.Errorf("normalize COPY destination: %w", err) }
	source, err := resolveSource(call.Runtime, stage, call.Spec.From)
	if err != nil { return stage, fmt.Errorf("resolve COPY source: %w", err) }
	info := copyInfo(call.Spec.FileOpts, call.Runtime, stage, false)
	action := (*llb.FileAction)(nil)
	for _, raw := range call.Spec.Src {
		src, err := call.Runtime.Expand(raw, stage)
		if err != nil { return call.Stage, err }
		if src == "" { return call.Stage, errors.New("COPY source cannot be empty") }
		target := destination
		if call.Spec.Parents { target = path.Join(destination, preservedSourcePath(src)) }
		copyOpts, err := expandedCopyOptions(info, call.Spec.FileOpts, call.Runtime, stage)
		if err != nil { return call.Stage, err }
		if action == nil { action = llb.Copy(source, rootPath(src), target, copyOpts...) } else {
			action = action.Copy(source, rootPath(src), target, copyOpts...)
		}
	}
	constraints := operationConstraints(call.Runtime, stage, call.Spec.Common, call.Spec.String())
	if call.Spec.Link {
		copyState := llb.Scratch().Platform(stage.Platform).Dir(stage.Image.Config.WorkingDir).File(action, constraints...)
		stage.State = llb.Merge([]llb.State{stage.State, copyState}, constraints...)
	} else {
		stage.State = stage.State.File(action, constraints...)
	}
	return stage, nil
}

func handleAdd(_ context.Context, call Call[AddSpec]) (Stage, error) {
	stage := call.Stage
	if len(call.Spec.Src) == 0 { return stage, errors.New("ADD requires at least one source") }
	if call.Spec.Dest == "" { return stage, errors.New("ADD destination cannot be empty") }
	destination, err := call.Runtime.Expand(call.Spec.Dest, stage)
	if err != nil { return stage, err }
	destination, err = system.NormalizePath(stage.Image.Config.WorkingDir, destination, stage.Platform.OS, true)
	if err != nil { return stage, fmt.Errorf("normalize ADD destination: %w", err) }

	var action *llb.FileAction
	for _, source := range call.Spec.Src {
		state, sourcePath, local, err := resolveAddSource(call.Runtime, stage, source, call.Spec)
		if err != nil { return call.Stage, fmt.Errorf("resolve ADD source %T: %w", source, err) }
		unpack := local
		if call.Spec.Unpack != nil { unpack = *call.Spec.Unpack }
		opts, err := expandedCopyOptions(copyInfo(call.Spec.FileOpts, call.Runtime, stage, unpack), call.Spec.FileOpts, call.Runtime, stage)
		if err != nil { return call.Stage, err }
		if action == nil { action = llb.Copy(state, sourcePath, destination, opts...) } else { action = action.Copy(state, sourcePath, destination, opts...) }
	}
	constraints := operationConstraints(call.Runtime, stage, call.Spec.Common, call.Spec.String())
	if call.Spec.Link {
		copyState := llb.Scratch().Platform(stage.Platform).Dir(stage.Image.Config.WorkingDir).File(action, constraints...)
		stage.State = llb.Merge([]llb.State{stage.State, copyState}, constraints...)
	} else {
		stage.State = stage.State.File(action, constraints...)
	}
	return stage, nil
}

func resolveAddSource(runtime *Runtime, stage Stage, source AddSource, spec AddSpec) (llb.State, string, bool, error) {
	switch value := source.(type) {
	case LocalPath:
		expanded, err := runtime.Expand(string(value), stage)
		if err != nil { return llb.State{}, "", false, err }
		if strings.TrimSpace(expanded) == "" { return llb.State{}, "", false, errors.New("local path is empty") }
		return runtime.BuildContext, rootPath(expanded), true, nil
	case URL:
		raw, err := runtime.Expand(string(value), stage)
		if err != nil { return llb.State{}, "", false, err }
		parsed, err := url.Parse(raw)
		if err != nil { return llb.State{}, "", false, fmt.Errorf("parse URL: %w", err) }
		if parsed.Scheme != "http" && parsed.Scheme != "https" { return llb.State{}, "", false, errors.New("remote ADD URL must use http or https") }
		filename := path.Base(parsed.Path)
		if filename == "." || filename == "/" || filename == "" { filename = "download" }
		opts := []llb.HTTPOption{llb.Filename(filename)}
		if spec.Checksum != "" { opts = append(opts, llb.Checksum(spec.Checksum)) }
		return llb.HTTP(raw, opts...), "/"+filename, false, nil
	case GitRepo:
		repoURL, err := runtime.Expand(value.URL, stage)
		if err != nil { return llb.State{}, "", false, err }
		ref, err := runtime.Expand(value.Ref, stage)
		if err != nil { return llb.State{}, "", false, err }
		subdir, err := runtime.Expand(value.Subdir, stage)
		if err != nil { return llb.State{}, "", false, err }
		if repoURL == "" { return llb.State{}, "", false, errors.New("Git URL is empty") }
		opts := []llb.GitOption{}
		if spec.KeepGitDir { opts = append(opts, llb.KeepGitDir()) }
		if subdir != "" { opts = append(opts, llb.GitSubDir(subdir)) }
		return llb.Git(repoURL, ref, opts...), "/", false, nil
	default:
		return llb.State{}, "", false, fmt.Errorf("unsupported ADD source %T", source)
	}
}

func resolveSource(runtime *Runtime, stage Stage, source CopySource) (llb.State, error) {
	if source == nil {
		if runtime == nil { return llb.State{}, errors.New("no build context configured") }
		return runtime.BuildContext, nil
	}
	switch value := source.(type) {
	case StageRef:
		if runtime.ResolveStage == nil { return llb.State{}, errors.New("stage resolver is not configured") }
		stage, err := runtime.ResolveStage(value.Name)
		if err != nil { return llb.State{}, err }
		return stage.State, nil
	case ContextRef:
		named, ok := runtime.NamedContexts[value.Name]
		if !ok { return llb.State{}, fmt.Errorf("named context %q not found", value.Name) }
		return named.State, nil
	case ImageRef:
		ref, err := runtime.Expand(value.Ref, stage)
		if err != nil { return llb.State{}, err }
		if ref == "" { return llb.State{}, errors.New("image reference is empty") }
		return llb.Image(ref, value.Options...).Platform(stage.Platform), nil
	default:
		return llb.State{}, fmt.Errorf("unsupported copy source %T", source)
	}
}

func copyInfo(fileOpts FileOpts, runtime *Runtime, stage Stage, unpack bool) *llb.CopyInfo {
	return &llb.CopyInfo{
		ChownOpt: fileOpts.Chown,
		Mode: fileOpts.Chmod,
		ExcludePatterns: append([]string(nil), fileOpts.Exclude...),
		CreateDestPath: true,
		AllowWildcard: true,
		FollowSymlinks: true,
		AttemptUnpack: unpack,
	}
}
func expandedCopyOptions(info *llb.CopyInfo, fileOpts FileOpts, runtime *Runtime, stage Stage) ([]llb.CopyOption, error) {
	if fileOpts.ChownUser != "" && runtime != nil {
		user, err := runtime.Expand(fileOpts.ChownUser, stage)
		if err != nil { return nil, err }
		if user == "" { return nil, errors.New("expanded --chown value cannot be empty") }
		chown := llb.WithUser(user).(llb.ChownOpt)
		info.ChownOpt = &chown
	}
	for index, pattern := range info.ExcludePatterns {
		if runtime != nil {
			value, err := runtime.Expand(pattern, stage)
			if err != nil { return nil, err }
			info.ExcludePatterns[index] = value
		}
	}
	if info.Mode != nil && info.Mode.ModeStr != "" && runtime != nil {
		value, err := runtime.Expand(info.Mode.ModeStr, stage)
		if err != nil { return nil, err }
		mode := *info.Mode
		mode.ModeStr = value
		info.Mode = &mode
	}
	options := []llb.CopyOption{info}
	if runtime != nil && runtime.CreatedAt != nil {
		options = append(options, llb.WithCreatedTime(*runtime.CreatedAt))
	}
	return options, nil
}

func handleStopSignal(_ context.Context, call Call[StopSignalSpec]) (Stage, error) {
	stage := call.Stage
	if call.Spec.Signal == "" { return stage, errors.New("STOPSIGNAL cannot be empty") }
	stage.Image.Config.StopSignal = string(call.Spec.Signal)
	return stage, nil
}

func handleHealthcheck(_ context.Context, call Call[HealthcheckSpec]) (Stage, error) {
	stage := call.Stage
	if call.Spec.None {
		stage.Image.Config.Healthcheck = &dockerspec.HealthcheckConfig{Test: []string{"NONE"}}
		return stage, nil
	}
	var test []string
	switch command := call.Spec.Command.(type) {
	case ExecForm:
		if len(command) == 0 { return stage, errors.New("HEALTHCHECK exec command cannot be empty") }
		test = append([]string{"CMD"}, []string(command)...)
	case ShellForm:
		if strings.TrimSpace(string(command)) == "" { return stage, errors.New("HEALTHCHECK shell command cannot be empty") }
		test = []string{"CMD-SHELL", string(command)}
	default:
		return stage, fmt.Errorf("unsupported HEALTHCHECK command form %T", call.Spec.Command)
	}
	if call.Spec.Interval < 0 || call.Spec.Timeout < 0 || call.Spec.StartPeriod < 0 || call.Spec.StartInterval < 0 || call.Spec.Retries < 0 {
		return call.Stage, errors.New("HEALTHCHECK durations and retries cannot be negative")
	}
	interval, timeout, startInterval, retries := call.Spec.Interval, call.Spec.Timeout, call.Spec.StartInterval, call.Spec.Retries
	if interval == 0 { interval = 30*time.Second }
	if timeout == 0 { timeout = 30*time.Second }
	if startInterval == 0 { startInterval = 5*time.Second }
	if retries == 0 { retries = 3 }
	if interval < time.Millisecond || timeout < time.Millisecond || startInterval < time.Millisecond || (call.Spec.StartPeriod > 0 && call.Spec.StartPeriod < time.Millisecond) {
		return call.Stage, errors.New("HEALTHCHECK non-zero durations must be at least 1ms")
	}
	stage.Image.Config.Healthcheck = &dockerspec.HealthcheckConfig{
		Test: test, Interval: interval, Timeout: timeout,
		StartPeriod: call.Spec.StartPeriod, StartInterval: startInterval, Retries: retries,
	}
	return stage, nil
}

func handleOnBuild(_ context.Context, call Call[OnBuildSpec]) (Stage, error) {
	stage := call.Stage
	if call.Spec.Trigger == nil { return stage, errors.New("ONBUILD trigger is required") }
	switch call.Spec.Trigger.Kind() {
	case KindFrom, KindResetStage, KindMaintainer, KindOnBuild:
		return call.Stage, fmt.Errorf("%s cannot be used as an ONBUILD trigger", call.Spec.Trigger.Name())
	}
	expression, err := call.Spec.Trigger.OnBuildExpression()
	if err != nil { return call.Stage, err }
	stage.Triggers = append(stage.Triggers, call.Spec.Trigger)
	stage.Image.Config.OnBuild = append(stage.Image.Config.OnBuild, expression)
	return stage, nil
}

func commandArgs(command Command, configuredShell []string, osName string) ([]string, error) {
	switch value := command.(type) {
	case ExecForm:
		if len(value) == 0 { return nil, errors.New("exec form cannot be empty") }
		return append([]string(nil), value...), nil
	case ShellForm:
		shell := configuredShell
		if len(shell) == 0 { shell = defaultShell(osName) }
		return append(append([]string(nil), shell...), string(value)), nil
	default:
		return nil, fmt.Errorf("unsupported command form %T", command)
	}
}
func defaultShell(osName string) []string {
	if strings.EqualFold(osName, "windows") { return []string{"cmd", "/S", "/C"} }
	return []string{"/bin/sh", "-c"}
}
func setEnv(values []string, key, value string) []string {
	result := make([]string, 0, len(values)+1)
	replaced := false
	for _, item := range values {
		current, _, _ := strings.Cut(item, "=")
		if current == key {
			if !replaced { result = append(result, key+"="+value); replaced = true }
			continue
		}
		result = append(result, item)
	}
	if !replaced { result = append(result, key+"="+value) }
	return result
}
func cloneStringMap(source map[string]string) map[string]string {
	if source == nil { return map[string]string{} }
	result := make(map[string]string, len(source))
	for key, value := range source { result[key] = value }
	return result
}
func ensureStringMap(source map[string]string) map[string]string {
	if source == nil { return make(map[string]string) }
	return source
}
func rootPath(source string) string {
	source = strings.ReplaceAll(source, "\\", "/")
	return path.Clean("/" + strings.TrimPrefix(source, "/"))
}
func preservedSourcePath(source string) string {
	source = strings.ReplaceAll(source, "\\", "/")
	if index := strings.Index(source, "/./"); index >= 0 { source = source[index+3:] } else { source = strings.TrimPrefix(source, "./") }
	cleaned := strings.TrimPrefix(path.Clean(source), "/")
	for strings.HasPrefix(cleaned, "../") { cleaned = strings.TrimPrefix(cleaned, "../") }
	if cleaned == "." || cleaned == "" || cleaned == ".." { return path.Base(source) }
	return cleaned
}


func normalizeMountTarget(runtime *Runtime, stage Stage, raw string) (string, error) {
	if strings.TrimSpace(raw) == "" { return "", errors.New("mount target is required") }
	value, err := runtime.Expand(raw, stage)
	if err != nil { return "", err }
	if value == "" { return "", errors.New("expanded mount target is empty") }
	normalized, err := system.NormalizePath(stage.Image.Config.WorkingDir, value, stage.Platform.OS, true)
	if err != nil { return "", err }
	if normalized == "/" { return "", errors.New("mount target cannot be the root directory") }
	return normalized, nil
}

func buildkitCustomName(runtime *Runtime, stage Stage, title string) string {
	parts := make([]string, 0, 3)
	if stage.Platform.OS != "" && stage.Platform.Architecture != "" {
		platform := stage.Platform.OS + "/" + stage.Platform.Architecture
		if stage.Platform.Variant != "" { platform += "/" + stage.Platform.Variant }
		parts = append(parts, platform)
	}
	stageName := stage.Name
	if stageName == "" && runtime != nil { stageName = runtime.StageName }
	if stageName != "" { parts = append(parts, string(stageName)) }
	prefix := "[" + strings.Join(parts, " ")
	if runtime != nil && runtime.InstructionIndex > 0 {
		total := runtime.InstructionTotal
		if total <= 0 { total = runtime.InstructionIndex }
		if len(parts) > 0 { prefix += " " }
		prefix += fmt.Sprintf("%*d/%d", len(strconv.Itoa(total)), runtime.InstructionIndex, total)
	}
	prefix += "] "
	if runtime != nil && runtime.OnBuild { prefix += "ONBUILD " }
	return prefix + title
}

func combinedConstraints(runtime *Runtime, instruction ...llb.ConstraintsOpt) []llb.ConstraintsOpt {
	var result []llb.ConstraintsOpt
	if runtime != nil {
		result = append(result, runtime.Constraints...)
		result = append(result, runtime.StageConstraints...)
	}
	result = append(result, instruction...)
	return result
}

func operationConstraints(runtime *Runtime, stage Stage, common Common, fallback string) []llb.ConstraintsOpt {
	constraints := combinedConstraints(runtime, common.Constraints...)
	name := common.CustomName
	if name == "" { name = fallback }
	return append(constraints, llb.WithCustomName(buildkitCustomName(runtime, stage, name)))
}
