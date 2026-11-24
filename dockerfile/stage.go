package dockerfile

import (
	"fmt"

	"github.com/moby/buildkit/client/llb"
	dockerspec "github.com/moby/docker-image-spec/specs-go/v1"
	ocispecs "github.com/opencontainers/image-spec/specs-go/v1"
)

type StageName string

type BaseImage struct {
	Ref string
	State llb.State
	Image dockerspec.DockerOCIImage
	Platform ocispecs.Platform
}

type Stage struct {
	Name StageName
	Base BaseImage
	State llb.State
	Image dockerspec.DockerOCIImage
	Platform ocispecs.Platform
	Args map[string]string
	Env map[string]string
	Triggers []Trigger
	RawOnBuild []string
	CmdSet bool
}

type Base interface{ base() }
type CopySource interface{ copySource() }

type ImageRef struct {
	Ref string
	Options []llb.ImageOption
}
type StageRef struct{ Name StageName }
type ContextRef struct{ Name string }
type ScratchBase struct{}

func (ImageRef) base() {}
func (StageRef) base() {}
func (ContextRef) base() {}
func (ScratchBase) base() {}
func (ImageRef) copySource() {}
func (StageRef) copySource() {}
func (ContextRef) copySource() {}

func Image(ref string, options ...llb.ImageOption) ImageRef {
	return ImageRef{Ref: ref, Options: append([]llb.ImageOption(nil), options...)}
}
func StageOf(name StageName) StageRef { return StageRef{Name: name} }
func Context(name string) ContextRef { return ContextRef{Name: name} }
func Scratch() ScratchBase { return ScratchBase{} }

func (r StageRef) String() string { return string(r.Name) }
func (r ImageRef) String() string { return r.Ref }
func (r ContextRef) String() string { return r.Name }
func (ScratchBase) String() string { return "scratch" }
func (s Stage) String() string {
	if s.Name != "" { return string(s.Name) }
	return fmt.Sprintf("stage(%s)", s.Base.Ref)
}
