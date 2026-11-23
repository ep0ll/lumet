package llb

import (
	"context"
	"fmt"
	"regexp"
	"sort"
	"strings"

	buildkitllb "github.com/moby/buildkit/client/llb"
	ocispecs "github.com/opencontainers/image-spec/specs-go/v1"
	"github.com/moby/buildkit/util/apicaps"
	"go.starlark.net/starlark"
	"go.starlark.net/starlarkstruct"
)

// StateValue is the immutable Starlark representation of BuildKit's llb.State.
type StateValue struct {
	state buildkitllb.State
}

// WrapState adapts a native BuildKit State for Starlark globals and callbacks.
func WrapState(state buildkitllb.State) *StateValue {
	return &StateValue{state: state}
}

// Unwrap returns the native immutable BuildKit state.
func (v *StateValue) Unwrap() buildkitllb.State {
	if v == nil {
		return buildkitllb.State{}
	}
	return v.state
}

func (v *StateValue) String() string        { return "<llb.state>" }
func (v *StateValue) Type() string          { return "llb.state" }
func (v *StateValue) Freeze()               {}
func (v *StateValue) Truth() starlark.Bool  { return starlark.Bool(v != nil) }
func (v *StateValue) Hash() (uint32, error) { return 0, fmt.Errorf("unhashable: llb.state") }

// OutputValue wraps the BuildKit Output interface. The interface is kept
// private because only BuildKit implementations can safely satisfy it.
type OutputValue struct {
	output buildkitllb.Output
}

// WrapOutput adapts a native BuildKit Output.
func WrapOutput(output buildkitllb.Output) *OutputValue {
	return &OutputValue{output: output}
}

// Unwrap returns the native BuildKit Output.
func (v *OutputValue) Unwrap() buildkitllb.Output {
	if v == nil {
		return nil
	}
	return v.output
}

func (v *OutputValue) String() string        { return "<llb.output>" }
func (v *OutputValue) Type() string          { return "llb.output" }
func (v *OutputValue) Freeze()               {}
func (v *OutputValue) Truth() starlark.Bool  { return starlark.Bool(v != nil && v.output != nil) }
func (v *OutputValue) Hash() (uint32, error) { return 0, fmt.Errorf("unhashable: llb.output") }

// VertexValue wraps BuildKit's Vertex interface for Go-defined operations.
type VertexValue struct {
	vertex buildkitllb.Vertex
}

// WrapVertex adapts a native BuildKit Vertex.
func WrapVertex(vertex buildkitllb.Vertex) *VertexValue {
	return &VertexValue{vertex: vertex}
}

// Unwrap returns the native BuildKit Vertex.
func (v *VertexValue) Unwrap() buildkitllb.Vertex {
	if v == nil {
		return nil
	}
	return v.vertex
}

func (v *VertexValue) String() string        { return "<llb.vertex>" }
func (v *VertexValue) Type() string          { return "llb.vertex" }
func (v *VertexValue) Freeze()               {}
func (v *VertexValue) Truth() starlark.Bool  { return starlark.Bool(v != nil && v.vertex != nil) }
func (v *VertexValue) Hash() (uint32, error) { return 0, fmt.Errorf("unhashable: llb.vertex") }

// ConstraintsValue wraps a BuildKit Constraints value. Its native fields
// remain private so that option application continues to be owned by BuildKit.
type ConstraintsValue struct {
	constraints buildkitllb.Constraints
}

// WrapConstraints adapts a native BuildKit Constraints value.
func WrapConstraints(constraints buildkitllb.Constraints) *ConstraintsValue {
	return &ConstraintsValue{constraints: constraints}
}

// Unwrap returns a copy of the native constraints value.
func (v *ConstraintsValue) Unwrap() buildkitllb.Constraints {
	if v == nil {
		return buildkitllb.Constraints{}
	}
	return v.constraints
}

func (v *ConstraintsValue) String() string        { return "<llb.constraints>" }
func (v *ConstraintsValue) Type() string          { return "llb.constraints" }
func (v *ConstraintsValue) Freeze()               {}
func (v *ConstraintsValue) Truth() starlark.Bool  { return starlark.Bool(v != nil) }
func (v *ConstraintsValue) Hash() (uint32, error) { return 0, fmt.Errorf("unhashable: llb.constraints") }

func (v *ConstraintsValue) Attr(name string) (starlark.Value, error) {
		if v == nil {
		return nil, fmt.Errorf("llb.constraints: nil receiver")
	}
	switch name {
	case "platform":
		if v.constraints.Platform == nil {
			return starlark.None, nil
		}
		return WrapOCIPlatform(*v.constraints.Platform), nil
	case "worker_constraints":
		return stringSliceValue(v.constraints.WorkerConstraints), nil
	case "metadata":
		return WrapOpMetadata(v.constraints.Metadata), nil
	case "local_unique_id":
		return starlark.String(v.constraints.LocalUniqueID), nil
	case "source_locations":
		values := make([]starlark.Value, len(v.constraints.SourceLocations))
		for i, location := range v.constraints.SourceLocations {
			values[i] = WrapSourceLocation(location)
		}
		return starlark.NewList(values), nil
	case "has_capabilities":
		return starlark.Bool(v.constraints.Caps != nil), nil
	default:
		return nil, nil
	}
}

func (v *ConstraintsValue) AttrNames() []string {
	return SnakeCaseNames([]string{"has_capabilities", "local_unique_id", "metadata", "platform", "source_locations", "worker_constraints",
		"has_capabilities", "local_unique_id", "metadata", "source_locations", "worker_constraints"})
}

// OpMetadataValue wraps BuildKit's user-facing LLB OpMetadata structure.
type OpMetadataValue struct {
	metadata buildkitllb.OpMetadata
}

// WrapOpMetadata adapts BuildKit operation metadata.
func WrapOpMetadata(metadata buildkitllb.OpMetadata) *OpMetadataValue {
	return &OpMetadataValue{metadata: metadata}
}

// Unwrap returns a copy of the native metadata value.
func (v *OpMetadataValue) Unwrap() buildkitllb.OpMetadata {
	if v == nil {
		return buildkitllb.OpMetadata{}
	}
	return v.metadata
}

