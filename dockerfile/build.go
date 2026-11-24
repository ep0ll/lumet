package dockerfile

import (
	"context"
	"errors"
	"fmt"
	"maps"
	"time"

	"github.com/ep0ll/lumet/dockerfile/progress"
	"github.com/moby/buildkit/client/llb"
)

func (d *Dockerfile) Build(ctx context.Context) (Result, error) {
	if d == nil {
		return Result{}, errors.New("dockerfile: nil definition")
	}
	if len(d.stages) == 0 {
		return Result{}, errors.New("dockerfile: no stages defined")
	}
	runtime := d.newRuntime()
	if err := d.evaluateGlobal(ctx, runtime); err != nil {
		return Result{}, err
	}

	stageIndexes := make(map[string]int, len(d.stages))
	for index, stage := range d.stages {
		stageIndexes[normalizeStageName(stage.Name)] = index
	}
	target := d.target
	if target == "" {
		target = d.stages[len(d.stages)-1].Name
	}
	targetIndex, ok := stageIndexes[normalizeStageName(target)]
	if !ok {
		return Result{}, fmt.Errorf("dockerfile: target stage %q does not exist", target)
	}

	states := make(map[StageName]State, len(d.stages))
	status := make([]uint8, len(d.stages))
	var evaluate func(int) error
	evaluate = func(index int) error {
		switch status[index] {
		case 2:
			return nil
		case 1:
			return fmt.Errorf("dockerfile: circular stage dependency involving %q", d.stages[index].Name)
		}
		status[index] = 1
		plan := d.stages[index]
		for _, dependency := range plan.Dependencies() {
			dependencyIndex, found := stageIndexes[normalizeStageName(dependency)]
			if !found {
				return fmt.Errorf("dockerfile: stage %q depends on unknown stage %q", plan.Name, dependency)
			}
			if dependencyIndex == index {
				return fmt.Errorf("dockerfile: stage %q depends on itself", plan.Name)
			}
			if err := evaluate(dependencyIndex); err != nil {
				return err
			}
		}

		runtime.StageName = plan.Name
		runtime.StageIndex = index + 1
		runtime.StageTotal = len(d.stages)
		runtime.StageConstraints = append([]llb.ConstraintsOpt(nil), plan.Constraints...)
		runtime.InstructionIndex = 1
		runtime.InstructionTotal = 1 + len(plan.Instructions)
		runtime.OnBuild = false

		stageEvent := StageEvent{Index: runtime.StageIndex, Total: runtime.StageTotal}
		if err := callStageHook(ctx, d.hooks.BeforeStage, stageEvent); err != nil {
			return err
		}
		if err := callStageHook(ctx, plan.Hooks.Before, stageEvent); err != nil {
			return err
		}

		platform := d.Platform
		if requested, ok := plan.From.Platform(); ok {
			platform = requested
		}
		started := time.Now()
		if err := d.report(ctx, runtime, stageProgress(progress.Started, plan.Name, runtime, platform, started, nil)); err != nil {
			return err
		}
		state, err := plan.From.Apply(ctx, runtime)
		if err != nil {
			d.reportFailure(ctx, runtime, stageProgress(progress.Failed, plan.Name, runtime, platform, started, err))
			failure := StageEvent{Index: runtime.StageIndex, Total: runtime.StageTotal, Err: err}
			_ = callStageHook(ctx, plan.Hooks.Failed, failure)
			_ = callStageHook(ctx, d.hooks.StageFailed, failure)
			return fmt.Errorf("dockerfile: initialize stage %q: %w", plan.Name, err)
		}
		state.StageName = plan.Name
		if err := d.report(ctx, runtime, stageProgress(progress.Completed, plan.Name, runtime, state.Platform, started, nil)); err != nil {
			return err
		}

		var triggerCount int
		state, triggerCount, err = d.fireOnBuild(ctx, runtime, plan, state)
		if err != nil {
			return fmt.Errorf("dockerfile: stage %q ONBUILD: %w", plan.Name, err)
		}

		instructionTotal := 1 + triggerCount + len(plan.Instructions)
		for index, instruction := range plan.Instructions {
			if err := ctx.Err(); err != nil {
				return err
			}
			runtime.InstructionIndex = index + triggerCount + 2
			runtime.InstructionTotal = instructionTotal
			state, err = d.applyInstruction(ctx, runtime, instruction, state)
			if err != nil {
				failure := StageEvent{
					Stage: state.Clone(), Index: runtime.StageIndex,
					Total: runtime.StageTotal, Err: err,
				}
				_ = callStageHook(ctx, plan.Hooks.Failed, failure)
				_ = callStageHook(ctx, d.hooks.StageFailed, failure)
				return fmt.Errorf("dockerfile: stage %q instruction %s: %w", plan.Name, instruction.name, err)
			}
		}

		completed := StageEvent{
			Stage: state.Clone(), Index: runtime.StageIndex, Total: runtime.StageTotal,
		}
		if err := callStageHook(ctx, plan.Hooks.After, completed); err != nil {
			return err
		}
		if err := callStageHook(ctx, d.hooks.AfterStage, completed); err != nil {
			return err
		}
		states[plan.Name] = state.Clone()
		status[index] = 2
		return nil
	}

	runtime.ResolveStage = func(name StageName) (State, error) {
		previousName := runtime.StageName
		previousStageIndex := runtime.StageIndex
		previousStageTotal := runtime.StageTotal
		previousInstructionIndex := runtime.InstructionIndex
		previousInstructionTotal := runtime.InstructionTotal
		previousStageConstraints := append([]llb.ConstraintsOpt(nil), runtime.StageConstraints...)
		previousOnBuild := runtime.OnBuild
		defer func() {
			runtime.StageName = previousName
			runtime.StageIndex = previousStageIndex
			runtime.StageTotal = previousStageTotal
			runtime.InstructionIndex = previousInstructionIndex
			runtime.InstructionTotal = previousInstructionTotal
			runtime.StageConstraints = previousStageConstraints
			runtime.OnBuild = previousOnBuild
		}()

		dependencyIndex, found := stageIndexes[normalizeStageName(name)]
		if !found {
			return State{}, fmt.Errorf("dockerfile: unknown stage %q", name)
		}
		if err := evaluate(dependencyIndex); err != nil {
			return State{}, err
		}
		resolved, found := states[d.stages[dependencyIndex].Name]
		if !found {
			return State{}, fmt.Errorf("dockerfile: stage %q was not evaluated", name)
		}
		return resolved.Clone(), nil
	}

	if err := evaluate(targetIndex); err != nil {
		return Result{}, err
	}
	return Result{
		Target: states[d.stages[targetIndex].Name].Clone(),
		Stages: cloneStates(states),
		GlobalArgs: maps.Clone(runtime.GlobalArgs),
	}, nil
}

