package dockerfile

import (
	"context"
	"fmt"

	ocispecs "github.com/opencontainers/image-spec/specs-go/v1"
)

// Call is everything a handler receives. Handlers gives access to sibling
// handlers (e.g. ONBUILD applying its trigger, mounts resolving stages).
type Call[S any] struct {
	Kind     Kind
	Stage    Stage
	Spec     S
	Handlers *Handlers
}

// Handler is the extension point: YOU implement the behaviour per instruction.
type Handler[S any] func(ctx context.Context, call Call[S]) (Stage, error)

// HistoryEvent is passed to history callbacks.
type HistoryEvent struct {
	Kind  Kind
	Stage Stage // stage before the instruction ran
	Spec  any   // the typed *Spec value
}

// HistoryFunc builds the history entry for one instruction.
type HistoryFunc func(ctx context.Context, ev HistoryEvent) (ocispecs.History, error)

type ApplyFunc func(ctx context.Context, st Stage) (Stage, error)

// Interceptor wraps every instruction (logging, tracing, source maps, caching...).
type Interceptor func(ctx context.Context, st Stage, in Instruction, next ApplyFunc) (Stage, error)

type Handlers struct {
	From        Handler[FromSpec]
	ResetStage  Handler[ResetSpec]
	Arg         Handler[ArgSpec]
	Env         Handler[EnvSpec]
	Label       Handler[LabelSpec]
	Maintainer  Handler[MaintainerSpec]
	Run         Handler[RunSpec]
	Cmd         Handler[CmdSpec]
	Entrypoint  Handler[EntrypointSpec]
	Shell       Handler[ShellSpec]
	Expose      Handler[ExposeSpec]
	Volume      Handler[VolumeSpec]
	User        Handler[UserSpec]
	Workdir     Handler[WorkdirSpec]
	Copy        Handler[CopySpec]
	Add         Handler[AddSpec]
	StopSignal  Handler[StopSignalSpec]
	Healthcheck Handler[HealthcheckSpec]
	OnBuild     Handler[OnBuildSpec]

	// Default per-instruction history strategy; a spec's own History overrides it.
	History      map[Kind]HistoryFunc
	Interceptors []Interceptor
}

type ErrNoHandler struct{ Kind Kind }

func (e *ErrNoHandler) Error() string { return fmt.Sprintf("dockerfile: no handler registered for %s", e.Kind) }

// Instruction is a declarative, immutable description of one Dockerfile instruction.
type Instruction interface {
	Kind() Kind
	Apply(ctx context.Context, st Stage, h *Handlers) (Stage, error)
}

// StageInit may start a stage (FROM). Trigger may appear inside ONBUILD.
// FROM, ONBUILD and MAINTAINER are not Triggers: ONBUILD(From(...)) does not compile.
type StageInit interface {
	Instruction
	isStageInit()
}
type Trigger interface {
	Instruction
	isTrigger()
}

// Spec is implemented by every *Spec struct (via embedded Common).
type Spec interface{ commonOf() Common }

type instruction[S Spec] struct {
	kind Kind
	spec S
	pick func(*Handlers) Handler[S]
}

func (i instruction[S]) Kind() Kind { return i.kind }

func (i instruction[S]) Apply(ctx context.Context, st Stage, h *Handlers) (Stage, error) {
	run := ApplyFunc(func(ctx context.Context, st Stage) (Stage, error) {
		fn := i.pick(h)
		if fn == nil {
			return st, &ErrNoHandler{Kind: i.kind}
		}
		return fn(ctx, Call[S]{Kind: i.kind, Stage: st, Spec: i.spec, Handlers: h})
	})
	for n := len(h.Interceptors) - 1; n >= 0; n-- {
		ic, next := h.Interceptors[n], run
		run = func(ctx context.Context, st Stage) (Stage, error) { return ic(ctx, st, i, next) }
	}
	return run(ctx, st)
}

type stageInit[S Spec] struct{ instruction[S] }
type trigger[S Spec] struct{ instruction[S] }

func (stageInit[S]) isStageInit() {}
func (trigger[S]) isTrigger()     {}

// Compose is the only sequencing helper: a stage must begin with a StageInit.
func Compose(ctx context.Context, h *Handlers, first StageInit, body ...Instruction) (Stage, error) {
	st, err := first.Apply(ctx, Stage{}, h)
	if err != nil {
		return st, err
	}
	for _, in := range body {
		if st, err = in.Apply(ctx, st, h); err != nil {
			return st, err
		}
	}
	return st, nil
}
