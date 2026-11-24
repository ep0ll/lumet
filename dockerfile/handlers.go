package dockerfile

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/moby/buildkit/client/llb"
	ocispecs "github.com/opencontainers/image-spec/specs-go/v1"
)

type Spec interface { CommonOptions() Common }

type Common struct {
	History HistoryFunc
	HistoryMode HistoryMode
	Constraints []llb.ConstraintsOpt
	ProgressGroup string
	ProgressTitle string
	CustomName string
	DependsOn []StageName
	Hooks InstructionHooks
}
func (c Common) CommonOptions() Common {
	c.Constraints = append([]llb.ConstraintsOpt(nil), c.Constraints...)
	c.DependsOn = append([]StageName(nil), c.DependsOn...)
	return c
}

type CommonOption func(*Common)
type Handler[S Spec] func(context.Context, Call[S]) (Stage, error)
type Call[S Spec] struct {
	Kind Kind
	Stage Stage
	Spec S
	Handlers *Handlers
	Runtime *Runtime
}
type HistoryEvent struct {
	Kind Kind
	Stage Stage
	Spec any
	Instruction Instruction
}
type HistoryFunc func(context.Context, HistoryEvent) (ocispecs.History, error)
type ApplyFunc func(context.Context, Stage) (Stage, error)
type Interceptor func(context.Context, Stage, Instruction, ApplyFunc) (Stage, error)
type handlerAdapter func(context.Context, Stage, any, *Handlers, *Runtime) (Stage, error)

type Handlers struct {
	handlers map[Kind]handlerAdapter
	History map[Kind]HistoryFunc
	Interceptors []Interceptor
}
func NewHandlers() *Handlers {
	return &Handlers{handlers: make(map[Kind]handlerAdapter), History: make(map[Kind]HistoryFunc)}
}
func RegisterHandler[S Spec](h *Handlers, kind Kind, fn Handler[S]) error {
	if h == nil { return errors.New("dockerfile: nil handler registry") }
	if kind == "" { return errors.New("dockerfile: instruction kind cannot be empty") }
	if fn == nil { return fmt.Errorf("dockerfile: nil handler for %q", kind) }
	if h.handlers == nil { h.handlers = make(map[Kind]handlerAdapter) }
	h.handlers[kind] = func(ctx context.Context, stage Stage, spec any, handlers *Handlers, runtime *Runtime) (Stage, error) {
		typed, ok := spec.(S)
		if !ok {
			var zero S
			return stage, fmt.Errorf("dockerfile: handler %q expected spec %T, got %T", kind, zero, spec)
		}
		return fn(ctx, Call[S]{Kind: kind, Stage: stage, Spec: typed, Handlers: handlers, Runtime: runtime})
	}
	return nil
}
type ErrNoHandler struct{ Kind Kind }
func (e *ErrNoHandler) Error() string { return fmt.Sprintf("dockerfile: no handler registered for %s", e.Kind) }

type Instruction interface {
	Kind() Kind
	Name() string
	String() string
	SpecValue() any
	CommonOptions() Common
	Dependencies() []StageName
	OnBuildExpression() (string, error)
	Apply(context.Context, Stage, *Handlers, *Runtime) (Stage, error)
}
type StageInit interface { Instruction; isStageInit() }
type Trigger interface { Instruction; isTrigger() }

type instruction[S Spec] struct { kind Kind; name string; spec S }
func (i instruction[S]) Kind() Kind { return i.kind }
func (i instruction[S]) Name() string { return i.name }
func (i instruction[S]) SpecValue() any { return i.spec }
func (i instruction[S]) CommonOptions() Common { return i.spec.CommonOptions() }
func (i instruction[S]) String() string {
	if value, ok := any(i.spec).(fmt.Stringer); ok { return value.String() }
	return i.name
}
func (i instruction[S]) OnBuildExpression() (string, error) {
	if value, ok := any(i.spec).(interface{ OnBuildExpression() (string, error) }); ok {
		return value.OnBuildExpression()
	}
	return i.String(), nil
}
func (i instruction[S]) Dependencies() []StageName {
	common := i.spec.CommonOptions()
	dependencies := append([]StageName(nil), common.DependsOn...)
	if provider, ok := any(i.spec).(interface{ StageDependencies() []StageName }); ok {
		dependencies = append(dependencies, provider.StageDependencies()...)
	}
	return uniqueStageNames(dependencies)
}
func (i instruction[S]) Apply(ctx context.Context, stage Stage, handlers *Handlers, runtime *Runtime) (Stage, error) {
	if handlers == nil { return stage, errors.New("dockerfile: nil handler registry") }
	adapter := handlers.handlers[i.kind]
	if adapter == nil { return stage, &ErrNoHandler{Kind: i.kind} }
	next := ApplyFunc(func(ctx context.Context, current Stage) (Stage, error) {
		return adapter(ctx, current, i.spec, handlers, runtime)
	})
	for index := len(handlers.Interceptors)-1; index >= 0; index-- {
		interceptor, following := handlers.Interceptors[index], next
		next = func(ctx context.Context, current Stage) (Stage, error) {
			return interceptor(ctx, current, i, following)
		}
	}
	result, err := next(ctx, cloneStage(stage))
	if err != nil { return stage, err }
	return appendHistory(ctx, result, stage, i, handlers, runtime)
}

