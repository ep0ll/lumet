package dockerfile

import (
	"context"
	"time"

	"github.com/ep0ll/lumet/dockerfile/progress"
	ocispecs "github.com/opencontainers/image-spec/specs-go/v1"
)

func (d *Dockerfile) reportInstruction(
	ctx context.Context,
	runtime *Runtime,
	instruction Instruction,
	state State,
	phase progress.Phase,
	started time.Time,
) error {
	if runtime.ProgressReporter == nil {
		return nil
	}
	metadata := instruction.Metadata()
	group := metadata.ProgressGroup
	if group == "" {
		group = string(runtime.StageName)
	}
	if group == "" {
		group = "build"
	}
	message := metadata.ProgressTitle
	if message == "" {
		message = instruction.String()
	}
	now := time.Now()
	event := progress.Event{
		Phase: phase, Group: group, Message: message,
		Instruction: instruction.name,
		InstructionIndex: runtime.InstructionIndex,
		InstructionTotal: runtime.InstructionTotal,
		StageName: string(runtime.StageName),
		StageIndex: runtime.StageIndex, StageTotal: runtime.StageTotal,
		Platform: state.Platform,
		PlatformSpecified: state.Platform.OS != "" && state.Platform.Architecture != "",
		Err: nil,
	}
	event.StartedAt = started
	if phase != progress.Started {
		event.FinishedAt = now
	}
	return runtime.ProgressReporter.Report(ctx, event)
}

func (d *Dockerfile) reportInstructionFailure(ctx context.Context, runtime *Runtime, instruction Instruction, state State, started time.Time, cause error) {
	if runtime.ProgressReporter == nil {
		return
	}
	metadata := instruction.Metadata()
	group := metadata.ProgressGroup
	if group == "" {
		group = string(runtime.StageName)
	}
	if group == "" {
		group = "build"
	}
	message := metadata.ProgressTitle
	if message == "" {
		message = instruction.String()
	}
	_ = runtime.ProgressReporter.Report(ctx, progress.Event{
		Phase: progress.Failed, Group: group, Message: message,
		Instruction: instruction.name, InstructionIndex: runtime.InstructionIndex,
		InstructionTotal: runtime.InstructionTotal, StageName: string(runtime.StageName),
		StageIndex: runtime.StageIndex, StageTotal: runtime.StageTotal,
		Platform: state.Platform,
		PlatformSpecified: state.Platform.OS != "" && state.Platform.Architecture != "",
		StartedAt: started, FinishedAt: time.Now(), Err: cause,
	})
}

func (d *Dockerfile) reportFailure(ctx context.Context, runtime *Runtime, event progress.Event) {
	if runtime.ProgressReporter != nil {
		_ = runtime.ProgressReporter.Report(ctx, event)
	}
}

func (d *Dockerfile) report(ctx context.Context, runtime *Runtime, event progress.Event) error {
	if runtime.ProgressReporter == nil {
		return nil
	}
	return runtime.ProgressReporter.Report(ctx, event)
}

func stageProgress(phase progress.Phase, name StageName, runtime *Runtime, platform ocispecs.Platform, started time.Time, err error) progress.Event {
	event := progress.Event{
		Phase: phase, Group: string(name), Message: "FROM " + string(name),
		Instruction: "FROM", InstructionIndex: 1,
		InstructionTotal: runtime.InstructionTotal,
		StageName: string(name), StageIndex: runtime.StageIndex,
		StageTotal: runtime.StageTotal, Platform: platform,
		PlatformSpecified: platform.OS != "" && platform.Architecture != "",
		Err: err,
	}
	event.StartedAt = started
	if phase != progress.Started {
		event.FinishedAt = time.Now()
	}
	return event
}

