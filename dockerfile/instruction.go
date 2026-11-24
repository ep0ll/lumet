package dockerfile

import (
	"context"
	"errors"
	"fmt"

	"github.com/moby/buildkit/client/llb"
	ocispecs "github.com/opencontainers/image-spec/specs-go/v1"
)

// InstructionAction is a state transition. Constructors capture instruction
// arguments in this closure; Apply supplies the current LLB and image state.
// An action must update both State.LLB and State.Image when the instruction
// changes both representations.
type InstructionAction func(context.Context, *State, *Runtime, InstructionMetadata) error

// Instruction is a named closure plus the metadata needed by the Dockerfile
// evaluator. It is intentionally a concrete value, not a handler interface or
// a dynamically-dispatched instruction specification.
type Instruction struct {
	name         string
	action       InstructionAction
	metadata     InstructionMetadata
	dependencies []StageName
	globalARG    bool
}

// NewInstruction creates a custom instruction from a state-transforming
// closure. Options attach instruction-local progress, history, LLB and hook
// metadata. The closure receives a private candidate state; failures do not
// partially mutate the caller's state.
func NewInstruction(name string, action InstructionAction, options ...InstructionOption) Instruction {
	instruction := Instruction{name: name, action: action}
	instruction = instruction.With(options...)
	return instruction
}

func (i Instruction) Name() string { return i.name }

func (i Instruction) String() string {
	if i.metadata.Description != "" {
		return i.metadata.Description
	}
	return i.name
}

func (i Instruction) Metadata() InstructionMetadata {
	return cloneInstructionMetadata(i.metadata)
}

func (i Instruction) Dependencies() []StageName {
	return append([]StageName(nil), i.dependencies...)
}

// With returns a copy with the supplied instruction-local settings applied.
func (i Instruction) With(options ...InstructionOption) Instruction {
	i.metadata = cloneInstructionMetadata(i.metadata)
	for _, option := range options {
		if option != nil {
			option(&i.metadata)
		}
	}
	i.dependencies = append(i.dependencies[:0:0], i.metadata.Dependencies...)
	return i
}

// Apply executes the instruction transactionally. BuildKit's State is
// persistent; the image config and maps are copied before invoking the closure.
func (i Instruction) Apply(ctx context.Context, current State, runtime *Runtime) (State, error) {
	if i.action == nil {
		return current, errors.New("dockerfile: instruction has no action")
	}
	if i.name == "" {
		return current, errors.New("dockerfile: instruction name cannot be empty")
	}

	candidate := current.Clone()
	metadata := cloneInstructionMetadata(i.metadata)
	if err := i.action(ctx, &candidate, runtime, metadata); err != nil {
		return current, fmt.Errorf("%s: %w", i.name, err)
	}
	next, err := appendHistory(ctx, current, candidate, i, metadata, runtime)
	if err != nil {
		return current, err
	}
	return next, nil
}

// OnBuildExpression returns the OCI image-config string for this instruction.
// Only instructions that explicitly provide a serializer can be ONBUILD
// triggers; arbitrary Go closures cannot be reconstructed from an OCI image.
func (i Instruction) OnBuildExpression() (string, error) {
	if i.metadata.OnBuildExpression == nil {
		return "", fmt.Errorf("dockerfile: %s cannot be serialized as an ONBUILD trigger", i.name)
	}
	return i.metadata.OnBuildExpression()
}

type HistoryMode uint8

const (
	HistoryDefault HistoryMode = iota
	HistoryLayer
	HistoryEmptyLayer
	HistorySkip
)

// HistoryEvent provides before/after snapshots for custom history creation.
type HistoryEvent struct {
	Instruction Instruction
	Before      State
	After       State
}

// HistoryFunc creates one OCI history entry for an instruction.
type HistoryFunc func(context.Context, HistoryEvent) (ocispecs.History, error)

// InstructionHook observes or validates one instruction transition.
type InstructionHook func(context.Context, InstructionEvent) error

// InstructionEvent is shared by instruction hooks and Dockerfile-level hooks.
type InstructionEvent struct {
	Instruction Instruction
	Before      State
	After       State
	StageIndex  int
	StageTotal  int
	Index       int
	Total       int
	Err         error
}

// InstructionHooks are attached to an individual instruction.
type InstructionHooks struct {
	Before InstructionHook
	After  InstructionHook
	Failed InstructionHook
}

// InstructionMetadata contains behavior that decorates an instruction but
// does not belong to the instruction's operation-specific arguments.
type InstructionMetadata struct {
	Description       string
	ProgressGroup     string
	ProgressTitle     string
	CustomName        string
	Constraints       []llb.ConstraintsOpt
	Dependencies      []StageName
	HistoryMode       HistoryMode
	History           HistoryFunc
	DeferredDependencies bool
	Hooks             InstructionHooks
	OnBuildExpression func() (string, error)
}

func cloneInstructionMetadata(src InstructionMetadata) InstructionMetadata {
	dst := src
	dst.Constraints = append([]llb.ConstraintsOpt(nil), src.Constraints...)
	dst.Dependencies = append([]StageName(nil), src.Dependencies...)
	return dst
}
