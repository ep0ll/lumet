package dockerfile

import (
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"
)

// FlagType is the declared value type of an instruction flag.
type FlagType uint8

const (
	FlagString FlagType = iota
	FlagBool
	FlagInt
	FlagUint
	FlagDuration
	FlagStringSlice
	FlagEnum
	FlagCustom
)

func (t FlagType) String() string {
	switch t {
	case FlagString:
		return "string"
	case FlagBool:
		return "bool"
	case FlagInt:
		return "int"
	case FlagUint:
		return "uint"
	case FlagDuration:
		return "duration"
	case FlagStringSlice:
		return "string-slice"
	case FlagEnum:
		return "enum"
	case FlagCustom:
		return "custom"
	default:
		return "unknown"
	}
}

// FlagValueCloner lets application-owned mutable values provide independent
// copies when instruction snapshots are created.
type FlagValueCloner interface {
	CloneDockerfileFlagValue() any
}

// FlagDefinition describes one instruction flag. Additions to this structure
// are schema additions; callers should use keyed literals.
type FlagDefinition struct {
	Name          string
	Aliases       []string
	Type          FlagType
	Description   string
	Default       any
	Required      bool
	Repeatable    bool
	Hidden        bool
	Deprecated    string
	AllowedValues []string
	MinInt        *int64
	MaxInt        *int64
	Parse         func(string) (any, error)
	Validate      func(any) error
	_             struct{}
}

// NormalizeFlagName accepts names with or without leading dashes and
// normalizes underscores to hyphens.
func NormalizeFlagName(name string) string {
	name = strings.TrimLeft(strings.TrimSpace(name), "-")
	return strings.ToLower(strings.ReplaceAll(name, "_", "-"))
}

func cloneFlagDefinition(in FlagDefinition) FlagDefinition {
	out := in
	out.Aliases = append([]string(nil), in.Aliases...)
	out.AllowedValues = append([]string(nil), in.AllowedValues...)
	if in.MinInt != nil {
		value := *in.MinInt
		out.MinInt = &value
	}
	if in.MaxInt != nil {
		value := *in.MaxInt
		out.MaxInt = &value
	}
	out.Default = cloneFlagValue(in.Default)
	return out
}

func cloneFlagValue(value any) any {
	switch value := value.(type) {
	case []string:
		return append([]string(nil), value...)
	case []int:
		return append([]int(nil), value...)
	case []int64:
		return append([]int64(nil), value...)
	case []uint64:
		return append([]uint64(nil), value...)
	case []any:
		copy := make([]any, len(value))
		for index, item := range value {
			copy[index] = cloneFlagValue(item)
		}
		return copy
	case map[string]string:
		copy := make(map[string]string, len(value))
		for key, item := range value {
			copy[key] = item
		}
		return copy
	case map[string]any:
		copy := make(map[string]any, len(value))
		for key, item := range value {
			copy[key] = cloneFlagValue(item)
		}
		return copy
	case FlagValueCloner:
		return value.CloneDockerfileFlagValue()
	default:
		return value
	}
}

