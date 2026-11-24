package dockerfile

import (
	"context"
	"runtime"
	"strings"
	"time"

	"github.com/ep0ll/lumet/dockerfile/expand"
	"github.com/ep0ll/lumet/dockerfile/progress"
	"github.com/moby/buildkit/client/llb"
	dockerspec "github.com/moby/docker-image-spec/specs-go/v1"
	ocispecs "github.com/opencontainers/image-spec/specs-go/v1"
)

// ImageResolver supplies the image config that LLB's image source alone does
// not expose to the Dockerfile evaluator.
type ImageResolver func(context.Context, string, ocispecs.Platform) (dockerspec.DockerOCIImage, error)

// OnBuildDecoder converts persisted Config.OnBuild strings from external
// images into executable typed instructions.
type OnBuildDecoder func(context.Context, []string) ([]Instruction, error)

// VariableExpander can replace the default expander without coupling the core
// evaluator to a parser or shell implementation.
type VariableExpander func(string, expand.Lookup) (string, error)

// NamedContext is a build context supplied by name.
type NamedContext struct {
	LLB      llb.State
	Image    dockerspec.DockerOCIImage
	Platform ocispecs.Platform
}

// Runtime contains dependencies and data available while an instruction
// closure is being evaluated. It is created per Build call.
type Runtime struct {
	BuildArgs     map[string]string
	GlobalArgs    map[string]string
	BuildContext  llb.State
	NamedContexts map[string]NamedContext
	Platform      ocispecs.Platform
	CreatedAt     *time.Time

	Constraints      []llb.ConstraintsOpt
	StageConstraints []llb.ConstraintsOpt
	CacheIDNamespace string
	BindMountNoOutput bool

	ImageResolver    ImageResolver
	OnBuildDecoder   OnBuildDecoder
	VariableExpander VariableExpander
	ExpansionConfig  expand.Config
	ProgressReporter progress.Reporter

	StageName         StageName
	StageIndex        int
	StageTotal        int
	InstructionIndex int
	InstructionTotal int
	OnBuild           bool

	ResolveStage func(StageName) (State, error)
}

// Expand resolves values in stage ENV, stage ARG, then global ARG scope. Global
// build arguments are visible before a stage begins, not implicitly in stages.
func (r *Runtime) Expand(value string, state *State) (string, error) {
	if r == nil {
		return value, nil
	}
	lookup := func(key string) (string, bool) {
		if state != nil {
			if found, ok := state.Env[key]; ok {
				return found, true
			}
			if found, ok := state.Args[key]; ok {
				return found, true
			}
			if state.StageName != "" {
				return "", false
			}
		}
		found, ok := r.GlobalArgs[key]
		return found, ok
	}
	if r.VariableExpander != nil {
		return r.VariableExpander(value, lookup)
	}
	return expand.New(r.ExpansionConfig, lookup).Expand(value)
}

func defaultPlatformArgs(target ocispecs.Platform) map[string]string {
	build := ocispecs.Platform{OS: runtime.GOOS, Architecture: runtime.GOARCH}
	return map[string]string{
		"BUILDPLATFORM":   platformString(build),
		"BUILDOS":         build.OS,
		"BUILDARCH":       build.Architecture,
		"BUILDVARIANT":    build.Variant,
		"BUILDOSVERSION":  build.OSVersion,
		"BUILDOSFEATURES": strings.Join(build.OSFeatures, ","),
		"TARGETPLATFORM":  platformString(target),
		"TARGETOS":        target.OS,
		"TARGETARCH":      target.Architecture,
		"TARGETVARIANT":   target.Variant,
		"TARGETOSVERSION": target.OSVersion,
		"TARGETOSFEATURES": strings.Join(target.OSFeatures, ","),
	}
}

func platformString(platform ocispecs.Platform) string {
	value := platform.OS + "/" + platform.Architecture
	if platform.Variant != "" {
		value += "/" + platform.Variant
	}
	return value
}
