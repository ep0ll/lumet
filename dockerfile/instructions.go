package dockerfile

import (
	"context"
	"encoding/json"
	"maps"
	"fmt"
	"os"
	"path"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/moby/buildkit/client/llb"
	"github.com/moby/buildkit/util/system"
	dockerspec "github.com/moby/docker-image-spec/specs-go/v1"
)

// KeyValue represents an ENV or LABEL entry.
type KeyValue struct {
	Key   string
	Value string
}

// KV constructs a key/value pair for ENV and LABEL.
func KV(key, value string) KeyValue { return KeyValue{Key: key, Value: value} }

// Command is a shell-form or exec-form command.
type Command interface {
	commandForm()
}

// ShellForm is a command string interpreted by the image's configured shell.
type ShellForm string

// ExecForm is an argv-style command that is executed directly.
type ExecForm []string

func (ShellForm) commandForm() {}
func (ExecForm) commandForm()  {}

// Sh creates a shell-form command.
func Sh(command string) ShellForm { return ShellForm(command) }

// Exec creates an exec-form command.
func Exec(args ...string) ExecForm { return ExecForm(args) }

// From starts a build stage from an image, scratch, a previous stage, a named
// context or an explicit LLB state. The returned function applies the stage.
func From(reference string, opts ...Option) Instruction {
	options := collectOptions(opts)
	return func(ctx context.Context, b *Build) error {
		if b == nil {
			return fmt.Errorf("FROM: nil build")
		}
		b.saveStage()

		ref := b.expandGlobal(reference)
		if ref == "" && options.BaseState == nil {
			return fmt.Errorf("FROM: image reference cannot be empty")
		}
		if options.StageName != "" {
			if _, exists := b.lookupStage(options.StageName); exists {
				return fmt.Errorf("FROM: duplicate stage name %q", options.StageName)
			}
		}
		platform := clonePlatform(b.Platform)
		if options.Platform != nil {
			platform = clonePlatform(*options.Platform)
		}

		var (
			state         llb.State
			image         = emptyImage(platform)
			inheritedArgs = make(map[string]string)
			inheritedEnv  map[string]string
		)
		switch {
		case options.BaseState != nil:
			state = *options.BaseState
		case strings.EqualFold(ref, "scratch"):
			state = llb.Scratch()
		default:
			if stage, ok := b.lookupStage(ref); ok {
				state = stage.State
				image = cloneImage(stage.Image)
				inheritedArgs = maps.Clone(stage.Args)
				inheritedEnv = maps.Clone(stage.BuildEnv)
				if options.Platform == nil {
					platform = clonePlatform(stage.Platform)
				}
			} else if named, ok := b.namedContexts[ref]; ok {
				state = named
			} else {
				state = llb.Image(ref, options.ImageOptions...)
				if b.ImageResolver != nil {
					resolved, err := b.ImageResolver(ctx, ref, platform)
					if err != nil {
						return fmt.Errorf("resolve base image %q: %w", ref, err)
					}
					image = cloneImage(resolved)
					config, err := json.Marshal(resolved)
					if err != nil {
						return fmt.Errorf("encode base image config %q: %w", ref, err)
					}
					state, err = state.WithImageConfig(config)
					if err != nil {
						return fmt.Errorf("apply base image config %q: %w", ref, err)
					}
				}
			}
		}

		if inheritedEnv == nil {
			inheritedEnv = environmentFromConfig(image.Config.Env)
		}

		image.OS = platform.OS
		image.Architecture = platform.Architecture
		image.Variant = platform.Variant
		image.OSVersion = platform.OSVersion
		image.OSFeatures = append([]string(nil), platform.OSFeatures...)

		state = state.Platform(platform)
		if image.Config.User != "" {
			state = state.User(image.Config.User)
		}

		b.State = state
		b.Image = image
		b.Platform = platform
		b.stageArgs = inheritedArgs
		b.stageEnv = inheritedEnv
		b.stageName = options.StageName
		b.started = true
		return nil
	}
}

