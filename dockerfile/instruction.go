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
type InstructionAction func(context.Context, State, *Runtime, InstructionMetadata) (State, error)

type instructionMutation func(context.Context, *State, *Runtime, InstructionMetadata) error

// Instruction is a named closure plus the metadata needed by the Dockerfile
// evaluator. It is intentionally a concrete value, not a handler interface or
// a dynamically-dispatched instruction specification.
type Instruction struct {
	name         string
	action       InstructionAction
	metadata     InstructionMetadata
	dependencies []StageName
}

// NewInstruction creates a pure state-transition instruction. The closure
// receives a detached snapshot of the current state and returns its successor.
// The evaluator supplies a clone, so mutating the snapshot cannot alter the
// caller's state even when the action returns an error.
func NewInstruction(name string, action InstructionAction, options ...InstructionOption) Instruction {
	instruction := Instruction{name: name, action: action}
	return instruction.With(options...)
}

// newMutationInstruction adapts concise in-place state edits used by built-in
// operations to the public input/output transition contract.
func newMutationInstruction(name string, mutation instructionMutation, options ...InstructionOption) Instruction {
	return NewInstruction(name, func(ctx context.Context, current State, runtime *Runtime, metadata InstructionMetadata) (State, error) {
		next := current
		if err := mutation(ctx, &next, runtime, metadata); err != nil {
			return current, err
		}
		return next, nil
	}, options...)
}

func (i Instruction) Name() string { return i.name }

// Valid reports whether the instruction has a usable transition closure.
func (i Instruction) Valid() bool { return i.name != "" && i.action != nil }

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
	next, err := i.action(ctx, candidate, runtime, metadata)
	if err != nil {
		return current, fmt.Errorf("%s: %w", i.name, err)
	}
	next = next.Clone()
	next, err = appendHistory(ctx, current, next, i, metadata, runtime)
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
	GlobalARG             bool
	Hooks                 InstructionHooks
	OnBuildExpression func() (string, error)
}

func cloneInstructionMetadata(src InstructionMetadata) InstructionMetadata {
	dst := src
	dst.Constraints = append([]llb.ConstraintsOpt(nil), src.Constraints...)
	dst.Dependencies = append([]StageName(nil), src.Dependencies...)
	return dst
}
