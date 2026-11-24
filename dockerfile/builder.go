package dockerfile

import (
	"context"
	"errors"
	"fmt"
	"maps"
	"runtime"
	"slices"
	"strings"
	"time"

	"github.com/ep0ll/lumet/dockerfile/expand"
	"github.com/ep0ll/lumet/dockerfile/progress"
	"github.com/moby/buildkit/client/llb"
	dockerspec "github.com/moby/docker-image-spec/specs-go/v1"
	ocispecs "github.com/opencontainers/image-spec/specs-go/v1"
)

// ImageResolver supplies Docker image configuration metadata for an external
// image. LLB image source resolution remains BuildKit's responsibility.
type ImageResolver func(context.Context, string, ocispecs.Platform) (dockerspec.DockerOCIImage, error)

// OnBuildDecoder is required only when a base image comes from outside this
// typed build graph and already contains raw Config.OnBuild strings.
type OnBuildDecoder func(context.Context, []string) ([]Trigger, error)

// VariableExpander expands values using an explicit variable lookup. It has no
// Dockerfile source escape setting because this API does not parse source text.
type VariableExpander interface {
	Expand(string, expand.Lookup) (string, error)
}
type VariableExpanderFunc func(string, expand.Lookup) (string, error)
func (fn VariableExpanderFunc) Expand(value string, lookup expand.Lookup) (string, error) {
	return fn(value, lookup)
}

// NamedContext is a named external build context.
type NamedContext struct {
	State    llb.State
	Image    dockerspec.DockerOCIImage
	Platform ocispecs.Platform
}

// Runtime is the scoped execution environment shared by handlers. Applications
// normally receive it through Call and do not need to mutate it directly.
type Runtime struct {
	BuildArgs    map[string]string
	GlobalArgs   map[string]string
	BuildContext llb.State
	NamedContexts map[string]NamedContext
	Platform     ocispecs.Platform
	CreatedAt    *time.Time
	Constraints  []llb.ConstraintsOpt
	StageConstraints []llb.ConstraintsOpt
	ImageResolver ImageResolver
	OnBuildDecoder OnBuildDecoder
	VariableExpander VariableExpander
	ExpansionConfig expand.Config
	ProgressReporter progress.Reporter

	StageName StageName
	StageIndex int
	StageTotal int
	InstructionIndex int
	InstructionTotal int
	OnBuild bool

	ResolveStage func(StageName) (Stage, error)
}

// StageEvent describes one stage lifecycle transition.
type StageEvent struct {
	Stage Stage
	Index int
	Total int
	Err   error
}
type StageHook func(context.Context, StageEvent) error

// InstructionEvent describes an instruction lifecycle transition.
type InstructionEvent struct {
	Instruction Instruction
	Stage       Stage
	StageIndex  int
	StageTotal  int
	Index       int
	Total       int
	Err         error
}
type InstructionHook func(context.Context, InstructionEvent) error

type InstructionHooks struct {
	Before InstructionHook
	After  InstructionHook
	Failed InstructionHook
}

// DockerfileHooks are build-level bindings. Instruction-specific hooks belong
// on Common; stage-specific hooks belong on StagePlan.
type DockerfileHooks struct {
	BeforeStage       StageHook
	AfterStage        StageHook
	StageFailed       StageHook
	BeforeInstruction InstructionHook
	AfterInstruction  InstructionHook
	InstructionFailed InstructionHook
}

// BuildOption configures the Dockerfile evaluator.
type BuildOption func(*Dockerfile)

type Dockerfile struct {
	Handlers *Handlers
	Platform ocispecs.Platform

	buildArgs      map[string]string
	buildContext   llb.State
	namedContexts  map[string]NamedContext
	imageResolver  ImageResolver
	onBuildDecoder OnBuildDecoder
	expander       VariableExpander
	expansionConfig expand.Config
	createdAt      *time.Time
	constraints    []llb.ConstraintsOpt
	reporter       progress.Reporter
	hooks          DockerfileHooks
	target         StageName

	global []Instruction
	stages []StagePlan
}

// StagePlan binds a stage initializer, its instructions, and stage-local
// lifecycle hooks. Stage-specific policy is kept with the stage rather than
// leaking into individual instruction constructors.
type StagePlan struct {
	Name     StageName
	Init     StageInit
	Body     []Instruction
	Hooks    StageHooks
	Handlers *Handlers
	Constraints []llb.ConstraintsOpt
}

