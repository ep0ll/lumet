package dockerfile

import (
	"errors"
	"fmt"
	"sort"
	"strings"
	"sync"
)

var (
	ErrNotFound         = errors.New("dockerfile: not found")
	ErrAlreadyExists    = errors.New("dockerfile: already exists")
	ErrRevisionConflict = errors.New("dockerfile: definition changed during update")
	ErrInvalidDefinition = errors.New("dockerfile: invalid definition")
)

// InstructionClass determines whether an instruction may begin a build stage.
type InstructionClass uint8

const (
	InstructionStandard InstructionClass = iota
	InstructionStageInitializer
)

// ArgumentDefinition describes one positional argument. A variadic argument
// may appear before a final required argument (COPY/ADD sources then dest).
type ArgumentDefinition struct {
	Name        string
	Description string
	Required    bool
	Variadic    bool
	_           struct{}
}

// InstructionValidator validates the complete instruction after generic
// argument and flag validation.
type InstructionValidator func(args []string, flags FlagValues, payload any) error

// InstructionDefinition is the schema for one instruction. Definitions are
// copied on read/write and versioned by Registry.
type InstructionDefinition struct {
	Name        string
	Aliases     []string
	Kind        Kind
	Class       InstructionClass
	Description string
	Arguments   []ArgumentDefinition
	Flags       []FlagDefinition
	OnBuildable bool
	Deprecated  string
	History     HistoryFunc
	HistoryMode HistoryMode
	Validate    InstructionValidator
	Revision    uint64
	_           struct{}
}

// Registry stores instruction definitions and their flag schemas. Its zero
// value is ready to use. Instruction instances use immutable snapshots.
type Registry struct {
	mu           sync.RWMutex
	instructions map[string]InstructionDefinition
	aliases      map[string]string
	generation   uint64
}

// NewRegistry returns an empty, ready-to-use registry.
func NewRegistry() *Registry { return &Registry{} }

// NewDefaultRegistry returns a registry containing all standard Dockerfile
// instructions and their currently supported flags.
func NewDefaultRegistry() (*Registry, error) {
	registry := NewRegistry()
	for _, definition := range BuiltinInstructions() {
		if err := registry.CreateInstruction(definition); err != nil {
			return nil, err
		}
	}
	return registry, nil
}

// Generation returns the monotonic registry generation.
func (r *Registry) Generation() uint64 {
	if r == nil {
		return 0
	}
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.generation
}

// NormalizeInstructionName canonicalizes instruction names and aliases.
func NormalizeInstructionName(name string) string {
	return strings.ToUpper(strings.TrimSpace(name))
}

func validInstructionName(name string) bool {
	if name == "" {
		return false
	}
	for index, char := range name {
		valid := char >= 'A' && char <= 'Z' ||
			char >= '0' && char <= '9' ||
			char == '_' || char == '-' || char == '.'
		if !valid || index == 0 && (char < 'A' || char > 'Z') {
			return false
		}
	}
	return true
}

func cloneInstructionDefinition(in InstructionDefinition) InstructionDefinition {
	out := in
	out.Aliases = append([]string(nil), in.Aliases...)
	out.Arguments = append([]ArgumentDefinition(nil), in.Arguments...)
	out.Flags = make([]FlagDefinition, len(in.Flags))
	for index := range in.Flags {
		out.Flags[index] = cloneFlagDefinition(in.Flags[index])
	}
	return out
}

