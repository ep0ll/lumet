package instruction

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path"
	"strings"

	"github.com/moby/buildkit/client/llb"
	"github.com/moby/buildkit/util/system"
)

// Sources is the ordered set of COPY source paths.
type Sources []string

type FileOptions struct {
	Chown     *llb.ChownOpt
	ChownUser string
	Chmod     *llb.ChmodOpt
	Exclude   []string
	Link      bool
}

type copyConfig struct {
	fileOptions FileOptions
	from        CopySource
	parents     bool
}

type CopyOption func(*copyConfig)

func CopyFrom(source CopySource) CopyOption {
	source = cloneCopySource(source)
	return func(config *copyConfig) { config.from = source }
}
func Parents() CopyOption {
	return func(config *copyConfig) { config.parents = true }
}
func Chown(user string) CopyOption {
	return func(config *copyConfig) { config.fileOptions.ChownUser = user }
}
func ChownIDs(uid, gid int) CopyOption {
	return func(config *copyConfig) {
		value := llb.WithUIDGID(uid, gid).(llb.ChownOpt)
		config.fileOptions.Chown = &value
	}
}
func Chmod(mode os.FileMode) CopyOption {
	return func(config *copyConfig) {
		config.fileOptions.Chmod = &llb.ChmodOpt{Mode: mode}
	}
}
func ChmodSymbolic(mode string) CopyOption {
	return func(config *copyConfig) {
		config.fileOptions.Chmod = &llb.ChmodOpt{ModeStr: mode}
	}
}
func Exclude(patterns ...string) CopyOption {
	return func(config *copyConfig) {
		config.fileOptions.Exclude = append(config.fileOptions.Exclude, patterns...)
	}
}
func Link() CopyOption {
	return func(config *copyConfig) { config.fileOptions.Link = true }
}

// Copy returns a filesystem transition closure. --from stage references are
// recorded as explicit DAG dependencies as well as resolved at execution time.
func Copy(sources Sources, destination string, options ...CopyOption) Instruction {
	config := copyConfig{}
	for _, option := range options {
		if option != nil {
			option(&config)
		}
	}
	config.fileOptions.Exclude = append([]string(nil), config.fileOptions.Exclude...)
	config.from = cloneCopySource(config.from)
	sources = append(Sources(nil), sources...)
	description := "COPY " + strings.Join(sources, " ") + " " + destination
	dependencies := []StageName{}
	if stage, ok := config.from.(StageRef); ok && !strings.Contains(string(stage.Name), "$") {
		dependencies = append(dependencies, stage.Name)
	}

	instruction := newMutationInstruction("COPY", func(_ context.Context, state *State, runtime *Runtime, metadata InstructionMetadata) error {
		if runtime == nil {
			return errors.New("COPY requires a build runtime")
		}
		if len(sources) == 0 {
			return errors.New("COPY requires at least one source")
		}
		if destination == "" {
			return errors.New("COPY destination cannot be empty")
		}

		dest, err := runtime.Expand(destination, state)
		if err != nil {
			return fmt.Errorf("expand COPY destination: %w", err)
		}
		dest, err = system.NormalizePath(state.Image.Config.WorkingDir, dest, state.Platform.OS, true)
		if err != nil {
			return fmt.Errorf("normalize COPY destination: %w", err)
		}
		source, err := resolveCopySource(runtime, state, config.from)
		if err != nil {
			return fmt.Errorf("resolve COPY source: %w", err)
		}

		info := makeCopyInfo(config.fileOptions, false)
		var action *llb.FileAction
		for _, rawSource := range sources {
			sourcePath, err := runtime.Expand(rawSource, state)
			if err != nil {
				return fmt.Errorf("expand COPY source %q: %w", rawSource, err)
			}
			if sourcePath == "" {
				return errors.New("COPY source cannot be empty")
			}
			target := dest
			if config.parents {
				target = path.Join(dest, preservedSourcePath(sourcePath))
			}
			copyOptions, err := expandedCopyOptions(info, config.fileOptions, runtime, state)
			if err != nil {
				return err
			}
			if action == nil {
				action = llb.Copy(source, rootPath(sourcePath), target, copyOptions...)
			} else {
				action = action.Copy(source, rootPath(sourcePath), target, copyOptions...)
			}
		}

		constraints := OperationConstraints(runtime, state, metadata, description)
		if config.fileOptions.Link {
			copyState := llb.Scratch().Platform(state.Platform).
				Dir(state.Image.Config.WorkingDir).
				File(action, constraints...)
			state.LLB = llb.Merge([]llb.State{state.LLB, copyState}, constraints...)
		} else {
			state.LLB = state.LLB.File(action, constraints...)
		}
		return nil
	}, WithDescription(description),
		WithOnBuildExpression(func() (string, error) {
			return copyOnBuildExpression(sources, destination, config)
		}))
	return instruction.With(DependsOn(dependencies...))
}