// StageHooks are attached to one stage declaration.
type StageHooks struct {
	Before StageHook
	After  StageHook
	Failed StageHook
}

type StageOption func(*StagePlan)
func WithStageName(name StageName) StageOption {
	return func(plan *StagePlan) { plan.Name = name }
}
func WithStageHooks(hooks StageHooks) StageOption {
	return func(plan *StagePlan) { plan.Hooks = hooks }
}
func WithStageHandlers(handlers *Handlers) StageOption {
	return func(plan *StagePlan) { plan.Handlers = handlers }
}
func WithStageConstraints(options ...llb.ConstraintsOpt) StageOption {
	return func(plan *StagePlan) { plan.Constraints = append(plan.Constraints, options...) }
}

// Plan constructs a stage plan without mutating a Dockerfile definition.
func Plan(init StageInit, body []Instruction, options ...StageOption) StagePlan {
	plan := StagePlan{Init: init, Body: append([]Instruction(nil), body...)}
	for _, option := range options {
		if option != nil { option(&plan) }
	}
	if plan.Name == "" && init != nil {
		switch spec := init.SpecValue().(type) {
		case FromSpec:
			plan.Name = spec.Name
		case ResetSpec:
			plan.Name = spec.Name
		}
	}
	return plan
}

// Result contains the target stage and all stages evaluated to build it.
type Result struct {
	Target     Stage
	Stages     map[StageName]Stage
	GlobalArgs map[string]string
}

// New creates a Dockerfile evaluator with standard BuildKit-backed handlers.
func New(options ...BuildOption) *Dockerfile {
	platform := ocispecs.Platform{OS: runtime.GOOS, Architecture: runtime.GOARCH}
	d := &Dockerfile{
		Handlers: NewLLBHandlers(),
		Platform: platform,
		buildArgs: make(map[string]string),
		buildContext: llb.Local("context"),
		namedContexts: make(map[string]NamedContext),
	}
	for _, option := range options {
		if option != nil { option(d) }
	}
	return d
}

func WithHandlers(handlers *Handlers) BuildOption {
	return func(d *Dockerfile) { d.Handlers = handlers }
}
func WithPlatform(platform ocispecs.Platform) BuildOption {
	return func(d *Dockerfile) {
		platform.OSFeatures = append([]string(nil), platform.OSFeatures...)
		d.Platform = platform
	}
}
func WithBuildArgs(args map[string]string) BuildOption {
	return func(d *Dockerfile) {
		for key, value := range args { d.buildArgs[key] = value }
	}
}
func WithBuildContext(state llb.State) BuildOption {
	return func(d *Dockerfile) { d.buildContext = state }
}
func WithNamedContext(name string, state llb.State) BuildOption {
	return func(d *Dockerfile) { d.namedContexts[name] = NamedContext{State: state, Platform: d.Platform} }
}
func WithNamedContextImage(name string, state llb.State, image dockerspec.DockerOCIImage, platform ocispecs.Platform) BuildOption {
	return func(d *Dockerfile) {
		d.namedContexts[name] = NamedContext{State: state, Image: cloneImage(image), Platform: platform}
	}
}
func WithImageResolver(resolve ImageResolver) BuildOption {
	return func(d *Dockerfile) { d.imageResolver = resolve }
}
func WithOnBuildDecoder(decode OnBuildDecoder) BuildOption {
	return func(d *Dockerfile) { d.onBuildDecoder = decode }
}
func WithVariableExpander(expander VariableExpander) BuildOption {
	return func(d *Dockerfile) { d.expander = expander }
}
func WithExpansionConfig(config expand.Config) BuildOption {
	return func(d *Dockerfile) { d.expansionConfig = config }
}
func WithProgressReporter(reporter progress.Reporter) BuildOption {
	return func(d *Dockerfile) { d.reporter = reporter }
}
func WithCreatedAt(timestamp time.Time) BuildOption {
	return func(d *Dockerfile) { t := timestamp; d.createdAt = &t }
}
func WithLLBConstraints(options ...llb.ConstraintsOpt) BuildOption {
	return func(d *Dockerfile) { d.constraints = append(d.constraints, options...) }
}
func WithDockerfileHooks(hooks DockerfileHooks) BuildOption {
	return func(d *Dockerfile) { d.hooks = hooks }
}
func WithTarget(name StageName) BuildOption {
	return func(d *Dockerfile) { d.target = name }
}