func (v *OpMetadataValue) String() string        { return "<llb.op_metadata>" }
func (v *OpMetadataValue) Type() string          { return "llb.op_metadata" }
func (v *OpMetadataValue) Freeze()               {}
func (v *OpMetadataValue) Truth() starlark.Bool  { return starlark.Bool(v != nil) }
func (v *OpMetadataValue) Hash() (uint32, error) { return 0, fmt.Errorf("unhashable: llb.op_metadata") }

func (v *OpMetadataValue) Attr(name string) (starlark.Value, error) {
		if v == nil {
		return nil, fmt.Errorf("llb.op_metadata: nil receiver")
	}
	switch name {
	case "to_pb":
		return boundBuiltin("op_metadata.to_pb", func(_ *starlark.Thread, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
			if err := arity("op_metadata.to_pb", args, kwargs, 0, 0); err != nil { return nil, err }
			return WrapOpMetadataProto(v.metadata.ToPB()), nil
		}), nil
	case "from_pb":
		return boundBuiltin("op_metadata.from_pb", func(_ *starlark.Thread, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
			if err := arity("op_metadata.from_pb", args, kwargs, 1, 1); err != nil { return nil, err }
			protoValue, ok := args[0].(*OpMetadataProtoValue)
			if !ok || protoValue == nil || protoValue.metadata == nil {
				return nil, fmt.Errorf("op_metadata.from_pb: got %s, want llb.op_metadata_pb", valueType(args[0]))
			}
			metadata := buildkitllb.OpMetadata{}
			metadata.FromPB(protoValue.metadata)
			return WrapOpMetadata(metadata), nil
		}), nil
	case "caps":
		dict := starlark.NewDict(len(v.metadata.Caps))
		for key, enabled := range v.metadata.Caps {
			if err := dict.SetKey(starlark.String(string(key)), starlark.Bool(enabled)); err != nil { return nil, err }
		}
		return dict, nil
	case "ignore_cache":
		return starlark.Bool(v.metadata.IgnoreCache), nil
	case "description":
		return stringDictValue(v.metadata.Description), nil
	case "export_cache":
		if v.metadata.ExportCache == nil {
			return starlark.None, nil
		}
		return starlark.Bool(v.metadata.ExportCache.Value), nil
	case "progress_group":
		if v.metadata.ProgressGroup == nil {
			return starlark.None, nil
		}
		return starlarkstructValue("llb.progress_group", starlark.StringDict{
			"id":   starlark.String(v.metadata.ProgressGroup.Id),
			"name": starlark.String(v.metadata.ProgressGroup.Name),
			"weak": starlark.Bool(v.metadata.ProgressGroup.Weak),
		}), nil
	case "linux_resources":
		if v.metadata.LinuxResources == nil {
			return starlark.None, nil
		}
		return starlarkstructValue("llb.linux_resources", starlark.StringDict{
			"memory":      starlark.MakeInt64(v.metadata.LinuxResources.Memory),
			"memory_swap": starlark.MakeInt64(v.metadata.LinuxResources.MemorySwap),
			"cpu_shares":  starlark.MakeUint64(v.metadata.LinuxResources.CpuShares),
			"cpu_period":  starlark.MakeUint64(v.metadata.LinuxResources.CpuPeriod),
			"cpu_quota":   starlark.MakeInt64(v.metadata.LinuxResources.CpuQuota),
			"cpuset_cpus": starlark.String(v.metadata.LinuxResources.CpusetCpus),
			"cpuset_mems": starlark.String(v.metadata.LinuxResources.CpusetMems),
		}), nil
	default:
		return nil, nil
	}
}

func (v *OpMetadataValue) AttrNames() []string {
	return SnakeCaseNames([]string{
		"caps", "description", "export_cache", "ignore_cache", "linux_resources", "progress_group", "to_pb", "from_pb",
		"caps", "description", "export_cache", "ignore_cache", "linux_resources", "progress_group", "to_pb", "from_pb",
	})
}

// OCIPlatformValue carries a normalized OCI platform for State.Platform.
type OCIPlatformValue struct {
	platform ocispecs.Platform
}

// WrapOCIPlatform adapts an OCI platform value.
func WrapOCIPlatform(platform ocispecs.Platform) *OCIPlatformValue {
	return &OCIPlatformValue{platform: platform}
}

// Unwrap returns the OCI platform value.
func (v *OCIPlatformValue) Unwrap() ocispecs.Platform {
	if v == nil {
		return ocispecs.Platform{}
	}
	return v.platform
}

func (v *OCIPlatformValue) String() string {
	if v == nil {
		return "<llb.oci_platform nil>"
	}
	return fmt.Sprintf("<llb.oci_platform %s/%s>", v.platform.OS, v.platform.Architecture)
}
func (v *OCIPlatformValue) Type() string          { return "llb.oci_platform" }
func (v *OCIPlatformValue) Freeze()               {}
func (v *OCIPlatformValue) Truth() starlark.Bool  { return starlark.Bool(v != nil) }
func (v *OCIPlatformValue) Hash() (uint32, error) { return 0, fmt.Errorf("unhashable: llb.oci_platform") }

func (v *OCIPlatformValue) Attr(name string) (starlark.Value, error) {
		if v == nil {
		return nil, fmt.Errorf("llb.oci_platform: nil receiver")
	}
	switch name {
	case "os":
		return starlark.String(v.platform.OS), nil
	case "architecture":
		return starlark.String(v.platform.Architecture), nil
	case "variant":
		return starlark.String(v.platform.Variant), nil
	case "os_version":
		return starlark.String(v.platform.OSVersion), nil
	case "os_features":
		return stringSliceValue(v.platform.OSFeatures), nil
	default:
		return nil, nil
	}
}

func (v *OCIPlatformValue) AttrNames() []string {
	return SnakeCaseNames([]string{"architecture", "os", "os_features", "os_version", "variant",
		"architecture", "os", "os_features", "os_version", "variant"})
}

func stringDictValue(values map[string]string) starlark.Value {
	dict := starlark.NewDict(len(values))
	for key, value := range values {
		_ = dict.SetKey(starlark.String(key), starlark.String(value))
	}
	return dict
}