// Arg declares a build-time argument. A build value supplied with
// WithBuildArgs takes precedence over the optional default.
func Arg(name string, opts ...Option) Instruction {
	options := collectOptions(opts)
	return func(ctx context.Context, b *Build) error {
		if b == nil {
			return fmt.Errorf("ARG: nil build")
		}
		if name == "" {
			return fmt.Errorf("ARG: name cannot be empty")
		}
		value, exists := b.BuildArgs[name]
		if !exists && options.ArgDefault != nil {
			if b.started {
				value = b.expand(*options.ArgDefault)
			} else {
				value = b.expandGlobal(*options.ArgDefault)
			}
			exists = true
		}
		if b.started && !exists {
			value, exists = b.stageArgs[name]
		}
		if b.started && !exists {
			value, exists = b.globalArgs[name]
		}
		if !exists && !b.started {
			value = ""
		}
		if b.started {
			if exists {
				b.stageArgs[name] = value
				b.stageEnv[name] = value
				b.State = b.State.AddEnv(name, value)
			}
		} else {
			b.globalArgs[name] = value
		}
		return b.recordHistory(ctx, "ARG "+name, true)
	}
}

// Env sets environment variables in both the LLB state and image config.
func Env(values ...KeyValue) Instruction {
	return func(ctx context.Context, b *Build) error {
		if err := b.requireStage("ENV"); err != nil {
			return err
		}
		parts := make([]string, 0, len(values))
		// Dockerfile variable replacement for a single ENV instruction uses
		// the environment that existed before the instruction is applied.
		previousEnv := maps.Clone(b.stageEnv)
		for _, item := range values {
			if item.Key == "" {
				return fmt.Errorf("ENV: key cannot be empty")
			}
			value := expandVariables(item.Value, previousEnv, nil)
			b.State = b.State.AddEnv(item.Key, value)
			b.Image.Config.Env = setEnv(b.Image.Config.Env, item.Key, value)
			b.stageEnv[item.Key] = value
			parts = append(parts, item.Key+"="+value)
		}
		return b.recordHistory(ctx, "ENV "+strings.Join(parts, " "), true)
	}
}

// Label adds image labels.
func Label(values ...KeyValue) Instruction {
	return func(ctx context.Context, b *Build) error {
		if err := b.requireStage("LABEL"); err != nil {
			return err
		}
		if b.Image.Config.Labels == nil {
			b.Image.Config.Labels = make(map[string]string, len(values))
		}
		parts := make([]string, 0, len(values))
		for _, item := range values {
			if item.Key == "" {
				return fmt.Errorf("LABEL: key cannot be empty")
			}
			value := b.expand(item.Value)
			b.Image.Config.Labels[item.Key] = value
			parts = append(parts, item.Key+"="+value)
		}
		return b.recordHistory(ctx, "LABEL "+strings.Join(parts, " "), true)
	}
}

// Maintainer sets the deprecated Docker image author metadata.
func Maintainer(name string) Instruction {
	return func(ctx context.Context, b *Build) error {
		if err := b.requireStage("MAINTAINER"); err != nil {
			return err
		}
		b.Image.Author = name
		return b.recordHistory(ctx, "MAINTAINER "+name, true)
	}
}

// Cmd sets the image's default command.
func Cmd(command Command) Instruction {
	return func(ctx context.Context, b *Build) error {
		if err := b.requireStage("CMD"); err != nil {
			return err
		}
		args, err := commandArgs(command, b.Image.Config.Shell, b.Platform.OS)
		if err != nil {
			return fmt.Errorf("CMD: %w", err)
		}
		b.Image.Config.Cmd = args
		return b.recordHistory(ctx, "CMD "+formatCommand(command), true)
	}
}