// AddGlobal appends metadata instructions that run before every stage. Only
// ARG is accepted here, matching Docker build-argument scope rules.
func (d *Dockerfile) AddGlobal(instructions ...Instruction) error {
	if d == nil { return errors.New("dockerfile: nil definition") }
	for _, in := range instructions {
		if in == nil { return errors.New("dockerfile: nil global instruction") }
		if in.Kind() != KindArg {
			return fmt.Errorf("dockerfile: %s is not allowed before the first stage", in.Name())
		}
		d.global = append(d.global, in)
	}
	return nil
}

// AddStage appends a new stage. Use AddStagePlan when attaching stage hooks or
// a stage-local handler table.
func (d *Dockerfile) AddStage(init StageInit, body ...Instruction) error {
	return d.AddStagePlan(Plan(init, body, nil))
}
func (d *Dockerfile) AddStagePlan(plan StagePlan) error {
	if d == nil { return errors.New("dockerfile: nil definition") }
	if plan.Init == nil { return errors.New("dockerfile: stage initializer is required") }
	for _, in := range plan.Body {
		if in == nil { return errors.New("dockerfile: nil instruction in stage body") }
		if _, ok := in.(StageInit); ok {
			return fmt.Errorf("dockerfile: %s cannot appear inside a stage body", in.Name())
		}
	}
	if plan.Name == "" {
		plan.Name = StageName(fmt.Sprintf("stage-%d", len(d.stages)))
	}
	for _, existing := range d.stages {
		if normalizeStageName(existing.Name) == normalizeStageName(plan.Name) {
			return fmt.Errorf("dockerfile: duplicate stage name %q", plan.Name)
		}
	}
	plan.Body = append([]Instruction(nil), plan.Body...)
	plan.Constraints = append([]llb.ConstraintsOpt(nil), plan.Constraints...)
	d.stages = append(d.stages, plan)
	return nil
}

