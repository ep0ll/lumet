package dockerfile

import "github.com/moby/buildkit/client/llb"

// StageName identifies a stage. Names are resolved case-insensitively.
type StageName string

// Base is a supported source for a stage initializer.
type Base interface{ isBase() }

// CopySource is a supported source for COPY and RUN filesystem mounts.
type CopySource interface{ isCopySource() }

type ImageRef struct {
	Ref     string
	Options []llb.ImageOption
}
type StageRef struct{ Name StageName }
type ContextRef struct{ Name string }
type ScratchBase struct{}

func (ImageRef) isBase()       {}
func (StageRef) isBase()       {}
func (ContextRef) isBase()     {}
func (ScratchBase) isBase()    {}
func (ImageRef) isCopySource() {}
func (StageRef) isCopySource() {}
func (ContextRef) isCopySource() {}

// Image constructs an external image reference.
func Image(ref string, options ...llb.ImageOption) ImageRef {
	return ImageRef{Ref: ref, Options: append([]llb.ImageOption(nil), options...)}
}

// StageOf references a stage in the current Dockerfile DAG.
func StageOf(name StageName) StageRef { return StageRef{Name: name} }

// Context references a named build context.
func Context(name string) ContextRef { return ContextRef{Name: name} }

// Scratch returns an empty filesystem base.
func Scratch() ScratchBase { return ScratchBase{} }

func (r StageRef) String() string   { return string(r.Name) }
func (r ImageRef) String() string   { return r.Ref }
func (r ContextRef) String() string { return r.Name }
func (ScratchBase) String() string { return "scratch" }
