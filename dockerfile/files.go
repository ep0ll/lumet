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
		copyOpts := copyOptions(b, options, false)
		for _, source := range sources {
			source = b.expand(source)
			sourcePath := rootPath(source)
			target := destination
			if options.Parents {
				target = path.Join(destination, preservedSourcePath(source))
			}

			if options.Link {
				copyState := llb.Scratch().Platform(b.Platform).Dir(b.Image.Config.WorkingDir)
				copyState = copyState.File(llb.Copy(sourceState, sourcePath, target, copyOpts...), combineConstraints(b, options)...)
				b.State = llb.Merge([]llb.State{b.State, copyState}, combineConstraints(b, options)...)
				continue
			}
			b.State = b.State.File(
				llb.Copy(sourceState, sourcePath, target, copyOpts...),
				combineConstraints(b, options)...,
			)
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
		if destination == "" {
			return fmt.Errorf("ADD: destination cannot be empty")
		}

		unpack := true
		if options.Unpack != nil {
			unpack = *options.Unpack
		}
		copyOpts := copyOptions(b, options, unpack)
		for _, source := range sources {
			state, sourcePath, err := resolveAddSource(b, source, options)
			if err != nil {
				return fmt.Errorf("ADD %q: %w", source.Value, err)
			}
			b.State = b.State.File(
				llb.Copy(state, sourcePath, destination, copyOpts...),
				combineConstraints(b, options)...,
			)
		}
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
	if stage, ok := b.stages[options.From]; ok {
		return stage.State, nil
	}
	if state, ok := b.namedContexts[options.From]; ok {
		return state, nil
	}
	return llb.State{}, fmt.Errorf("source stage or context %q does not exist", options.From)
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
		return b.BuildContext, rootPath(b.expand(source.Value)), nil
	case AddRemote:
		if source.Value == "" {
			return llb.State{}, "", fmt.Errorf("remote URL cannot be empty")
		}
		parsed, err := url.Parse(source.Value)
		if err != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") {
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
		gitOptions := append([]llb.GitOption(nil), source.GitOptions...)
		if options.KeepGitDir {
			gitOptions = append(gitOptions, llb.KeepGitDir())
		}
		if source.Checksum != "" {
			gitOptions = append(gitOptions, llb.GitChecksum(source.Checksum.String()))
		}
		if options.AddChecksum != "" {
			gitOptions = append(gitOptions, llb.GitChecksum(options.AddChecksum.String()))
		}
		if source.Subdir != "" {
			gitOptions = append(gitOptions, llb.GitSubDir(source.Subdir))
		}
		return llb.Git(source.Value, source.Ref, gitOptions...), "/", nil
	default:
		return llb.State{}, "", fmt.Errorf("unsupported source kind %d", source.Kind)
	}
}

func copyOptions(b *Build, options InstructionOptions, unpack bool) []llb.CopyOption {
	info := &llb.CopyInfo{
		CreateDestPath: true,
		AllowWildcard:  true,
		FollowSymlinks: true,
		AttemptUnpack:  unpack,
	}
	if options.Chmod != nil {
		info.Mode = options.Chmod
	}
	if len(options.Excludes) > 0 {
		info.ExcludePatterns = append(info.ExcludePatterns, options.Excludes...)
	}
	result := make([]llb.CopyOption, 0, len(options.CopyOptions)+5)
	result = append(result, info)
	result = append(result, options.CopyOptions...)
	if options.Chown != "" {
		result = append(result, llb.WithUser(b.expand(options.Chown)))
	}
	if options.Chmod != nil {
		result = append(result, *options.Chmod)
	}
	if len(options.Excludes) > 0 {
		result = append(result, llb.WithExcludePatterns(options.Excludes))
	}
	if b.createdAt != nil {
		result = append(result, llb.WithCreatedTime(*b.createdAt))
	}
	return result
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