func starlarkstructValue(name string, values starlark.StringDict) starlark.Value {
	normalized, err := NormalizeMembers(values)
	if err != nil {
		panic(err)
	}
	return starlarkstruct.FromStringDict(starlark.String(name), normalized)
}

func (v *StateValue) Attr(name string) (starlark.Value, error) {
		if v == nil {
		return nil, fmt.Errorf("llb.state: nil receiver")
	}
	switch name {
	case "with_value":
		return boundBuiltin("state.with_value", func(_ *starlark.Thread, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
			if err := arity("state.with_value", args, kwargs, 2, 2); err != nil {
				return nil, err
			}
			key, ok := args[0].(starlark.String)
			if !ok {
				return nil, fmt.Errorf("state.with_value: key must be a string, got %s", valueType(args[0]))
			}
			return WrapState(v.state.WithValue(string(key), goScalar(args[1]))), nil
		}), nil
	case "value":
		return boundBuiltin("state.value", func(thread *starlark.Thread, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
			if err := arity("state.value", args, kwargs, 1, -1); err != nil {
				return nil, err
			}
			key, ok := args[0].(starlark.String)
			if !ok {
				return nil, fmt.Errorf("state.value: key must be a string, got %s", valueType(args[0]))
			}
			opts, err := optionsFromArgs[buildkitllb.ConstraintsOpt](args, 1, "constraints_opt")
			if err != nil {
				return nil, err
			}
			result, err := v.state.Value(callContext(thread), string(key), opts...)
			if err != nil {
				return nil, err
			}
			return fromGoValue(result), nil
		}), nil
	case "output":
		return boundBuiltin("state.output", func(_ *starlark.Thread, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
			if err := arity("state.output", args, kwargs, 0, 0); err != nil {
				return nil, err
			}
			output := v.state.Output()
			if output == nil {
				return starlark.None, nil
			}
			return WrapOutput(output), nil
		}), nil
	case "with_output":
		return boundBuiltin("state.with_output", func(_ *starlark.Thread, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
			if err := arity("state.with_output", args, kwargs, 1, 1); err != nil {
				return nil, err
			}
			output, ok := args[0].(*OutputValue)
			if !ok || output == nil {
				return nil, fmt.Errorf("state.with_output: got %s, want llb.output", valueType(args[0]))
			}
			return WrapState(v.state.WithOutput(output.output)), nil
		}), nil
	case "validate":
		return boundBuiltin("state.validate", func(thread *starlark.Thread, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
			if err := arity("state.validate", args, kwargs, 0, -1); err != nil {
				return nil, err
			}
			constraints, err := constraintsFromArgs(args, 0)
			if err != nil {
				return nil, err
			}
			if err := v.state.Validate(callContext(thread), constraints); err != nil {
				return nil, err
			}
			return starlark.None, nil
		}), nil
	default:
		for _, provider := range stateAttrProviders {
			value, found, err := provider(v, name)
			if err != nil || found {
				return value, err
			}
		}
		return nil, nil
	}
}

func (v *StateValue) AttrNames() []string {
	names := []string{
		"output", "value", "validate", "with_output", "with_value",
		"output", "value", "validate", "with_output", "with_value",
	}
	for _, provider := range stateAttrNames {
		names = append(names, provider...)
	}
	sort.Strings(names)
	return SnakeCaseNames(compactStrings(names))
}

var stateAttrProviders = []func(*StateValue, string) (starlark.Value, bool, error){
	stateAsyncAttr,
	stateMetaAttr,
	stateExecAttr,
	stateFileAttr,
	stateMarshalAttr,
}

var stateAttrNames = [][]string{
	stateAsyncAttrNames,
	stateMetaAttrNames,
	stateExecAttrNames,
	stateFileAttrNames,
	stateMarshalAttrNames,
}

func constraintsFromArgs(args starlark.Tuple, start int) (*buildkitllb.Constraints, error) {
	options, err := optionsFromArgs[buildkitllb.ConstraintsOpt](args, start, "constraints_opt")
	if err != nil {
		return nil, err
	}
	return buildkitllb.NewConstraints(options...), nil
}

func fromGoValue(value any) starlark.Value {
	switch v := value.(type) {
	case nil:
		return starlark.None
	case string:
		return starlark.String(v)
	case []byte:
		return starlark.Bytes(v)
	case bool:
		return starlark.Bool(v)
	case int:
		return starlark.MakeInt(v)
	case int8:
		return starlark.MakeInt(int(v))
	case int16:
		return starlark.MakeInt(int(v))
	case int32:
		return starlark.MakeInt(int(v))
	case int64:
		return starlark.MakeInt64(v)
	case uint:
		return starlark.MakeUint(uint(v))
	case uint8:
		return starlark.MakeInt(int(v))
	case uint16:
		return starlark.MakeInt(int(v))
	case uint32:
		return starlark.MakeUint(uint(v))
	case uint64:
		return starlark.MakeUint64(v)
	case []string:
		return stringSliceValue(v)
	case *OCIPlatformValue:
		if v == nil {
			return starlark.None
		}
		return v
	default:
		return starlark.String(fmt.Sprint(value))
	}
}

func compactStrings(values []string) []string {
	if len(values) < 2 {
		return values
	}
	sort.Strings(values)
	n := 1
	for i := 1; i < len(values); i++ {
		if values[i] != values[n-1] {
			values[n] = values[i]
			n++
		}
	}
	return values[:n]
}

func stateBuiltins() map[string]starlark.Value {
	members := map[string]starlark.Value{
		"new_state": builtin("new_state", newStateBuiltin),
		"new_constraints": builtin("new_constraints", newConstraintsBuiltin),
		"new_op_metadata": builtin("new_op_metadata", newOpMetadataBuiltin),
		"oci_platform": builtin("oci_platform", ociPlatformBuiltin),
	}
	for name, value := range stateConstraintBuiltins() {
		if _, exists := members[name]; exists { panic("starlark/llb: duplicate state builtin " + name) }
		members[name] = value
	}
	return members
}

