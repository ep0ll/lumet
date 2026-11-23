package dockerfile

import (
	"github.com/moby/buildkit/client/llb"
	ocispecs "github.com/opencontainers/image-spec/specs-go/v1"
)

type StageName string

// BaseImage is what a stage was started from (kept so handlers can diff against it).
type BaseImage struct {
	Ref   string
	State llb.State
	Image *ocispecs.Image
}

// Stage is the value that flows through every instruction: the LLB state
// plus the OCI image config being built (history lives in Image.History).
type Stage struct {
	Name  StageName
	Base  BaseImage
	State llb.State
	Image ocispecs.Image
}

// Base is anything valid after FROM. CopySource is anything valid for COPY --from / --mount from=.
// Both are sealed, so invalid combinations fail at compile time.
type Base interface{ base() }
type CopySource interface{ copySource() }

type ImageRef struct {
	Ref     string
	Options []llb.ImageOption // reuse llb: WithMetaResolver, ResolveMode, WithLayerLimit...
}
type StageRef struct{ Stage Stage }
type ContextRef struct{ Name string } // named build context
type ScratchBase struct{}

func (ImageRef) base()       {}
func (StageRef) base()       {}
func (ContextRef) base()     {}
func (ScratchBase) base()    {}
func (ImageRef) copySource() {}
func (StageRef) copySource() {}
func (ContextRef) copySource() {}

func Image(ref string, opts ...llb.ImageOption) ImageRef { return ImageRef{Ref: ref, Options: opts} }
func StageOf(s Stage) StageRef                           { return StageRef{Stage: s} }
func Context(name string) ContextRef                     { return ContextRef{Name: name} }
func Scratch() ScratchBase                               { return ScratchBase{} }
