// Package expand provides a Dockerfile-oriented variable expander without shell parsing.
package expand

import (
	"fmt"
	"strings"
)

// Lookup resolves a variable and reports whether it exists.
type Lookup func(string) (string, bool)

// Config controls the source escape rune and unset-variable behavior.
type Config struct {
	Escape        rune
	PreserveUnset bool
	Strict        bool
}

// Expander expands variable references using a caller-provided lookup.
type Expander struct {
	config Config
	lookup Lookup
}

// New creates an expander. A zero escape rune selects backslash.
func New(config Config, lookup Lookup) *Expander {
	if config.Escape == 0 {
		config.Escape = '\\'
	}
	if lookup == nil {
		lookup = func(string) (string, bool) { return "", false }
	}
	return &Expander{config: config, lookup: lookup}
}

// MapLookup adapts a string map to a Lookup.
func MapLookup(values map[string]string) Lookup {
	return func(key string) (string, bool) {
		value, ok := values[key]
		return value, ok
	}
}

// Expand expands $NAME, the braced NAME form, and default/alternate/required forms.
func (e *Expander) Expand(input string) (string, error) {
	if e == nil {
		return input, nil
	}
	var out strings.Builder
	out.Grow(len(input))
	for i := 0; i < len(input); {
		if rune(input[i]) == e.config.Escape && i+1 < len(input) && input[i+1] == '$' {
			out.WriteByte('$')
			i += 2
			continue
		}
		if input[i] != '$' {
			out.WriteByte(input[i])
			i++
			continue
		}
		if i+1 < len(input) && input[i+1] == '$' {
			out.WriteByte('$')
			i += 2
			continue
		}
		if i+1 < len(input) && input[i+1] == '{' {
			end, err := closingBrace(input, i+2)
			if err != nil {
				return "", err
			}
			value, err := e.expandBraced(input[i+2 : end])
			if err != nil {
				return "", err
			}
			out.WriteString(value)
			i = end + 1
			continue
		}
		start, end := i+1, i+1
		for end < len(input) && isNameByte(input[end], end == start) {
			end++
		}
		if end == start {
			out.WriteByte('$')
			i++
			continue
		}
		name := input[start:end]
		value, ok := e.lookup(name)
		if !ok {
			if e.config.Strict {
				return "", fmt.Errorf("variable %q is not set", name)
			}
			if e.config.PreserveUnset {
				out.WriteString(input[i:end])
			}
		} else {
			out.WriteString(value)
		}
		i = end
	}
	return out.String(), nil
}

func (e *Expander) expandBraced(expression string) (string, error) {
	name, operator, operand := splitExpression(expression)
	if !validName(name) {
		return "", fmt.Errorf("invalid variable expression: %s", expression)
	}
	value, exists := e.lookup(name)
	set := exists && value != ""
	switch operator {
	case "":
		if !exists {
			return e.unset(name, "$"+"{"+expression+"}")
		}
		return value, nil
	case "-", ":-":
		if exists && (operator == "-" || set) {
			return value, nil
		}
		return e.Expand(operand)
	case "+", ":+":
		if exists && (operator == "+" || set) {
			return e.Expand(operand)
		}
		return "", nil
	case "?", ":?":
		if exists && (operator == "?" || set) {
			return value, nil
		}
		message := operand
		if message == "" {
			message = name + " is required"
		}
		message, err := e.Expand(message)
		if err != nil {
			return "", err
		}
		return "", fmt.Errorf("%s", message)
	default:
		return "", fmt.Errorf("unsupported expansion operator %q", operator)
	}
}

func (e *Expander) unset(name, original string) (string, error) {
	if e.config.Strict {
		return "", fmt.Errorf("variable %q is not set", name)
	}
	if e.config.PreserveUnset {
		return original, nil
	}
	return "", nil
}

func splitExpression(expression string) (name, operator, operand string) {
	for i := 0; i < len(expression); i++ {
		if expression[i] == ':' && i+1 < len(expression) && strings.ContainsRune("-+?", rune(expression[i+1])) {
			return expression[:i], expression[i : i+2], expression[i+2:]
		}
		if strings.ContainsRune("-+?", rune(expression[i])) {
			return expression[:i], expression[i : i+1], expression[i+1:]
		}
	}
	return expression, "", ""
}

func closingBrace(input string, start int) (int, error) {
	depth := 1
	for i := start; i < len(input); i++ {
		switch input[i] {
		case '{':
			depth++
		case '}':
			depth--
			if depth == 0 {
				return i, nil
			}
		}
	}
	return 0, fmt.Errorf("unterminated variable expansion")
}

func isNameByte(value byte, first bool) bool {
	if value == '_' || value >= 'a' && value <= 'z' || value >= 'A' && value <= 'Z' {
		return true
	}
	return !first && value >= '0' && value <= '9'
}

func validName(value string) bool {
	if value == "" || !isNameByte(value[0], true) {
		return false
	}
	for i := 1; i < len(value); i++ {
		if !isNameByte(value[i], false) {
			return false
		}
	}
	return true
}