func normalizeInstructionDefinition(definition InstructionDefinition) (InstructionDefinition, error) {
	definition = cloneInstructionDefinition(definition)
	definition.Name = NormalizeInstructionName(definition.Name)
	if !validInstructionName(definition.Name) {
		return InstructionDefinition{}, fmt.Errorf("%w: invalid instruction name %q", ErrInvalidDefinition, definition.Name)
	}
	if definition.Kind > KindCustom {
		return InstructionDefinition{}, fmt.Errorf("%w: invalid instruction kind %d", ErrInvalidDefinition, definition.Kind)
	}
	if definition.Class > InstructionStageInitializer {
		return InstructionDefinition{}, fmt.Errorf("%w: invalid class for %s", ErrInvalidDefinition, definition.Name)
	}
	if definition.HistoryMode > HistorySkip {
		return InstructionDefinition{}, fmt.Errorf("%w: invalid history mode for %s", ErrInvalidDefinition, definition.Name)
	}
	if definition.Class == InstructionStageInitializer && definition.OnBuildable {
		return InstructionDefinition{}, fmt.Errorf("%w: stage initializer %s cannot be ONBUILD-able", ErrInvalidDefinition, definition.Name)
	}

	names := map[string]struct{}{definition.Name: {}}
	for index, alias := range definition.Aliases {
		alias = NormalizeInstructionName(alias)
		if !validInstructionName(alias) {
			return InstructionDefinition{}, fmt.Errorf("%w: invalid alias %q for %s", ErrInvalidDefinition, alias, definition.Name)
		}
		if _, exists := names[alias]; exists {
			return InstructionDefinition{}, fmt.Errorf("%w: duplicate instruction name or alias %q", ErrInvalidDefinition, alias)
		}
		names[alias] = struct{}{}
		definition.Aliases[index] = alias
	}

	flagNames := make(map[string]string)
	for index := range definition.Flags {
		flag := cloneFlagDefinition(definition.Flags[index])
		flag.Name = NormalizeFlagName(flag.Name)
		if err := validateFlagDefinition(flag); err != nil {
			return InstructionDefinition{}, fmt.Errorf("instruction %s: %w", definition.Name, err)
		}
		for aliasIndex, alias := range flag.Aliases {
			flag.Aliases[aliasIndex] = NormalizeFlagName(alias)
		}
		identifiers := append([]string{flag.Name}, flag.Aliases...)
		for _, identifier := range identifiers {
			identifier = NormalizeFlagName(identifier)
			if previous, exists := flagNames[identifier]; exists {
				return InstructionDefinition{}, fmt.Errorf(
					"%w: flag --%s conflicts with --%s on %s",
					ErrInvalidDefinition, identifier, previous, definition.Name,
				)
			}
			flagNames[identifier] = flag.Name
		}
		definition.Flags[index] = flag
	}

	argumentNames := make(map[string]struct{}, len(definition.Arguments))
	variadicCount := 0
	optionalSeen := false
	for _, argument := range definition.Arguments {
		if strings.TrimSpace(argument.Name) == "" {
			return InstructionDefinition{}, fmt.Errorf("%w: unnamed argument on %s", ErrInvalidDefinition, definition.Name)
		}
		if _, exists := argumentNames[argument.Name]; exists {
			return InstructionDefinition{}, fmt.Errorf("%w: duplicate argument %q on %s", ErrInvalidDefinition, argument.Name, definition.Name)
		}
		argumentNames[argument.Name] = struct{}{}
		if argument.Variadic {
			variadicCount++
		}
		if argument.Required {
			if optionalSeen {
				return InstructionDefinition{}, fmt.Errorf("%w: required argument %q follows an optional argument on %s", ErrInvalidDefinition, argument.Name, definition.Name)
			}
		} else {
			optionalSeen = true
		}
	}
	if variadicCount > 1 {
		return InstructionDefinition{}, fmt.Errorf("%w: only one variadic argument is allowed on %s", ErrInvalidDefinition, definition.Name)
	}
	if variadicCount == 1 {
		for index, argument := range definition.Arguments {
			if !argument.Variadic {
				continue
			}
			for _, trailing := range definition.Arguments[index+1:] {
				if !trailing.Required {
					return InstructionDefinition{}, fmt.Errorf("%w: optional argument %q follows a variadic argument on %s", ErrInvalidDefinition, trailing.Name, definition.Name)
				}
			}
		}
	}

	return definition, nil
}

func (r *Registry) ensureMapsLocked() {
	if r.instructions == nil {
		r.instructions = make(map[string]InstructionDefinition)
	}
	if r.aliases == nil {
		r.aliases = make(map[string]string)
	}
}

func (r *Registry) resolveLocked(name string) (string, bool) {
	name = NormalizeInstructionName(name)
	if _, exists := r.instructions[name]; exists {
		return name, true
	}
	canonical, exists := r.aliases[name]
	return canonical, exists
}

