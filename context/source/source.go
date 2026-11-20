package source

import "github.com/moby/buildkit/client/llb"

// SourceKind classifies how a source was obtained.
type SourceKind int

const (
	SourceUnknown SourceKind = iota
	SourceLocal
	SourceGit
	SourceHTTP
	SourceInputs
	SourceImage
	SourceOCI
)

// ResolvedSource is the result of resolving one input (context or dexfile).
type ResolvedSource struct {
	State     *llb.State // nil ⇒ caller should use Local
	LocalName string
	Filename  string // only meaningful for the dexfile side
	Kind      SourceKind
}
