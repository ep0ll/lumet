package dockerfile

import (
	"fmt"
	"os"
	"strings"

	"github.com/moby/buildkit/client/llb"
	"github.com/moby/buildkit/solver/pb"
)

// InstructionOptions is the construction-time configuration shared by every
// instruction. Its storage is private so the package can evolve internally.
// External packages extend options through the methods below.
type InstructionOptions struct {
	flags       map[string]any
	extensions  map[string]any
	payload     any
	buildKit    BuildKitOptions
	history     HistoryFunc
	historyMode *HistoryMode
}

// Option configures any current or future Dockerfile instruction.
type Option func(*InstructionOptions) error

// SetFlag adds or replaces a flag value in construction-time options.
func (o *InstructionOptions) SetFlag(name string, value any) {
	if o.flags == nil {
		o.flags = make(map[string]any)
	}
	o.flags[NormalizeFlagName(name)] = cloneFlagValue(value)
}

// SetExtension adds or replaces an application-specific extension value.
func (o *InstructionOptions) SetExtension(name string, value any) {
	if o.extensions == nil {
		o.extensions = make(map[string]any)
	}
	o.extensions[name] = cloneFlagValue(value)
}

// SetPayload sets structured, instruction-specific data.
func (o *InstructionOptions) SetPayload(value any) {
	o.payload = clonePayload(value)
}

// AddBuildKitOption appends a native BuildKit option.
func (o *InstructionOptions) AddBuildKitOption(option BuildKitOption) {
	if option != nil {
		option(&o.buildKit)
	}
}

// SetHistory overrides the definition's history callback for this instruction.
func (o *InstructionOptions) SetHistory(history HistoryFunc) {
	o.history = history
}

// SetHistoryMode overrides the definition's history mode for this instruction.
func (o *InstructionOptions) SetHistoryMode(mode HistoryMode) {
	o.historyMode = &mode
}

// WithFlag sets one instruction flag. Its value is validated against the
// instruction's registered flag definition during construction.
func WithFlag(name string, value any) Option {
	name = NormalizeFlagName(name)
	value = cloneFlagValue(value)
	if name == "" {
		return func(*InstructionOptions) error { return ErrInvalidDefinition }
	}
	return func(options *InstructionOptions) error {
		options.SetFlag(name, value)
		return nil
	}
}

// WithFlags sets several instruction flags.
func WithFlags(values map[string]any) Option {
	values = cloneValues(values)
	return func(options *InstructionOptions) error {
		for name, value := range values {
			options.SetFlag(name, value)
		}
		return nil
	}
}

// WithPayload attaches structured, instruction-specific data.
func WithPayload(value any) Option {
	value = clonePayload(value)
	return func(options *InstructionOptions) error {
		options.SetPayload(value)
		return nil
	}
}

// WithExtension attaches an application-specific value without changing the
// instruction schema or the package's public types.
func WithExtension(name string, value any) Option {
	if strings.TrimSpace(name) == "" {
		return func(*InstructionOptions) error { return ErrInvalidDefinition }
	}
	value = cloneFlagValue(value)
	return func(options *InstructionOptions) error {
		options.SetExtension(name, value)
		return nil
	}
}

// WithOSSignal stores an os.Signal as structured payload for STOPSIGNAL
// handlers. This package deliberately does not maintain its own signal enum.
func WithOSSignal(signal os.Signal) Option {
	return WithPayload(signal)
}

// WithHistory overrides the definition's history callback for one instance.
func WithHistory(history HistoryFunc) Option {
	return func(options *InstructionOptions) error {
		options.SetHistory(history)
		return nil
	}
}

// WithHistoryMode overrides the definition's history mode for one instance.
func WithHistoryMode(mode HistoryMode) Option {
	return func(options *InstructionOptions) error {
		options.SetHistoryMode(mode)
		return nil
	}
}

// BuildKitOptions is an immutable snapshot of native BuildKit options. Internal
// fields are private so the option bundle can be extended without changing
// downstream struct literals.
type BuildKitOptions struct {
	constraints []llb.ConstraintsOpt
	image       []llb.ImageOption
	run         []llb.RunOption
	copy        []llb.CopyOption
	network     *pb.NetMode
	security    *pb.SecurityMode
}

