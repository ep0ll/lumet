package instruction

import (
	"errors"
	"fmt"
	"os"
	"path"
	"strings"

	"github.com/moby/buildkit/client/llb"
	"github.com/moby/buildkit/util/system"
)

// Mount is a RUN filesystem or credential mount.
type Mount interface{ isMount() }

type BindMount struct {
	Target    string
	From      CopySource
	Source    string
	ReadWrite bool
}
type CacheMount struct {
	Target   string
	ID       string
	From     CopySource
	Source   string
	Sharing  llb.CacheMountSharingMode
	ReadOnly bool
	Mode     os.FileMode
	UID      int
	GID      int
}
type TmpfsMount struct {
	Target string
	Size   int64
}
type SecretMount struct {
	ID       string
	Target   string
	Env      string
	Required bool
	Mode     os.FileMode
	UID      int
	GID      int
}
type SSHMount struct {
	ID       string
	Target   string
	Required bool
	Mode     os.FileMode
	UID      int
	GID      int
}

func (BindMount) isMount()   {}
func (CacheMount) isMount()  {}
func (TmpfsMount) isMount()  {}
func (SecretMount) isMount() {}
func (SSHMount) isMount()    {}

func Bind(target string, source CopySource, sourcePath string, readWrite bool) Mount {
	return BindMount{Target: target, From: cloneCopySource(source), Source: sourcePath, ReadWrite: readWrite}
}

type CacheOption func(*CacheMount)

func Cache(target string, options ...CacheOption) Mount {
	config := CacheMount{Target: target, Sharing: llb.CacheMountShared}
	for _, option := range options {
		if option != nil {
			option(&config)
		}
	}
	return config
}
func CacheID(id string) CacheOption {
	return func(config *CacheMount) { config.ID = id }
}
func CacheFrom(source CopySource, sourcePath string) CacheOption {
	return func(config *CacheMount) {
		config.From = cloneCopySource(source)
		config.Source = sourcePath
	}
}
func CacheSharing(mode llb.CacheMountSharingMode) CacheOption {
	return func(config *CacheMount) { config.Sharing = mode }
}
func CacheReadOnly() CacheOption {
	return func(config *CacheMount) { config.ReadOnly = true }
}
func CacheOwnership(uid, gid int, mode os.FileMode) CacheOption {
	return func(config *CacheMount) {
		config.UID, config.GID, config.Mode = uid, gid, mode
	}
}
func Tmpfs(target string, size int64) Mount {
	return TmpfsMount{Target: target, Size: size}
}

type SecretOption func(*SecretMount)

func Secret(id, target string, options ...SecretOption) Mount {
	config := SecretMount{ID: id, Target: target, Required: true, Mode: 0o400}
	for _, option := range options {
		if option != nil {
			option(&config)
		}
	}
	return config
}
func SecretOptional() SecretOption {
	return func(config *SecretMount) { config.Required = false }
}
func SecretRequired() SecretOption {
	return func(config *SecretMount) { config.Required = true }
}
func SecretEnv(name string) SecretOption {
	return func(config *SecretMount) { config.Env = name }
}
func SecretFile(uid, gid int, mode os.FileMode) SecretOption {
	return func(config *SecretMount) {
		config.UID, config.GID, config.Mode = uid, gid, mode
	}
}

type SSHOption func(*SSHMount)

func SSH(id, target string, options ...SSHOption) Mount {
	config := SSHMount{ID: id, Target: target, Required: false, Mode: 0o600}
	for _, option := range options {
		if option != nil {
			option(&config)
		}
	}
	return config
}
func SSHOptional() SSHOption {
	return func(config *SSHMount) { config.Required = false }
}
func SSHRequired() SSHOption {
	return func(config *SSHMount) { config.Required = true }
}
func SSHSocket(uid, gid int, mode os.FileMode) SSHOption {
	return func(config *SSHMount) {
		config.UID, config.GID, config.Mode = uid, gid, mode
	}
}

