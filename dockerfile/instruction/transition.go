package instruction

import (
	"context"

	dockerfile "github.com/ep0ll/lumet/dockerfile"
)

// mutation is a compact implementation helper for built-ins. Public custom
// instructions use dockerfile.NewInstruction's pure State -> State contract.
type mutation func(context.Context, *State, *Runtime, InstructionMetadata) error

func newMutationInstruction(name string, apply mutation, options ...InstructionOption) Instruction {
	return dockerfile.NewInstruction(name, func(
		ctx context.Context,
		current State,
		runtime *Runtime,
		metadata InstructionMetadata,
	) (State, error) {
		next := current
		if err := apply(ctx, &next, runtime, metadata); err != nil {
			return current, err
		}
		return next, nil
	}, options...)
}