// Build evaluates the requested target and its dependency DAG. With no target
// configured, the last declared stage is the target.
func (d *Dockerfile) Build(ctx context.Context) (Result, error) {
	if d == nil { return Result{}, errors.New("dockerfile: nil definition") }
	if len(d.stages) == 0 { return Result{}, errors.New("dockerfile: no stages defined") }
	if d.Handlers == nil { return Result{}, errors.New("dockerfile: no handler registry configured") }

	runtime := &Runtime{
		BuildArgs: maps.Clone(d.buildArgs),
		GlobalArgs: defaultPlatformArgs(d.Platform),
		BuildContext: d.buildContext,
		NamedContexts: maps.Clone(d.namedContexts),
		Platform: d.Platform,
		CreatedAt: d.createdAt,
		Constraints: append([]llb.ConstraintsOpt(nil), d.constraints...),
		ImageResolver: d.imageResolver,
		OnBuildDecoder: d.onBuildDecoder,
		VariableExpander: d.expander,
		ExpansionConfig: d.expansionConfig,
		ProgressReporter: d.reporter,
	}
	if runtime.BuildArgs == nil { runtime.BuildArgs = make(map[string]string) }
	runtime.StageTotal = len(d.stages)
	runtime.InstructionTotal = len(d.global)

	// Global ARGs have no stage state. They populate the global lookup before
	// FROM references are evaluated.
	for index, in := range d.global {
		if err := ctx.Err(); err != nil { return Result{}, err }
		runtime.StageName = ""
		runtime.StageIndex = 0
		runtime.InstructionIndex = index + 1
		started := time.Now()
		if err := d.report(ctx, runtime, eventFor(progress.Started, in, "", 0, len(d.stages), index+1, len(d.global), d.Platform, in.CommonOptions(), started, nil)); err != nil { return Result{}, err }
		stage, err := in.Apply(ctx, Stage{}, d.Handlers, runtime)
		if err != nil { return Result{}, fmt.Errorf("dockerfile: global %s: %w", in.Name(), err) }
		_ = stage
		if err := d.report(ctx, runtime, eventFor(progress.Completed, in, "", 0, len(d.stages), index+1, len(d.global), d.Platform, in.CommonOptions(), started, nil)); err != nil { return Result{}, err }
	}

	plans := make(map[string]int, len(d.stages))
	for index, plan := range d.stages { plans[normalizeStageName(plan.Name)] = index }
	target := d.target
	if target == "" { target = d.stages[len(d.stages)-1].Name }
	targetIndex, ok := plans[normalizeStageName(target)]
	if !ok { return Result{}, fmt.Errorf("dockerfile: target stage %q not found", target) }

	states := make(map[StageName]Stage, len(d.stages))
	state := make(map[int]uint8, len(d.stages))
	var evaluate func(int) error
	evaluate = func(index int) error {
		switch state[index] {
		case 2:
			return nil
		case 1:
			return fmt.Errorf("dockerfile: circular stage dependency involving %q", d.stages[index].Name)
		}
		state[index] = 1
		plan := d.stages[index]
		for _, dependency := range planDependencies(plan) {
			depIndex, found := plans[normalizeStageName(dependency)]
			if !found {
				return fmt.Errorf("dockerfile: stage %q depends on unknown stage %q", plan.Name, dependency)
			}
			if depIndex == index {
				return fmt.Errorf("dockerfile: stage %q depends on itself", plan.Name)
			}
			if err := evaluate(depIndex); err != nil { return err }
		}

		runtime.ResolveStage = func(name StageName) (Stage, error) {
			for stageName, result := range states {
				if normalizeStageName(stageName) == normalizeStageName(name) { return cloneStage(result), nil }
			}
			return Stage{}, fmt.Errorf("dockerfile: stage %q has not been resolved", name)
		}
		handlers := plan.Handlers
		if handlers == nil { handlers = d.Handlers }

		stageIndex := index + 1
		total := len(d.stages)
		runtime.StageName = plan.Name
		runtime.StageIndex = stageIndex
		runtime.StageTotal = total
		runtime.StageConstraints = append([]llb.ConstraintsOpt(nil), plan.Constraints...)
		runtime.InstructionTotal = 1 + len(plan.Body)
		runtime.InstructionIndex = 1
		if err := callStageHook(ctx, d.hooks.BeforeStage, StageEvent{Index: stageIndex, Total: total}); err != nil { return err }
		if err := callStageHook(ctx, plan.Hooks.Before, StageEvent{Index: stageIndex, Total: total}); err != nil { return err }

		platform := d.Platform
		if requested, ok := stagePlatform(plan.Init); ok { platform = requested }
		initCommon := plan.Init.CommonOptions()
		if err := d.report(ctx, runtime, eventFor(progress.Started, plan.Init, plan.Name, stageIndex, total, 1, 1+len(plan.Body), platform, initCommon, time.Now(), nil)); err != nil { return err }
		started := time.Now()
		stage, err := plan.Init.Apply(ctx, Stage{}, handlers, runtime)
		if err != nil {
			_ = d.report(ctx, runtime, eventFor(progress.Failed, plan.Init, plan.Name, stageIndex, total, 1, 1+len(plan.Body), platform, initCommon, started, err))
			_ = callStageHook(ctx, plan.Hooks.Failed, StageEvent{Index: stageIndex, Total: total, Err: err})
			_ = callStageHook(ctx, d.hooks.StageFailed, StageEvent{Index: stageIndex, Total: total, Err: err})
			return fmt.Errorf("dockerfile: stage %q FROM: %w", plan.Name, err)
		}
		stage.Name = plan.Name
		if err := d.report(ctx, runtime, eventFor(progress.Completed, plan.Init, stage.Name, stageIndex, total, 1, 1+len(plan.Body), stage.Platform, initCommon, started, nil)); err != nil { return err }

		var onBuildCount int
		stage, onBuildCount, err = d.fireOnBuild(ctx, stage, handlers, runtime, plan, stageIndex, total)
		if err != nil { return fmt.Errorf("dockerfile: stage %q ONBUILD: %w", plan.Name, err) }

		instructionTotal := 1 + onBuildCount + len(plan.Body)
		for bodyIndex, in := range plan.Body {
			if err := ctx.Err(); err != nil { return err }
			runtime.StageName = plan.Name
			runtime.StageIndex = stageIndex
			runtime.StageTotal = total
			runtime.InstructionIndex = bodyIndex + onBuildCount + 2
			runtime.InstructionTotal = instructionTotal
			stage, err = d.applyInstruction(ctx, runtime, handlers, in, stage, stageIndex, total, bodyIndex+onBuildCount+2, instructionTotal)
			if err != nil {
				failure := StageEvent{Stage: cloneStage(stage), Index: stageIndex, Total: total, Err: err}
				_ = callStageHook(ctx, plan.Hooks.Failed, failure)
				_ = callStageHook(ctx, d.hooks.StageFailed, failure)
				return fmt.Errorf("dockerfile: stage %q instruction %s: %w", plan.Name, in.Name(), err)
			}
		}
		if err := callStageHook(ctx, plan.Hooks.After, StageEvent{Stage: cloneStage(stage), Index: stageIndex, Total: total}); err != nil { return err }
		if err := callStageHook(ctx, d.hooks.AfterStage, StageEvent{Stage: cloneStage(stage), Index: stageIndex, Total: total}); err != nil { return err }
		states[plan.Name] = cloneStage(stage)
		state[index] = 2
		return nil
	}
	if err := evaluate(targetIndex); err != nil { return Result{}, err }

	targetStage := states[d.stages[targetIndex].Name]
	return Result{Target: cloneStage(targetStage), Stages: cloneStageMap(states), GlobalArgs: maps.Clone(runtime.GlobalArgs)}, nil
}

