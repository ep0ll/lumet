package dockerfile

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"maps"
	"strings"

	"github.com/moby/buildkit/client/llb"
	dockerspec "github.com/moby/docker-image-spec/specs-go/v1"
	ocispecs "github.com/opencontainers/image-spec/specs-go/v1"
)

func initializeFrom(
	ctx context.Context,
	state *State,
	runtime *Runtime,
	metadata InstructionMetadata,
	base Base,
	config stageFromConfig,
) error {
	if runtime == nil {
		return errors.New("FROM requires a build runtime")
	}
	if base == nil {
		return errors.New("FROM base is required")
	}

	platform := runtime.Platform
	if config.platform != nil {
		platform = *config.platform
	}

	var (
		llbState         llb.State
		image            = emptyImage(platform)
		baseRef          string
		inheritedArgs    = map[string]string{}
		inheritedTriggers []Instruction
		rawOnBuild       []string
		llbConfigApplied bool
	)
	switch value := base.(type) {
	case ScratchBase:
		llbState = llb.Scratch()
		baseRef = "scratch"

	case ImageRef:
		image = unresolvedImage(platform)
		ref, err := runtime.Expand(value.Ref, &State{Platform: platform})
		if err != nil {
			return fmt.Errorf("expand FROM image reference: %w", err)
		}
		if strings.TrimSpace(ref) == "" {
			return errors.New("FROM image reference cannot be empty")
		}
		baseRef = ref
		imageOptions := append([]llb.ImageOption(nil), value.Options...)
		imageOptions = append(imageOptions, config.options...)
		customName := metadata.CustomName
		if customName == "" {
			customName = "FROM " + ref
		}
		constraints := RuntimeConstraints(runtime, metadata)
		constraints = append(constraints, llb.WithCustomName(BuildKitCustomName(runtime, &State{Platform: platform, StageName: state.StageName}, customName)))
		for _, constraint := range constraints {
			imageOptions = append(imageOptions, constraint)
		}
		llbState = llb.Image(ref, imageOptions...)
		if runtime.ImageResolver != nil {
			resolved, err := runtime.ImageResolver(ctx, ref, platform)
			if err != nil {
				return fmt.Errorf("resolve FROM image config: %w", err)
			}
			image = cloneImage(resolved)
			if image.Config.WorkingDir == "" {
				image.Config.WorkingDir = "/"
			}
			rawOnBuild = append([]string(nil), image.Config.OnBuild...)
			configBytes, err := json.Marshal(image)
			if err != nil {
				return fmt.Errorf("marshal FROM image config: %w", err)
			}
			llbState, err = llbState.WithImageConfig(configBytes)
			if err != nil {
				return fmt.Errorf("apply FROM image config to LLB state: %w", err)
			}
			llbConfigApplied = true
		} else {
			return errors.New("external FROM images require WithImageResolver so image configuration can be reflected in the stage state")
		}

	case StageRef:
		if runtime.ResolveStage == nil {
			return errors.New("FROM stage reference requires a stage resolver")
		}
		parent, err := runtime.ResolveStage(value.Name)
		if err != nil {
			return err
		}
		llbState = parent.LLB
		image = cloneImage(parent.Image)
		llbConfigApplied = true
		if config.platform == nil {
			platform = parent.Platform
		}
		baseRef = string(value.Name)
		inheritedArgs = maps.Clone(parent.Args)
		inheritedTriggers = append([]Instruction(nil), parent.Triggers...)
		rawOnBuild = append([]string(nil), parent.RawOnBuild...)

	case ContextRef:
		named, ok := runtime.NamedContexts[value.Name]
		if !ok {
			return fmt.Errorf("named context %q not found", value.Name)
		}
		llbState = named.LLB
		image = cloneImage(named.Image)
		if named.Image.Config.WorkingDir != "" || len(named.Image.Config.Env) > 0 || named.Image.Config.User != "" {
			configBytes, err := json.Marshal(named.Image)
			if err != nil {
				return fmt.Errorf("marshal named-context image config: %w", err)
			}
			llbState, err = llbState.WithImageConfig(configBytes)
			if err != nil {
				return fmt.Errorf("apply named-context image config: %w", err)
			}
			llbConfigApplied = true
		}
		if config.platform == nil && named.Platform.OS != "" {
			platform = named.Platform
		}
		baseRef = "context:" + value.Name
		rawOnBuild = append([]string(nil), image.Config.OnBuild...)

	default:
		return fmt.Errorf("unsupported FROM base %T", base)
	}

	image.OS = platform.OS
	image.Architecture = platform.Architecture
	image.Variant = platform.Variant
	image.OSVersion = platform.OSVersion
	image.OSFeatures = append([]string(nil), platform.OSFeatures...)
	if image.Config.WorkingDir == "" {
		image.Config.WorkingDir = "/"
	}

	llbState = llbState.Platform(platform)
	if !llbConfigApplied {
		for _, entry := range image.Config.Env {
			key, value, ok := strings.Cut(entry, "=")
			if ok {
				llbState = llbState.AddEnv(key, value)
			}
		}
		if image.Config.WorkingDir != "" {
			llbState = llbState.Dir(image.Config.WorkingDir)
		}
	}
	if image.Config.User != "" {
		llbState = llbState.User(image.Config.User)
	}

	state.LLB = llbState
	state.Image = image
	state.Platform = platform
	state.BaseRef = baseRef
	state.Args = inheritedArgs
	state.Env = environmentFromConfig(image.Config.Env)
	state.Triggers = inheritedTriggers
	state.RawOnBuild = rawOnBuild
	state.CmdSet = false

	if config.reset {
		state.Args = make(map[string]string)
		state.Triggers = nil
		state.RawOnBuild = nil
		state.Image.Config.OnBuild = nil
	}
	return nil
}

func unresolvedImage(platform ocispecs.Platform) dockerspec.DockerOCIImage {
	image := dockerspec.DockerOCIImage{
		OS: platform.OS, Architecture: platform.Architecture,
		Variant: platform.Variant, OSVersion: platform.OSVersion,
		OSFeatures: append([]string(nil), platform.OSFeatures...),
	}
	image.RootFS.Type = "layers"
	image.Config.WorkingDir = "/"
	return image
}
