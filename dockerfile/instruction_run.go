package dockerfile

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/moby/buildkit/client/llb"
	"github.com/moby/buildkit/solver/pb"
)

// Command is one of the two Docker command forms.
type Command interface{ isCommand() }
type ShellForm string
type ExecForm []string

func (ShellForm) isCommand() {}
func (ExecForm) isCommand()  {}

func Sh(command string) ShellForm { return ShellForm(command) }
func Exec(args ...string) ExecForm {
	return append(ExecForm(nil), args...)
}

type NetworkMode uint8

const (
	NetworkDefault NetworkMode = iota
	NetworkNone
	NetworkHost
)

type SecurityMode uint8

const (
	SecuritySandbox SecurityMode = iota
	SecurityInsecure
)

type Ulimit struct {
	Name llb.UlimitName
	Soft int64
	Hard int64
}

type runConfig struct {
	command      Command
	mounts       []Mount
	network      NetworkMode
	security     SecurityMode
	readOnlyRoot bool
	devices      []llb.CDIDeviceInfo
	ulimits      []Ulimit
	options      []llb.RunOption
	dependencies []StageName
}

type RunOption func(*runConfig)

func Mounts(mounts ...Mount) RunOption {
	return func(config *runConfig) { config.mounts = append(config.mounts, mounts...) }
}
func Network(mode NetworkMode) RunOption {
	return func(config *runConfig) { config.network = mode }
}
func Security(mode SecurityMode) RunOption {
	return func(config *runConfig) { config.security = mode }
}
func ReadOnlyRootFS() RunOption {
	return func(config *runConfig) { config.readOnlyRoot = true }
}
func Devices(devices ...llb.CDIDeviceInfo) RunOption {
	return func(config *runConfig) { config.devices = append(config.devices, devices...) }
}
func Device(name string, optional bool) llb.CDIDeviceInfo {
	return llb.CDIDeviceInfo{Name: name, Optional: optional}
}
func Ulimits(limits ...Ulimit) RunOption {
	return func(config *runConfig) { config.ulimits = append(config.ulimits, limits...) }
}
func LLBRunOptions(options ...llb.RunOption) RunOption {
	return func(config *runConfig) { config.options = append(config.options, options...) }
}

// Run returns an instruction closure. It captures the command and execution
// options, while the evaluator supplies the current LLB/image state at Apply.
func Run(command Command, options ...RunOption) Instruction {
	config := runConfig{command: cloneCommand(command)}
	for _, option := range options {
		if option != nil {
			option(&config)
		}
	}
	config.mounts = append([]Mount(nil), config.mounts...)
	for index, mount := range config.mounts {
		config.mounts[index] = cloneMount(mount)
	}
	config.devices = append([]llb.CDIDeviceInfo(nil), config.devices...)
	config.ulimits = append([]Ulimit(nil), config.ulimits...)
	config.options = append([]llb.RunOption(nil), config.options...)
	for _, mount := range config.mounts {
		config.dependencies = append(config.dependencies, mountDependencies(mount)...)
	}
	config.dependencies = uniqueStageNames(config.dependencies)

	description := "RUN " + formatCommand(command)
	instruction := NewInstruction("RUN", func(_ context.Context, state *State, runtime *Runtime, metadata InstructionMetadata) error {
		args, err := commandArgs(config.command, state.Image.Config.Shell, state.Platform.OS)
		if err != nil {
			return err
		}
		options := make([]llb.RunOption, 0, 12+len(config.options)+len(config.mounts))
		options = append(options, llb.Args(args))

		title := metadata.CustomName
		if title == "" {
			title = metadata.ProgressTitle
		}
		if title == "" {
			title = description
		}
		options = append(options, llb.WithCustomName(buildkitCustomName(runtime, state, title)))

		for _, mount := range config.mounts {
			mountOptions, err := runMountOptions(runtime, state, mount)
			if err != nil {
				return err
			}
			options = append(options, mountOptions...)
		}

		switch config.network {
		case NetworkDefault:
		case NetworkNone:
			options = append(options, llb.Network(pb.NetMode_NONE))
		case NetworkHost:
			options = append(options, llb.Network(pb.NetMode_HOST))
		default:
			return fmt.Errorf("unsupported RUN network mode %d", config.network)
		}

		switch config.security {
		case SecuritySandbox:
			// BuildKit's default execution mode is sandboxed; no extra op option is needed.
		case SecurityInsecure:
			options = append(options, llb.Security(pb.SecurityMode_INSECURE))
		default:
			return fmt.Errorf("unsupported RUN security mode %d", config.security)
		}

		for _, device := range config.devices {
			name, err := runtime.Expand(device.Name, state)
			if err != nil {
				return fmt.Errorf("expand RUN device name: %w", err)
			}
			if name == "" {
				return errors.New("RUN device name cannot be empty")
			}
			deviceOptions := []llb.CDIDeviceOption{llb.CDIDeviceName(name)}
			if device.Optional {
				deviceOptions = append(deviceOptions, llb.CDIDeviceOptional)
			}
			options = append(options, llb.AddCDIDevice(deviceOptions...))
		}

		for _, limit := range config.ulimits {
			if limit.Name == "" || limit.Soft < 0 || limit.Hard < 0 || limit.Soft > limit.Hard {
				return fmt.Errorf("invalid RUN ulimit %q: soft=%d hard=%d", limit.Name, limit.Soft, limit.Hard)
			}
			options = append(options, llb.AddUlimit(limit.Name, limit.Soft, limit.Hard))
		}
		if config.readOnlyRoot {
			options = append(options, llb.ReadonlyRootFS())
		}
		options = append(options, config.options...)
		for _, constraint := range runtimeConstraints(runtime, metadata) {
			options = append(options, constraint)
		}
		state.LLB = state.LLB.Run(options...).Root()
		return nil
	}, WithDescription(description),
		WithOnBuildExpression(func() (string, error) {
			return runOnBuildExpression(config)
		}))
	return instruction.With(DependsOn(config.dependencies...))
}

func commandArgs(command Command, configuredShell []string, osName string) ([]string, error) {
	switch value := command.(type) {
	case ExecForm:
		if len(value) == 0 {
			return nil, errors.New("RUN exec form cannot be empty")
		}
		return append([]string(nil), value...), nil
	case ShellForm:
		if strings.TrimSpace(string(value)) == "" {
			return nil, errors.New("RUN shell command cannot be empty")
		}
		shell := configuredShell
		if len(shell) == 0 {
			shell = defaultShell(osName)
		}
		return append(append([]string(nil), shell...), string(value)), nil
	default:
		return nil, fmt.Errorf("unsupported command form %T", command)
	}
}

func defaultShell(osName string) []string {
	if strings.EqualFold(osName, "windows") {
		return []string{"cmd", "/S", "/C"}
	}
	return []string{"/bin/sh", "-c"}
}

func formatCommand(command Command) string {
	switch value := command.(type) {
	case ShellForm:
		return string(value)
	case ExecForm:
		encoded, _ := json.Marshal([]string(value))
		return string(encoded)
	default:
		return fmt.Sprintf("%v", command)
	}
}