func (d *Dockerfile) applyInstruction(ctx context.Context, runtime *Runtime, handlers *Handlers, in Instruction, stage Stage, stageIndex, stageTotal, index, total int) (Stage, error) {
	common := in.CommonOptions()
	started := time.Now()
	event := eventFor(progress.Started, in, stage.Name, stageIndex, stageTotal, index, total, stage.Platform, common, started, nil)
	if err := d.report(ctx, runtime, event); err != nil { return stage, err }
	instructionEvent := InstructionEvent{Instruction: in, Stage: cloneStage(stage), StageIndex: stageIndex, StageTotal: stageTotal, Index: index, Total: total}
	if common.Hooks.Before != nil {
		if err := common.Hooks.Before(ctx, instructionEvent); err != nil { return stage, err }
	}
	if d.hooks.BeforeInstruction != nil {
		if err := d.hooks.BeforeInstruction(ctx, instructionEvent); err != nil { return stage, err }
	}
	next, err := in.Apply(ctx, stage, handlers, runtime)
	if err != nil {
		_ = d.report(ctx, runtime, eventFor(progress.Failed, in, stage.Name, stageIndex, stageTotal, index, total, stage.Platform, common, started, err))
		instructionEvent.Err = err
		if common.Hooks.Failed != nil { _ = common.Hooks.Failed(ctx, instructionEvent) }
		if d.hooks.InstructionFailed != nil { _ = d.hooks.InstructionFailed(ctx, instructionEvent) }
		return stage, err
	}
	if err := d.report(ctx, runtime, eventFor(progress.Completed, in, next.Name, stageIndex, stageTotal, index, total, next.Platform, common, started, nil)); err != nil {
		return next, err
	}
	instructionEvent.Stage = cloneStage(next)
	if common.Hooks.After != nil {
		if err := common.Hooks.After(ctx, instructionEvent); err != nil { return next, err }
	}
	if d.hooks.AfterInstruction != nil {
		if err := d.hooks.AfterInstruction(ctx, instructionEvent); err != nil { return next, err }
	}
	return next, nil
}

