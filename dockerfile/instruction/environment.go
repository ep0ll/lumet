package instruction

import (
	"context"
	"errors"
	"fmt"
	"strings"
)

// KeyValue is a key/value pair used by ENV and LABEL.
type KeyValue struct {
	Key   string
	Value string
}

func KV(key, value string) KeyValue { return KeyValue{Key: key, Value: value} }

// Env applies every value against the environment that existed before this
// instruction, matching Dockerfile's same-instruction expansion behavior.
func Env(values []KeyValue) Instruction {
	pairs := append([]KeyValue(nil), values...)
	description := "ENV " + formatKeyValues(pairs)
	return newMutationInstruction("ENV", func(_ context.Context, state *State, runtime *Runtime, _ InstructionMetadata) error {
		if runtime == nil {
			return errors.New("ENV requires a build runtime")
		}
		if len(pairs) == 0 {
			return errors.New("ENV requires at least one key/value pair")
		}
		previous := state.Clone()
		for _, pair := range pairs {
			if pair.Key == "" {
				return errors.New("ENV key cannot be empty")
			}
			value, err := runtime.Expand(pair.Value, &previous)
			if err != nil {
				return fmt.Errorf("expand ENV %s: %w", pair.Key, err)
			}
			state.LLB = state.LLB.AddEnv(pair.Key, value)
			state.Image.Config.Env = setEnv(state.Image.Config.Env, pair.Key, value)
			if state.Env == nil {
				state.Env = make(map[string]string)
			}
			state.Env[pair.Key] = value
		}
		return nil
	}, WithDescription(description),
		WithOnBuildExpression(func() (string, error) {
			if len(pairs) == 0 {
				return "", errors.New("ENV requires values")
			}
			parts := []string{"ENV"}
			for _, pair := range pairs {
				if pair.Key == "" {
					return "", errors.New("ENV key cannot be empty")
				}
				parts = append(parts, pair.Key+"="+dockerWord(pair.Value))
			}
			return strings.Join(parts, " "), nil
		}))
}