// Entrypoint sets the image's entrypoint.
func Entrypoint(command Command) Instruction {
	return func(ctx context.Context, b *Build) error {
		if err := b.requireStage("ENTRYPOINT"); err != nil {
			return err
		}
		args, err := commandArgs(command, b.Image.Config.Shell, b.Platform.OS)
		if err != nil {
			return fmt.Errorf("ENTRYPOINT: %w", err)
		}
		b.Image.Config.Entrypoint = args
		return b.recordHistory(ctx, "ENTRYPOINT "+formatCommand(command), true)
	}
}

// Shell changes the shell used for shell-form RUN, CMD and ENTRYPOINT.
func Shell(args ...string) Instruction {
	return func(ctx context.Context, b *Build) error {
		if err := b.requireStage("SHELL"); err != nil {
			return err
		}
		if len(args) == 0 {
			return fmt.Errorf("SHELL: at least one executable argument is required")
		}
		b.Image.Config.Shell = append([]string(nil), args...)
		return b.recordHistory(ctx, "SHELL "+strings.Join(args, " "), true)
	}
}

// Expose declares one or more container ports. Ports may include a protocol
// suffix and a range, for example 8080/tcp or 8000-8005/udp.
func Expose(ports ...string) Instruction {
	return func(ctx context.Context, b *Build) error {
		if err := b.requireStage("EXPOSE"); err != nil {
			return err
		}
		if b.Image.Config.ExposedPorts == nil {
			b.Image.Config.ExposedPorts = make(map[string]struct{})
		}
		for _, value := range ports {
			expanded, err := expandPorts(b.expand(value))
			if err != nil {
				return fmt.Errorf("EXPOSE: %w", err)
			}
			for _, port := range expanded {
				b.Image.Config.ExposedPorts[port] = struct{}{}
			}
		}
		return b.recordHistory(ctx, "EXPOSE "+strings.Join(ports, " "), true)
	}
}

// Volume declares one or more persistent data paths in the image config.
func Volume(paths ...string) Instruction {
	return func(ctx context.Context, b *Build) error {
		if err := b.requireStage("VOLUME"); err != nil {
			return err
		}
		if b.Image.Config.Volumes == nil {
			b.Image.Config.Volumes = make(map[string]struct{}, len(paths))
		}
		for _, value := range paths {
			if value == "" {
				return fmt.Errorf("VOLUME: path cannot be empty")
			}
			value = b.expand(value)
			if !strings.HasPrefix(value, "/") {
				value = path.Join(b.Image.Config.WorkingDir, value)
			}
			b.Image.Config.Volumes[path.Clean(value)] = struct{}{}
		}
		return b.recordHistory(ctx, "VOLUME "+strings.Join(paths, " "), true)
	}
}

// User sets the default process user in the LLB state and image config.
// Values such as user, uid, user:group and uid:gid are passed through to BuildKit.
func User(user string) Instruction {
	return func(ctx context.Context, b *Build) error {
		if err := b.requireStage("USER"); err != nil {
			return err
		}
		if user == "" {
			return fmt.Errorf("USER: user cannot be empty")
		}
		user = b.expand(user)
		b.State = b.State.User(user)
		b.Image.Config.User = user
		return b.recordHistory(ctx, "USER "+user, true)
	}
}

// Workdir changes the working directory and creates it in the filesystem.
func Workdir(directory string) Instruction {
	return func(ctx context.Context, b *Build) error {
		if err := b.requireStage("WORKDIR"); err != nil {
			return err
		}
		directory = b.expand(directory)
		normalized, err := system.NormalizeWorkdir(b.Image.Config.WorkingDir, directory, b.Platform.OS)
		if err != nil {
			return fmt.Errorf("WORKDIR: normalize path: %w", err)
		}
		b.Image.Config.WorkingDir = normalized
		slashPath := system.ToSlash(normalized, b.Platform.OS)
		b.State = b.State.Dir(slashPath)
		if slashPath != "/" {
			mkdirOptions := []llb.MkdirOption{llb.WithParents(true)}
			if b.Image.Config.User != "" {
				mkdirOptions = append(mkdirOptions, llb.WithUser(b.Image.Config.User))
			}
			if b.createdAt != nil {
				mkdirOptions = append(mkdirOptions, llb.WithCreatedTime(*b.createdAt))
			}
			b.State = b.State.File(llb.Mkdir(slashPath, 0o755, mkdirOptions...), b.constraints...)
		}
		return b.recordHistory(ctx, "WORKDIR "+normalized, slashPath != "/")
	}
}

