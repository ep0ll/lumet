package llb

import (
	"fmt"
	"sort"

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

func (v *StateValue) String() string        { return "<llb.State>" }
func (v *StateValue) Type() string          { return "llb.State" }
func (v *StateValue) Freeze()               {}
func (v *StateValue) Truth() starlark.Bool  { return starlark.Bool(v != nil) }
func (v *StateValue) Hash() (uint32, error) { return 0, fmt.Errorf("unhashable: llb.State") }

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

func (v *OutputValue) String() string        { return "<llb.Output>" }
func (v *OutputValue) Type() string          { return "llb.Output" }
func (v *OutputValue) Freeze()               {}
func (v *OutputValue) Truth() starlark.Bool  { return starlark.Bool(v != nil && v.output != nil) }
func (v *OutputValue) Hash() (uint32, error) { return 0, fmt.Errorf("unhashable: llb.Output") }

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

func (v *VertexValue) String() string        { return "<llb.Vertex>" }
func (v *VertexValue) Type() string          { return "llb.Vertex" }
func (v *VertexValue) Freeze()               {}
func (v *VertexValue) Truth() starlark.Bool  { return starlark.Bool(v != nil && v.vertex != nil) }
func (v *VertexValue) Hash() (uint32, error) { return 0, fmt.Errorf("unhashable: llb.Vertex") }

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

func (v *ConstraintsValue) String() string        { return "<llb.Constraints>" }
func (v *ConstraintsValue) Type() string          { return "llb.Constraints" }
func (v *ConstraintsValue) Freeze()               {}
func (v *ConstraintsValue) Truth() starlark.Bool  { return starlark.Bool(v != nil) }
func (v *ConstraintsValue) Hash() (uint32, error) { return 0, fmt.Errorf("unhashable: llb.Constraints") }

func (v *ConstraintsValue) Attr(name string) (starlark.Value, error) {
	if v == nil {
		return nil, fmt.Errorf("llb.Constraints: nil receiver")
	}
	switch name {
	case "platform":
		if v.constraints.Platform == nil {
			return starlark.None, nil
		}
		return WrapOCIPlatform(*v.constraints.Platform), nil
	case "worker_constraints", "WorkerConstraints":
		return stringSliceValue(v.constraints.WorkerConstraints), nil
	case "metadata", "Metadata":
		return WrapOpMetadata(v.constraints.Metadata), nil
	case "local_unique_id", "LocalUniqueID":
		return starlark.String(v.constraints.LocalUniqueID), nil
	case "source_locations", "SourceLocations":
		values := make([]starlark.Value, len(v.constraints.SourceLocations))
		for i, location := range v.constraints.SourceLocations {
			values[i] = WrapSourceLocation(location)
		}
		return starlark.NewList(values), nil
	case "has_capabilities", "HasCapabilities":
		return starlark.Bool(v.constraints.Caps != nil), nil
	default:
		return nil, nil
	}
}

func (v *ConstraintsValue) AttrNames() []string {
	return []string{"has_capabilities", "local_unique_id", "metadata", "platform", "source_locations", "worker_constraints",
		"HasCapabilities", "LocalUniqueID", "Metadata", "SourceLocations", "WorkerConstraints"}
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

func (v *OpMetadataValue) String() string        { return "<llb.OpMetadata>" }
func (v *OpMetadataValue) Type() string          { return "llb.OpMetadata" }
func (v *OpMetadataValue) Freeze()               {}
func (v *OpMetadataValue) Truth() starlark.Bool  { return starlark.Bool(v != nil) }
func (v *OpMetadataValue) Hash() (uint32, error) { return 0, fmt.Errorf("unhashable: llb.OpMetadata") }

func (v *OpMetadataValue) Attr(name string) (starlark.Value, error) {
	if v == nil {
		return nil, fmt.Errorf("llb.OpMetadata: nil receiver")
	}
	switch name {
	case "ignore_cache", "IgnoreCache":
		return starlark.Bool(v.metadata.IgnoreCache), nil
	case "description", "Description":
		return stringDictValue(v.metadata.Description), nil
	case "export_cache", "ExportCache":
		if v.metadata.ExportCache == nil {
			return starlark.None, nil
		}
		return starlark.Bool(v.metadata.ExportCache.Value), nil
	case "progress_group", "ProgressGroup":
		if v.metadata.ProgressGroup == nil {
			return starlark.None, nil
		}
		return starlarkstructValue("llb.ProgressGroup", starlark.StringDict{
			"id":   starlark.String(v.metadata.ProgressGroup.Id),
			"name": starlark.String(v.metadata.ProgressGroup.Name),
			"weak": starlark.Bool(v.metadata.ProgressGroup.Weak),
		}), nil
	case "linux_resources", "LinuxResources":
		if v.metadata.LinuxResources == nil {
			return starlark.None, nil
		}
		return starlarkstructValue("llb.LinuxResources", starlark.StringDict{
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
	return []string{
		"Description", "ExportCache", "IgnoreCache", "LinuxResources", "ProgressGroup",
		"description", "export_cache", "ignore_cache", "linux_resources", "progress_group",
	}
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
		return "<llb.OCIPlatform nil>"
	}
	return fmt.Sprintf("<llb.OCIPlatform %s/%s>", v.platform.OS, v.platform.Architecture)
}
func (v *OCIPlatformValue) Type() string          { return "llb.OCIPlatform" }
func (v *OCIPlatformValue) Freeze()               {}
func (v *OCIPlatformValue) Truth() starlark.Bool  { return starlark.Bool(v != nil) }
func (v *OCIPlatformValue) Hash() (uint32, error) { return 0, fmt.Errorf("unhashable: llb.OCIPlatform") }

func (v *OCIPlatformValue) Attr(name string) (starlark.Value, error) {
	if v == nil {
		return nil, fmt.Errorf("llb.OCIPlatform: nil receiver")
	}
	switch name {
	case "os", "OS":
		return starlark.String(v.platform.OS), nil
	case "architecture", "Architecture":
		return starlark.String(v.platform.Architecture), nil
	case "variant", "Variant":
		return starlark.String(v.platform.Variant), nil
	case "os_version", "OSVersion":
		return starlark.String(v.platform.OSVersion), nil
	case "os_features", "OSFeatures":
		return stringSliceValue(v.platform.OSFeatures), nil
	default:
		return nil, nil
	}
}

func (v *OCIPlatformValue) AttrNames() []string {
	return []string{"architecture", "os", "os_features", "os_version", "variant",
		"Architecture", "OS", "OSFeatures", "OSVersion", "Variant"}
}

func stringDictValue(values map[string]string) starlark.Value {
	dict := starlark.NewDict(len(values))
	for key, value := range values {
		_ = dict.SetKey(starlark.String(key), starlark.String(value))
	}
	return dict
}

func starlarkstructValue(name string, values starlark.StringDict) starlark.Value {
	return starlarkstruct.FromStringDict(starlark.String(name), values)
}

func (v *StateValue) Attr(name string) (starlark.Value, error) {
	if v == nil {
		return nil, fmt.Errorf("llb.State: nil receiver")
	}
	switch name {
	case "WithValue", "with_value":
		return boundBuiltin("State.WithValue", func(_ *starlark.Thread, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
			if err := arity("State.WithValue", args, kwargs, 2, 2); err != nil {
				return nil, err
			}
			key, ok := args[0].(starlark.String)
			if !ok {
				return nil, fmt.Errorf("State.WithValue: key must be a string, got %s", valueType(args[0]))
			}
			return WrapState(v.state.WithValue(string(key), goScalar(args[1]))), nil
		}), nil
	case "Value", "value":
		return boundBuiltin("State.Value", func(thread *starlark.Thread, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
			if err := arity("State.Value", args, kwargs, 1, -1); err != nil {
				return nil, err
			}
			key, ok := args[0].(starlark.String)
			if !ok {
				return nil, fmt.Errorf("State.Value: key must be a string, got %s", valueType(args[0]))
			}
			opts, err := optionsFromArgs[buildkitllb.ConstraintsOpt](args, 1, "ConstraintsOpt")
			if err != nil {
				return nil, err
			}
			result, err := v.state.Value(callContext(thread), string(key), opts...)
			if err != nil {
				return nil, err
			}
			return fromGoValue(result), nil
		}), nil
	case "Output", "output":
		return boundBuiltin("State.Output", func(_ *starlark.Thread, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
			if err := arity("State.Output", args, kwargs, 0, 0); err != nil {
				return nil, err
			}
			output := v.state.Output()
			if output == nil {
				return starlark.None, nil
			}
			return WrapOutput(output), nil
		}), nil
	case "WithOutput", "with_output":
		return boundBuiltin("State.WithOutput", func(_ *starlark.Thread, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
			if err := arity("State.WithOutput", args, kwargs, 1, 1); err != nil {
				return nil, err
			}
			output, ok := args[0].(*OutputValue)
			if !ok || output == nil {
				return nil, fmt.Errorf("State.WithOutput: got %s, want llb.Output", valueType(args[0]))
			}
			return WrapState(v.state.WithOutput(output.output)), nil
		}), nil
	case "Validate", "validate":
		return boundBuiltin("State.Validate", func(thread *starlark.Thread, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
			if err := arity("State.Validate", args, kwargs, 0, -1); err != nil {
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
		"Output", "Value", "Validate", "WithOutput", "WithValue",
		"output", "value", "validate", "with_output", "with_value",
	}
	for _, provider := range stateAttrNames {
		names = append(names, provider...)
	}
	sort.Strings(names)
	return compactStrings(names)
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
	options, err := optionsFromArgs[buildkitllb.ConstraintsOpt](args, start, "ConstraintsOpt")
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
		"NewState": builtin("NewState", newStateBuiltin),
		"NewConstraints": builtin("NewConstraints", newConstraintsBuiltin),
		"OCIPlatform": builtin("OCIPlatform", ociPlatformBuiltin),
	}
	for name, value := range stateConstraintBuiltins() {
		if _, exists := members[name]; exists { panic("starlark/llb: duplicate state builtin " + name) }
		members[name] = value
	}
	return members
}

func stateConstraintBuiltins() map[string]starlark.Value {
	return map[string]starlark.Value{
		"WithDescription": builtin("WithDescription", withDescriptionBuiltin),
		"WithCustomName": stringOptionBuiltin("WithCustomName", "ConstraintsOpt", buildkitllb.WithCustomName),
		"WithCustomNamef": builtin("WithCustomNamef", withCustomNamefBuiltin),
		"WithExportCache": builtin("WithExportCache", withExportCacheBuiltin),
		"WithoutExportCache": builtin("WithoutExportCache", withoutExportCacheBuiltin),
		"WithoutDefaultExportCache": builtin("WithoutDefaultExportCache", withoutDefaultExportCacheBuiltin),
		"WithLinuxResources": builtin("WithLinuxResources", withLinuxResourcesBuiltin),
		"Platform": builtin("Platform", platformConstraintBuiltin),
		"LocalUniqueID": stringOptionBuiltin("LocalUniqueID", "ConstraintsOpt", buildkitllb.LocalUniqueID),
		"ProgressGroup": builtin("ProgressGroup", progressGroupBuiltin),
		"MemoryLimit": int64OptionBuiltin("MemoryLimit", "ConstraintsOpt", buildkitllb.MemoryLimit),
		"MemorySwapLimit": int64OptionBuiltin("MemorySwapLimit", "ConstraintsOpt", buildkitllb.MemorySwapLimit),
		"CPUShares": uint64OptionBuiltin("CPUShares", "ConstraintsOpt", buildkitllb.CPUShares),
		"CPUPeriod": uint64OptionBuiltin("CPUPeriod", "ConstraintsOpt", buildkitllb.CPUPeriod),
		"CPUQuota": int64OptionBuiltin("CPUQuota", "ConstraintsOpt", buildkitllb.CPUQuota),
		"CpusetCpus": stringOptionBuiltin("CpusetCpus", "ConstraintsOpt", buildkitllb.CpusetCpus),
		"CpusetMems": stringOptionBuiltin("CpusetMems", "ConstraintsOpt", buildkitllb.CpusetMems),
		"Require": builtin("Require", requireBuiltin),
		"WithCaps": builtin("WithCaps", withCapsBuiltin),
		"OpMetadata": builtin("OpMetadata", opMetadataBuiltin),
	}
}

func newStateBuiltin(_ *starlark.Thread, _ *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	if err := arity("NewState", args, kwargs, 1, 1); err != nil {
		return nil, err
	}
	output, ok := args[0].(*OutputValue)
	if !ok || output == nil {
		return nil, fmt.Errorf("NewState: got %s, want llb.Output", valueType(args[0]))
	}
	return WrapState(buildkitllb.NewState(output.output)), nil
}

func newConstraintsBuiltin(_ *starlark.Thread, _ *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	if len(kwargs) != 0 {
		return nil, fmt.Errorf("NewConstraints: options must be passed positionally")
	}
	options, err := optionsFromArgs[buildkitllb.ConstraintsOpt](args, 0, "ConstraintsOpt")
	if err != nil {
		return nil, err
	}
	return WrapConstraints(*buildkitllb.NewConstraints(options...)), nil
}

func ociPlatformBuiltin(_ *starlark.Thread, _ *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	var platform ocispecs.Platform
	if err := starlark.UnpackArgs("OCIPlatform", args, kwargs,
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
		"LinuxAmd64":   wrapOption("ConstraintsOpt", buildkitllb.LinuxAmd64),
		"LinuxArmhf":   wrapOption("ConstraintsOpt", buildkitllb.LinuxArmhf),
		"LinuxArm":     wrapOption("ConstraintsOpt", buildkitllb.LinuxArm),
		"LinuxArmel":   wrapOption("ConstraintsOpt", buildkitllb.LinuxArmel),
		"LinuxArm64":   wrapOption("ConstraintsOpt", buildkitllb.LinuxArm64),
		"LinuxS390x":   wrapOption("ConstraintsOpt", buildkitllb.LinuxS390x),
		"LinuxPpc64":   wrapOption("ConstraintsOpt", buildkitllb.LinuxPpc64),
		"LinuxPpc64le": wrapOption("ConstraintsOpt", buildkitllb.LinuxPpc64le),
		"LinuxRiscv64": wrapOption("ConstraintsOpt", buildkitllb.LinuxRiscv64),
		"Darwin":       wrapOption("ConstraintsOpt", buildkitllb.Darwin),
		"Windows":      wrapOption("ConstraintsOpt", buildkitllb.Windows),
	}
}

func stateConstraintValues() map[string]starlark.Value {
	return map[string]starlark.Value{
		"IgnoreCache": wrapOption("ConstraintsOpt", buildkitllb.IgnoreCache),
	}
}

func (v *OutputValue) Attr(name string) (starlark.Value, error) {
	if v == nil || v.output == nil {
		return nil, fmt.Errorf("llb.Output: nil receiver")
	}
	switch name {
	case "Vertex", "vertex":
		return boundBuiltin("Output.Vertex", func(thread *starlark.Thread, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
			if err := arity("Output.Vertex", args, kwargs, 0, -1); err != nil { return nil, err }
			constraints, err := constraintsFromArgs(args, 0)
			if err != nil { return nil, err }
			return WrapVertex(v.output.Vertex(callContext(thread), constraints)), nil
		}), nil
	case "ToInput", "to_input":
		return boundBuiltin("Output.ToInput", func(thread *starlark.Thread, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
			if err := arity("Output.ToInput", args, kwargs, 0, -1); err != nil { return nil, err }
			constraints, err := constraintsFromArgs(args, 0)
			if err != nil { return nil, err }
			input, err := v.output.ToInput(callContext(thread), constraints)
			if err != nil { return nil, err }
			if input == nil { return starlark.None, nil }
			return starlarkstructValue("llb.Input", starlark.StringDict{
				"digest": starlark.String(input.Digest),
				"index": starlark.MakeInt64(input.Index),
			}), nil
		}), nil
	default:
		return nil, nil
	}
}

func (v *OutputValue) AttrNames() []string {
	return []string{"Vertex", "ToInput", "vertex", "to_input"}
}

func (v *VertexValue) Attr(name string) (starlark.Value, error) {
	if v == nil || v.vertex == nil {
		return nil, fmt.Errorf("llb.Vertex: nil receiver")
	}
	switch name {
	case "Output", "output":
		return boundBuiltin("Vertex.Output", func(_ *starlark.Thread, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
			if err := arity("Vertex.Output", args, kwargs, 0, 0); err != nil { return nil, err }
			output := v.vertex.Output()
			if output == nil { return starlark.None, nil }
			return WrapOutput(output), nil
		}), nil
	case "Inputs", "inputs":
		return boundBuiltin("Vertex.Inputs", func(_ *starlark.Thread, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
			if err := arity("Vertex.Inputs", args, kwargs, 0, 0); err != nil { return nil, err }
			return outputListValue(v.vertex.Inputs()), nil
		}), nil
	case "Validate", "validate":
		return boundBuiltin("Vertex.Validate", func(thread *starlark.Thread, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
			if err := arity("Vertex.Validate", args, kwargs, 0, -1); err != nil { return nil, err }
			constraints, err := constraintsFromArgs(args, 0)
			if err != nil { return nil, err }
			if err := v.vertex.Validate(callContext(thread), constraints); err != nil { return nil, err }
			return starlark.None, nil
		}), nil
	case "Marshal", "marshal":
		return boundBuiltin("Vertex.Marshal", func(thread *starlark.Thread, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
			if err := arity("Vertex.Marshal", args, kwargs, 0, -1); err != nil { return nil, err }
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
	return []string{"Output", "Inputs", "Validate", "Marshal", "output", "inputs", "validate", "marshal"}
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
	if err := arity("WithDescription", args, kwargs, 1, 1); err != nil { return nil, err }
	values, err := stringDict(args[0], "WithDescription")
	if err != nil { return nil, err }
	return wrapOption("ConstraintsOpt", buildkitllb.WithDescription(values)), nil
}

func withCustomNamefBuiltin(_ *starlark.Thread, _ *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	if err := arity("WithCustomNamef", args, kwargs, 1, -1); err != nil { return nil, err }
	format, err := stringAt("WithCustomNamef", args, 0)
	if err != nil { return nil, err }
	values := make([]any, 0, len(args)-1)
	for _, arg := range args[1:] { values = append(values, goScalar(arg)) }
	return wrapOption("ConstraintsOpt", buildkitllb.WithCustomNamef(format, values...)), nil
}

func withExportCacheBuiltin(_ *starlark.Thread, _ *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	if err := arity("WithExportCache", args, kwargs, 0, 0); err != nil { return nil, err }
	return wrapOption("ConstraintsOpt", buildkitllb.WithExportCache()), nil
}

func withoutExportCacheBuiltin(_ *starlark.Thread, _ *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	if err := arity("WithoutExportCache", args, kwargs, 0, 0); err != nil { return nil, err }
	return wrapOption("ConstraintsOpt", buildkitllb.WithoutExportCache()), nil
}

func withoutDefaultExportCacheBuiltin(_ *starlark.Thread, _ *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	if err := arity("WithoutDefaultExportCache", args, kwargs, 0, 0); err != nil { return nil, err }
	return wrapOption("ConstraintsOpt", buildkitllb.WithoutDefaultExportCache()), nil
}

func withLinuxResourcesBuiltin(_ *starlark.Thread, _ *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	var resources buildkitllb.LinuxResources
	if len(args) == 1 && len(kwargs) == 0 {
		value, ok := args[0].(*LinuxResourcesValue)
		if !ok || value == nil { return nil, fmt.Errorf("WithLinuxResources: got %s, want llb.LinuxResources", valueType(args[0])) }
		resources = value.resources
	} else {
		if len(args) != 0 { return nil, fmt.Errorf("WithLinuxResources: use either one llb.LinuxResources value or named fields") }
		if err := starlark.UnpackArgs("WithLinuxResources", args, kwargs,
			"memory?", &resources.Memory,
			"memory_swap?", &resources.MemorySwap,
			"cpu_shares?", &resources.CPUShares,
			"cpu_period?", &resources.CPUPeriod,
			"cpu_quota?", &resources.CPUQuota,
			"cpuset_cpus?", &resources.CpusetCpus,
			"cpuset_mems?", &resources.CpusetMems,
		); err != nil { return nil, err }
	}
	return wrapOption("ConstraintsOpt", buildkitllb.WithLinuxResources(resources)), nil
}

func platformConstraintBuiltin(_ *starlark.Thread, _ *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	var platform ocispecs.Platform
	if err := starlark.UnpackArgs("Platform", args, kwargs,
		"os", &platform.OS,
		"architecture", &platform.Architecture,
		"variant?", &platform.Variant,
		"os_version?", &platform.OSVersion,
		"os_features?", &platform.OSFeatures,
	); err != nil { return nil, err }
	return wrapOption("ConstraintsOpt", buildkitllb.Platform(platform)), nil
}

func progressGroupBuiltin(_ *starlark.Thread, _ *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	if err := arity("ProgressGroup", args, kwargs, 3, 3); err != nil { return nil, err }
	id, err := stringAt("ProgressGroup", args, 0); if err != nil { return nil, err }
	name, err := stringAt("ProgressGroup", args, 1); if err != nil { return nil, err }
	weak, err := boolAt("ProgressGroup", args, 2); if err != nil { return nil, err }
	return wrapOption("ConstraintsOpt", buildkitllb.ProgressGroup(id, name, weak)), nil
}

func requireBuiltin(_ *starlark.Thread, _ *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	if err := arity("Require", args, kwargs, 0, -1); err != nil { return nil, err }
	var filters []string
	if len(args) == 1 {
		if sequence, ok := args[0].(starlark.Indexable); ok {
			values, err := stringList(sequence); if err != nil { return nil, err }
			filters = values
		} else {
			value, err := stringAt("Require", args, 0); if err != nil { return nil, err }
			filters = []string{value}
		}
	} else {
		filters = make([]string, len(args))
		for i := range args {
			value, err := stringAt("Require", args, i); if err != nil { return nil, err }
			filters[i] = value
		}
	}
	return wrapOption("ConstraintsOpt", buildkitllb.Require(filters...)), nil
}

// CapSetValue is an opaque adapter for BuildKit's negotiated API capability set.
// It can be injected by Go code; Starlark cannot fabricate server capabilities.
type CapSetValue struct { capSet apicaps.CapSet }

// WrapCapSet adapts a BuildKit API capability set.
func WrapCapSet(capSet apicaps.CapSet) *CapSetValue { return &CapSetValue{capSet: capSet} }

func (v *CapSetValue) String() string        { return "<llb.CapSet>" }
func (v *CapSetValue) Type() string          { return "llb.CapSet" }
func (v *CapSetValue) Freeze()               {}
func (v *CapSetValue) Truth() starlark.Bool  { return starlark.Bool(v != nil) }
func (v *CapSetValue) Hash() (uint32, error) { return 0, fmt.Errorf("unhashable: llb.CapSet") }

func withCapsBuiltin(_ *starlark.Thread, _ *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	if err := arity("WithCaps", args, kwargs, 1, 1); err != nil { return nil, err }
	caps, ok := args[0].(*CapSetValue)
	if !ok || caps == nil { return nil, fmt.Errorf("WithCaps: got %s, want llb.CapSet", valueType(args[0])) }
	return wrapOption("ConstraintsOpt", buildkitllb.WithCaps(caps.capSet)), nil
}

func opMetadataBuiltin(_ *starlark.Thread, _ *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	var metadata buildkitllb.OpMetadata
	var description starlark.Value
	if err := starlark.UnpackArgs("OpMetadata", args, kwargs,
		"ignore_cache?", &metadata.IgnoreCache,
		"description?", &description,
	); err != nil { return nil, err }
	if description != nil && description != starlark.None {
		values, err := stringDict(description, "OpMetadata.description")
		if err != nil { return nil, err }
		metadata.Description = values
	}
	return WrapOpMetadata(metadata), nil
}
