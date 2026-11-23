// Package llb provides a Starlark facade over BuildKit's client/llb API.
package llb

import (
	"fmt"

	"go.starlark.net/starlark"
)

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
	return &optionValue{kind: "GoOption", value: option}
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
