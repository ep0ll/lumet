// Package dockerfile provides a functional, Dockerfile-inspired API for building
// BuildKit LLB states and Docker/OCI image configuration.
//
// Each instruction constructor returns an Instruction. Instructions are applied
// in order to a Build, which carries both the immutable LLB state and mutable
// image metadata. The package intentionally does not parse Dockerfile text.
// Build values are mutable and are not safe for concurrent instruction application.
package dockerfile

import (
	"context"
	"fmt"
	"maps"
	"reflect"
	"runtime"
	"slices"
	"strings"
	"time"

	"github.com/moby/buildkit/client/llb"
	"github.com/ep0ll/lumet/dockerfile/expand"
	dockerspec "github.com/moby/docker-image-spec/specs-go/v1"
	ocispecs "github.com/opencontainers/image-spec/specs-go/v1"
)

// Instruction is a build step. Constructors such as From, Run and Copy return
// Instructions, which can be composed with Build.Apply.
type Instruction func(context.Context, *Build) error

// ImageResolver resolves the image configuration for a FROM instruction.
// LLB image resolution remains BuildKit's responsibility; this callback gives
// the builder the Docker-specific config needed for history and metadata.
type ImageResolver func(context.Context, string, ocispecs.Platform) (dockerspec.DockerOCIImage, error)

// VariableExpander is a pluggable variable expansion strategy. It receives a
// lookup scoped to the current instruction and the configured source escape rune.
type VariableExpander interface {
	Expand(input string, lookup expand.Lookup, config expand.Config) (string, error)
}

// VariableExpanderFunc adapts a function to VariableExpander.
type VariableExpanderFunc func(string, expand.Lookup, expand.Config) (string, error)

func (fn VariableExpanderFunc) Expand(input string, lookup expand.Lookup, config expand.Config) (string, error) {
	return fn(input, lookup, config)
}

// HistoryRecorder observes history entries after they have been appended.
type HistoryRecorder func(context.Context, ocispecs.History) error

// BuildOption configures a new Build.
type BuildOption func(*Build)

// Build is the mutable instruction-application context. Its State value remains
// immutable: each LLB operation returns a new state graph.
type Build struct {
	State        llb.State
	Image        dockerspec.DockerOCIImage
	Platform     ocispecs.Platform
	BuildContext llb.State
	BuildArgs    map[string]string

	ImageResolver   ImageResolver
	HistoryRecorder HistoryRecorder
	ProgressReporter progress.Reporter
	VariableExpander VariableExpander
	EscapeToken      rune

	namedContexts map[string]llb.State
	stages        map[string]Stage
	stageName     string
	started       bool
	cmdSet        bool
	stageIndex    int
	progressGroup string
	progressTitle string
	createdAt     *time.Time
	constraints   []llb.ConstraintsOpt
	globalArgs    map[string]string
	stageArgs     map[string]string
	stageEnv      map[string]string
}

// Stage is a named, immutable snapshot that can be used by a later FROM or
// COPY --from instruction.
type Stage struct {
	Name     string
	State    llb.State
	Image    dockerspec.DockerOCIImage
	Platform ocispecs.Platform
	Args     map[string]string
	BuildEnv map[string]string
}

// Result is the output of a completed build sequence.
type Result struct {
	State    llb.State
	Image    dockerspec.DockerOCIImage
	Platform ocispecs.Platform
	Stages   map[string]Stage
}