// StopSignal sets the image's stop signal. It accepts any standard-library
// os.Signal implementation; syscall.Signal values are serialized numerically,
// avoiding a package-maintained list of platform-specific signal constants.
func StopSignal(signal os.Signal) Instruction {
	return func(ctx context.Context, b *Build) error {
		if err := b.requireStage("STOPSIGNAL"); err != nil {
			return err
		}
		if signal == nil {
			return fmt.Errorf("STOPSIGNAL: signal cannot be nil")
		}
		value := signal.String()
		if number, ok := signal.(syscall.Signal); ok && number > 0 {
			value = strconv.Itoa(int(number))
		}
		b.Image.Config.StopSignal = value
		return b.recordHistory(ctx, "STOPSIGNAL "+value, true)
	}
}

// Healthcheck sets the image health check. Use HealthcheckNone to disable an
// inherited health check.
func Healthcheck(command Command, opts ...Option) Instruction {
	options := collectOptions(opts)
	return func(ctx context.Context, b *Build) error {
		if err := b.requireStage("HEALTHCHECK"); err != nil {
			return err
		}
		var test []string
		switch command := command.(type) {
		case ExecForm:
			if len(command) == 0 {
				return fmt.Errorf("HEALTHCHECK: exec form cannot be empty")
			}
			test = append([]string{"CMD"}, []string(command)...)
		case ShellForm:
			if strings.TrimSpace(string(command)) == "" {
				return fmt.Errorf("HEALTHCHECK: command cannot be empty")
			}
			test = []string{"CMD-SHELL", string(command)}
		default:
			return fmt.Errorf("HEALTHCHECK: unsupported command form %T", command)
		}
		if options.HealthInterval < 0 || options.HealthTimeout < 0 ||
			options.HealthStartPeriod < 0 || options.HealthStartInterval < 0 ||
			options.HealthRetries < 0 {
			return fmt.Errorf("HEALTHCHECK: durations and retries cannot be negative")
		}

		interval := options.HealthInterval
		if interval == 0 {
			interval = 30 * time.Second
		}
		timeout := options.HealthTimeout
		if timeout == 0 {
			timeout = 30 * time.Second
		}
		startPeriod := options.HealthStartPeriod
		startInterval := options.HealthStartInterval
		if startInterval == 0 {
			startInterval = 5 * time.Second
		}
		retries := options.HealthRetries
		if retries == 0 {
			retries = 3
		}
		if interval < time.Millisecond || timeout < time.Millisecond ||
			startInterval < time.Millisecond ||
			(startPeriod > 0 && startPeriod < time.Millisecond) {
			return fmt.Errorf("HEALTHCHECK: non-zero durations must be at least 1ms")
		}

		b.Image.Config.Healthcheck = &dockerspec.HealthcheckConfig{
			Test: test, Interval: interval,
			Timeout: timeout, StartPeriod: startPeriod,
			StartInterval: startInterval, Retries: retries,
		}
		return b.recordHistory(ctx, "HEALTHCHECK "+formatCommand(command), true)
	}
}

// HealthcheckNone disables an inherited health check.
func HealthcheckNone() Instruction {
	return func(ctx context.Context, b *Build) error {
		if err := b.requireStage("HEALTHCHECK"); err != nil {
			return err
		}
		b.Image.Config.Healthcheck = &dockerspec.HealthcheckConfig{Test: []string{"NONE"}}
		return b.recordHistory(ctx, "HEALTHCHECK NONE", true)
	}
}

