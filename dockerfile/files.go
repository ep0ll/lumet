package dockerfile

import (
	"context"
	"fmt"
	"net/url"
	"path"
	"strings"

	"github.com/moby/buildkit/client/llb"
)

// Copy copies one or more paths from the build context, a named stage, a named
// context or an explicit LLB state into the current root filesystem.
func Copy(sources []string, destination string, opts ...Option) Instruction {
	options := collectOptions(opts)
	return func(ctx context.Context, b *Build) error {
		if err := b.requireStage("COPY"); err != nil {
			return err
		}
		if len(sources) == 0 {
			return fmt.Errorf("COPY: at least one source is required")
		}
		destination = b.expand(destination)
		if destination == "" {
			return fmt.Errorf("COPY: destination cannot be empty")
		}

		sourceState, err := resolveCopySource(b, options)
		if err != nil {
			return fmt.Errorf("COPY: %w", err)
		}
		copyOpts, err := copyOptions(b, options, false)
		if err != nil {
			return fmt.Errorf("COPY: %w", err)
		}
		var action *llb.FileAction
		for _, source := range sources {
			source = b.expand(source)
			if source == "" {
				return fmt.Errorf("COPY: expanded source path cannot be empty")
			}
			sourcePath := rootPath(source)
			target := destination
			if options.Parents {
				target = path.Join(destination, preservedSourcePath(source))
			}
			if action == nil {
				action = llb.Copy(sourceState, sourcePath, target, copyOpts...)
			} else {
				action = action.Copy(sourceState, sourcePath, target, copyOpts...)
			}
		}

		constraints := combineConstraints(b, options)
		if options.Link {
			copyState := llb.Scratch().Platform(b.Platform).Dir(b.Image.Config.WorkingDir)
			copyState = copyState.File(action, constraints...)
			b.State = llb.Merge([]llb.State{b.State, copyState}, constraints...)
		} else {
			b.State = b.State.File(action, constraints...)
		}
		return b.recordHistory(ctx, "COPY "+strings.Join(sources, " ")+" "+destination, false)
	}
}

// Add adds local, HTTP(S) or Git sources. Unlike COPY, ADD enables BuildKit's
// archive unpack behavior by default. Use Unpack(false) to disable extraction.
func Add(sources []AddSource, destination string, opts ...Option) Instruction {
	options := collectOptions(opts)
	return func(ctx context.Context, b *Build) error {
		if err := b.requireStage("ADD"); err != nil {
			return err
		}
		if len(sources) == 0 {
			return fmt.Errorf("ADD: at least one source is required")
		}
		destination = b.expand(destination)
		if destination == "" {
			return fmt.Errorf("ADD: destination cannot be empty")
		}

		var action *llb.FileAction
		for _, source := range sources {
			state, sourcePath, err := resolveAddSource(b, source, options)
			if err != nil {
				return fmt.Errorf("ADD %q: %w", source.Value, err)
			}

			// Docker's implicit archive extraction applies to local ADD sources.
			// Remote URLs remain files unless the caller explicitly opts in.
			unpack := source.Kind == AddLocal
			if options.Unpack != nil {
				unpack = *options.Unpack
			}
			copyOpts, err := copyOptions(b, options, unpack)
			if err != nil {
				return fmt.Errorf("ADD: %w", err)
			}
			if action == nil {
				action = llb.Copy(state, sourcePath, destination, copyOpts...)
			} else {
				action = action.Copy(state, sourcePath, destination, copyOpts...)
			}
		}
		b.State = b.State.File(action, combineConstraints(b, options)...)
		descriptions := make([]string, 0, len(sources))
		for _, source := range sources {
			descriptions = append(descriptions, source.Value)
		}
		return b.recordHistory(ctx, "ADD "+strings.Join(descriptions, " ")+" "+destination, false)
	}
}

func resolveCopySource(b *Build, options InstructionOptions) (llb.State, error) {
	if options.FromState != nil {
		return *options.FromState, nil
	}
	if options.From == "" {
		return b.BuildContext, nil
	}
	if stage, ok := b.lookupStage(options.From); ok {
		return stage.State, nil
	}
	if state, ok := b.namedContexts[options.From]; ok {
		return state, nil
	}
	// Dockerfile COPY --from also accepts an image reference, not only a
	// previously declared stage or named context.
	return llb.Image(options.From, options.ImageOptions...).Platform(b.Platform), nil
}