func resolveCopySource(runtime *Runtime, state *State, source CopySource) (llb.State, error) {
	if source == nil {
		return runtime.BuildContext, nil
	}
	switch value := source.(type) {
	case StageRef:
		if runtime.ResolveStage == nil {
			return llb.State{}, errors.New("stage resolver is not configured")
		}
		name, err := runtime.Expand(string(value.Name), state)
		if err != nil {
			return llb.State{}, err
		}
		resolved, err := runtime.ResolveStage(StageName(name))
		if err != nil {
			return llb.State{}, err
		}
		return resolved.LLB, nil
	case ContextRef:
		name, err := runtime.Expand(value.Name, state)
		if err != nil {
			return llb.State{}, fmt.Errorf("expand named context reference: %w", err)
		}
		named, ok := runtime.NamedContexts[name]
		if !ok {
			return llb.State{}, fmt.Errorf("named context %q not found", name)
		}
		return named.LLB, nil
	case ImageRef:
		ref, err := runtime.Expand(value.Ref, state)
		if err != nil {
			return llb.State{}, err
		}
		if ref == "" {
			return llb.State{}, errors.New("COPY image source reference cannot be empty")
		}
		options := append([]llb.ImageOption(nil), value.Options...)
		return llb.Image(ref, options...).Platform(state.Platform), nil
	default:
		return llb.State{}, fmt.Errorf("unsupported COPY source type %T", source)
	}
}

func makeCopyInfo(options FileOptions, unpack bool) *llb.CopyInfo {
	return &llb.CopyInfo{
		ChownOpt:        options.Chown,
		Mode:            options.Chmod,
		ExcludePatterns: append([]string(nil), options.Exclude...),
		CreateDestPath:  true,
		AllowWildcard:   true,
		FollowSymlinks:  true,
		AttemptUnpack:   unpack,
	}
}

func expandedCopyOptions(
	info *llb.CopyInfo,
	fileOptions FileOptions,
	runtime *Runtime,
	state *State,
) ([]llb.CopyOption, error) {
	if fileOptions.ChownUser != "" {
		user, err := runtime.Expand(fileOptions.ChownUser, state)
		if err != nil {
			return nil, fmt.Errorf("expand --chown: %w", err)
		}
		if user == "" {
			return nil, errors.New("expanded --chown value cannot be empty")
		}
		value := llb.WithUser(user).(llb.ChownOpt)
		info.ChownOpt = &value
	}
	for index, pattern := range info.ExcludePatterns {
		value, err := runtime.Expand(pattern, state)
		if err != nil {
			return nil, fmt.Errorf("expand --exclude: %w", err)
		}
		info.ExcludePatterns[index] = value
	}
	if info.Mode != nil && info.Mode.ModeStr != "" {
		value, err := runtime.Expand(info.Mode.ModeStr, state)
		if err != nil {
			return nil, fmt.Errorf("expand --chmod: %w", err)
		}
		mode := *info.Mode
		mode.ModeStr = value
		info.Mode = &mode
	}
	return []llb.CopyOption{info}, nil
}

func rootPath(value string) string {
	value = strings.ReplaceAll(value, "\\", "/")
	return path.Clean("/" + strings.TrimPrefix(value, "/"))
}

func preservedSourcePath(value string) string {
	value = strings.ReplaceAll(value, "\\", "/")
	if index := strings.Index(value, "/./"); index >= 0 {
		value = value[index+3:]
	} else {
		value = strings.TrimPrefix(value, "./")
	}
	cleaned := strings.TrimPrefix(path.Clean(value), "/")
	for strings.HasPrefix(cleaned, "../") {
		cleaned = strings.TrimPrefix(cleaned, "../")
	}
	if cleaned == "." || cleaned == "" || cleaned == ".." {
		return path.Base(value)
	}
	return cleaned
}