func stateConstraintBuiltins() map[string]starlark.Value {
	return map[string]starlark.Value{
		"with_description": builtin("with_description", withDescriptionBuiltin),
		"with_custom_name": stringOptionBuiltin("with_custom_name", "constraints_opt", buildkitllb.WithCustomName),
		"with_custom_namef": builtin("with_custom_namef", withCustomNamefBuiltin),
		"with_export_cache": builtin("with_export_cache", withExportCacheBuiltin),
		"without_export_cache": builtin("without_export_cache", withoutExportCacheBuiltin),
		"without_default_export_cache": builtin("without_default_export_cache", withoutDefaultExportCacheBuiltin),
		"with_linux_resources": builtin("with_linux_resources", withLinuxResourcesBuiltin),
		"platform": builtin("platform", platformConstraintBuiltin),
		"local_unique_id": stringOptionBuiltin("local_unique_id", "constraints_opt", buildkitllb.LocalUniqueID),
		"progress_group": builtin("progress_group", progressGroupBuiltin),
		"memory_limit": int64OptionBuiltin("memory_limit", "constraints_opt", buildkitllb.MemoryLimit),
		"memory_swap_limit": int64OptionBuiltin("memory_swap_limit", "constraints_opt", buildkitllb.MemorySwapLimit),
		"cpu_shares": uint64OptionBuiltin("cpu_shares", "constraints_opt", buildkitllb.CPUShares),
		"cpu_period": uint64OptionBuiltin("cpu_period", "constraints_opt", buildkitllb.CPUPeriod),
		"cpu_quota": int64OptionBuiltin("cpu_quota", "constraints_opt", buildkitllb.CPUQuota),
		"cpuset_cpus": stringOptionBuiltin("cpuset_cpus", "constraints_opt", buildkitllb.CpusetCpus),
		"cpuset_mems": stringOptionBuiltin("cpuset_mems", "constraints_opt", buildkitllb.CpusetMems),
		"require": builtin("require", requireBuiltin),
		"with_caps": builtin("with_caps", withCapsBuiltin),
		"op_metadata": builtin("op_metadata", opMetadataBuiltin),
	}
}

func newStateBuiltin(_ *starlark.Thread, _ *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	if err := arity("new_state", args, kwargs, 1, 1); err != nil {
		return nil, err
	}
	output, ok := args[0].(*OutputValue)
	if !ok || output == nil {
		return nil, fmt.Errorf("new_state: got %s, want llb.output", valueType(args[0]))
	}
	return WrapState(buildkitllb.NewState(output.output)), nil
}

func newConstraintsBuiltin(_ *starlark.Thread, _ *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	if len(kwargs) != 0 {
		return nil, fmt.Errorf("new_constraints: options must be passed positionally")
	}
	options, err := optionsFromArgs[buildkitllb.ConstraintsOpt](args, 0, "constraints_opt")
	if err != nil {
		return nil, err
	}
	return WrapConstraints(*buildkitllb.NewConstraints(options...)), nil
}

func ociPlatformBuiltin(_ *starlark.Thread, _ *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	var platform ocispecs.Platform
	if err := UnpackArgs("oci_platform", args, kwargs,
		"os", &platform.OS,
		"architecture", &platform.Architecture,
		"variant?", &platform.Variant,
		"os_version?", &platform.OSVersion,
		"os_features?", &platform.OSFeatures,
	); err != nil {
		return nil, err
	}
	return WrapOCIPlatform(platform), nil
}

func platformConstraintValues() map[string]starlark.Value {
	return map[string]starlark.Value{
		"linux_amd64":   wrapOption("constraints_opt", buildkitllb.LinuxAmd64),
		"linux_armhf":   wrapOption("constraints_opt", buildkitllb.LinuxArmhf),
		"linux_arm":     wrapOption("constraints_opt", buildkitllb.LinuxArm),
		"linux_armel":   wrapOption("constraints_opt", buildkitllb.LinuxArmel),
		"linux_arm64":   wrapOption("constraints_opt", buildkitllb.LinuxArm64),
		"linux_s390x":   wrapOption("constraints_opt", buildkitllb.LinuxS390x),
		"linux_ppc64":   wrapOption("constraints_opt", buildkitllb.LinuxPpc64),
		"linux_ppc64le": wrapOption("constraints_opt", buildkitllb.LinuxPpc64le),
		"linux_riscv64": wrapOption("constraints_opt", buildkitllb.LinuxRiscv64),
		"darwin":       wrapOption("constraints_opt", buildkitllb.Darwin),
		"windows":      wrapOption("constraints_opt", buildkitllb.Windows),
	}
}

func stateConstraintValues() map[string]starlark.Value {
	return map[string]starlark.Value{
		"ignore_cache": wrapOption("constraints_opt", buildkitllb.IgnoreCache),
	}
}

func (v *OutputValue) Attr(name string) (starlark.Value, error) {
		if v == nil || v.output == nil {
		return nil, fmt.Errorf("llb.output: nil receiver")
	}
	switch name {
	case "vertex":
		return boundBuiltin("output.vertex", func(thread *starlark.Thread, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
			if err := arity("output.vertex", args, kwargs, 0, -1); err != nil { return nil, err }
			constraints, err := constraintsFromArgs(args, 0)
			if err != nil { return nil, err }
			return WrapVertex(v.output.Vertex(callContext(thread), constraints)), nil
		}), nil
	case "to_input":
		return boundBuiltin("output.to_input", func(thread *starlark.Thread, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
			if err := arity("output.to_input", args, kwargs, 0, -1); err != nil { return nil, err }
			constraints, err := constraintsFromArgs(args, 0)
			if err != nil { return nil, err }
			input, err := v.output.ToInput(callContext(thread), constraints)
			if err != nil { return nil, err }
			if input == nil { return starlark.None, nil }
			return starlarkstructValue("llb.input", starlark.StringDict{
				"digest": starlark.String(input.Digest),
				"index": starlark.MakeInt64(input.Index),
			}), nil
		}), nil
	default:
		return nil, nil
	}
}

func (v *OutputValue) AttrNames() []string {
	return SnakeCaseNames([]string{"vertex", "to_input", "vertex", "to_input"})
}

