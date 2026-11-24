// Package progress defines a UI-independent event stream for Dockerfile builds.
package progress

import (
	"context"
	"time"

	ocispecs "github.com/opencontainers/image-spec/specs-go/v1"
)

type Phase string

const (
	Started Phase = "started"
	Completed Phase = "completed"
	Failed Phase = "failed"
)

type Event struct {
	Phase Phase
	Group string
	Message string
	Instruction string
	InstructionIndex int
	InstructionTotal int
	StageName string
	StageIndex int
	StageTotal int
	Platform ocispecs.Platform
	PlatformSpecified bool
	StartedAt time.Time
	FinishedAt time.Time
	Err error
}

type Reporter interface {
	Report(context.Context, Event) error
}
type ReporterFunc func(context.Context, Event) error
func (fn ReporterFunc) Report(ctx context.Context, event Event) error { return fn(ctx, event) }
