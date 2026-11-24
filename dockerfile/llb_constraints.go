package dockerfile

import (
	"fmt"
	"strings"

	"github.com/moby/buildkit/client/llb"
)

func RuntimeConstraints(runtime *Runtime, metadata InstructionMetadata) []llb.ConstraintsOpt {
	var options []llb.ConstraintsOpt
	if runtime != nil {
		options = append(options, runtime.Constraints...)
		options = append(options, runtime.StageConstraints...)
	}
	return append(options, metadata.Constraints...)
}

// operationConstraints adds scoped BuildKit constraints and a stable progress
// description to filesystem operations.
func OperationConstraints(runtime *Runtime, state *State, metadata InstructionMetadata, fallback string) []llb.ConstraintsOpt {
	options := RuntimeConstraints(runtime, metadata)
	title := metadata.CustomName
	if title == "" {
		title = metadata.ProgressTitle
	}
	if title == "" {
		title = fallback
	}
	return append(options, llb.WithCustomName(BuildKitCustomName(runtime, state, title)))
}

func BuildKitCustomName(runtime *Runtime, state *State, title string) string {
	parts := make([]string, 0, 3)
	if state != nil && state.Platform.OS != "" && state.Platform.Architecture != "" {
		platform := state.Platform.OS + "/" + state.Platform.Architecture
		if state.Platform.Variant != "" {
			platform += "/" + state.Platform.Variant
		}
		parts = append(parts, platform)
	}
	stageName := StageName("")
	if state != nil {
		stageName = state.StageName
	}
	if stageName == "" && runtime != nil {
		stageName = runtime.StageName
	}
	if stageName != "" {
		parts = append(parts, string(stageName))
	}
	prefix := "[" + strings.Join(parts, " ")
	if runtime != nil && runtime.InstructionIndex > 0 {
		total := runtime.InstructionTotal
		if total <= 0 {
			total = runtime.InstructionIndex
		}
		if len(parts) > 0 {
			prefix += " "
		}
		prefix += fmt.Sprintf("%*d/%d", len(fmt.Sprint(total)), runtime.InstructionIndex, total)
	}
	prefix += "] "
	if runtime != nil && runtime.OnBuild {
		prefix += "ONBUILD "
	}
	return prefix + title
}