// OnBuild appends a trigger expression to the image's ONBUILD metadata. The
// expression is stored as Dockerfile text and runs when this image is a base.
func OnBuild(trigger string) Instruction {
	return func(ctx context.Context, b *Build) error {
		if err := b.requireStage("ONBUILD"); err != nil {
			return err
		}
		trigger = strings.TrimSpace(trigger)
		if trigger == "" {
			return fmt.Errorf("ONBUILD: trigger cannot be empty")
		}
		opcode := strings.ToUpper(strings.Fields(trigger)[0])
		switch opcode {
		case "FROM", "MAINTAINER", "ONBUILD":
			return fmt.Errorf("ONBUILD: %s cannot be used as a trigger", opcode)
		}
		b.Image.Config.OnBuild = append(b.Image.Config.OnBuild, trigger)
		return b.recordHistory(ctx, "ONBUILD "+trigger, true)
	}
}

func commandArgs(command Command, configuredShell []string, osName string) ([]string, error) {
	switch value := command.(type) {
	case ExecForm:
		if len(value) == 0 {
			return nil, fmt.Errorf("exec form cannot be empty")
		}
		return append([]string(nil), value...), nil
	case ShellForm:
		shell := configuredShell
		if len(shell) == 0 {
			shell = defaultShell(osName)
		}
		if len(shell) == 0 {
			return nil, fmt.Errorf("no default shell for OS %q", osName)
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
		return "[" + strings.Join([]string(value), " ") + "]"
	default:
		return fmt.Sprintf("%v", command)
	}
}

func setEnv(env []string, key, value string) []string {
	result := make([]string, 0, len(env)+1)
	replaced := false
	for _, item := range env {
		currentKey, _, _ := strings.Cut(item, "=")
		if currentKey == key {
			if !replaced {
				result = append(result, key+"="+value)
				replaced = true
			}
			continue
		}
		result = append(result, item)
	}
	if !replaced {
		result = append(result, key+"="+value)
	}
	return result
}

func expandVariables(value string, args map[string]string, env []string) string {
	environment := make(map[string]string, len(args)+len(env))
	for key, current := range args {
		environment[key] = current
	}
	for _, item := range env {
		key, current, found := strings.Cut(item, "=")
		if found {
			environment[key] = current
		}
	}
	return os.Expand(value, func(key string) string {
		return environment[key]
	})
}

func expandPorts(value string) ([]string, error) {
	parts := strings.Split(value, "/")
	if len(parts) > 2 || parts[0] == "" {
		return nil, fmt.Errorf("invalid port %q", value)
	}
	protocol := "tcp"
	if len(parts) == 2 {
		protocol = strings.ToLower(parts[1])
	}
	if protocol != "tcp" && protocol != "udp" && protocol != "sctp" {
		return nil, fmt.Errorf("unsupported protocol in %q", value)
	}

	bounds := strings.Split(parts[0], "-")
	if len(bounds) > 2 {
		return nil, fmt.Errorf("invalid port range %q", value)
	}
	start, err := strconv.Atoi(bounds[0])
	if err != nil || start < 1 || start > 65535 {
		return nil, fmt.Errorf("invalid port %q", value)
	}
	end := start
	if len(bounds) == 2 {
		end, err = strconv.Atoi(bounds[1])
		if err != nil || end < start || end > 65535 {
			return nil, fmt.Errorf("invalid port range %q", value)
		}
	}
	result := make([]string, 0, end-start+1)
	for port := start; port <= end; port++ {
		result = append(result, strconv.Itoa(port)+"/"+protocol)
	}
	return result, nil
}

// Keep the standard-library signal contract explicit at compile time.
var _ os.Signal = syscall.Signal(0)