// BuildKitOption appends native BuildKit configuration to an option bundle.
type BuildKitOption func(*BuildKitOptions)

// LLBConstraints returns a BuildKit-only constraint option.
func LLBConstraints(values ...llb.ConstraintsOpt) BuildKitOption {
	options := append([]llb.ConstraintsOpt(nil), values...)
	return func(target *BuildKitOptions) {
		target.constraints = append(target.constraints, options...)
	}
}

// LLBImageOptions returns BuildKit image source options.
func LLBImageOptions(values ...llb.ImageOption) BuildKitOption {
	options := append([]llb.ImageOption(nil), values...)
	return func(target *BuildKitOptions) {
		target.image = append(target.image, options...)
	}
}

// LLBRunOptions returns BuildKit RUN execution options.
func LLBRunOptions(values ...llb.RunOption) BuildKitOption {
	options := append([]llb.RunOption(nil), values...)
	return func(target *BuildKitOptions) {
		target.run = append(target.run, options...)
	}
}

// LLBCopyOptions returns BuildKit file copy options.
func LLBCopyOptions(values ...llb.CopyOption) BuildKitOption {
	options := append([]llb.CopyOption(nil), values...)
	return func(target *BuildKitOptions) {
		target.copy = append(target.copy, options...)
	}
}

// LLBNetworkMode returns a BuildKit state network option.
func LLBNetworkMode(mode pb.NetMode) BuildKitOption {
	return func(target *BuildKitOptions) {
		value := mode
		target.network = &value
	}
}

// LLBSecurityMode returns a BuildKit state security option.
func LLBSecurityMode(mode pb.SecurityMode) BuildKitOption {
	return func(target *BuildKitOptions) {
		value := mode
		target.security = &value
	}
}

// WithLLBConstraints adds native BuildKit constraints to an instruction.
func WithLLBConstraints(values ...llb.ConstraintsOpt) Option {
	return func(options *InstructionOptions) error {
		options.AddBuildKitOption(LLBConstraints(values...))
		return nil
	}
}

// WithLLBImageOptions adds native BuildKit image source options.
func WithLLBImageOptions(values ...llb.ImageOption) Option {
	return func(options *InstructionOptions) error {
		options.AddBuildKitOption(LLBImageOptions(values...))
		return nil
	}
}

// WithLLBRunOptions adds native BuildKit RUN options, including mounts,
// secrets, SSH, CDI devices, and future BuildKit exec options.
func WithLLBRunOptions(values ...llb.RunOption) Option {
	return func(options *InstructionOptions) error {
		options.AddBuildKitOption(LLBRunOptions(values...))
		return nil
	}
}

// WithLLBCopyOptions adds native BuildKit file-copy options.
func WithLLBCopyOptions(values ...llb.CopyOption) Option {
	return func(options *InstructionOptions) error {
		options.AddBuildKitOption(LLBCopyOptions(values...))
		return nil
	}
}

// WithLLBNetworkMode configures the BuildKit state network mode.
func WithLLBNetworkMode(mode pb.NetMode) Option {
	return func(options *InstructionOptions) error {
		options.AddBuildKitOption(LLBNetworkMode(mode))
		return nil
	}
}

// WithLLBSecurityMode configures the BuildKit state security mode.
func WithLLBSecurityMode(mode pb.SecurityMode) Option {
	return func(options *InstructionOptions) error {
		options.AddBuildKitOption(LLBSecurityMode(mode))
		return nil
	}
}

// WithConstraints is a concise alias for WithLLBConstraints.
func WithConstraints(values ...llb.ConstraintsOpt) Option {
	return WithLLBConstraints(values...)
}

// WithRunOption aliases WithLLBRunOptions.
func WithRunOption(values ...llb.RunOption) Option {
	return WithLLBRunOptions(values...)
}

// WithCopyOption aliases WithLLBCopyOptions.
func WithCopyOption(values ...llb.CopyOption) Option {
	return WithLLBCopyOptions(values...)
}