func (v *VertexValue) Attr(name string) (starlark.Value, error) {
		if v == nil || v.vertex == nil {
		return nil, fmt.Errorf("llb.vertex: nil receiver")
	}
	switch name {
	case "output":
		return boundBuiltin("vertex.output", func(_ *starlark.Thread, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
			if err := arity("vertex.output", args, kwargs, 0, 0); err != nil { return nil, err }
			output := v.vertex.Output()
			if output == nil { return starlark.None, nil }
			return WrapOutput(output), nil
		}), nil
	case "inputs":
		return boundBuiltin("vertex.inputs", func(_ *starlark.Thread, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
			if err := arity("vertex.inputs", args, kwargs, 0, 0); err != nil { return nil, err }
			return outputListValue(v.vertex.Inputs()), nil
		}), nil
	case "validate":
		return boundBuiltin("vertex.validate", func(thread *starlark.Thread, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
			if err := arity("vertex.validate", args, kwargs, 0, -1); err != nil { return nil, err }
			constraints, err := constraintsFromArgs(args, 0)
			if err != nil { return nil, err }
			if err := v.vertex.Validate(callContext(thread), constraints); err != nil { return nil, err }
			return starlark.None, nil
		}), nil
	case "marshal":
		return boundBuiltin("vertex.marshal", func(thread *starlark.Thread, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
			if err := arity("vertex.marshal", args, kwargs, 0, -1); err != nil { return nil, err }
			constraints, err := constraintsFromArgs(args, 0)
			if err != nil { return nil, err }
			dgst, data, metadata, sources, err := v.vertex.Marshal(callContext(thread), constraints)
			if err != nil { return nil, err }
			return marshalResult(dgst, data, metadata, sources), nil
		}), nil
	default:
		return nil, nil
	}
}

func (v *VertexValue) AttrNames() []string {
	return SnakeCaseNames([]string{"output", "inputs", "validate", "marshal", "output", "inputs", "validate", "marshal"})
}

var (
	_ starlark.Value     = (*StateValue)(nil)
	_ starlark.HasAttrs  = (*StateValue)(nil)
	_ starlark.Value     = (*OutputValue)(nil)
	_ starlark.HasAttrs  = (*OutputValue)(nil)
	_ starlark.Value     = (*VertexValue)(nil)
	_ starlark.Value     = (*ConstraintsValue)(nil)
	_ starlark.HasAttrs  = (*ConstraintsValue)(nil)
	_ starlark.Value     = (*OpMetadataValue)(nil)
	_ starlark.HasAttrs  = (*OpMetadataValue)(nil)
	_ starlark.Value     = (*OCIPlatformValue)(nil)
	_ starlark.HasAttrs  = (*OCIPlatformValue)(nil)
)


func withDescriptionBuiltin(_ *starlark.Thread, _ *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	if err := arity("with_description", args, kwargs, 1, 1); err != nil { return nil, err }
	values, err := stringDict(args[0], "with_description")
	if err != nil { return nil, err }
	return wrapOption("constraints_opt", buildkitllb.WithDescription(values)), nil
}

func withCustomNamefBuiltin(_ *starlark.Thread, _ *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	if err := arity("with_custom_namef", args, kwargs, 1, -1); err != nil { return nil, err }
	format, err := stringAt("with_custom_namef", args, 0)
	if err != nil { return nil, err }
	values := make([]any, 0, len(args)-1)
	for _, arg := range args[1:] { values = append(values, goScalar(arg)) }
	return wrapOption("constraints_opt", buildkitllb.WithCustomNamef(format, values...)), nil
}

func withExportCacheBuiltin(_ *starlark.Thread, _ *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	if err := arity("with_export_cache", args, kwargs, 0, 0); err != nil { return nil, err }
	return wrapOption("constraints_opt", buildkitllb.WithExportCache()), nil
}

func withoutExportCacheBuiltin(_ *starlark.Thread, _ *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	if err := arity("without_export_cache", args, kwargs, 0, 0); err != nil { return nil, err }
	return wrapOption("constraints_opt", buildkitllb.WithoutExportCache()), nil
}

func withoutDefaultExportCacheBuiltin(_ *starlark.Thread, _ *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	if err := arity("without_default_export_cache", args, kwargs, 0, 0); err != nil { return nil, err }
	return wrapOption("constraints_opt", buildkitllb.WithoutDefaultExportCache()), nil
}

func withLinuxResourcesBuiltin(_ *starlark.Thread, _ *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	var resources buildkitllb.LinuxResources
	if len(args) == 1 && len(kwargs) == 0 {
		value, ok := args[0].(*LinuxResourcesValue)
		if !ok || value == nil { return nil, fmt.Errorf("with_linux_resources: got %s, want llb.linux_resources", valueType(args[0])) }
		resources = value.resources
	} else {
		if len(args) != 0 { return nil, fmt.Errorf("with_linux_resources: use either one llb.linux_resources value or named fields") }
		if err := UnpackArgs("with_linux_resources", args, kwargs,
			"memory?", &resources.Memory,
			"memory_swap?", &resources.MemorySwap,
			"cpu_shares?", &resources.CPUShares,
			"cpu_period?", &resources.CPUPeriod,
			"cpu_quota?", &resources.CPUQuota,
			"cpuset_cpus?", &resources.CpusetCpus,
			"cpuset_mems?", &resources.CpusetMems,
		); err != nil { return nil, err }
	}
	return wrapOption("constraints_opt", buildkitllb.WithLinuxResources(resources)), nil
}

func platformConstraintBuiltin(_ *starlark.Thread, _ *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	var platform ocispecs.Platform
	if err := UnpackArgs("platform", args, kwargs,
		"os", &platform.OS,
		"architecture", &platform.Architecture,
		"variant?", &platform.Variant,
		"os_version?", &platform.OSVersion,
		"os_features?", &platform.OSFeatures,
	); err != nil { return nil, err }
	return wrapOption("constraints_opt", buildkitllb.Platform(platform)), nil
}

func progressGroupBuiltin(_ *starlark.Thread, _ *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	if err := arity("progress_group", args, kwargs, 3, 3); err != nil { return nil, err }
	id, err := stringAt("progress_group", args, 0); if err != nil { return nil, err }
	name, err := stringAt("progress_group", args, 1); if err != nil { return nil, err }
	weak, err := boolAt("progress_group", args, 2); if err != nil { return nil, err }
	return wrapOption("constraints_opt", buildkitllb.ProgressGroup(id, name, weak)), nil
}

