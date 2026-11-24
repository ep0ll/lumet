package dockerfile

import (
	"context"

	"github.com/moby/buildkit/client/llb"
)

// StagePlan binds one FROM closure to the instructions and lifecycle hooks
// that transform the resulting stage state.
type StagePlan struct {
	Name        StageName
	From        StageFrom
	Instructions []Instruction
	Hooks       StageHooks
	Constraints []llb.ConstraintsOpt
}

type StageHooks struct {
	Before StageHook
	After  StageHook
	Failed StageHook
}

type StageEvent struct {
	Stage State
	Index int
	Total int
	Err   error
}

type StageHook func(context.Context, StageEvent) error

type StageOption func(*StagePlan)

func WithStageName(name StageName) StageOption {
	return func(plan *StagePlan) { plan.Name = name }
}
func WithStageHooks(hooks StageHooks) StageOption {
	return func(plan *StagePlan) { plan.Hooks = hooks }
}
func WithStageConstraints(options ...llb.ConstraintsOpt) StageOption {
	return func(plan *StagePlan) {
		plan.Constraints = append(plan.Constraints, options...)
	}
}

// Plan constructs a stage plan. Its input slices are copied so later caller
// mutations do not silently change the Dockerfile definition.
func Plan(from StageFrom, instructions []Instruction, options ...StageOption) StagePlan {
	plan := StagePlan{
		Name: from.Name(),
		From: from,
		Instructions: append([]Instruction(nil), instructions...),
	}
	for _, option := range options {
		if option != nil {
			option(&plan)
		}
	}
	return plan
}

func (plan StagePlan) Dependencies() []StageName {
	dependencies := append([]StageName(nil), plan.From.Dependencies()...)
	for _, instruction := range plan.Instructions {
		// ONBUILD is deferred to a derived stage. Its dynamic dependencies
		// are resolved only when the stored trigger is actually executed.
		if instruction.Metadata().DeferredDependencies {
			continue
		}
		dependencies = append(dependencies, instruction.Dependencies()...)
	}
	return uniqueStageNames(dependencies)
}

func uniqueStageNames(names []StageName) []StageName {
	seen := make(map[string]struct{}, len(names))
	result := make([]StageName, 0, len(names))
	for _, name := range names {
		key := normalizeStageName(name)
		if key == "" {
			continue
		}
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		result = append(result, name)
	}
	return result
}