type stageInit[S Spec] struct{ instruction[S] }
type trigger[S Spec] struct{ instruction[S] }
func (stageInit[S]) isStageInit() {}
func (trigger[S]) isTrigger() {}

func NewInstruction[S Spec](kind Kind, name string, spec S) Instruction {
	return instruction[S]{kind: kind, name: name, spec: spec}
}
func NewStageInit[S Spec](kind Kind, name string, spec S) StageInit {
	return stageInit[S]{instruction: instruction[S]{kind: kind, name: name, spec: spec}}
}
func NewTrigger[S Spec](kind Kind, name string, spec S) Trigger {
	return trigger[S]{instruction: instruction[S]{kind: kind, name: name, spec: spec}}
}

func Compose(ctx context.Context, handlers *Handlers, runtime *Runtime, first StageInit, body ...Instruction) (Stage, error) {
	if first == nil { return Stage{}, errors.New("dockerfile: stage initializer is required") }
	if runtime == nil { runtime = newRuntime() }
	stage, err := first.Apply(ctx, Stage{}, handlers, runtime)
	if err != nil { return stage, err }
	for _, in := range body {
		if in == nil { return stage, errors.New("dockerfile: nil instruction") }
		if _, ok := in.(StageInit); ok { return stage, fmt.Errorf("dockerfile: %s cannot appear inside a stage body", in.Name()) }
		stage, err = in.Apply(ctx, stage, handlers, runtime)
		if err != nil { return stage, fmt.Errorf("dockerfile: %s: %w", in.Name(), err) }
	}
	return stage, nil
}

func appendHistory(ctx context.Context, result, before Stage, in Instruction, handlers *Handlers, runtime *Runtime) (Stage, error) {
	mode := in.CommonOptions().HistoryMode
	if mode == HistoryDefault { mode = defaultHistoryMode(in.Kind()) }
	if mode == HistorySkip { return result, nil }
	var entry ocispecs.History
	historyFn := in.CommonOptions().History
	if historyFn == nil && handlers != nil { historyFn = handlers.History[in.Kind()] }
	if historyFn != nil {
		var err error
		entry, err = historyFn(ctx, HistoryEvent{Kind: in.Kind(), Stage: cloneStage(before), Spec: in.SpecValue(), Instruction: in})
		if err != nil { return before, fmt.Errorf("dockerfile: history for %s: %w", in.Name(), err) }
	} else {
		entry.CreatedBy = in.String()
		entry.EmptyLayer = mode != HistoryLayer
		entry.Author = result.Image.Author
		if runtime != nil && runtime.CreatedAt != nil { created := *runtime.CreatedAt; entry.Created = &created }
	}
	result.Image.History = append(result.Image.History, entry)
	return result, nil
}
func defaultHistoryMode(kind Kind) HistoryMode {
	switch kind {
	case KindFrom, KindResetStage, KindOnBuild: return HistorySkip
	case KindRun, KindCopy, KindAdd, KindWorkdir: return HistoryLayer
	default: return HistoryEmptyLayer
	}
}
func uniqueStageNames(input []StageName) []StageName {
	seen := make(map[string]struct{}, len(input))
	out := make([]StageName, 0, len(input))
	for _, name := range input {
		key := normalizeStageName(name)
		if key == "" { continue }
		if _, exists := seen[key]; exists { continue }
		seen[key] = struct{}{}
		out = append(out, name)
	}
	return out
}
func normalizeStageName(name StageName) string { return strings.ToLower(strings.TrimSpace(string(name))) }
