package instruction

import (
	"context"
	"errors"
	"fmt"
	"strings"
)

// ArgOption configures one ARG closure.
type ArgOption func(*argConfig)

type argConfig struct {
	defaultValue *string
}

func ArgDefault(value string) ArgOption {
	return func(config *argConfig) {
		copy := value
		config.defaultValue = &copy
	}
}

// Arg returns an instruction closure. In global scope it populates global
// arguments for FROM; inside a stage it makes the value available to later
// instructions and RUN without adding it to the final image Config.Env.
func Arg(name string, options ...ArgOption) Instruction {
	config := argConfig{}
	for _, option := range options {
		if option != nil {
			option(&config)
		}
	}
	description := "ARG " + name
	if config.defaultValue != nil {
		description += "=" + *config.defaultValue
	}
	instruction := newMutationInstruction("ARG", func(_ context.Context, state *State, runtime *Runtime, _ InstructionMetadata) error {
		if runtime == nil {
			return errors.New("ARG requires a build runtime")
		}
		if !validVariableName(name) {
			return fmt.Errorf("invalid ARG name %q", name)
		}

		if state.StageName == "" {
			value, found := runtime.BuildArgs[name]
			if !found && config.defaultValue != nil {
				expanded, err := runtime.Expand(*config.defaultValue, state)
				if err != nil {
					return fmt.Errorf("expand ARG %s default: %w", name, err)
				}
				value, found = expanded, true
			}
			if !found {
				value, found = runtime.GlobalArgs[name]
			}
			if found {
				runtime.GlobalArgs[name] = value
			}
			return nil
		}

		value, found := runtime.BuildArgs[name]
		if !found && config.defaultValue != nil {
			expanded, err := runtime.Expand(*config.defaultValue, state)
			if err != nil {
				return fmt.Errorf("expand ARG %s default: %w", name, err)
			}
			value, found = expanded, true
		}
		if !found {
			value, found = state.Args[name]
		}
		if !found {
			value, found = runtime.GlobalArgs[name]
		}
		if !found {
			return nil
		}

		if state.Args == nil {
			state.Args = make(map[string]string)
		}
		state.Args[name] = value
		state.LLB = state.LLB.AddEnv(name, value)
		return nil
	}, WithDescription(description),
		WithOnBuildExpression(func() (string, error) {
			if config.defaultValue == nil {
				return "ARG " + name, nil
			}
			return "ARG " + name + "=" + dockerWord(*config.defaultValue), nil
		}))
	return instruction.With(WithGlobalARG())
}

func validVariableName(name string) bool {
	if name == "" || strings.TrimSpace(name) != name {
		return false
	}
	for index, r := range name {
		if index == 0 {
			if !(r == '_' || r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z') {
				return false
			}
			continue
		}
		if !(r == '_' || r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9') {
			return false
		}
	}
	return true
}
