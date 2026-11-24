package instruction

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/moby/buildkit/client/llb"
	"github.com/moby/buildkit/util/system"
)

// Workdir changes the LLB working directory and the image's Config.WorkingDir.
// The target directory is created when it does not already exist.
func Workdir(directory string) Instruction {
	description := "WORKDIR " + directory
	return newMutationInstruction("WORKDIR", func(_ context.Context, state *State, runtime *Runtime, metadata InstructionMetadata) error {
		if runtime == nil {
			return errors.New("WORKDIR requires a build runtime")
		}
		value, err := runtime.Expand(directory, state)
		if err != nil {
			return fmt.Errorf("expand WORKDIR: %w", err)
		}
		if strings.TrimSpace(value) == "" {
			return errors.New("WORKDIR path cannot be empty")
		}
		normalized, err := normalizeWorkdir(state.Image.Config.WorkingDir, value, state.Platform.OS)
		if err != nil {
			return fmt.Errorf("normalize WORKDIR: %w", err)
		}
		state.Image.Config.WorkingDir = normalized
		state.LLB = state.LLB.Dir(system.ToSlash(normalized, state.Platform.OS))
		if system.ToSlash(normalized, state.Platform.OS) == "/" {
			return nil
		}

		mkdirOptions := []llb.MkdirOption{llb.WithParents(true)}
		if state.Image.Config.User != "" {
			mkdirOptions = append(mkdirOptions, llb.WithUser(state.Image.Config.User))
		}
		if runtime.CreatedAt != nil {
			mkdirOptions = append(mkdirOptions, llb.WithCreatedTime(*runtime.CreatedAt))
		}
		constraints := OperationConstraints(runtime, state, metadata, description)
		state.LLB = state.LLB.File(
			llb.Mkdir(system.ToSlash(normalized, state.Platform.OS), 0o755, mkdirOptions...),
			constraints...,
		)
		return nil
	}, WithDescription(description),
		WithOnBuildExpression(func() (string, error) {
			if strings.TrimSpace(directory) == "" {
				return "", errors.New("WORKDIR path cannot be empty")
			}
			return "WORKDIR " + dockerWord(directory), nil
		}))
}

func normalizeWorkdir(current, value, osName string) (string, error) {
	return system.NormalizeWorkdir(current, value, osName)
}
