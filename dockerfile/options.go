package dockerfile

import (
	"os"
	"time"

	"github.com/moby/buildkit/client/llb"
	digest "github.com/opencontainers/go-digest"
	ocispecs "github.com/opencontainers/image-spec/specs-go/v1"
)

// Option configures one instruction. Its public configuration value also makes
// it possible for callers to define reusable options without coupling their
// code to individual Dockerfile constructors.
type Option func(*InstructionOptions)

// InstructionOptions holds the common and instruction-specific settings
// consumed by instruction functions. Additional flags can be introduced as
// fields without changing existing instruction signatures.
type InstructionOptions struct {
	StageName   string
	Platform    *ocispecs.Platform
	BaseState   *llb.State
	ImageOptions []llb.ImageOption
	RunOptions   []llb.RunOption
	CopyOptions  []llb.CopyOption
	Constraints  []llb.ConstraintsOpt
	CustomName   string

	From      string
	FromState *llb.State
	Parents   bool
	Link      bool
	Chown     string
	Chmod     *llb.ChmodOpt
	Excludes  []string

	AddChecksum digest.Digest
	KeepGitDir  bool
	Unpack      *bool

	ArgDefault *string

	HealthInterval      time.Duration
	HealthTimeout       time.Duration
	HealthStartPeriod   time.Duration
	HealthStartInterval time.Duration
	HealthRetries       int
}

// As names the stage created by FROM.
func As(name string) Option {
	return func(o *InstructionOptions) { o.StageName = name }
}

// OnPlatform selects the platform for one FROM stage.
func OnPlatform(platform ocispecs.Platform) Option {
	return func(o *InstructionOptions) {
		p := platform
		o.Platform = &p
	}
}

// FromState uses an explicit LLB state as the base for FROM.
func FromState(state llb.State) Option {
	return func(o *InstructionOptions) {
		s := state
		o.BaseState = &s
	}
}

// ImageOptions forwards options to llb.Image.
func ImageOptions(opts ...llb.ImageOption) Option {
	return func(o *InstructionOptions) { o.ImageOptions = append(o.ImageOptions, opts...) }
}

// RunOptions forwards BuildKit RUN options, including mounts, network,
// security, secrets, SSH agents, ulimits and device options.
func RunOptions(opts ...llb.RunOption) Option {
	return func(o *InstructionOptions) { o.RunOptions = append(o.RunOptions, opts...) }
}

// CopyOptions forwards BuildKit file-copy options.
func CopyOptions(opts ...llb.CopyOption) Option {
	return func(o *InstructionOptions) { o.CopyOptions = append(o.CopyOptions, opts...) }
}

// LLBConstraints adds constraints to the LLB operation created by an
// instruction. It is useful for source maps, progress groups and operation
// metadata.
func LLBConstraints(opts ...llb.ConstraintsOpt) Option {
	return func(o *InstructionOptions) { o.Constraints = append(o.Constraints, opts...) }
}

// CustomName sets the BuildKit progress name for an LLB operation.
func CustomName(name string) Option {
	return func(o *InstructionOptions) { o.CustomName = name }
}

// CopyFrom selects a named stage or named context as the source for COPY.
func CopyFrom(name string) Option {
	return func(o *InstructionOptions) { o.From = name }
}

// CopyFromState selects an explicit LLB state as the source for COPY.
func CopyFromState(state llb.State) Option {
	return func(o *InstructionOptions) {
		s := state
		o.FromState = &s
	}
}

// Parents preserves source path components when copying into a directory.
// As in Dockerfile COPY --parents, a /./ marker in a source determines where
// the preserved portion of the source path begins.
func Parents() Option {
	return func(o *InstructionOptions) { o.Parents = true }
}

// Link creates COPY's new files in an independent scratch state and merges
// that state with the current root filesystem.
func Link() Option {
	return func(o *InstructionOptions) { o.Link = true }
}

// Chown configures file ownership using a user, uid, group or uid:gid value.
func Chown(owner string) Option {
	return func(o *InstructionOptions) { o.Chown = owner }
}

// Chmod configures copied file permissions.
func Chmod(mode os.FileMode) Option {
	return func(o *InstructionOptions) { o.Chmod = &llb.ChmodOpt{Mode: mode} }
}

// ChmodSymbolic configures symbolic file permissions, for example u+x,g=u.
func ChmodSymbolic(mode string) Option {
	return func(o *InstructionOptions) { o.Chmod = &llb.ChmodOpt{ModeStr: mode} }
}

// Exclude excludes source paths matching any of the supplied patterns.
func Exclude(patterns ...string) Option {
	return func(o *InstructionOptions) { o.Excludes = append(o.Excludes, patterns...) }
}

// AddChecksum pins a remote ADD source to the specified digest.
func AddChecksum(value digest.Digest) Option {
	return func(o *InstructionOptions) { o.AddChecksum = value }
}

// KeepGitDir keeps the .git directory when ADD downloads a Git repository.
func KeepGitDir() Option {
	return func(o *InstructionOptions) { o.KeepGitDir = true }
}

// Unpack controls archive extraction for ADD. The default is enabled.
func Unpack(enabled bool) Option {
	return func(o *InstructionOptions) {
		value := enabled
		o.Unpack = &value
	}
}

// ArgDefault defines an ARG default value. A value supplied through
// WithBuildArgs takes precedence.
func ArgDefault(value string) Option {
	return func(o *InstructionOptions) {
		value := value
		o.ArgDefault = &value
	}
}

// HealthInterval sets the HEALTHCHECK interval.
func HealthInterval(value time.Duration) Option {
	return func(o *InstructionOptions) { o.HealthInterval = value }
}

// HealthTimeout sets the HEALTHCHECK timeout.
func HealthTimeout(value time.Duration) Option {
	return func(o *InstructionOptions) { o.HealthTimeout = value }
}

// HealthStartPeriod sets the HEALTHCHECK start period.
func HealthStartPeriod(value time.Duration) Option {
	return func(o *InstructionOptions) { o.HealthStartPeriod = value }
}

// HealthStartInterval sets the HEALTHCHECK start interval.
func HealthStartInterval(value time.Duration) Option {
	return func(o *InstructionOptions) { o.HealthStartInterval = value }
}

// HealthRetries sets the number of consecutive failures before the container
// is considered unhealthy.
func HealthRetries(value int) Option {
	return func(o *InstructionOptions) { o.HealthRetries = value }
}

func collectOptions(opts []Option) InstructionOptions {
	var result InstructionOptions
	for _, opt := range opts {
		if opt != nil {
			opt(&result)
		}
	}
	return result
}

// AddSourceKind identifies the source type used by ADD.
type AddSourceKind uint8

const (
	AddLocal AddSourceKind = iota
	AddRemote
	AddGit
)

// AddSource describes a local file, HTTP(S) resource or Git repository.
// Use LocalSource, RemoteSource or GitSource to construct a source.
type AddSource struct {
	Kind       AddSourceKind
	Value      string
	Ref        string
	Subdir     string
	Checksum   digest.Digest
	GitOptions []llb.GitOption
}

// LocalSource adds a path from the build context.
func LocalSource(path string) AddSource {
	return AddSource{Kind: AddLocal, Value: path}
}

// RemoteSource adds a file from an HTTP or HTTPS URL.
func RemoteSource(address string) AddSource {
	return AddSource{Kind: AddRemote, Value: address}
}

// GitSource adds a Git repository. The ref may be a branch, tag or commit.
func GitSource(remote, ref string, opts ...llb.GitOption) AddSource {
	return AddSource{Kind: AddGit, Value: remote, Ref: ref, GitOptions: opts}
}
