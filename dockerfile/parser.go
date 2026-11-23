package dockerfile

import (
	"fmt"
	"strings"
)

// ParseTokens parses already-tokenized Dockerfile instruction operands. It
// handles the instruction flag layer, not shell quoting, continuations,
// heredocs, JSON arrays, or environment expansion.
//
// Supported forms are --name=value, --name value, -a value, boolean --name,
// boolean --no-name, repeatable flags, and the -- end-of-flags marker. Parsing
// is transactional: an error never leaves partial flag changes on the target.
func (i *Instruction) ParseTokens(tokens ...string) ([]string, error) {
	if i == nil {
		return nil, fmt.Errorf("%w: nil instruction", ErrInvalidInstruction)
	}
	snapshot := i.Clone()
	baseVersion := snapshot.version
	arguments, err := snapshot.parseTokens(tokens...)
	if err != nil {
		return nil, err
	}
	arguments = append([]string(nil), arguments...)
	flags := snapshot.Flags().All()

	i.mu.Lock()
	defer i.mu.Unlock()
	if i.version != baseVersion {
		return nil, ErrInstructionConflict
	}
	i.arguments = arguments
	i.flags = flags
	i.version++
	return append([]string(nil), arguments...), nil
}

func (i *Instruction) parseTokens(tokens ...string) ([]string, error) {
	arguments := make([]string, 0, len(tokens))
	parsingFlags := true
	for index := 0; index < len(tokens); index++ {
		token := tokens[index]
		if !parsingFlags {
			arguments = append(arguments, token)
			continue
		}
		if token == "--" {
			arguments = append(arguments, tokens[index+1:]...)
			break
		}
		if !strings.HasPrefix(token, "-") || token == "-" {
			arguments = append(arguments, token)
			parsingFlags = false
			continue
		}

		nameValue := strings.TrimLeft(token, "-")
		if nameValue == "" {
			return nil, fmt.Errorf("%w: invalid flag token %q", ErrInvalidInstruction, token)
		}
		name, raw, hasValue := strings.Cut(nameValue, "=")
		lookup := name
		definition, exists := i.flagDefinition(name)
		negated := false
		if !exists && strings.HasPrefix(name, "no-") {
			lookup = strings.TrimPrefix(name, "no-")
			definition, exists = i.flagDefinition(lookup)
			negated = exists
		}
		if !exists {
			return nil, fmt.Errorf("%w: unknown flag %q for %s", ErrInvalidInstruction, token, i.Name())
		}
		if negated {
			if hasValue || definition.Type != FlagBool {
				return nil, fmt.Errorf("%w: %q is only valid for boolean flags without an explicit value", ErrInvalidInstruction, token)
			}
			if err := i.SetFlag(lookup, false); err != nil {
				return nil, err
			}
			continue
		}

		if !hasValue {
			if definition.Type == FlagBool {
				raw = "true"
			} else {
				if index+1 >= len(tokens) || tokens[index+1] == "--" {
					return nil, fmt.Errorf("%w: flag --%s requires a value", ErrInvalidInstruction, definition.Name)
				}
				index++
				raw = tokens[index]
			}
		}

		var err error
		if definition.Repeatable && definition.Type == FlagStringSlice {
			err = i.AppendFlag(lookup, raw)
		} else {
			err = i.SetFlag(lookup, raw)
		}
		if err != nil {
			return nil, err
		}
	}
	if err := i.SetArguments(arguments...); err != nil {
		return nil, err
	}
	return i.Arguments(), nil
}

func (i *Instruction) flagDefinition(name string) (FlagDefinition, bool) {
	if i == nil {
		return FlagDefinition{}, false
	}
	i.mu.RLock()
	index := findFlagIndex(i.definition.Flags, name)
	if index < 0 {
		i.mu.RUnlock()
		return FlagDefinition{}, false
	}
	definition := i.definition.Flags[index]
	i.mu.RUnlock()
	return cloneFlagDefinition(definition), true
}