func (d *Dockerfile) fireOnBuild(ctx context.Context, stage Stage, handlers *Handlers, runtime *Runtime, plan StagePlan, stageIndex, stageTotal int) (Stage, int, error) {
	triggers := append([]Trigger(nil), stage.Triggers...)
	rawTriggers := append([]string(nil), stage.RawOnBuild...)
	if len(rawTriggers) == 0 && len(stage.Triggers) == 0 && len(stage.Image.Config.OnBuild) > 0 {
		rawTriggers = append(rawTriggers, stage.Image.Config.OnBuild...)
	}
	if len(rawTriggers) > 0 {
		if runtime.OnBuildDecoder == nil {
			return stage, 0, fmt.Errorf("base image contains raw ONBUILD metadata; configure WithOnBuildDecoder to convert it into typed instructions")
		}
		decoded, err := runtime.OnBuildDecoder(ctx, rawTriggers)
		if err != nil { return stage, 0, fmt.Errorf("decode base image ONBUILD metadata: %w", err) }
		triggers = append(decoded, triggers...)
	}
	stage.Triggers = nil
	stage.RawOnBuild = nil
	stage.Image.Config.OnBuild = nil
	for index, trigger := range triggers {
		if trigger == nil { return stage, index, errors.New("nil ONBUILD trigger") }
		runtime.StageName = plan.Name
		runtime.StageIndex = stageIndex
		runtime.StageTotal = stageTotal
		runtime.InstructionIndex = index + 2
		runtime.InstructionTotal = 1 + len(triggers) + len(plan.Body)
		runtime.OnBuild = true
		var err error
		stage, err = d.applyInstruction(ctx, runtime, handlers, trigger, stage, stageIndex, stageTotal, index+2, 1+len(triggers)+len(plan.Body))
		runtime.OnBuild = false
		if err != nil { return stage, index+1, fmt.Errorf("trigger %s: %w", trigger.Name(), err) }
	}
	return stage, len(triggers), nil
}

func (d *Dockerfile) report(ctx context.Context, runtime *Runtime, event progress.Event) error {
	if runtime.ProgressReporter == nil { return nil }
	return runtime.ProgressReporter.Report(ctx, event)
}

func eventFor(phase progress.Phase, in Instruction, stageName StageName, stageIndex, stageTotal, index, total int, platform ocispecs.Platform, common Common, started time.Time, err error) progress.Event {
	group := common.ProgressGroup
	if group == "" { group = string(stageName) }
	if group == "" { group = "build" }
	message := common.ProgressTitle
	if message == "" { message = in.String() }
	finished := time.Time{}
	if phase != progress.Started { finished = time.Now() }
	return progress.Event{
		Phase: phase, Group: group, Message: message, Instruction: in.Name(),
		InstructionIndex: index, InstructionTotal: total, StageName: string(stageName),
		StageIndex: stageIndex, StageTotal: stageTotal, Platform: platform,
		PlatformSpecified: platformSpecified(platform), StartedAt: started,
		FinishedAt: finished, Err: err,
	}
}

func platformSpecified(platform ocispecs.Platform) bool {
	return platform.OS != "" && platform.Architecture != ""
}

func callStageHook(ctx context.Context, hook StageHook, event StageEvent) error {
	if hook == nil { return nil }
	return hook(ctx, event)
}

func planDependencies(plan StagePlan) []StageName {
	var result []StageName
	result = append(result, plan.Init.Dependencies()...)
	for _, in := range plan.Body { result = append(result, in.Dependencies()...) }
	return uniqueStageNames(result)
}

func stagePlatform(init StageInit) (ocispecs.Platform, bool) {
	switch spec := init.SpecValue().(type) {
	case FromSpec:
		if spec.Platform != nil { return *spec.Platform, true }
	case ResetSpec:
		if spec.Platform != nil { return *spec.Platform, true }
	}
	return ocispecs.Platform{}, false
}

func cloneStage(src Stage) Stage {
	dst := src
	dst.Image = cloneImage(src.Image)
	dst.Platform.OSFeatures = slices.Clone(src.Platform.OSFeatures)
	dst.Base.Image = cloneImage(src.Base.Image)
	dst.Base.Platform.OSFeatures = slices.Clone(src.Base.Platform.OSFeatures)
	dst.Args = maps.Clone(src.Args)
	dst.Env = maps.Clone(src.Env)
	dst.Triggers = append([]Trigger(nil), src.Triggers...)
	dst.RawOnBuild = append([]string(nil), src.RawOnBuild...)
	return dst
}

func cloneStageMap(src map[StageName]Stage) map[StageName]Stage {
	dst := make(map[StageName]Stage, len(src))
	for name, stage := range src { dst[name] = cloneStage(stage) }
	return dst
}