func runMountOptions(runtime *Runtime, state *State, mount Mount) ([]llb.RunOption, error) {
	if runtime == nil {
		return nil, errors.New("RUN mount requires a build runtime")
	}
	switch value := mount.(type) {
	case BindMount:
		target, err := normalizeMountTarget(runtime, state, value.Target)
		if err != nil {
			return nil, fmt.Errorf("bind mount target: %w", err)
		}
		source, err := resolveCopySource(runtime, state, value.From)
		if err != nil {
			return nil, fmt.Errorf("bind mount source: %w", err)
		}
		options := make([]llb.MountOption, 0, 2)
		if value.Source != "" {
			sourcePath, err := runtime.Expand(value.Source, state)
			if err != nil {
				return nil, err
			}
			options = append(options, llb.SourcePath(rootPath(sourcePath)))
		}
		if !value.ReadWrite {
			options = append(options, llb.Readonly)
		} else if runtime.BindMountNoOutput {
			options = append(options, llb.ForceNoOutput)
		}
		return []llb.RunOption{llb.AddMount(target, source, options...)}, nil

	case CacheMount:
		target, err := normalizeMountTarget(runtime, state, value.Target)
		if err != nil {
			return nil, fmt.Errorf("cache mount target: %w", err)
		}
		source := llb.Scratch()
		if value.From != nil {
			source, err = resolveCopySource(runtime, state, value.From)
			if err != nil {
				return nil, fmt.Errorf("cache mount source: %w", err)
			}
		}
		sourcePath := value.Source
		if sourcePath != "" {
			sourcePath, err = runtime.Expand(sourcePath, state)
			if err != nil {
				return nil, err
			}
			sourcePath = rootPath(sourcePath)
		}
		if value.UID != 0 || value.GID != 0 || value.Mode != 0 {
			mode := value.Mode
			if mode == 0 {
				mode = 0o755
			}
			ownership := llb.WithUIDGID(value.UID, value.GID).(llb.ChownOpt)
			if sourcePath == "" {
				sourcePath = "/cache"
			}
			source = source.File(
				llb.Mkdir(sourcePath, mode, llb.WithParents(true), ownership),
				RuntimeConstraints(runtime, InstructionMetadata{})...,
			)
		}
		id := value.ID
		if id != "" {
			id, err = runtime.Expand(id, state)
			if err != nil {
				return nil, err
			}
		}
		if id == "" {
			id = strings.TrimPrefix(path.Clean(target), "/")
		}
		if runtime.CacheIDNamespace != "" {
			id = path.Join(runtime.CacheIDNamespace, id)
		}
		if id == "" || id == "." {
			return nil, errors.New("cache mount ID cannot be derived from its target")
		}
		options := []llb.MountOption{llb.AsPersistentCacheDir(id, value.Sharing)}
		if sourcePath != "" {
			options = append(options, llb.SourcePath(sourcePath))
		}
		if value.ReadOnly {
			options = append(options, llb.Readonly)
		}
		return []llb.RunOption{llb.AddMount(target, source, options...)}, nil

	case TmpfsMount:
		target, err := normalizeMountTarget(runtime, state, value.Target)
		if err != nil {
			return nil, fmt.Errorf("tmpfs mount target: %w", err)
		}
		if value.Size < 0 {
			return nil, errors.New("tmpfs mount size cannot be negative")
		}
		return []llb.RunOption{
			llb.AddMount(target, llb.Scratch(), llb.Tmpfs(llb.TmpfsSize(value.Size))),
		}, nil

	case SecretMount:
		id := value.ID
		var err error
		if id != "" {
			id, err = runtime.Expand(id, state)
			if err != nil {
				return nil, err
			}
		}
		target := value.Target
		if target != "" {
			target, err = normalizeMountTarget(runtime, state, target)
			if err != nil {
				return nil, fmt.Errorf("secret mount target: %w", err)
			}
		}
		if id == "" && target != "" {
			id = path.Base(target)
		}
		if id == "" {
			return nil, errors.New("secret mount requires an ID or target")
		}

		var destination *string
		if target != "" {
			destination = &target
		} else if value.Env == "" {
			defaultTarget := "/run/secrets/" + id
			destination = &defaultTarget
		}
		options := []llb.SecretOption{
			llb.SecretID(id),
			llb.SecretFileOpt(value.UID, value.GID, int(modeOr(value.Mode, 0o400))),
		}
		if !value.Required {
			options = append(options, llb.SecretOptional)
		}
		if value.Env != "" {
			envName, err := runtime.Expand(value.Env, state)
			if err != nil {
				return nil, err
			}
			options = append(options, llb.SecretAsEnvName(envName))
		}
		return []llb.RunOption{llb.AddSecretWithDest(id, destination, options...)}, nil

	case SSHMount:
		id := value.ID
		var err error
		if id != "" {
			id, err = runtime.Expand(id, state)
			if err != nil {
				return nil, err
			}
		}
		if id == "" {
			id = "default"
		}
		target := value.Target
		if target != "" {
			target, err = normalizeMountTarget(runtime, state, target)
			if err != nil {
				return nil, fmt.Errorf("SSH mount target: %w", err)
			}
		}
		options := []llb.SSHOption{llb.SSHID(id)}
		if target != "" {
			options = append(options, llb.SSHSocketTarget(target))
		}
		if !value.Required {
			options = append(options, llb.SSHOptional)
		}
		if value.UID != 0 || value.GID != 0 || value.Mode != 0 {
			options = append(options, llb.SSHSocketOpt(target, value.UID, value.GID, int(modeOr(value.Mode, 0o600))))
		}
		return []llb.RunOption{llb.AddSSHSocket(options...)}, nil

	default:
		return nil, fmt.Errorf("unsupported RUN mount type %T", mount)
	}
}

func normalizeMountTarget(runtime *Runtime, state *State, raw string) (string, error) {
	if strings.TrimSpace(raw) == "" {
		return "", errors.New("mount target is required")
	}
	value, err := runtime.Expand(raw, state)
	if err != nil {
		return "", err
	}
	if value == "" {
		return "", errors.New("expanded mount target is empty")
	}
	target, err := system.NormalizePath(state.Image.Config.WorkingDir, value, state.Platform.OS, true)
	if err != nil {
		return "", err
	}
	if target == "/" {
		return "", errors.New("mount target cannot be the root directory")
	}
	return target, nil
}

func mountDependencies(mount Mount) []StageName {
	switch value := mount.(type) {
	case BindMount:
		if stage, ok := value.From.(StageRef); ok && !strings.Contains(string(stage.Name), "$") {
			return []StageName{stage.Name}
		}
	case CacheMount:
		if stage, ok := value.From.(StageRef); ok && !strings.Contains(string(stage.Name), "$") {
			return []StageName{stage.Name}
		}
	}
	return nil
}

func modeOr(value, fallback os.FileMode) os.FileMode {
	if value == 0 {
		return fallback
	}
	return value
}
