package llb

import (
	"fmt"

	buildkitllb "github.com/moby/buildkit/client/llb"
	"go.starlark.net/starlark"
)

// StateValue is the Starlark representation of an immutable BuildKit llb.State.
// Its underlying value is intentionally private so Starlark cannot construct
// invalid states by setting implementation fields.
type StateValue struct {
	state buildkitllb.State
}

// WrapState adapts a native BuildKit state for use in Starlark globals.
func WrapState(state buildkitllb.State) *StateValue { return &StateValue{state: state} }

// WrapExecState adapts a native BuildKit execution state for Starlark globals.
func WrapExecState(state buildkitllb.ExecState) *ExecStateValue {
	return &ExecStateValue{exec: state}
}

// WrapFileAction adapts a native BuildKit file action for Starlark globals.
func WrapFileAction(action *buildkitllb.FileAction) *FileActionValue {
	return &FileActionValue{action: action}
}

// WrapDefinition adapts a native BuildKit definition for Starlark globals.
func WrapDefinition(definition *buildkitllb.Definition) *DefinitionValue {
	return &DefinitionValue{definition: definition}
}

// WrapOption adapts a native BuildKit option into an opaque Starlark value.
// The underlying value must implement the appropriate BuildKit option
// interface (for example llb.RunOption or llb.ConstraintsOpt). The receiving
// builtin validates that interface before applying the option.
func WrapOption(option any) starlark.Value {
	if option == nil {
		return starlark.None
	}
	return &optionValue{kind: "GoOption", value: option}
}

// Unwrap returns the immutable BuildKit state represented by v.
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

// ExecStateValue represents the result of State.Run. BuildKit's ExecState
// retains the execution operation so additional mounts can be attached before
// Root is selected.
type ExecStateValue struct {
	exec buildkitllb.ExecState
}

// Unwrap returns the native BuildKit execution state.
func (v *ExecStateValue) Unwrap() buildkitllb.ExecState {
	if v == nil {
		return buildkitllb.ExecState{}
	}
	return v.exec
}

func (v *ExecStateValue) String() string        { return "<llb.ExecState>" }
func (v *ExecStateValue) Type() string          { return "llb.ExecState" }
func (v *ExecStateValue) Freeze()               {}
func (v *ExecStateValue) Truth() starlark.Bool  { return starlark.Bool(v != nil) }
func (v *ExecStateValue) Hash() (uint32, error) { return 0, fmt.Errorf("unhashable: llb.ExecState") }

// FileActionValue represents a BuildKit file operation. File actions are
// persistent values: chaining a method returns a new action.
type FileActionValue struct {
	action *buildkitllb.FileAction
}

// Unwrap returns the native BuildKit file action.
func (v *FileActionValue) Unwrap() *buildkitllb.FileAction {
	if v == nil {
		return nil
	}
	return v.action
}

func (v *FileActionValue) String() string        { return "<llb.FileAction>" }
func (v *FileActionValue) Type() string          { return "llb.FileAction" }
func (v *FileActionValue) Freeze()               {}
func (v *FileActionValue) Truth() starlark.Bool  { return starlark.Bool(v != nil && v.action != nil) }
func (v *FileActionValue) Hash() (uint32, error) { return 0, fmt.Errorf("unhashable: llb.FileAction") }

// optionValue carries a native, typed BuildKit option. The Starlark-facing
// constructor determines the interface the value satisfies; consumers assert
// that interface before invoking BuildKit.
type optionValue struct {
	kind  string
	value any
}

func (v *optionValue) String() string {
	return fmt.Sprintf("<llb.%s>", v.kind)
}
func (v *optionValue) Type() string          { return "llb." + v.kind }
func (v *optionValue) Freeze()               {}
func (v *optionValue) Truth() starlark.Bool  { return starlark.Bool(v != nil && v.value != nil) }
func (v *optionValue) Hash() (uint32, error) { return 0, fmt.Errorf("unhashable: %s", v.Type()) }

// stateFromValue validates and unwraps a Starlark LLB state.
func stateFromValue(v starlark.Value) (buildkitllb.State, error) {
	state, ok := v.(*StateValue)
	if !ok || state == nil {
		return buildkitllb.State{}, fmt.Errorf("got %s, want llb.State", valueType(v))
	}
	return state.state, nil
}

func actionFromValue(v starlark.Value) (*buildkitllb.FileAction, error) {
	action, ok := v.(*FileActionValue)
	if !ok || action == nil || action.action == nil {
		return nil, fmt.Errorf("got %s, want llb.FileAction", valueType(v))
	}
	return action.action, nil
}

func valueType(v starlark.Value) string {
	if v == nil {
		return "nil"
	}
	return v.Type()
}

func optionFromValue[T any](v starlark.Value, want string) (T, error) {
	var zero T
	option, ok := v.(*optionValue)
	if !ok || option == nil {
		return zero, fmt.Errorf("got %s, want llb.%s option", valueType(v), want)
	}
	result, ok := option.value.(T)
	if !ok {
		return zero, fmt.Errorf("llb.%s cannot be used as %s option", option.kind, want)
	}
	return result, nil
}

func optionsFromArgs[T any](args starlark.Tuple, start int, want string) ([]T, error) {
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


// DefinitionValue is a marshalled BuildKit LLB definition. It is suitable for
// passing to the native BuildKit client after the Starlark program completes.
type DefinitionValue struct {
	definition *buildkitllb.Definition
}

// Unwrap returns the native BuildKit definition.
func (v *DefinitionValue) Unwrap() *buildkitllb.Definition {
	if v == nil {
		return nil
	}
	return v.definition
}

func (v *DefinitionValue) String() string        { return "<llb.Definition>" }
func (v *DefinitionValue) Type() string          { return "llb.Definition" }
func (v *DefinitionValue) Freeze()               {}
func (v *DefinitionValue) Truth() starlark.Bool  { return starlark.Bool(v != nil && v.definition != nil) }
func (v *DefinitionValue) Hash() (uint32, error) { return 0, fmt.Errorf("unhashable: llb.Definition") }

func (v *DefinitionValue) Attr(name string) (starlark.Value, error) {
	if v == nil || v.definition == nil {
		return nil, fmt.Errorf("llb.Definition: nil receiver")
	}
	switch name {
	case "Head", "head":
		return boundBuiltin("Definition.Head", func(_ *starlark.Thread, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
			if err := arity("Definition.Head", args, kwargs, 0, 0); err != nil {
				return nil, err
			}
			head, err := v.definition.Head()
			if err != nil {
				return nil, err
			}
			return starlark.String(head.String()), nil
		}), nil
	default:
		return nil, nil
	}
}

func (v *DefinitionValue) AttrNames() []string {
	return []string{"Head", "head"}
}

var _ starlark.HasAttrs = (*DefinitionValue)(nil)
