package dockerfile

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"os"
	"path"
	"strings"

	"github.com/moby/buildkit/client/llb"
	"github.com/moby/buildkit/util/system"
	digest "github.com/opencontainers/go-digest"
)

// AddSource is a typed ADD input. LocalPath, URL and GitRepo keep source
// interpretation explicit rather than inferring it from path strings.
type AddSource interface{ isAddSource() }
type LocalPath string
type URL string
type GitRepo struct {
	URL    string
	Ref    string
	Subdir string
}

func (LocalPath) isAddSource() {}
func (URL) isAddSource()       {}
func (GitRepo) isAddSource()   {}

type addConfig struct {
	fileOptions FileOptions
	keepGitDir  bool
	checksum    digest.Digest
	unpack      *bool
}

type AddOption func(*addConfig)

func AddKeepGitDir() AddOption {
	return func(config *addConfig) { config.keepGitDir = true }
}
func AddChecksum(value digest.Digest) AddOption {
	return func(config *addConfig) { config.checksum = value }
}
func AddUnpack(enabled bool) AddOption {
	return func(config *addConfig) {
		copy := enabled
		config.unpack = &copy
	}
}
func AddChown(user string) AddOption {
	return func(config *addConfig) { config.fileOptions.ChownUser = user }
}
func AddChownIDs(uid, gid int) AddOption {
	return func(config *addConfig) {
		value := llb.WithUIDGID(uid, gid).(llb.ChownOpt)
		config.fileOptions.Chown = &value
	}
}
func AddChmod(mode os.FileMode) AddOption {
	return func(config *addConfig) {
		config.fileOptions.Chmod = &llb.ChmodOpt{Mode: mode}
	}
}
func AddChmodSymbolic(mode string) AddOption {
	return func(config *addConfig) {
		config.fileOptions.Chmod = &llb.ChmodOpt{ModeStr: mode}
	}
}
func AddExclude(patterns ...string) AddOption {
	return func(config *addConfig) {
		config.fileOptions.Exclude = append(config.fileOptions.Exclude, patterns...)
	}
}
func AddLink() AddOption {
	return func(config *addConfig) { config.fileOptions.Link = true }
}

// Add creates a file operation closure for local, HTTP and Git inputs.
func Add(sources []AddSource, destination string, options ...AddOption) Instruction {
	config := addConfig{}
	for _, option := range options {
		if option != nil {
			option(&config)
		}
	}
	config.fileOptions.Exclude = append([]string(nil), config.fileOptions.Exclude...)
	sources = append([]AddSource(nil), sources...)
	description := fmt.Sprintf("ADD %v %s", sources, destination)

	instruction := NewInstruction("ADD", func(_ context.Context, state *State, runtime *Runtime, metadata InstructionMetadata) error {
		if runtime == nil {
			return errors.New("ADD requires a build runtime")
		}
		if len(sources) == 0 {
			return errors.New("ADD requires at least one source")
		}
		if destination == "" {
			return errors.New("ADD destination cannot be empty")
		}

		dest, err := runtime.Expand(destination, state)
		if err != nil {
			return fmt.Errorf("expand ADD destination: %w", err)
		}
		dest, err = system.NormalizePath(state.Image.Config.WorkingDir, dest, state.Platform.OS, true)
		if err != nil {
			return fmt.Errorf("normalize ADD destination: %w", err)
		}

		var action *llb.FileAction
		for _, source := range sources {
			sourceState, sourcePath, local, err := resolveAddSource(runtime, state, source, config)
			if err != nil {
				return fmt.Errorf("resolve ADD source %T: %w", source, err)
			}
			unpack := local
			if config.unpack != nil {
				unpack = *config.unpack
			}
			copyOptions, err := expandedCopyOptions(
				makeCopyInfo(config.fileOptions, unpack),
				config.fileOptions, runtime, state,
			)
			if err != nil {
				return err
			}
			if action == nil {
				action = llb.Copy(sourceState, sourcePath, dest, copyOptions...)
			} else {
				action = action.Copy(sourceState, sourcePath, dest, copyOptions...)
			}
		}

		constraints := operationConstraints(runtime, state, metadata, description)
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
			return addOnBuildExpression(sources, destination, config)
		}))
	return instruction
}

func resolveAddSource(
	runtime *Runtime,
	state *State,
	source AddSource,
	config addConfig,
) (llb.State, string, bool, error) {
	switch value := source.(type) {
	case LocalPath:
		expanded, err := runtime.Expand(string(value), state)
		if err != nil {
			return llb.State{}, "", false, err
		}
		if strings.TrimSpace(expanded) == "" {
			return llb.State{}, "", false, errors.New("ADD local path cannot be empty")
		}
		return runtime.BuildContext, rootPath(expanded), true, nil

	case URL:
		raw, err := runtime.Expand(string(value), state)
		if err != nil {
			return llb.State{}, "", false, err
		}
		parsed, err := url.Parse(raw)
		if err != nil {
			return llb.State{}, "", false, fmt.Errorf("parse ADD URL: %w", err)
		}
		if parsed.Scheme != "http" && parsed.Scheme != "https" {
			return llb.State{}, "", false, errors.New("remote ADD URL must use http or https")
		}
		filename := path.Base(parsed.Path)
		if filename == "." || filename == "/" || filename == "" {
			filename = "download"
		}
		options := []llb.HTTPOption{llb.Filename(filename)}
		if config.checksum != "" {
			options = append(options, llb.Checksum(config.checksum))
		}
		return llb.HTTP(raw, options...), "/"+filename, false, nil

	case GitRepo:
		repoURL, err := runtime.Expand(value.URL, state)
		if err != nil {
			return llb.State{}, "", false, err
		}
		ref, err := runtime.Expand(value.Ref, state)
		if err != nil {
			return llb.State{}, "", false, err
		}
		subdir, err := runtime.Expand(value.Subdir, state)
		if err != nil {
			return llb.State{}, "", false, err
		}
		if repoURL == "" {
			return llb.State{}, "", false, errors.New("ADD Git URL cannot be empty")
		}
		options := make([]llb.GitOption, 0, 2)
		if config.keepGitDir {
			options = append(options, llb.KeepGitDir())
		}
		if subdir != "" {
			options = append(options, llb.GitSubDir(subdir))
		}
		return llb.Git(repoURL, ref, options...), "/", false, nil

	default:
		return llb.State{}, "", false, fmt.Errorf("unsupported ADD source type %T", source)
	}
}
