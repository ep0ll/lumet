// Package progress defines a UI-agnostic build progress event contract.
package progress

import (
	"context"
	"time"

	ocispecs "github.com/opencontainers/image-spec/specs-go/v1"
)

// Phase identifies an instruction lifecycle event.
type Phase string

const (
	Started   Phase = "started"
	Completed Phase = "completed"
	Failed    Phase = "failed"
)

// Event is a point-in-time snapshot of build progress.
type Event struct {
	Phase             Phase
	Group             string
	Message           string
	Instruction       string
	InstructionIndex  int
	InstructionTotal  int
	StageName         string
	StageIndex        int
	StageTotal        int
	Platform          ocispecs.Platform
	PlatformSpecified bool
	StartedAt         time.Time
	FinishedAt        time.Time
	Err               error
}

// Reporter consumes progress events independently of any UI implementation.
type Reporter interface {
	Report(context.Context, Event) error
}

// ReporterFunc adapts a function to Reporter.
type ReporterFunc func(context.Context, Event) error

func (fn ReporterFunc) Report(ctx context.Context, event Event) error {
	return fn(ctx, event)
}
