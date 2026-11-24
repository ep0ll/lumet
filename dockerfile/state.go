package dockerfile

import (
	"maps"
	"slices"
	"strings"
	"time"

	"github.com/moby/buildkit/client/llb"
	dockerspec "github.com/moby/docker-image-spec/specs-go/v1"
	ocispecs "github.com/opencontainers/image-spec/specs-go/v1"
)

// State is the complete result of evaluating a stage up to a given instruction.
// LLB is the filesystem/execution graph; Image is the OCI/Docker image config.
// Instructions receive a candidate State and update both representations.
type State struct {
	LLB       llb.State
	Image     dockerspec.DockerOCIImage
	Platform  ocispecs.Platform
	StageName StageName
	BaseRef   string

	Args map[string]string
	Env  map[string]string

	Triggers   []Instruction
	RawOnBuild []string
	CmdSet     bool
}

// Clone returns a detached copy of mutable metadata. llb.State is persistent,
// so copying its value is sufficient.
func (s State) Clone() State {
	out := s
	out.Image = cloneImage(s.Image)
	out.Platform.OSFeatures = slices.Clone(s.Platform.OSFeatures)
	out.Args = maps.Clone(s.Args)
	out.Env = maps.Clone(s.Env)
	out.Triggers = slices.Clone(s.Triggers)
	out.RawOnBuild = slices.Clone(s.RawOnBuild)
	return out
}

func cloneImage(src dockerspec.DockerOCIImage) dockerspec.DockerOCIImage {
	dst := src
	dst.OSFeatures = slices.Clone(src.OSFeatures)
	dst.RootFS.DiffIDs = slices.Clone(src.RootFS.DiffIDs)
	dst.History = slices.Clone(src.History)
	for i := range dst.History {
		if src.History[i].Created != nil {
			created := *src.History[i].Created
			dst.History[i].Created = &created
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
		created := *src.Created
		dst.Created = &created
	}
	if src.Config.Healthcheck != nil {
		health := *src.Config.Healthcheck
		health.Test = slices.Clone(src.Config.Healthcheck.Test)
		dst.Config.Healthcheck = &health
	}
	return dst
}

func emptyImage(platform ocispecs.Platform) dockerspec.DockerOCIImage {
	image := dockerspec.DockerOCIImage{}
	image.OS = platform.OS
	image.Architecture = platform.Architecture
	image.Variant = platform.Variant
	image.OSVersion = platform.OSVersion
	image.OSFeatures = slices.Clone(platform.OSFeatures)
	image.RootFS.Type = "layers"
	image.Config.WorkingDir = "/"
	if platform.OS != "windows" {
		image.Config.Env = []string{
			"PATH=/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin",
		}
	}
	return image
}

func environmentFromConfig(values []string) map[string]string {
	env := make(map[string]string, len(values))
	for _, entry := range values {
		key, value, ok := stringsCut(entry, "=")
		if ok {
			env[key] = value
		}
	}
	return env
}

func setEnv(values []string, key, value string) []string {
	out := make([]string, 0, len(values)+1)
	replaced := false
	for _, entry := range values {
		current, _, _ := stringsCut(entry, "=")
		if current == key {
			if !replaced {
				out = append(out, key+"="+value)
				replaced = true
			}
			continue
		}
		out = append(out, entry)
	}
	if !replaced {
		out = append(out, key+"="+value)
	}
	return out
}

func stringsCut(value, sep string) (before, after string, found bool) {
	index := strings.Index(value, sep)
	if index < 0 {
		return value, "", false
	}
	return value[:index], value[index+len(sep):], true
}

func cloneCreatedAt(src *time.Time) *time.Time {
	if src == nil {
		return nil
	}
	value := *src
	return &value
}
