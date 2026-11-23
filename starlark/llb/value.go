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
func (v *ExecStateValue) Truth() starlark.Bool  { return v != nil }
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