func validateFlagDefinition(def FlagDefinition) error {
	name := NormalizeFlagName(def.Name)
	if name == "" || strings.ContainsAny(name, " \t\r\n/=") {
		return fmt.Errorf("%w: invalid flag name %q", ErrInvalidDefinition, def.Name)
	}
	if def.Type > FlagCustom {
		return fmt.Errorf("%w: unsupported type %d for flag --%s", ErrInvalidDefinition, def.Type, name)
	}
	if def.Repeatable && def.Type != FlagStringSlice {
		return fmt.Errorf("%w: repeatable flag --%s must use string-slice type", ErrInvalidDefinition, name)
	}
	if def.Type == FlagCustom && def.Parse == nil {
		return fmt.Errorf("%w: custom flag --%s requires a parser", ErrInvalidDefinition, name)
	}
	if def.Type == FlagEnum {
		if len(def.AllowedValues) == 0 {
			return fmt.Errorf("%w: enum flag --%s has no allowed values", ErrInvalidDefinition, name)
		}
		seen := make(map[string]struct{}, len(def.AllowedValues))
		for _, value := range def.AllowedValues {
			value = strings.TrimSpace(value)
			key := strings.ToLower(value)
			if value == "" {
				return fmt.Errorf("%w: enum flag --%s contains an empty value", ErrInvalidDefinition, name)
			}
			if _, exists := seen[key]; exists {
				return fmt.Errorf("%w: enum flag --%s repeats value %q", ErrInvalidDefinition, name, value)
			}
			seen[key] = struct{}{}
		}
	} else if len(def.AllowedValues) != 0 {
		return fmt.Errorf("%w: allowed values require enum type for --%s", ErrInvalidDefinition, name)
	}
	if (def.MinInt != nil || def.MaxInt != nil) && def.Type != FlagInt {
		return fmt.Errorf("%w: integer bounds require int type for --%s", ErrInvalidDefinition, name)
	}
	if def.MinInt != nil && def.MaxInt != nil && *def.MinInt > *def.MaxInt {
		return fmt.Errorf("%w: minimum exceeds maximum for --%s", ErrInvalidDefinition, name)
	}

	seen := map[string]struct{}{name: {}}
	for _, alias := range def.Aliases {
		alias = NormalizeFlagName(alias)
		if alias == "" || strings.ContainsAny(alias, " \t\r\n/=") {
			return fmt.Errorf("%w: invalid alias %q for --%s", ErrInvalidDefinition, alias, name)
		}
		if _, exists := seen[alias]; exists {
			return fmt.Errorf("%w: duplicate alias %q for --%s", ErrInvalidDefinition, alias, name)
		}
		seen[alias] = struct{}{}
	}

	if def.Default != nil {
		value, err := parseFlagValue(def, def.Default)
		if err != nil {
			return fmt.Errorf("%w: invalid default for --%s: %v", ErrInvalidDefinition, name, err)
		}
		if err := validateFlagValue(def, value); err != nil {
			return fmt.Errorf("%w: invalid default for --%s: %v", ErrInvalidDefinition, name, err)
		}
	}
	return nil
}

func parseFlagValue(def FlagDefinition, raw any) (any, error) {
	switch def.Type {
	case FlagString:
		value, ok := raw.(string)
		if !ok {
			return nil, fmt.Errorf("want string, got %T", raw)
		}
		return value, nil
	case FlagBool:
		switch value := raw.(type) {
		case bool:
			return value, nil
		case string:
			parsed, err := strconv.ParseBool(value)
			if err != nil {
				return nil, fmt.Errorf("want boolean: %w", err)
			}
			return parsed, nil
		default:
			return nil, fmt.Errorf("want bool or string, got %T", raw)
		}
	case FlagInt:
		switch value := raw.(type) {
		case int:
			return int64(value), nil
		case int8:
			return int64(value), nil
		case int16:
			return int64(value), nil
		case int32:
			return int64(value), nil
		case int64:
			return value, nil
		case uint:
			if uint64(value) > math.MaxInt64 {
				return nil, fmt.Errorf("integer overflows int64")
			}
			return int64(value), nil
		case uint64:
			if value > math.MaxInt64 {
				return nil, fmt.Errorf("integer overflows int64")
			}
			return int64(value), nil
		case string:
			parsed, err := strconv.ParseInt(value, 10, 64)
			if err != nil {
				return nil, fmt.Errorf("want integer: %w", err)
			}
			return parsed, nil
		default:
			return nil, fmt.Errorf("want integer, got %T", raw)
		}
	case FlagUint:
		switch value := raw.(type) {
		case uint:
			return uint64(value), nil
		case uint8:
			return uint64(value), nil
		case uint16:
			return uint64(value), nil
		case uint32:
			return uint64(value), nil
		case uint64:
			return value, nil
		case int:
			if value < 0 {
				return nil, fmt.Errorf("unsigned integer cannot be negative")
			}
			return uint64(value), nil
		case int64:
			if value < 0 {
				return nil, fmt.Errorf("unsigned integer cannot be negative")
			}
			return uint64(value), nil
		case string:
			parsed, err := strconv.ParseUint(value, 10, 64)
			if err != nil {
				return nil, fmt.Errorf("want unsigned integer: %w", err)
			}
			return parsed, nil
		default:
			return nil, fmt.Errorf("want unsigned integer, got %T", raw)
		}
	case FlagDuration:
		switch value := raw.(type) {
		case time.Duration:
			return value, nil
		case string:
			parsed, err := time.ParseDuration(value)
			if err != nil {
				return nil, fmt.Errorf("want duration: %w", err)
			}
			return parsed, nil
		default:
			return nil, fmt.Errorf("want duration or string, got %T", raw)
		}
	case FlagStringSlice:
		switch value := raw.(type) {
		case string:
			return []string{value}, nil
		case []string:
			return append([]string(nil), value...), nil
		case []any:
			result := make([]string, len(value))
			for index, item := range value {
				text, ok := item.(string)
				if !ok {
					return nil, fmt.Errorf("item %d: want string, got %T", index, item)
				}
				result[index] = text
			}
			return result, nil
		default:
			return nil, fmt.Errorf("want string or string slice, got %T", raw)
		}
	case FlagEnum:
		value, ok := raw.(string)
		if !ok {
			return nil, fmt.Errorf("want enum string, got %T", raw)
		}
		for _, candidate := range def.AllowedValues {
			if strings.EqualFold(value, candidate) {
				return candidate, nil
			}
		}
		return nil, fmt.Errorf("must be one of: %s", strings.Join(def.AllowedValues, ", "))
	case FlagCustom:
		if text, ok := raw.(string); ok {
			return def.Parse(text)
		}
		return raw, nil
	default:
		return nil, fmt.Errorf("unsupported flag type %d", def.Type)
	}
}

