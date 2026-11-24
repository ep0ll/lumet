package dockerfile

import (
	"errors"
	"fmt"
	"runtime"
	"time"

	"github.com/ep0ll/lumet/dockerfile/expand"
	"github.com/ep0ll/lumet/dockerfile/progress"
	"github.com/moby/buildkit/client/llb"
	dockerspec "github.com/moby/docker-image-spec/specs-go/v1"
	ocispecs "github.com/opencontainers/image-spec/specs-go/v1"
)

type DockerfileHooks struct {
	BeforeStage       StageHook
	AfterStage        StageHook
	StageFailed       StageHook
	BeforeInstruction InstructionHook
	AfterInstruction  InstructionHook
	InstructionFailed InstructionHook
}

type BuildOption func(*Dockerfile)

type Dockerfile struct {
	Platform ocispecs.Platform
	hooks    DockerfileHooks

	buildArgs      map[string]string
	buildContext   llb.State
	namedContexts  map[string]NamedContext
	imageResolver  ImageResolver
	onBuildDecoder OnBuildDecoder
	expander       VariableExpander
	expansion      expand.Config
	createdAt      *time.Time
	constraints    []llb.ConstraintsOpt
	reporter       progress.Reporter
	cacheIDNamespace string
	bindMountNoOutput bool
	target         StageName

	global []Instruction
	stages []StagePlan
}

type Result struct {
	Target     State
	Stages     map[StageName]State
	GlobalArgs map[string]string
}

// New creates an empty Dockerfile definition with the current platform and a
// default local build context.
func New(options ...BuildOption) *Dockerfile {
	platform := ocispecs.Platform{
		OS: runtime.GOOS, Architecture: runtime.GOARCH,
	}
	d := &Dockerfile{
		Platform: platform,
		buildArgs: make(map[string]string),
		buildContext: llb.Local("context"),
		namedContexts: make(map[string]NamedContext),
	}
	for _, option := range options {
		if option != nil {
			option(d)
		}
	}
	return d
}

func WithPlatform(platform ocispecs.Platform) BuildOption {
	return func(d *Dockerfile) {
		platform.OSFeatures = append([]string(nil), platform.OSFeatures...)
		d.Platform = platform
	}
}
func WithBuildArgs(args map[string]string) BuildOption {
	return func(d *Dockerfile) {
		for key, value := range args {
			d.buildArgs[key] = value
		}
	}
}
func WithBuildContext(state llb.State) BuildOption {
	return func(d *Dockerfile) { d.buildContext = state }
}
func WithNamedContext(name string, state llb.State) BuildOption {
	return func(d *Dockerfile) {
		d.namedContexts[name] = NamedContext{LLB: state}
	}
}
func WithNamedContextImage(name string, state llb.State, image dockerspec.DockerOCIImage, platform ocispecs.Platform) BuildOption {
	return func(d *Dockerfile) {
		d.namedContexts[name] = NamedContext{
			LLB: state, Image: cloneImage(image), Platform: platform,
		}
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
	return func(d *Dockerfile) { d.expansion = config }
}
func WithProgressReporter(reporter progress.Reporter) BuildOption {
	return func(d *Dockerfile) { d.reporter = reporter }
}
func WithCreatedAt(timestamp time.Time) BuildOption {
	return func(d *Dockerfile) {
		value := timestamp
		d.createdAt = &value
	}
}
func WithBuildConstraints(options ...llb.ConstraintsOpt) BuildOption {
	return func(d *Dockerfile) {
		d.constraints = append(d.constraints, options...)
	}
}
func WithCacheIDNamespace(namespace string) BuildOption {
	return func(d *Dockerfile) { d.cacheIDNamespace = namespace }
}
func WithBindMountNoOutput(supported bool) BuildOption {
	return func(d *Dockerfile) { d.bindMountNoOutput = supported }
}
func WithDockerfileHooks(hooks DockerfileHooks) BuildOption {
	return func(d *Dockerfile) { d.hooks = hooks }
}
func WithTarget(name StageName) BuildOption {
	return func(d *Dockerfile) { d.target = name }
}

// AddGlobal appends global ARG instructions. Global ARGs are evaluated before
// FROM references so they can participate in image names and platform values.
func (d *Dockerfile) AddGlobal(instructions ...Instruction) error {
	if d == nil {
		return errors.New("dockerfile: nil definition")
	}
	for _, instruction := range instructions {
		if instruction.name == "" || instruction.action == nil {
			return errors.New("dockerfile: invalid global instruction")
		}
		if !instruction.Metadata().GlobalARG {
			return fmt.Errorf("dockerfile: %s is not permitted in global scope", instruction.name)
		}
		d.global = append(d.global, instruction)
	}
	return nil
}

func (d *Dockerfile) AddStage(from StageFrom, instructions ...Instruction) error {
	return d.AddStagePlan(Plan(from, instructions, nil))
}

func (d *Dockerfile) AddStagePlan(plan StagePlan) error {
	if d == nil {
		return errors.New("dockerfile: nil definition")
	}
	if plan.From.action == nil {
		return errors.New("dockerfile: stage FROM closure is required")
	}
	for _, instruction := range plan.Instructions {
		if instruction.action == nil || instruction.name == "" {
			return errors.New("dockerfile: stage contains an invalid instruction")
		}
	}
	if plan.Name == "" {
		plan.Name = plan.From.Name()
	}
	if plan.Name == "" {
		plan.Name = StageName(fmt.Sprintf("stage-%d", len(d.stages)))
	}
	for _, existing := range d.stages {
		if normalizeStageName(existing.Name) == normalizeStageName(plan.Name) {
			return fmt.Errorf("dockerfile: duplicate stage name %q", plan.Name)
		}
	}
	if plan.From.Name() == "" {
		plan.From = plan.From.With(WithDescription("FROM "+string(plan.Name)))
		plan.From.name = plan.Name
	}
	plan.Instructions = append([]Instruction(nil), plan.Instructions...)
	plan.Constraints = append([]llb.ConstraintsOpt(nil), plan.Constraints...)
	d.stages = append(d.stages, plan)
	return nil
}

// Build resolves only the stages reachable from the target. A stage is
// evaluated once even when several downstream stages depend on it.