func requireBuiltin(_ *starlark.Thread, _ *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	if err := arity("require", args, kwargs, 0, -1); err != nil { return nil, err }
	var filters []string
	if len(args) == 1 {
		if sequence, ok := args[0].(starlark.Indexable); ok {
			values, err := stringList(sequence); if err != nil { return nil, err }
			filters = values
		} else {
			value, err := stringAt("require", args, 0); if err != nil { return nil, err }
			filters = []string{value}
		}
	} else {
		filters = make([]string, len(args))
		for i := range args {
			value, err := stringAt("require", args, i); if err != nil { return nil, err }
			filters[i] = value
		}
	}
	return wrapOption("constraints_opt", buildkitllb.Require(filters...)), nil
}

// CapSetValue is an opaque adapter for BuildKit's negotiated API capability set.
// It can be injected by Go code; Starlark cannot fabricate server capabilities.
type CapSetValue struct { capSet apicaps.CapSet }

// WrapCapSet adapts a BuildKit API capability set.
func WrapCapSet(capSet apicaps.CapSet) *CapSetValue { return &CapSetValue{capSet: capSet} }

func (v *CapSetValue) String() string        { return "<llb.cap_set>" }
func (v *CapSetValue) Type() string          { return "llb.cap_set" }
func (v *CapSetValue) Freeze()               {}
func (v *CapSetValue) Truth() starlark.Bool  { return starlark.Bool(v != nil) }
func (v *CapSetValue) Hash() (uint32, error) { return 0, fmt.Errorf("unhashable: llb.cap_set") }

func withCapsBuiltin(_ *starlark.Thread, _ *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	if err := arity("with_caps", args, kwargs, 1, 1); err != nil { return nil, err }
	caps, ok := args[0].(*CapSetValue)
	if !ok || caps == nil { return nil, fmt.Errorf("with_caps: got %s, want llb.cap_set", valueType(args[0])) }
	return wrapOption("constraints_opt", buildkitllb.WithCaps(caps.capSet)), nil
}

func opMetadataBuiltin(_ *starlark.Thread, _ *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	var metadata buildkitllb.OpMetadata
	var description starlark.Value
	if err := UnpackArgs("op_metadata", args, kwargs,
		"ignore_cache?", &metadata.IgnoreCache,
		"description?", &description,
	); err != nil { return nil, err }
	if description != nil && description != starlark.None {
		values, err := stringDict(description, "op_metadata.description")
		if err != nil { return nil, err }
		metadata.Description = values
	}
	return WrapOpMetadata(metadata), nil
}


// Starlark identifier and argument helpers.
var (
	snakeAcronyms = strings.NewReplacer("UIDGID", "UID_GID")
	snakeUpperBoundary = regexp.MustCompile(`([A-Z]+)([A-Z][a-z])`)
	snakeLowerBoundary = regexp.MustCompile(`([a-z0-9])([A-Z])`)
)

func SnakeCaseIdentifier(name string) string {
	name = strings.TrimSpace(name)
	name = strings.ReplaceAll(name, "-", "_")
	name = snakeAcronyms.Replace(name)
	name = snakeUpperBoundary.ReplaceAllString(name, `${1}_${2}`)
	name = snakeLowerBoundary.ReplaceAllString(name, `${1}_${2}`)
	name = strings.ToLower(name)
	for strings.Contains(name, "__") {
		name = strings.ReplaceAll(name, "__", "_")
	}
	return strings.Trim(name, "_")
}

func SnakeCaseNames(names []string) []string {
	result := make([]string, 0, len(names))
	seen := make(map[string]struct{}, len(names))
	for _, name := range names {
				if name == "" {
			continue
		}
		if _, ok := seen[name]; ok {
			continue
		}
		seen[name] = struct{}{}
		result = append(result, name)
	}
	sort.Strings(result)
	return result
}

func NormalizeMembers(members starlark.StringDict) (starlark.StringDict, error) {
	if members == nil {
		return nil, fmt.Errorf("starlark/llb: nil member dictionary")
	}
	result := make(starlark.StringDict, len(members))
	original := make(map[string]string, len(members))
	for name, value := range members {
		key := SnakeCaseIdentifier(name)
		if key == "" || value == nil {
			return nil, fmt.Errorf("starlark/llb: invalid member %q", name)
		}
		if previous, ok := original[key]; ok {
			return nil, fmt.Errorf("starlark/llb: members %q and %q both normalize to %q", previous, name, key)
		}
		original[key] = name
		result[key] = value
	}
	return result, nil
}

// UnpackArgs uses snake_case parameter names without accepting CamelCase
// keyword aliases. Every Starlark callable therefore has one canonical API.
func UnpackArgs(fnname string, args starlark.Tuple, kwargs []starlark.Tuple, pairs ...any) error {
	return starlark.UnpackArgs(SnakeCaseIdentifier(fnname), args, kwargs, pairs...)
}

// Shared Starlark value adapters and argument conversion helpers.
// optionValue is an opaque Starlark carrier for a native BuildKit option.
// The dynamic Go type is retained so consumers can validate the exact
// BuildKit interface family (RunOption, StateOption, ImageOption, and so on).
type optionValue struct {
	kind  string
	value any
}

func (v *optionValue) String() string {
	if v == nil {
		return "<llb.option nil>"
	}
	return "<llb." + SnakeCaseIdentifier(v.kind) + ">"
}

func (v *optionValue) Type() string {
	if v == nil || v.kind == "" {
		return "llb.option"
	}
	return "llb." + SnakeCaseIdentifier(v.kind)
}

func (v *optionValue) Freeze() {}

func (v *optionValue) Truth() starlark.Bool {
	return starlark.Bool(v != nil && v.value != nil)
}

func (v *optionValue) Hash() (uint32, error) {
	return 0, fmt.Errorf("unhashable: %s", v.Type())
}

// WrapOption injects a native BuildKit option into a Starlark environment.
// Its interface family is checked when the option is consumed.
func WrapOption(option any) starlark.Value {
	if option == nil {
		return starlark.None
	}
	return &optionValue{kind: "go_option", value: option}
}

func wrapOption(kind string, value any) starlark.Value {
	if value == nil {
		return starlark.None
	}
	return &optionValue{kind: kind, value: value}
}