func cloneImage(src dockerspec.DockerOCIImage) dockerspec.DockerOCIImage {
	dst := src
	dst.OSFeatures = slices.Clone(src.OSFeatures)
	dst.RootFS.DiffIDs = slices.Clone(src.RootFS.DiffIDs)
	dst.History = slices.Clone(src.History)
	for index := range dst.History {
		if src.History[index].Created != nil {
			created := *src.History[index].Created
			dst.History[index].Created = &created
		}
	}
	dst.Config.Env = slices.Clone(src.Config.Env)
	dst.Config.Entrypoint = slices.Clone(src.Config.Entrypoint)
	dst.Config.Cmd = slices.Clone(src.Config.Cmd)
	dst.Config.Shell = slices.Clone(src.Config.Shell)
	dst.Config.OnBuild = slices.Clone(src.Config.OnBuild)
	dst.Config.ExposedPorts = maps.Clone(src.Config.ExposedPorts)
	dst.Config.Volumes = maps.Clone(src.Config.Volumes)
	dst.Config.Labels = maps.Clone(src.Config.Labels)
	if src.Created != nil { created := *src.Created; dst.Created = &created }
	if src.Config.Healthcheck != nil {
		health := *src.Config.Healthcheck
		health.Test = slices.Clone(src.Config.Healthcheck.Test)
		dst.Config.Healthcheck = &health
	}
	return dst
}

// Expand expands against stage environment, then stage ARGs, then global ARGs.
// There is deliberately no source escape-token or Dockerfile parser dependency.
func (r *Runtime) Expand(value string, stage Stage) (string, error) {
	if r == nil {
		return value, nil
	}
	lookup := func(key string) (string, bool) {
		if v, ok := stage.Env[key]; ok { return v, true }
		if v, ok := stage.Args[key]; ok { return v, true }
		if stage.Name == "" {
			if v, ok := r.GlobalArgs[key]; ok { return v, true }
		}
		return "", false
	}
	if r.VariableExpander != nil { return r.VariableExpander.Expand(value, lookup) }
	return expand.New(r.ExpansionConfig, lookup).Expand(value)
}

func environmentFromConfig(values []string) map[string]string {
	result := make(map[string]string, len(values))
	for _, item := range values {
		key, value, ok := strings.Cut(item, "=")
		if ok { result[key] = value }
	}
	return result
}

func emptyImage(platform ocispecs.Platform) dockerspec.DockerOCIImage {
	image := dockerspec.DockerOCIImage{}
	image.OS, image.Architecture, image.Variant = platform.OS, platform.Architecture, platform.Variant
	image.OSVersion = platform.OSVersion
	image.OSFeatures = slices.Clone(platform.OSFeatures)
	image.RootFS.Type = "layers"
	image.Config.WorkingDir = "/"
	if !strings.EqualFold(platform.OS, "windows") {
		image.Config.Env = []string{"PATH=/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin"}
	}
	return image
}


// newRuntime provides the standalone execution context used by Compose.
func newRuntime() *Runtime {
	platform := ocispecs.Platform{OS: runtime.GOOS, Architecture: runtime.GOARCH}
	return &Runtime{
		BuildArgs: make(map[string]string),
		GlobalArgs: defaultPlatformArgs(platform),
		BuildContext: llb.Local("context"),
		NamedContexts: make(map[string]NamedContext),
		Platform: platform,
	}
}

// NewRuntime returns a ready-to-use runtime for composing a single stage
// outside the top-level Dockerfile DAG evaluator.
func NewRuntime() *Runtime { return newRuntime() }


// defaultPlatformArgs exposes BuildKit's automatic platform argument set.
// Values remain global until an ARG instruction makes a particular value
// available inside a stage.
func defaultPlatformArgs(target ocispecs.Platform) map[string]string {
	build := ocispecs.Platform{OS: runtime.GOOS, Architecture: runtime.GOARCH}
	values := map[string]string{
		"BUILDPLATFORM": platformString(build),
		"BUILDOS": build.OS,
		"BUILDARCH": build.Architecture,
		"BUILDVARIANT": build.Variant,
		"BUILDOSVERSION": build.OSVersion,
		"BUILDOSFEATURES": strings.Join(build.OSFeatures, ","),
		"TARGETPLATFORM": platformString(target),
		"TARGETOS": target.OS,
		"TARGETARCH": target.Architecture,
		"TARGETVARIANT": target.Variant,
		"TARGETOSVERSION": target.OSVersion,
		"TARGETOSFEATURES": strings.Join(target.OSFeatures, ","),
	}
	return values
}

func platformString(platform ocispecs.Platform) string {
	value := platform.OS + "/" + platform.Architecture
	if platform.Variant != "" { value += "/" + platform.Variant }
	return value
}
