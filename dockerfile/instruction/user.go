package instruction

import (
	"context"
	"errors"
	"fmt"
	"strconv"
)

// Principal is a named or numeric user/group identity.
type Principal struct {
	Name string
	ID   uint32
	ByID bool
}

func Named(name string) Principal { return Principal{Name: name} }
func Numeric(id uint32) Principal { return Principal{ID: id, ByID: true} }

type userConfig struct{ group *Principal }
type UserOption func(*userConfig)

func WithGroup(group Principal) UserOption {
	return func(config *userConfig) {
		copy := group
		config.group = &copy
	}
}

// User updates both the LLB execution user and image Config.User.
func User(user Principal, options ...UserOption) Instruction {
	config := userConfig{}
	for _, option := range options {
		if option != nil {
			option(&config)
		}
	}
	description := "USER " + principalString(user)
	return newMutationInstruction("USER", func(_ context.Context, state *State, runtime *Runtime, _ InstructionMetadata) error {
		value := principalString(user)
		if config.group != nil {
			value += ":" + principalString(*config.group)
		}
		if value == "" {
			return errors.New("USER cannot be empty")
		}
		if runtime != nil {
			expanded, err := runtime.Expand(value, state)
			if err != nil {
				return fmt.Errorf("expand USER: %w", err)
			}
			value = expanded
		}
		state.LLB = state.LLB.User(value)
		state.Image.Config.User = value
		return nil
	}, WithDescription(description),
		WithOnBuildExpression(func() (string, error) {
			value := principalString(user)
			if config.group != nil {
				value += ":" + principalString(*config.group)
			}
			if value == "" {
				return "", errors.New("USER cannot be empty")
			}
			return "USER " + dockerWord(value), nil
		}))
}

func principalString(value Principal) string {
	if value.ByID {
		return strconv.FormatUint(uint64(value.ID), 10)
	}
	return value.Name
}
