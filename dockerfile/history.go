package dockerfile

import (
	"context"
	"fmt"

	ocispecs "github.com/opencontainers/image-spec/specs-go/v1"
)

func appendHistory(
	ctx context.Context,
	before State,
	after State,
	instruction Instruction,
	metadata InstructionMetadata,
	runtime *Runtime,
) (State, error) {
	mode := metadata.HistoryMode
	if mode == HistoryDefault {
		mode = defaultHistoryMode(instruction.name)
	}
	if mode == HistorySkip {
		return after, nil
	}

	var entry ocispecs.History
	if metadata.History != nil {
		var err error
		entry, err = metadata.History(ctx, HistoryEvent{
			Instruction: instruction,
			Before:      before.Clone(),
			After:       after.Clone(),
		})
		if err != nil {
			return before, fmt.Errorf("dockerfile: create history for %s: %w", instruction.name, err)
		}
	} else {
		entry.CreatedBy = instruction.String()
		entry.EmptyLayer = mode == HistoryEmptyLayer
		entry.Author = after.Image.Author
		if runtime != nil && runtime.CreatedAt != nil {
			created := *runtime.CreatedAt
			entry.Created = &created
		}
	}
	after.Image.History = append(after.Image.History, entry)
	return after, nil
}

func defaultHistoryMode(name string) HistoryMode {
	switch name {
	case "RUN", "COPY", "ADD", "WORKDIR":
		return HistoryLayer
	case "ONBUILD":
		return HistorySkip
	default:
		return HistoryEmptyLayer
	}
}