func (r *Registry) checkNamesLocked(definition InstructionDefinition, except string) error {
	names := append([]string{definition.Name}, definition.Aliases...)
	for _, name := range names {
		if existing, exists := r.instructions[name]; exists && name != except {
			return fmt.Errorf("%w: instruction %s", ErrAlreadyExists, existing.Name)
		}
		if existing, exists := r.aliases[name]; exists && existing != except {
			return fmt.Errorf("%w: alias %s", ErrAlreadyExists, name)
		}
	}
	return nil
}

func (r *Registry) rebuildAliasesLocked() {
	r.aliases = make(map[string]string)
	for name, definition := range r.instructions {
		for _, alias := range definition.Aliases {
			r.aliases[alias] = name
		}
	}
}

// CreateInstruction creates a new instruction definition.
func (r *Registry) CreateInstruction(definition InstructionDefinition) error {
	if r == nil {
		return fmt.Errorf("%w: nil registry", ErrInvalidDefinition)
	}
	normalized, err := normalizeInstructionDefinition(definition)
	if err != nil {
		return err
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	r.ensureMapsLocked()
	if err := r.checkNamesLocked(normalized, ""); err != nil {
		return err
	}
	r.generation++
	normalized.Revision = r.generation
	r.instructions[normalized.Name] = normalized
	r.rebuildAliasesLocked()
	return nil
}

// RegisterInstruction is an alias for CreateInstruction.
func (r *Registry) RegisterInstruction(definition InstructionDefinition) error {
	return r.CreateInstruction(definition)
}

// UpdateInstruction applies a transactional update to a detached definition.
// Concurrent changes are detected and reported as ErrRevisionConflict.
func (r *Registry) UpdateInstruction(name string, update func(*InstructionDefinition) error) error {
	if r == nil || update == nil {
		return fmt.Errorf("%w: registry and update callback are required", ErrInvalidDefinition)
	}
	r.mu.RLock()
	canonical, exists := r.resolveLocked(name)
	if !exists {
		r.mu.RUnlock()
		return fmt.Errorf("%w: instruction %q", ErrNotFound, name)
	}
	current := r.instructions[canonical]
	revision := current.Revision
	r.mu.RUnlock()

	current = cloneInstructionDefinition(current)
	candidate := cloneInstructionDefinition(current)
	if err := update(&candidate); err != nil {
		return err
	}
	normalized, err := normalizeInstructionDefinition(candidate)
	if err != nil {
		return err
	}

	r.mu.Lock()
	defer r.mu.Unlock()
	latest, exists := r.instructions[canonical]
	if !exists || latest.Revision != revision {
		return fmt.Errorf("%w: instruction %q", ErrRevisionConflict, name)
	}
	if err := r.checkNamesLocked(normalized, canonical); err != nil {
		return err
	}
	delete(r.instructions, canonical)
	r.generation++
	normalized.Revision = r.generation
	r.instructions[normalized.Name] = normalized
	r.rebuildAliasesLocked()
	return nil
}

// DeleteInstruction deletes a definition and all of its aliases.
func (r *Registry) DeleteInstruction(name string) error {
	if r == nil {
		return fmt.Errorf("%w: nil registry", ErrNotFound)
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	canonical, exists := r.resolveLocked(name)
	if !exists {
		return fmt.Errorf("%w: instruction %q", ErrNotFound, name)
	}
	delete(r.instructions, canonical)
	r.generation++
	r.rebuildAliasesLocked()
	return nil
}

// GetInstruction returns a detached definition snapshot.
func (r *Registry) GetInstruction(name string) (InstructionDefinition, error) {
	if r == nil {
		return InstructionDefinition{}, fmt.Errorf("%w: nil registry", ErrNotFound)
	}
	r.mu.RLock()
	canonical, exists := r.resolveLocked(name)
	if !exists {
		r.mu.RUnlock()
		return InstructionDefinition{}, fmt.Errorf("%w: instruction %q", ErrNotFound, name)
	}
	definition := r.instructions[canonical]
	r.mu.RUnlock()
	return cloneInstructionDefinition(definition), nil
}

// ListInstructions returns definitions sorted by canonical name.
func (r *Registry) ListInstructions() []InstructionDefinition {
	if r == nil {
		return nil
	}
	r.mu.RLock()
	names := make([]string, 0, len(r.instructions))
	definitions := make(map[string]InstructionDefinition, len(r.instructions))
	for name, definition := range r.instructions {
		names = append(names, name)
		definitions[name] = definition
	}
	r.mu.RUnlock()

	sort.Strings(names)
	result := make([]InstructionDefinition, 0, len(names))
	for _, name := range names {
		result = append(result, cloneInstructionDefinition(definitions[name]))
	}
	return result
}

// CreateFlag adds a new flag to an existing instruction.
func (r *Registry) CreateFlag(instructionName string, flag FlagDefinition) error {
	return r.UpdateInstruction(instructionName, func(definition *InstructionDefinition) error {
		for _, current := range definition.Flags {
			if flagIdentifierMatches(current, flag.Name) {
				return fmt.Errorf("%w: flag --%s", ErrAlreadyExists, flag.Name)
			}
			for _, alias := range flag.Aliases {
				if flagIdentifierMatches(current, alias) {
					return fmt.Errorf("%w: flag alias --%s", ErrAlreadyExists, alias)
				}
			}
		}
		definition.Flags = append(definition.Flags, cloneFlagDefinition(flag))
		return nil
	})
}

// RegisterFlag is an alias for CreateFlag.
func (r *Registry) RegisterFlag(instructionName string, flag FlagDefinition) error {
	return r.CreateFlag(instructionName, flag)
}

// UpdateFlag transactionally updates a flag definition. The name or aliases
// may be changed as part of the update.
func (r *Registry) UpdateFlag(instructionName, flagName string, update func(*FlagDefinition) error) error {
	if update == nil {
		return fmt.Errorf("%w: update callback is required", ErrInvalidDefinition)
	}
	return r.UpdateInstruction(instructionName, func(definition *InstructionDefinition) error {
		index := findFlagIndex(definition.Flags, flagName)
		if index < 0 {
			return fmt.Errorf("%w: flag --%s on %s", ErrNotFound, flagName, definition.Name)
		}
		flag := cloneFlagDefinition(definition.Flags[index])
		if err := update(&flag); err != nil {
			return err
		}
		definition.Flags[index] = flag
		return nil
	})
}

// DeleteFlag removes one flag and its aliases.
func (r *Registry) DeleteFlag(instructionName, flagName string) error {
	return r.UpdateInstruction(instructionName, func(definition *InstructionDefinition) error {
		index := findFlagIndex(definition.Flags, flagName)
		if index < 0 {
			return fmt.Errorf("%w: flag --%s on %s", ErrNotFound, flagName, definition.Name)
		}
		definition.Flags = append(definition.Flags[:index], definition.Flags[index+1:]...)
		return nil
	})
}

// GetFlag returns a detached flag definition.
func (r *Registry) GetFlag(instructionName, flagName string) (FlagDefinition, error) {
	definition, err := r.GetInstruction(instructionName)
	if err != nil {
		return FlagDefinition{}, err
	}
	index := findFlagIndex(definition.Flags, flagName)
	if index < 0 {
		return FlagDefinition{}, fmt.Errorf("%w: flag --%s on %s", ErrNotFound, flagName, definition.Name)
	}
	return cloneFlagDefinition(definition.Flags[index]), nil
}

// ListFlags returns an instruction's flags sorted by canonical flag name.
func (r *Registry) ListFlags(instructionName string) ([]FlagDefinition, error) {
	definition, err := r.GetInstruction(instructionName)
	if err != nil {
		return nil, err
	}
	sort.Slice(definition.Flags, func(i, j int) bool {
		return definition.Flags[i].Name < definition.Flags[j].Name
	})
	return definition.Flags, nil
}

func findFlagIndex(flags []FlagDefinition, name string) int {
	name = NormalizeFlagName(name)
	for index, flag := range flags {
		if flagIdentifierMatches(flag, name) {
			return index
		}
	}
	return -1
}

func flagIdentifierMatches(flag FlagDefinition, name string) bool {
	name = NormalizeFlagName(name)
	if NormalizeFlagName(flag.Name) == name {
		return true
	}
	for _, alias := range flag.Aliases {
		if NormalizeFlagName(alias) == name {
			return true
		}
	}
	return false
}

	return definition, nil
}

func (r *Registry) ensureMapsLocked() {
	if r.instructions == nil {
		r.instructions = make(map[string]InstructionDefinition)
	}
	if r.aliases == nil {
		r.aliases = make(map[string]string)
	}
}

func (r *Registry) resolveLocked(name string) (string, bool) {
	name = NormalizeInstructionName(name)
	if _, exists := r.instructions[name]; exists {
		return name, true
	}
	canonical, exists := r.aliases[name]
	return canonical, exists
}

func (r *Registry) checkNamesLocked(definition InstructionDefinition, except string) error {
	names := append([]string{definition.Name}, definition.Aliases...)
	for _, name := range names {
		if existing, exists := r.instructions[name]; exists && name != except {
			return fmt.Errorf("%w: instruction %s", ErrAlreadyExists, existing.Name)
		}
		if existing, exists := r.aliases[name]; exists && existing != except {
			return fmt.Errorf("%w: alias %s", ErrAlreadyExists, name)
		}
	}
	return nil
}

func (r *Registry) rebuildAliasesLocked() {
	r.aliases = make(map[string]string)
	for name, definition := range r.instructions {
		for _, alias := range definition.Aliases {
			r.aliases[alias] = name
		}
	}
}

// CreateInstruction creates a new instruction definition.
func (r *Registry) CreateInstruction(definition InstructionDefinition) error {
	if r == nil {
		return fmt.Errorf("%w: nil registry", ErrInvalidDefinition)
	}
	normalized, err := normalizeInstructionDefinition(definition)
	if err != nil {
		return err
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	r.ensureMapsLocked()
	if err := r.checkNamesLocked(normalized, ""); err != nil {
		return err
	}
	r.generation++
	normalized.Revision = r.generation
	r.instructions[normalized.Name] = normalized
	r.rebuildAliasesLocked()
	return nil
}

// RegisterInstruction is an alias for CreateInstruction.
func (r *Registry) RegisterInstruction(definition InstructionDefinition) error {
	return r.CreateInstruction(definition)
}

// UpdateInstruction applies a transactional update to a detached definition.
// Concurrent changes are detected and reported as ErrRevisionConflict.
func (r *Registry) UpdateInstruction(name string, update func(*InstructionDefinition) error) error {
	if r == nil || update == nil {
		return fmt.Errorf("%w: registry and update callback are required", ErrInvalidDefinition)
	}
	r.mu.RLock()
	canonical, exists := r.resolveLocked(name)
	if !exists {
		r.mu.RUnlock()
		return fmt.Errorf("%w: instruction %q", ErrNotFound, name)
	}
	current := cloneInstructionDefinition(r.instructions[canonical])
	revision := current.Revision
	r.mu.RUnlock()

	candidate := cloneInstructionDefinition(current)
	if err := update(&candidate); err != nil {
		return err
	}
	normalized, err := normalizeInstructionDefinition(candidate)
	if err != nil {
		return err
	}

	r.mu.Lock()
	defer r.mu.Unlock()
	latest, exists := r.instructions[canonical]
	if !exists || latest.Revision != revision {
		return fmt.Errorf("%w: instruction %q", ErrRevisionConflict, name)
	}
	if err := r.checkNamesLocked(normalized, canonical); err != nil {
		return err
	}
	delete(r.instructions, canonical)
	r.generation++
	normalized.Revision = r.generation
	r.instructions[normalized.Name] = normalized
	r.rebuildAliasesLocked()
	return nil
}

// DeleteInstruction deletes a definition and all of its aliases.
func (r *Registry) DeleteInstruction(name string) error {
	if r == nil {
		return fmt.Errorf("%w: nil registry", ErrNotFound)
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	canonical, exists := r.resolveLocked(name)
	if !exists {
		return fmt.Errorf("%w: instruction %q", ErrNotFound, name)
	}
	delete(r.instructions, canonical)
	r.generation++
	r.rebuildAliasesLocked()
	return nil
}

// GetInstruction returns a detached definition snapshot.
func (r *Registry) GetInstruction(name string) (InstructionDefinition, error) {
	if r == nil {
		return InstructionDefinition{}, fmt.Errorf("%w: nil registry", ErrNotFound)
	}
	r.mu.RLock()
	defer r.mu.RUnlock()
	canonical, exists := r.resolveLocked(name)
	if !exists {
		return InstructionDefinition{}, fmt.Errorf("%w: instruction %q", ErrNotFound, name)
	}
	return cloneInstructionDefinition(r.instructions[canonical]), nil
}

// ListInstructions returns definitions sorted by canonical name.
func (r *Registry) ListInstructions() []InstructionDefinition {
	if r == nil {
		return nil
	}
	r.mu.RLock()
	defer r.mu.RUnlock()
	names := make([]string, 0, len(r.instructions))
	for name := range r.instructions {
		names = append(names, name)
	}
	sort.Strings(names)
	result := make([]InstructionDefinition, 0, len(names))
	for _, name := range names {
		result = append(result, cloneInstructionDefinition(r.instructions[name]))
	}
	return result
}

// CreateFlag adds a new flag to an existing instruction.
func (r *Registry) CreateFlag(instructionName string, flag FlagDefinition) error {
	return r.UpdateInstruction(instructionName, func(definition *InstructionDefinition) error {
		for _, current := range definition.Flags {
			if flagIdentifierMatches(current, flag.Name) {
				return fmt.Errorf("%w: flag --%s", ErrAlreadyExists, flag.Name)
			}
			for _, alias := range flag.Aliases {
				if flagIdentifierMatches(current, alias) {
					return fmt.Errorf("%w: flag alias --%s", ErrAlreadyExists, alias)
				}
			}
		}
		definition.Flags = append(definition.Flags, cloneFlagDefinition(flag))
		return nil
	})
}

// RegisterFlag is an alias for CreateFlag.
func (r *Registry) RegisterFlag(instructionName string, flag FlagDefinition) error {
	return r.CreateFlag(instructionName, flag)
}

// UpdateFlag transactionally updates a flag definition. The name or aliases
// may be changed as part of the update.
func (r *Registry) UpdateFlag(instructionName, flagName string, update func(*FlagDefinition) error) error {
	if update == nil {
		return fmt.Errorf("%w: update callback is required", ErrInvalidDefinition)
	}
	return r.UpdateInstruction(instructionName, func(definition *InstructionDefinition) error {
		index := findFlagIndex(definition.Flags, flagName)
		if index < 0 {
			return fmt.Errorf("%w: flag --%s on %s", ErrNotFound, flagName, definition.Name)
		}
		flag := cloneFlagDefinition(definition.Flags[index])
		if err := update(&flag); err != nil {
			return err
		}
		definition.Flags[index] = flag
		return nil
	})
}

// DeleteFlag removes one flag and its aliases.
func (r *Registry) DeleteFlag(instructionName, flagName string) error {
	return r.UpdateInstruction(instructionName, func(definition *InstructionDefinition) error {
		index := findFlagIndex(definition.Flags, flagName)
		if index < 0 {
			return fmt.Errorf("%w: flag --%s on %s", ErrNotFound, flagName, definition.Name)
		}
		definition.Flags = append(definition.Flags[:index], definition.Flags[index+1:]...)
		return nil
	})
}

// GetFlag returns a detached flag definition.
func (r *Registry) GetFlag(instructionName, flagName string) (FlagDefinition, error) {
	definition, err := r.GetInstruction(instructionName)
	if err != nil {
		return FlagDefinition{}, err
	}
	index := findFlagIndex(definition.Flags, flagName)
	if index < 0 {
		return FlagDefinition{}, fmt.Errorf("%w: flag --%s on %s", ErrNotFound, flagName, definition.Name)
	}
	return cloneFlagDefinition(definition.Flags[index]), nil
}

// ListFlags returns an instruction's flags sorted by canonical flag name.
func (r *Registry) ListFlags(instructionName string) ([]FlagDefinition, error) {
	definition, err := r.GetInstruction(instructionName)
	if err != nil {
		return nil, err
	}
	sort.Slice(definition.Flags, func(i, j int) bool {
		return definition.Flags[i].Name < definition.Flags[j].Name
	})
	return definition.Flags, nil
}

func findFlagIndex(flags []FlagDefinition, name string) int {
	name = NormalizeFlagName(name)
	for index, flag := range flags {
		if flagIdentifierMatches(flag, name) {
			return index
		}
	}
	return -1
}

func flagIdentifierMatches(flag FlagDefinition, name string) bool {
	name = NormalizeFlagName(name)
	if NormalizeFlagName(flag.Name) == name {
		return true
	}
	for _, alias := range flag.Aliases {
		if NormalizeFlagName(alias) == name {
			return true
		}
	}
	return false
}