func resolveAddSource(b *Build, source AddSource, options InstructionOptions) (llb.State, string, error) {
	source.Value = b.expand(source.Value)
	source.Ref = b.expand(source.Ref)
	source.Subdir = b.expand(source.Subdir)
	switch source.Kind {
	case AddLocal:
		if source.Value == "" {
			return llb.State{}, "", fmt.Errorf("local source path cannot be empty")
		}
		return b.BuildContext, rootPath(source.Value), nil
	case AddRemote:
		if source.Value == "" {
			return llb.State{}, "", fmt.Errorf("remote URL cannot be empty")
		}
		parsed, err := url.Parse(source.Value)
		if err != nil {
			return llb.State{}, "", fmt.Errorf("parse remote URL: %w", err)
		}
		scheme := strings.ToLower(parsed.Scheme)
		if scheme != "http" && scheme != "https" {
			return llb.State{}, "", fmt.Errorf("ADD remote source must be an HTTP(S) URL")
		}
		filename := path.Base(parsed.Path)
		if filename == "." || filename == "/" || filename == "" {
			filename = "download"
		}
		httpOptions := []llb.HTTPOption{llb.Filename(filename)}
		checksum := source.Checksum
		if checksum == "" {
			checksum = options.AddChecksum
		}
		if checksum != "" {
			httpOptions = append(httpOptions, llb.Checksum(checksum))
		}
		return llb.HTTP(source.Value, httpOptions...), "/" + filename, nil
	case AddGit:
		if source.Value == "" {
			return llb.State{}, "", fmt.Errorf("Git remote cannot be empty")
		}
		if source.Checksum != "" || options.AddChecksum != "" {
			return llb.State{}, "", fmt.Errorf("Git ADD does not use an HTTP digest; pin the ref or pass llb.GitChecksum through GitSource")
		}
		gitOptions := append([]llb.GitOption(nil), source.GitOptions...)
		if options.KeepGitDir {
			gitOptions = append(gitOptions, llb.KeepGitDir())
		}
		if source.Subdir != "" {
			gitOptions = append(gitOptions, llb.GitSubDir(source.Subdir))
		}
		return llb.Git(source.Value, source.Ref, gitOptions...), "/", nil
	default:
		return llb.State{}, "", fmt.Errorf("unsupported source kind %d", source.Kind)
	}
}

func copyOptions(b *Build, options InstructionOptions, unpack bool) ([]llb.CopyOption, error) {
	info := &llb.CopyInfo{
		CreateDestPath: true,
		AllowWildcard:  true,
		FollowSymlinks: true,
		AttemptUnpack:  unpack,
	}
	if options.Chmod != nil {
		mode := *options.Chmod
		if mode.ModeStr != "" {
			mode.ModeStr = b.expand(mode.ModeStr)
		}
		info.Mode = &mode
	}
	if len(options.Excludes) > 0 {
		for _, pattern := range options.Excludes {
			info.ExcludePatterns = append(info.ExcludePatterns, b.expand(pattern))
		}
	}
	result := make([]llb.CopyOption, 0, len(options.CopyOptions)+3)
	result = append(result, info)
	result = append(result, options.CopyOptions...)
	if options.Chown != "" {
		chown := b.expand(options.Chown)
		if chown == "" {
			return nil, fmt.Errorf("expanded --chown value cannot be empty")
		}
		result = append(result, llb.WithUser(chown))
	}
	if b.createdAt != nil {
		result = append(result, llb.WithCreatedTime(*b.createdAt))
	}
	return result, nil
}

func combineConstraints(b *Build, options InstructionOptions) []llb.ConstraintsOpt {
	result := make([]llb.ConstraintsOpt, 0, len(b.constraints)+len(options.Constraints))
	result = append(result, b.constraints...)
	result = append(result, options.Constraints...)
	return result
}

func rootPath(source string) string {
	source = strings.ReplaceAll(source, "\\", "/")
	if strings.HasPrefix(source, "/") {
		return path.Clean(source)
	}
	return path.Clean("/" + source)
}

func preservedSourcePath(source string) string {
	source = strings.ReplaceAll(source, "\\", "/")
	if index := strings.Index(source, "/./"); index >= 0 {
		source = source[index+3:]
	} else {
		source = strings.TrimPrefix(source, "./")
	}
	cleaned := strings.TrimPrefix(path.Clean(source), "/")
	for strings.HasPrefix(cleaned, "../") {
		cleaned = strings.TrimPrefix(cleaned, "../")
	}
	if cleaned == "." || cleaned == "" || cleaned == ".." {
		return path.Base(source)
	}
	return cleaned
}