func (d *Dockerfile) newRuntime() *Runtime {
	return &Runtime{
		BuildArgs: maps.Clone(d.buildArgs),
		GlobalArgs: defaultPlatformArgs(d.Platform),
		BuildContext: d.buildContext,
		NamedContexts: cloneNamedContexts(d.namedContexts, d.Platform),
		Platform: d.Platform,
		CreatedAt: cloneCreatedAt(d.createdAt),
		CacheIDNamespace: d.cacheIDNamespace,
		BindMountNoOutput: d.bindMountNoOutput,
		Constraints: append([]llb.ConstraintsOpt(nil), d.constraints...),
		ImageResolver: d.imageResolver,
		OnBuildDecoder: d.onBuildDecoder,
		VariableExpander: d.expander,
		ExpansionConfig: d.expansion,
		ProgressReporter: d.reporter,
	}
}

func (d *Dockerfile) evaluateGlobal(ctx context.Context, runtime *Runtime) error {
	runtime.StageName = ""
	runtime.StageIndex = 0
	runtime.StageTotal = len(d.stages)
	runtime.InstructionTotal = len(d.global)
	for index, instruction := range d.global {
		if err := ctx.Err(); err != nil {
			return err
		}
		runtime.InstructionIndex = index + 1
		state := State{
			LLB: llb.Scratch(), Image: emptyImage(d.Platform),
			Platform: d.Platform, Args: map[string]string{}, Env: map[string]string{},
		}
		started := time.Now()
		if err := d.reportInstruction(ctx, runtime, instruction, state, progress.Started, started); err != nil {
			return err
		}
		next, err := instruction.Apply(ctx, state, runtime)
		if err != nil {
			d.reportInstructionFailure(ctx, runtime, instruction, state, started, err)
			return fmt.Errorf("dockerfile: global %s: %w", instruction.name, err)
		}
		if err := d.reportInstruction(ctx, runtime, instruction, next, progress.Completed, started); err != nil {
			return err
		}
	}
	return nil
}

