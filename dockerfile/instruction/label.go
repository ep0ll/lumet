package instruction

import (
	"context"
	"errors"
	"fmt"
	"strings"
)

// Label updates OCI image config labels without changing filesystem state.
// Label modifies image config metadata without changing the filesystem state.
func Label(values []KeyValue) Instruction {
	pairs := append([]KeyValue(nil), values...)
	return newMutationInstruction("LABEL", func(_ context.Context, state *State, runtime *Runtime, _ InstructionMetadata) error {
		if len(pairs) == 0 {
			return errors.New("LABEL requires at least one key/value pair")
		}
		if state.Image.Config.Labels == nil {
			state.Image.Config.Labels = make(map[string]string, len(pairs))
		}
		for _, pair := range pairs {
			if pair.Key == "" {
				return errors.New("LABEL key cannot be empty")
			}
			value := pair.Value
			if runtime != nil {
				expanded, err := runtime.Expand(value, state)
				if err != nil {
					return fmt.Errorf("expand LABEL %s: %w", pair.Key, err)
				}
				value = expanded
			}
			state.Image.Config.Labels[pair.Key] = value
		}
		return nil
	}, WithDescription("LABEL "+formatKeyValues(pairs)),
		WithOnBuildExpression(func() (string, error) {
			if len(pairs) == 0 {
				return "", errors.New("LABEL requires values")
			}
			parts := []string{"LABEL"}
			for _, pair := range pairs {
				if pair.Key == "" {
					return "", errors.New("LABEL key cannot be empty")
				}
				parts = append(parts, pair.Key+"="+dockerWord(pair.Value))
			}
			return strings.Join(parts, " "), nil
		}))
}

func formatKeyValues(values []KeyValue) string {
	parts := make([]string, 0, len(values))
	for _, value := range values {
		parts = append(parts, value.Key+"="+value.Value)
	}
	return strings.Join(parts, " ")
}
