package instruction

import (
	"context"
	"errors"
	"strings"
)

// Maintainer updates the image author field.
func Maintainer(name string) Instruction {
	return newMutationInstruction("MAINTAINER", func(_ context.Context, state *State, _ *Runtime, _ InstructionMetadata) error {
		if strings.TrimSpace(name) == "" {
			return errors.New("MAINTAINER cannot be empty")
		}
		state.Image.Author = name
		return nil
	}, WithDescription("MAINTAINER "+name),
		WithOnBuildExpression(func() (string, error) {
			if strings.TrimSpace(name) == "" {
				return "", errors.New("MAINTAINER cannot be empty")
			}
			return "MAINTAINER " + dockerWord(name), nil
		}))
}

// Cmd sets the image's default command.
func Cmd(command Command) Instruction {
	command = cloneCommand(command)
	description := "CMD " + formatCommand(command)
	return newMutationInstruction("CMD", func(_ context.Context, state *State, _ *Runtime, _ InstructionMetadata) error {
		args, err := commandArgs(command, state.Image.Config.Shell, state.Platform.OS)
		if err != nil {
			return err
		}
		state.Image.Config.Cmd = args
		state.Image.Config.ArgsEscaped = true
		state.CmdSet = true
		return nil
	}, WithDescription(description),
		WithOnBuildExpression(func() (string, error) {
			expression, err := commandExpression(command)
			if err != nil {
				return "", err
			}
			return "CMD " + expression, nil
		}))
}

// Entrypoint configures the image's executable. It clears an inherited CMD
// unless the derived stage declares its own CMD instruction.
func Entrypoint(command Command) Instruction {
	command = cloneCommand(command)
	description := "ENTRYPOINT " + formatCommand(command)
	return newMutationInstruction("ENTRYPOINT", func(_ context.Context, state *State, _ *Runtime, _ InstructionMetadata) error {
		args, err := commandArgs(command, state.Image.Config.Shell, state.Platform.OS)
		if err != nil {
			return err
		}
		state.Image.Config.Entrypoint = args
		state.Image.Config.ArgsEscaped = true
		if !state.CmdSet {
			state.Image.Config.Cmd = nil
		}
		return nil
	}, WithDescription(description),
		WithOnBuildExpression(func() (string, error) {
			expression, err := commandExpression(command)
			if err != nil {
				return "", err
			}
			return "ENTRYPOINT " + expression, nil
		}))
}

// Shell sets the shell used by subsequent shell-form RUN, CMD and ENTRYPOINT.
func Shell(shell ExecForm) Instruction {
	value := append(ExecForm(nil), shell...)
	return newMutationInstruction("SHELL", func(_ context.Context, state *State, _ *Runtime, _ InstructionMetadata) error {
		if len(value) == 0 || strings.TrimSpace(value[0]) == "" {
			return errors.New("SHELL requires a non-empty executable")
		}
		state.Image.Config.Shell = append([]string(nil), value...)
		return nil
	}, WithDescription("SHELL "+formatCommand(value)),
		WithOnBuildExpression(func() (string, error) {
			if len(value) == 0 {
				return "", errors.New("SHELL requires an executable")
			}
			encoded, err := jsonMarshal([]string(value))
			if err != nil {
				return "", err
			}
			return "SHELL " + encoded, nil
		}))
}