func (d *Dockerfile) applyInstruction(ctx context.Context, runtime *Runtime, instruction Instruction, state State) (State, error) {
	metadata := instruction.Metadata()
	beforeEvent := InstructionEvent{
		Instruction: instruction, Before: state.Clone(),
		StageIndex: runtime.StageIndex, StageTotal: runtime.StageTotal,
		Index: runtime.InstructionIndex, Total: runtime.InstructionTotal,
	}
	started := time.Now()
	if err := d.reportInstruction(ctx, runtime, instruction, state, progress.Started, started); err != nil {
		return state, err
	}
	if metadata.Hooks.Before != nil {
		if err := metadata.Hooks.Before(ctx, beforeEvent); err != nil {
			return state, err
		}
	}
	if d.hooks.BeforeInstruction != nil {
		if err := d.hooks.BeforeInstruction(ctx, beforeEvent); err != nil {
			return state, err
		}
	}

	next, err := instruction.Apply(ctx, state, runtime)
	if err != nil {
		d.reportInstructionFailure(ctx, runtime, instruction, state, started, err)
		failedEvent := beforeEvent
		failedEvent.Err = err
		if metadata.Hooks.Failed != nil {
			_ = metadata.Hooks.Failed(ctx, failedEvent)
		}
		if d.hooks.InstructionFailed != nil {
			_ = d.hooks.InstructionFailed(ctx, failedEvent)
		}
		return state, err
	}

	if err := d.reportInstruction(ctx, runtime, instruction, next, progress.Completed, started); err != nil {
		return next, err
	}
	afterEvent := beforeEvent
	afterEvent.After = next.Clone()
	if metadata.Hooks.After != nil {
		if err := metadata.Hooks.After(ctx, afterEvent); err != nil {
			return next, err
		}
	}
	if d.hooks.AfterInstruction != nil {
		if err := d.hooks.AfterInstruction(ctx, afterEvent); err != nil {
			return next, err
		}
	}
	return next, nil
}

func (d *Dockerfile) fireOnBuild(ctx context.Context, runtime *Runtime, plan StagePlan, state State) (State, int, error) {
	triggers := append([]Instruction(nil), state.Triggers...)
	raw := append([]string(nil), state.RawOnBuild...)
	if len(raw) == 0 && len(triggers) == 0 && len(state.Image.Config.OnBuild) > 0 {
		raw = append(raw, state.Image.Config.OnBuild...)
	}
	if len(raw) > 0 {
		if runtime.OnBuildDecoder == nil {
			return state, 0, errors.New("base image has raw Config.OnBuild entries; configure an OnBuildDecoder")
		}
		decoded, err := runtime.OnBuildDecoder(ctx, raw)
		if err != nil {
			return state, 0, fmt.Errorf("decode base image ONBUILD metadata: %w", err)
		}
		triggers = append(decoded, triggers...)
	}

	state.Triggers = nil
	state.RawOnBuild = nil
	state.Image.Config.OnBuild = nil
	total := 1 + len(triggers) + len(plan.Instructions)
	for index, trigger := range triggers {
		if trigger.action == nil {
			return state, index, errors.New("nil ONBUILD trigger")
		}
		runtime.OnBuild = true
		runtime.InstructionIndex = index + 2
		runtime.InstructionTotal = total
		var err error
		state, err = d.applyInstruction(ctx, runtime, trigger, state)
		runtime.OnBuild = false
		if err != nil {
			return state, index + 1, fmt.Errorf("trigger %s: %w", trigger.name, err)
		}
	}
	return state, len(triggers), nil
}