func validateFlagValue(def FlagDefinition, value any) error {
	if def.MinInt != nil || def.MaxInt != nil {
		number, ok := value.(int64)
		if !ok {
			return fmt.Errorf("numeric bounds require int64 value, got %T", value)
		}
		if def.MinInt != nil && number < *def.MinInt {
			return fmt.Errorf("must be at least %d", *def.MinInt)
		}
		if def.MaxInt != nil && number > *def.MaxInt {
			return fmt.Errorf("must be at most %d", *def.MaxInt)
		}
	}
	if def.Validate != nil {
		return def.Validate(cloneFlagValue(value))
	}
	return nil
}

// FlagValues is a read-only snapshot of resolved values. Get returns defensive
// copies, and aliases resolve to the canonical flag name.
type FlagValues struct {
	values  map[string]any
	aliases map[string]string
}

func newFlagValues(definitions []FlagDefinition) (FlagValues, error) {
	values := make(map[string]any, len(definitions))
	aliases := make(map[string]string, len(definitions)*2)
	for _, definition := range definitions {
		name := NormalizeFlagName(definition.Name)
		aliases[name] = name
		for _, alias := range definition.Aliases {
			aliases[NormalizeFlagName(alias)] = name
		}
		if definition.Default == nil {
			continue
		}
		value, err := parseFlagValue(definition, definition.Default)
		if err != nil {
			return FlagValues{}, fmt.Errorf("default for --%s: %w", name, err)
		}
		if err := validateFlagValue(definition, value); err != nil {
			return FlagValues{}, fmt.Errorf("default for --%s: %w", name, err)
		}
		values[name] = cloneFlagValue(value)
	}
	return FlagValues{values: values, aliases: aliases}, nil
}

// Get returns a defensive copy of one value.
func (v FlagValues) Get(name string) (any, bool) {
	key := NormalizeFlagName(name)
	if canonical, ok := v.aliases[key]; ok {
		key = canonical
	}
	value, ok := v.values[key]
	return cloneFlagValue(value), ok
}

// String returns a string flag value.
func (v FlagValues) String(name string) (string, bool) {
	value, ok := v.Get(name)
	if !ok {
		return "", false
	}
	result, ok := value.(string)
	return result, ok
}

// Bool returns a boolean flag value.
func (v FlagValues) Bool(name string) (bool, bool) {
	value, ok := v.Get(name)
	if !ok {
		return false, false
	}
	result, ok := value.(bool)
	return result, ok
}

// Int returns a signed integer flag value.
func (v FlagValues) Int(name string) (int64, bool) {
	value, ok := v.Get(name)
	if !ok {
		return 0, false
	}
	result, ok := value.(int64)
	return result, ok
}

// Uint returns an unsigned integer flag value.
func (v FlagValues) Uint(name string) (uint64, bool) {
	value, ok := v.Get(name)
	if !ok {
		return 0, false
	}
	result, ok := value.(uint64)
	return result, ok
}

// Duration returns a duration flag value.
func (v FlagValues) Duration(name string) (time.Duration, bool) {
	value, ok := v.Get(name)
	if !ok {
		return 0, false
	}
	result, ok := value.(time.Duration)
	return result, ok
}

// Strings returns a repeatable string flag value.
func (v FlagValues) Strings(name string) ([]string, bool) {
	value, ok := v.Get(name)
	if !ok {
		return nil, false
	}
	result, ok := value.([]string)
	return result, ok
}

// Has reports whether a flag value is set.
func (v FlagValues) Has(name string) bool {
	_, ok := v.Get(name)
	return ok
}

// All returns a copy keyed by canonical flag name.
func (v FlagValues) All() map[string]any {
	result := make(map[string]any, len(v.values))
	for key, value := range v.values {
		result[key] = cloneFlagValue(value)
	}
	return result
}
