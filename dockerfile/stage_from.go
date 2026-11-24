package dockerfile

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/moby/buildkit/client/llb"
	ocispecs "github.com/opencontainers/image-spec/specs-go/v1"
)

// StageFrom is the closure that initializes the first State of a stage.
type StageFrom struct {
	name         StageName
	platform     *ocispecs.Platform
	action       InstructionAction
	metadata     InstructionMetadata
	dependencies []StageName
}

type StageFromOption func(*stageFromConfig)

type stageFromConfig struct {
	name     StageName
	platform *ocispecs.Platform
	options  []llb.ImageOption
	reset    bool
	metadata InstructionMetadata
}

func As(name StageName) StageFromOption {
	return func(config *stageFromConfig) { config.name = name }
}
func FromPlatform(platform ocispecs.Platform) StageFromOption {
	return func(config *stageFromConfig) {
		value := platform
		value.OSFeatures = append([]string(nil), platform.OSFeatures...)
		config.platform = &value
	}
}
func FromImageOptions(options ...llb.ImageOption) StageFromOption {
	return func(config *stageFromConfig) {
		config.options = append(config.options, options...)
	}
}
func FromInstructionOptions(options ...InstructionOption) StageFromOption {
	return func(config *stageFromConfig) {
		for _, option := range options {
			if option != nil {
				option(&config.metadata)
			}
		}
	}
}

// From returns a stage initialization closure. Arguments are captured now;
// the current build runtime and initial state are supplied when Apply runs.
func From(base Base, options ...StageFromOption) StageFrom {
	base = cloneBase(base)
	config := stageFromConfig{}
	for _, option := range options {
		if option != nil {
			option(&config)
		}
	}
	dependencies := uniqueStageNames(append(baseDependencies(base), config.metadata.Dependencies...))
	instruction := newMutationInstruction("FROM", func(ctx context.Context, state *State, runtime *Runtime, metadata InstructionMetadata) error {
		return initializeFrom(ctx, state, runtime, metadata, base, config)
	})
	return StageFrom{
		name: config.name, platform: config.platform, action: instruction.action,
		metadata: cloneInstructionMetadata(config.metadata),
		dependencies: dependencies,
	}
}

// ResetStage starts from a base while discarding inherited build ARG scope and
// deferred ONBUILD triggers. The filesystem and image metadata still derive
// from the selected base.
func ResetStage(base Base, options ...StageFromOption) StageFrom {
	base = cloneBase(base)
	config := stageFromConfig{reset: true}
	for _, option := range options {
		if option != nil {
			option(&config)
		}
	}
	dependencies := uniqueStageNames(append(baseDependencies(base), config.metadata.Dependencies...))
	instruction := newMutationInstruction("FROM", func(ctx context.Context, state *State, runtime *Runtime, metadata InstructionMetadata) error {
		return initializeFrom(ctx, state, runtime, metadata, base, config)
	})
	return StageFrom{
		name: config.name, platform: config.platform, action: instruction.action,
		metadata: cloneInstructionMetadata(config.metadata),
		dependencies: dependencies,
	}
}

// NewStageFrom creates a custom state initializer closure.
func NewStageFrom(name StageName, action InstructionAction, options ...InstructionOption) StageFrom {
	instruction := NewInstruction("FROM", action, options...)
	return StageFrom{
		name: name, action: instruction.action,
		metadata: instruction.metadata,
		dependencies: instruction.dependencies,
	}
}

func (from StageFrom) Name() StageName { return from.name }

func (from StageFrom) Platform() (ocispecs.Platform, bool) {
	if from.platform == nil {
		return ocispecs.Platform{}, false
	}
	return *from.platform, true
}

func (from StageFrom) Dependencies() []StageName {
	return append([]StageName(nil), from.dependencies...)
}

// With returns a StageFrom value with additional instruction metadata.
func (from StageFrom) With(options ...InstructionOption) StageFrom {
	from.metadata = cloneInstructionMetadata(from.metadata)
	for _, option := range options {
		if option != nil {
			option(&from.metadata)
		}
	}
	from.dependencies = uniqueStageNames(append(append([]StageName(nil), from.dependencies...), from.metadata.Dependencies...))
	return from
}

// Apply evaluates the FROM closure against a new stage state.
func (from StageFrom) Apply(ctx context.Context, runtime *Runtime) (State, error) {
	if from.action == nil {
		return State{}, errors.New("dockerfile: FROM closure is nil")
	}
	platform := ocispecs.Platform{}
	if runtime != nil {
		platform = runtime.Platform
	}
	if from.platform != nil {
		platform = *from.platform
	}
	state := State{
		LLB: fromEmptyState(runtime).Platform(platform),
		Image: emptyImage(platform),
		Platform: platform,
		Args: make(map[string]string),
		Env: make(map[string]string),
	}
	if from.name != "" {
		state.StageName = from.name
	}
	next, err := from.action(ctx, state, runtime, cloneInstructionMetadata(from.metadata))
	if err != nil {
		return State{}, fmt.Errorf("FROM %s: %w", from.name, err)
	}
	if from.name != "" {
		next.StageName = from.name
	}
	return next, nil
}

func fromEmptyState(runtime *Runtime) llb.State {
	if runtime == nil {
		return llb.Scratch()
	}
	return llb.Scratch().Platform(runtime.Platform)
}

func baseDependencies(base Base) []StageName {
	if stage, ok := base.(StageRef); ok {
		return []StageName{stage.Name}
	}
	return nil
}

func normalizeStageName(name StageName) string {
	return strings.ToLower(strings.TrimSpace(string(name)))
}