// ParseLLBNetworkMode parses Dockerfile RUN --network values.
func ParseLLBNetworkMode(value string) (pb.NetMode, error) {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "", "default":
		return pb.NetMode_UNSET, nil
	case "none":
		return pb.NetMode_NONE, nil
	case "host":
		return pb.NetMode_HOST, nil
	default:
		return pb.NetMode_UNSET, fmt.Errorf("unsupported RUN network mode %q", value)
	}
}

// ParseLLBSecurityMode parses Dockerfile RUN --security values.
func ParseLLBSecurityMode(value string) (pb.SecurityMode, error) {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "", "sandbox":
		return pb.SecurityMode_SANDBOX, nil
	case "insecure":
		return pb.SecurityMode_INSECURE, nil
	default:
		return pb.SecurityMode_SANDBOX, fmt.Errorf("unsupported RUN security mode %q", value)
	}
}

// AddConstraints appends BuildKit constraints to this option snapshot.
func (o *BuildKitOptions) AddConstraints(values ...llb.ConstraintsOpt) {
	o.constraints = append(o.constraints, values...)
}

// AddImageOptions appends BuildKit image source options.
func (o *BuildKitOptions) AddImageOptions(values ...llb.ImageOption) {
	o.image = append(o.image, values...)
}

// AddRunOptions appends BuildKit RUN options.
func (o *BuildKitOptions) AddRunOptions(values ...llb.RunOption) {
	o.run = append(o.run, values...)
}

// AddCopyOptions appends BuildKit file copy options.
func (o *BuildKitOptions) AddCopyOptions(values ...llb.CopyOption) {
	o.copy = append(o.copy, values...)
}

// SetNetworkMode sets the BuildKit state network mode.
func (o *BuildKitOptions) SetNetworkMode(mode pb.NetMode) {
	value := mode
	o.network = &value
}

// SetSecurityMode sets the BuildKit state security mode.
func (o *BuildKitOptions) SetSecurityMode(mode pb.SecurityMode) {
	value := mode
	o.security = &value
}

// Constraints returns a copy of the native BuildKit constraints.
func (o BuildKitOptions) Constraints() []llb.ConstraintsOpt {
	return append([]llb.ConstraintsOpt(nil), o.constraints...)
}

// ImageOptions returns a copy of the native BuildKit image options.
func (o BuildKitOptions) ImageOptions() []llb.ImageOption {
	return append([]llb.ImageOption(nil), o.image...)
}

// RunOptions returns a copy of the native BuildKit RUN options.
func (o BuildKitOptions) RunOptions() []llb.RunOption {
	return append([]llb.RunOption(nil), o.run...)
}

// CopyOptions returns a copy of the native BuildKit copy options.
func (o BuildKitOptions) CopyOptions() []llb.CopyOption {
	return append([]llb.CopyOption(nil), o.copy...)
}

// NetworkMode returns the configured state network mode, if specified.
func (o BuildKitOptions) NetworkMode() (pb.NetMode, bool) {
	if o.network == nil {
		return pb.NetMode_UNSET, false
	}
	return *o.network, true
}

// SecurityMode returns the configured state security mode, if specified.
func (o BuildKitOptions) SecurityMode() (pb.SecurityMode, bool) {
	if o.security == nil {
		return pb.SecurityMode_SANDBOX, false
	}
	return *o.security, true
}

// ApplyToState applies BuildKit state-level network and security configuration.
func (o BuildKitOptions) ApplyToState(state llb.State) llb.State {
	if o.network != nil {
		state = state.Network(*o.network)
	}
	if o.security != nil {
		state = state.Security(*o.security)
	}
	return state
}

func cloneBuildKitOptions(in BuildKitOptions) BuildKitOptions {
	out := BuildKitOptions{
		constraints: append([]llb.ConstraintsOpt(nil), in.constraints...),
		image:       append([]llb.ImageOption(nil), in.image...),
		run:         append([]llb.RunOption(nil), in.run...),
		copy:        append([]llb.CopyOption(nil), in.copy...),
	}
	if in.network != nil {
		value := *in.network
		out.network = &value
	}
	if in.security != nil {
		value := *in.security
		out.security = &value
	}
	return out
}