// New creates a Build with an empty root filesystem and the host platform.
// Configure the target platform and contexts with BuildOptions as needed.
func New(opts ...BuildOption) *Build {
	platform := ocispecs.Platform{OS: runtime.GOOS, Architecture: runtime.GOARCH}
	b := &Build{
		Platform:      platform,
		State:         llb.Scratch().Platform(platform),
		Image:         emptyImage(platform),
		BuildContext:  llb.Local("context"),
		BuildArgs:     make(map[string]string),
		globalArgs:    make(map[string]string),
		stageArgs:     make(map[string]string),
		stageEnv:      make(map[string]string),
		namedContexts: make(map[string]llb.State),
		stages:        make(map[string]Stage),
	}
	for _, opt := range opts {
		if opt != nil {
			opt(b)
		}
	}
	b.State = b.State.Platform(b.Platform)
	b.Image.OS = b.Platform.OS
	b.Image.Architecture = b.Platform.Architecture
	b.Image.Variant = b.Platform.Variant
	b.Image.OSVersion = b.Platform.OSVersion
	b.Image.OSFeatures = slices.Clone(b.Platform.OSFeatures)
	return b
}

// WithPlatform sets the target platform for the build.
func WithPlatform(platform ocispecs.Platform) BuildOption {
	return func(b *Build) {
		b.Platform = clonePlatform(platform)
		b.State = b.State.Platform(b.Platform)
	}
}

// WithBuildContext sets the default local state used by COPY and local ADD.
func WithBuildContext(state llb.State) BuildOption {
	return func(b *Build) { b.BuildContext = state }
}

// WithBuildArgs supplies build-time argument values. Instruction-level ARG
// declarations still control which values are made available to the build.
func WithBuildArgs(args map[string]string) BuildOption {
	return func(b *Build) {
		for key, value := range args {
			b.BuildArgs[key] = value
		}
	}
}

// WithImageResolver installs the callback used to obtain base-image metadata.
func WithImageResolver(resolve ImageResolver) BuildOption {
	return func(b *Build) { b.ImageResolver = resolve }
}

// WithNamedContext registers a named BuildKit context usable by FROM and
// COPY --from. A later registration replaces the earlier one.
func WithNamedContext(name string, state llb.State) BuildOption {
	return func(b *Build) { b.namedContexts[name] = state }
}

// WithCreatedAt fixes the timestamp written to image history. This is useful
// for reproducible image metadata.
func WithCreatedAt(t time.Time) BuildOption {
	return func(b *Build) {
		t := t
		b.createdAt = &t
	}
}

// WithProgressReporter installs a UI-agnostic observer for instruction lifecycle events.
func WithProgressReporter(reporter progress.Reporter) BuildOption {
	return func(b *Build) { b.ProgressReporter = reporter }
}

// WithVariableExpander installs a custom variable expansion strategy.
func WithVariableExpander(expander VariableExpander) BuildOption {
	return func(b *Build) { b.VariableExpander = expander }
}

// WithEscapeToken configures the source-format escape rune used when expanding
// variables. Dockerfile's default is backslash; a syntax directive may select backtick.
func WithEscapeToken(token rune) BuildOption {
	return func(b *Build) { b.EscapeToken = token }
}

// WithHistoryRecorder installs a callback invoked for each history entry.
func WithHistoryRecorder(recorder HistoryRecorder) BuildOption {
	return func(b *Build) { b.HistoryRecorder = recorder }
}

// WithConstraints adds LLB constraints to operations created by instructions,
// including RUN, COPY, ADD and WORKDIR.
func WithConstraints(opts ...llb.ConstraintsOpt) BuildOption {
	return func(b *Build) { b.constraints = append(b.constraints, opts...) }
}