func optionFromValue[T any](value starlark.Value, want string) (T, error) {
	var zero T

	option, ok := value.(*optionValue)
	if !ok || option == nil || option.value == nil {
		return zero, fmt.Errorf("got %s, want llb.%s option", valueType(value), want)
	}
	result, ok := option.value.(T)
	if !ok {
		return zero, fmt.Errorf("llb.%s cannot be used as %s option", option.kind, want)
	}
	return result, nil
}

func optionsFromArgs[T any](args starlark.Tuple, start int, want string) ([]T, error) {
	if start < 0 || start > len(args) {
		return nil, fmt.Errorf("invalid option argument start index %d", start)
	}

	options := make([]T, 0, len(args)-start)
	for i := start; i < len(args); i++ {
		option, err := optionFromValue[T](args[i], want)
		if err != nil {
			return nil, fmt.Errorf("argument %d: %w", i+1, err)
		}
		options = append(options, option)
	}
	return options, nil
}

func valueType(value starlark.Value) string {
	if value == nil {
		return "nil"
	}
	return value.Type()
}

func arity(name string, args starlark.Tuple, kwargs []starlark.Tuple, min, max int) error {
	if len(kwargs) != 0 {
		return fmt.Errorf("%s: keyword arguments are not supported; use the documented named arguments or pass options positionally", name)
	}
	if len(args) < min || (max >= 0 && len(args) > max) {
		switch {
		case max < 0:
			return fmt.Errorf("%s: got %d arguments, want at least %d", name, len(args), min)
		case min == max:
			return fmt.Errorf("%s: got %d arguments, want %d", name, len(args), min)
		default:
			return fmt.Errorf("%s: got %d arguments, want %d..%d", name, len(args), min, max)
		}
	}
	return nil
}

func stringAt(name string, args starlark.Tuple, index int) (string, error) {
	if index < 0 || index >= len(args) {
		return "", fmt.Errorf("%s: missing argument %d", name, index+1)
	}
	value, ok := args[index].(starlark.String)
	if !ok {
		return "", fmt.Errorf("%s: argument %d: got %s, want string", name, index+1, valueType(args[index]))
	}
	return string(value), nil
}

func boolAt(name string, args starlark.Tuple, index int) (bool, error) {
	if index < 0 || index >= len(args) {
		return false, fmt.Errorf("%s: missing argument %d", name, index+1)
	}
	value, ok := args[index].(starlark.Bool)
	if !ok {
		return false, fmt.Errorf("%s: argument %d: got %s, want bool", name, index+1, valueType(args[index]))
	}
	return bool(value), nil
}

func intAt(name string, args starlark.Tuple, index int) (int, error) {
	if index < 0 || index >= len(args) {
		return 0, fmt.Errorf("%s: missing argument %d", name, index+1)
	}
	value, ok := args[index].(starlark.Int)
	if !ok {
		return 0, fmt.Errorf("%s: argument %d: got %s, want int", name, index+1, valueType(args[index]))
	}
	result, ok := value.Int64()
	if !ok || int64(int(result)) != result {
		return 0, fmt.Errorf("%s: argument %d: integer out of range", name, index+1)
	}
	return int(result), nil
}

func int64At(name string, args starlark.Tuple, index int) (int64, error) {
	if index < 0 || index >= len(args) {
		return 0, fmt.Errorf("%s: missing argument %d", name, index+1)
	}
	value, ok := args[index].(starlark.Int)
	if !ok {
		return 0, fmt.Errorf("%s: argument %d: got %s, want int", name, index+1, valueType(args[index]))
	}
	result, ok := value.Int64()
	if !ok {
		return 0, fmt.Errorf("%s: argument %d: integer out of range for int64", name, index+1)
	}
	return result, nil
}

func uint64At(name string, args starlark.Tuple, index int) (uint64, error) {
	if index < 0 || index >= len(args) {
		return 0, fmt.Errorf("%s: missing argument %d", name, index+1)
	}
	value, ok := args[index].(starlark.Int)
	if !ok {
		return 0, fmt.Errorf("%s: argument %d: got %s, want int", name, index+1, valueType(args[index]))
	}
	result, ok := value.Uint64()
	if !ok {
		return 0, fmt.Errorf("%s: argument %d: expected a non-negative uint64", name, index+1)
	}
	return result, nil
}

func bytesAt(name string, args starlark.Tuple, index int) ([]byte, error) {
	if index < 0 || index >= len(args) {
		return nil, fmt.Errorf("%s: missing argument %d", name, index+1)
	}
	switch value := args[index].(type) {
	case starlark.Bytes:
		return []byte(value), nil
	case starlark.String:
		return []byte(value), nil
	default:
		return nil, fmt.Errorf("%s: argument %d: got %s, want bytes or string", name, index+1, valueType(args[index]))
	}
}

func stringList(value starlark.Value) ([]string, error) {
	sequence, ok := value.(starlark.Indexable)
	if !ok {
		return nil, fmt.Errorf("got %s, want list or tuple of strings", valueType(value))
	}
	result := make([]string, sequence.Len())
	for i := range result {
		item := sequence.Index(i)
		str, ok := item.(starlark.String)
		if !ok {
			return nil, fmt.Errorf("element %d: got %s, want string", i, valueType(item))
		}
		result[i] = string(str)
	}
	return result, nil
}

func stringSliceValue(values []string) starlark.Value {
	items := make([]starlark.Value, len(values))
	for i, value := range values {
		items[i] = starlark.String(value)
	}
	return starlark.NewList(items)
}

func goScalar(value starlark.Value) any {
	switch v := value.(type) {
	case starlark.String:
		return string(v)
	case starlark.Int:
		if n, ok := v.Int64(); ok {
			return n
		}
		return v.BigInt()
	case starlark.Float:
		return float64(v)
	case starlark.Bool:
		return bool(v)
	case starlark.NoneType:
		return nil
	default:
		return value.String()
	}
}

func stringDict(value starlark.Value, name string) (map[string]string, error) {
	dict, ok := value.(*starlark.Dict)
	if !ok {
		return nil, fmt.Errorf("%s: got %s, want dict", name, valueType(value))
	}
	result := make(map[string]string, dict.Len())
	for _, item := range dict.Items() {
		key, ok := item[0].(starlark.String)
		if !ok {
			return nil, fmt.Errorf("%s: dictionary keys must be strings", name)
		}
		val, ok := item[1].(starlark.String)
		if !ok {
			return nil, fmt.Errorf("%s: dictionary values must be strings", name)
		}
		result[string(key)] = string(val)
	}
	return result, nil
}

