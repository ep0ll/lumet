package dockerfile

import "github.com/moby/buildkit/client/llb"

// InstructionOption attaches reusable metadata to one instruction.
type InstructionOption func(*InstructionMetadata)

func WithDescription(description string) InstructionOption {
	return func(metadata *InstructionMetadata) { metadata.Description = description }
}
func WithProgressGroup(group string) InstructionOption {
	return func(metadata *InstructionMetadata) { metadata.ProgressGroup = group }
}
func WithProgressTitle(title string) InstructionOption {
	return func(metadata *InstructionMetadata) { metadata.ProgressTitle = title }
}
func WithCustomName(name string) InstructionOption {
	return func(metadata *InstructionMetadata) { metadata.CustomName = name }
}
func WithInstructionConstraints(options ...llb.ConstraintsOpt) InstructionOption {
	return func(metadata *InstructionMetadata) {
		metadata.Constraints = append(metadata.Constraints, options...)
	}
}
func DependsOn(stages ...StageName) InstructionOption {
	return func(metadata *InstructionMetadata) {
		metadata.Dependencies = append(metadata.Dependencies, stages...)
	}
}
func WithHistoryMode(mode HistoryMode) InstructionOption {
	return func(metadata *InstructionMetadata) { metadata.HistoryMode = mode }
}
func WithHistory(fn HistoryFunc) InstructionOption {
	return func(metadata *InstructionMetadata) { metadata.History = fn }
}
func WithInstructionHooks(hooks InstructionHooks) InstructionOption {
	return func(metadata *InstructionMetadata) { metadata.Hooks = hooks }
}
func WithOnBuildExpression(fn func() (string, error)) InstructionOption {
	return func(metadata *InstructionMetadata) { metadata.OnBuildExpression = fn }
}

func WithDeferredDependencies() InstructionOption {
	return func(metadata *InstructionMetadata) {
		metadata.DeferredDependencies = true
	}
}

func WithGlobalARG() InstructionOption {
	return func(metadata *InstructionMetadata) { metadata.GlobalARG = true }
}