// Apply executes instructions sequentially and reports lifecycle events.
// Instruction and stage totals are scoped to this Apply call.
func (b *Build) Apply(ctx context.Context, instructions ...Instruction) error {
	if b == nil {
		return fmt.Errorf("dockerfile: nil build")
	}
	stageTotal := 0
	for _, instruction := range instructions {
		if instruction != nil && instructionName(instruction) == "From" {
			stageTotal++
		}
	}
	stageIndex := b.stageIndex
	for index, instruction := range instructions {
		if err := ctx.Err(); err != nil {
			return fmt.Errorf("dockerfile: instruction %d: %w", index+1, err)
		}
		if instruction == nil {
			return fmt.Errorf("dockerfile: instruction %d is nil", index+1)
		}
		name := instructionName(instruction)
		b.progressGroup, b.progressTitle = "", ""
		if name == "From" {
			stageIndex++
		}
		started := time.Now()
		event := b.progressEvent(progress.Started, name, index+1, len(instructions), stageIndex, stageTotal, started, nil)
		if err := b.reportProgress(ctx, event); err != nil {
			return fmt.Errorf("dockerfile: progress start for instruction %d: %w", index+1, err)
		}
		if err := instruction(ctx, b); err != nil {
			failed := b.progressEvent(progress.Failed, name, index+1, len(instructions), stageIndex, stageTotal, started, err)
			_ = b.reportProgress(ctx, failed)
			return fmt.Errorf("dockerfile: instruction %d (%s): %w", index+1, name, err)
		}
		completed := b.progressEvent(progress.Completed, name, index+1, len(instructions), stageIndex, stageTotal, started, nil)
		if err := b.reportProgress(ctx, completed); err != nil {
			return fmt.Errorf("dockerfile: progress completion for instruction %d: %w", index+1, err)
		}
	}
	b.stageIndex = stageIndex
	return nil
}

func instructionName(instruction Instruction) string {
	fn := runtime.FuncForPC(reflect.ValueOf(instruction).Pointer())
	if fn == nil {
		return "instruction"
	}
	name := fn.Name()
	if at := strings.LastIndex(name, ".func"); at >= 0 {
		name = name[:at]
	}
	if at := strings.LastIndex(name, "."); at >= 0 {
		name = name[at+1:]
	}
	return name
}

func (b *Build) progressEvent(phase progress.Phase, instruction string, index, total, stageIndex, stageTotal int, started time.Time, err error) progress.Event {
	group := b.progressGroup
	if group == "" {
		group = b.stageName
	}
	if group == "" {
		group = "build"
	}
	message := b.progressTitle
	if message == "" {
		message = instruction
	}
	if b.stageName != "" {
		message = b.stageName + ": " + message
	}
	finished := time.Time{}
	if phase != progress.Started {
		finished = time.Now()
	}
	return progress.Event{
		Phase: phase, Group: group, Message: message, Instruction: instruction,
		InstructionIndex: index, InstructionTotal: total, StageName: b.stageName,
		StageIndex: stageIndex, StageTotal: stageTotal, Platform: clonePlatform(b.Platform),
		PlatformSpecified: b.Platform.OS != "" && b.Platform.Architecture != "",
		StartedAt: started, FinishedAt: finished, Err: err,
	}
}

func (b *Build) reportProgress(ctx context.Context, event progress.Event) error {
	if b.ProgressReporter == nil {
		return nil
	}
	return b.ProgressReporter.Report(ctx, event)
}

// Result returns a detached snapshot of the current state, image and named stages.
func (b *Build) Result() Result {
	if b == nil {
		return Result{}
	}
	stages := make(map[string]Stage, len(b.stages)+1)
	for name, stage := range b.stages {
		stages[name] = cloneStage(stage)
	}
	if b.started && b.stageName != "" {
		stages[b.stageName] = b.currentStage()
	}
	return Result{
		State: b.State, Image: cloneImage(b.Image),
		Platform: clonePlatform(b.Platform), Stages: stages,
	}
}

// Stage returns a previously completed stage, or the current stage when its
// name matches the requested name.
func (b *Build) Stage(name string) (Stage, bool) {
	if b == nil {
		return Stage{}, false
	}
	if b.started && strings.EqualFold(name, b.stageName) && b.stageName != "" {
		return cloneStage(b.currentStage()), true
	}
	stage, ok := b.lookupStage(name)
	if !ok {
		return Stage{}, false
	}
	return cloneStage(stage), true
}

