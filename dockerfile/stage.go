package dockerfile

import (
	"github.com/moby/buildkit/client/llb"
	ocispecs "github.com/opencontainers/image-spec/specs-go/v1"
)

// StageName identifies a named build stage.
type StageName string

// BaseImage records the base reference and resolved BuildKit/image metadata.
type BaseImage struct {
	Ref   string
	State llb.State
	Image *ocispecs.Image
	_     struct{}
}

// Stage is the value passed between instruction handlers.
type Stage struct {
	Name  StageName
	Base  BaseImage
	State llb.State
	Image ocispecs.Image
	_     struct{}
}

// SourceKind identifies the source represented by Source.
type SourceKind uint8

const (
	SourceImage SourceKind = iota
	SourceStage
	SourceContext
	SourceScratch
)

// Source is an open, data-oriented reference used by FROM, COPY --from, and
// RUN mounts. It replaces sealed source interfaces with one extensible value.
type Source struct {
	Kind         SourceKind
	Reference    string
	Stage        *Stage
	ImageOptions []llb.ImageOption
	_             struct{}
}

// Clone returns an independent Source value and option slice.
func (s Source) Clone() Source {
	copy := s
	copy.ImageOptions = append([]llb.ImageOption(nil), s.ImageOptions...)
	if s.Stage != nil {
		stage := *s.Stage
		copy.Stage = &stage
	}
	return copy
}

// Image creates an image source.
func Image(ref string, options ...llb.ImageOption) Source {
	return Source{
		Kind:         SourceImage,
		Reference:    ref,
		ImageOptions: append([]llb.ImageOption(nil), options...),
	}
}

// StageOf creates a reference to an existing stage.
func StageOf(stage Stage) Source {
	copy := stage
	return Source{Kind: SourceStage, Stage: &copy, Reference: string(stage.Name)}
}

// Context creates a named build-context source.
func Context(name string) Source {
	return Source{Kind: SourceContext, Reference: name}
}

// Scratch creates an empty base source.
func Scratch() Source { return Source{Kind: SourceScratch} }

// CommandForm distinguishes shell and exec command representations.
type CommandForm uint8

const (
	CommandShell CommandForm = iota
	CommandExec
)

// Command stores a command independently from a specific Dockerfile
// instruction. New command forms can be represented through Payload or
// instruction extensions without changing instruction constructors.
type Command struct {
	Form CommandForm
	Text string
	Args []string
	_    struct{}
}

// Sh creates a shell-form command.
func Sh(command string) Command {
	return Command{Form: CommandShell, Text: command}
}

// Exec creates an exec-form command.
func Exec(args ...string) Command {
	return Command{Form: CommandExec, Args: append([]string(nil), args...)}
}

// Clone returns an independent command value.
func (c Command) Clone() Command {
	c.Args = append([]string(nil), c.Args...)
	return c
}