func builtin(name string, fn func(*starlark.Thread, *starlark.Builtin, starlark.Tuple, []starlark.Tuple) (starlark.Value, error)) *starlark.Builtin {
	return starlark.NewBuiltin(SnakeCaseIdentifier(name), fn)
}

func boundBuiltin(name string, fn func(*starlark.Thread, starlark.Tuple, []starlark.Tuple) (starlark.Value, error)) *starlark.Builtin {
	return starlark.NewBuiltin(SnakeCaseIdentifier(name), func(thread *starlark.Thread, _ *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
		return fn(thread, args, kwargs)
	})
}

func optionBuiltin(name, kind string, build func(*starlark.Thread, starlark.Tuple, []starlark.Tuple) (any, error)) *starlark.Builtin {
	return builtin(name, func(thread *starlark.Thread, _ *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
		value, err := build(thread, args, kwargs)
		if err != nil {
			return nil, err
		}
		return wrapOption(kind, value), nil
	})
}

var (
	_ starlark.Value = (*optionValue)(nil)
)


func stringOptionBuiltin[T any](name, kind string, build func(string) T) *starlark.Builtin {
	return optionBuiltin(name, kind, func(_ *starlark.Thread, args starlark.Tuple, kwargs []starlark.Tuple) (any, error) {
		if err := arity(name, args, kwargs, 1, 1); err != nil { return nil, err }
		value, err := stringAt(name, args, 0)
		if err != nil { return nil, err }
		return build(value), nil
	})
}

func boolOptionBuiltin[T any](name, kind string, build func(bool) T) *starlark.Builtin {
	return optionBuiltin(name, kind, func(_ *starlark.Thread, args starlark.Tuple, kwargs []starlark.Tuple) (any, error) {
		if err := arity(name, args, kwargs, 1, 1); err != nil { return nil, err }
		value, err := boolAt(name, args, 0)
		if err != nil { return nil, err }
		return build(value), nil
	})
}

func intOptionBuiltin[T any](name, kind string, build func(int) T) *starlark.Builtin {
	return optionBuiltin(name, kind, func(_ *starlark.Thread, args starlark.Tuple, kwargs []starlark.Tuple) (any, error) {
		if err := arity(name, args, kwargs, 1, 1); err != nil { return nil, err }
		value, err := intAt(name, args, 0)
		if err != nil { return nil, err }
		return build(value), nil
	})
}

func int64OptionBuiltin[T any](name, kind string, build func(int64) T) *starlark.Builtin {
	return optionBuiltin(name, kind, func(_ *starlark.Thread, args starlark.Tuple, kwargs []starlark.Tuple) (any, error) {
		if err := arity(name, args, kwargs, 1, 1); err != nil { return nil, err }
		value, err := int64At(name, args, 0)
		if err != nil { return nil, err }
		return build(value), nil
	})
}

func uint64OptionBuiltin[T any](name, kind string, build func(uint64) T) *starlark.Builtin {
	return optionBuiltin(name, kind, func(_ *starlark.Thread, args starlark.Tuple, kwargs []starlark.Tuple) (any, error) {
		if err := arity(name, args, kwargs, 1, 1); err != nil { return nil, err }
		value, err := uint64At(name, args, 0)
		if err != nil { return nil, err }
		return build(value), nil
	})
}


// UnwrapOption extracts a native BuildKit option from a Starlark value.
// This is primarily useful to wrapper subpackages that compose LLB options.
func UnwrapOption[T any](value starlark.Value, family string) (T, error) {
	return optionFromValue[T](value, family)
}

// Root LLB builtin inventory. Module construction and registration remain in starlark/modules.
// BuiltinMembers returns the global members exported by the high-level
// BuildKit client/llb facade. It does not construct or register a module;
// module lifecycle and registration are owned by starlark/modules.
func BuiltinMembers() (starlark.StringDict, error) {
	members := make(starlark.StringDict)
	origins := make(map[string]string)

	add := func(group map[string]starlark.Value, kind string) error {
		for name, value := range group {
			if name == "" || value == nil {
				return fmt.Errorf("starlark/llb: invalid %s %q", kind, name)
			}
			if previous, exists := origins[name]; exists {
				return fmt.Errorf("starlark/llb: duplicate %s %q (already registered as %s)", kind, name, previous)
			}
			origins[name] = kind
			members[name] = value
		}
		return nil
	}

	builtinGroups := []func() map[string]starlark.Value{
		stateBuiltins,
		metaBuiltins,
		execBuiltins,
		fileOpBuiltins,
		sourceBuiltins,
		resolverBuiltins,
		diffBuiltins,
		mergeBuiltins,
		passthroughBuiltins,
		definitionBuiltins,
		marshalBuiltins,
		sourceMapBuiltins,
	}
	for _, group := range builtinGroups {
		if err := add(group(), "builtin"); err != nil {
			return nil, err
		}
	}

	valueGroups := []map[string]starlark.Value{
		platformConstraintValues(),
		stateConstraintValues(),
		imageOptionValues(),
		networkModeValues(),
		securityModeValues(),
		cacheMountSharingValues(),
		mountContentCacheValues(),
		ulimitValues(),
		cdiDeviceOptionValues(),
		sshOptionValues(),
		secretOptionValues(),
		diffTypeValues(),
		gitAuthValues(),
		resolveModeValues(),
	}
	for _, group := range valueGroups {
		if err := add(group, "constant"); err != nil {
			return nil, err
		}
	}

	normalized, err := NormalizeMembers(members)
	if err != nil {
		return nil, fmt.Errorf("starlark/llb: invalid builtin registry: %w", err)
	}
	return normalized, nil
}

// callContext returns the Go context attached to the current Starlark thread.
// callContext retrieves the Go context attached to the current Starlark
// thread. If no context was supplied, it returns context.Background.
func callContext(thread *starlark.Thread) context.Context {
	if thread != nil {
		if ctx, ok := thread.Local("context").(context.Context); ok && ctx != nil {
			return ctx
		}
	}
	return context.Background()
}