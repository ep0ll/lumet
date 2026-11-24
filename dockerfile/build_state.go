package dockerfile

import (
	ocispecs "github.com/opencontainers/image-spec/specs-go/v1"
)

func cloneStates(source map[StageName]State) map[StageName]State {
	out := make(map[StageName]State, len(source))
	for name, state := range source {
		out[name] = state.Clone()
	}
	return out
}

func cloneNamedContexts(source map[string]NamedContext, defaultPlatform ocispecs.Platform) map[string]NamedContext {
	out := make(map[string]NamedContext, len(source))
	for name, value := range source {
		value.Image = cloneImage(value.Image)
		value.Platform.OSFeatures = append([]string(nil), value.Platform.OSFeatures...)
		if value.Platform.OS == "" {
			value.Platform = defaultPlatform
			value.Platform.OSFeatures = append([]string(nil), defaultPlatform.OSFeatures...)
		}
		out[name] = value
	}
	return out
}
