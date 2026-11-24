package dockerfile

import (
	"context"
	"errors"
	"fmt"
	"strings"
)

// Volume declares image-level volume paths.
func Volume(paths ...string) Instruction {
	values := append([]string(nil), paths...)
	return NewInstruction("VOLUME", func(_ context.Context, state *State, runtime *Runtime, _ InstructionMetadata) error {
		if runtime == nil {
			return errors.New("VOLUME requires a build runtime")
		}
		if len(values) == 0 {
			return errors.New("VOLUME requires at least one path")
		}
		if state.Image.Config.Volumes == nil {
			state.Image.Config.Volumes = make(map[string]struct{})
		}
		for _, raw := range values {
			value, err := runtime.Expand(raw, state)
			if err != nil {
				return fmt.Errorf("expand VOLUME path: %w", err)
			}
			if value == "" {
				return errors.New("VOLUME path cannot be empty")
			}
			normalized, err := normalizeWorkdir(state.Image.Config.WorkingDir, value, state.Platform.OS)
			if err != nil {
				return fmt.Errorf("normalize VOLUME path: %w", err)
			}
			state.Image.Config.Volumes[normalized] = struct{}{}
		}
		return nil
	}, WithDescription("VOLUME "+strings.Join(values, " ")),
		WithOnBuildExpression(func() (string, error) {
			if len(values) == 0 {
				return "", errors.New("VOLUME requires paths")
			}
			encoded, err := jsonMarshal(values)
			if err != nil {
				return "", err
			}
			return "VOLUME " + encoded, nil
		}))
}