func (b *Build) currentStage() Stage {
	return Stage{
		Name: b.stageName, State: b.State,
		Image: cloneImage(b.Image), Platform: clonePlatform(b.Platform),
		Args: maps.Clone(b.stageArgs), BuildEnv: maps.Clone(b.stageEnv),
	}
}

func (b *Build) saveStage() {
	if !b.started || b.stageName == "" {
		return
	}
	b.stages[b.stageName] = b.currentStage()
}

func (b *Build) lookupStage(name string) (Stage, bool) {
	if stage, ok := b.stages[name]; ok {
		return stage, true
	}
	for stageName, stage := range b.stages {
		if strings.EqualFold(stageName, name) {
			return stage, true
		}
	}
	return Stage{}, false
}

func (b *Build) requireStage(instruction string) error {
	if b == nil {
		return fmt.Errorf("%s: nil build", instruction)
	}
	if !b.started {
		return fmt.Errorf("%s must follow FROM", instruction)
	}
	return nil
}

func (b *Build) expand(value string) string {
	return b.expandMap(value, b.stageEnv)
}

func (b *Build) expandGlobal(value string) string {
	return b.expandMap(value, b.globalArgs)
}

func (b *Build) expandMap(value string, values map[string]string) string {
	lookup := expand.MapLookup(values)
	strategy := b.VariableExpander
	if strategy == nil {
		strategy = VariableExpanderFunc(func(input string, lookup expand.Lookup, config expand.Config) (string, error) {
			return expand.New(config, lookup).Expand(input)
		})
	}
	result, err := strategy.Expand(value, lookup, expand.Config{Escape: b.EscapeToken})
	if err != nil {
		return value
	}
	return result
}

func (b *Build) recordHistory(ctx context.Context, createdBy string, emptyLayer bool) error {
	if !b.started {
		return nil
	}
	var created *time.Time
	if b.createdAt != nil {
		t := *b.createdAt
		created = &t
	}
	entry := ocispecs.History{
		Created: created, CreatedBy: createdBy,
		Author: b.Image.Author, EmptyLayer: emptyLayer,
	}
	b.Image.History = append(b.Image.History, entry)
	if b.Image.Created == nil && created != nil {
		t := *created
		b.Image.Created = &t
	}
	if b.HistoryRecorder != nil {
		observed := entry
		if entry.Created != nil {
			created := *entry.Created
			observed.Created = &created
		}
		return b.HistoryRecorder(ctx, observed)
	}
	return nil
}

func cloneStage(src Stage) Stage {
	return Stage{
		Name: src.Name, State: src.State,
		Image: cloneImage(src.Image), Platform: clonePlatform(src.Platform),
		Args: maps.Clone(src.Args), BuildEnv: maps.Clone(src.BuildEnv),
	}
}

func environmentFromConfig(values []string) map[string]string {
	environment := make(map[string]string, len(values))
	for _, item := range values {
		key, value, found := strings.Cut(item, "=")
		if found {
			environment[key] = value
		}
	}
	return environment
}

func emptyImage(platform ocispecs.Platform) dockerspec.DockerOCIImage {
	image := dockerspec.DockerOCIImage{}
	image.OS = platform.OS
	image.Architecture = platform.Architecture
	image.OSVersion = platform.OSVersion
	image.OSFeatures = slices.Clone(platform.OSFeatures)
	image.Variant = platform.Variant
	image.RootFS.Type = "layers"
	image.Config.WorkingDir = "/"
	return image
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
	if src.Created != nil {
		t := *src.Created
		dst.Created = &t
	}
	if src.Config.Healthcheck != nil {
		health := *src.Config.Healthcheck
		health.Test = slices.Clone(src.Config.Healthcheck.Test)
		dst.Config.Healthcheck = &health
	}
	return dst
}

func clonePlatform(src ocispecs.Platform) ocispecs.Platform {
	dst := src
	dst.OSFeatures = slices.Clone(src.OSFeatures)
	return dst
}
