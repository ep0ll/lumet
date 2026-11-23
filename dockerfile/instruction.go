package dockerfile

import (
	"errors"
	"fmt"
	"sync"
)

// Instruction is an instance of an instruction definition. It owns an
// immutable definition snapshot plus mutable, synchronized argument/flag
// values. Mutating the Registry never changes an existing instruction.
type Instruction struct {
	mu          sync.RWMutex
	definition  InstructionDefinition
	arguments   []string
	flags       map[string]any
	flagAliases map[string]string
	extensions  map[string]any
	payload     any
	buildKit    BuildKitOptions
	history     HistoryFunc
	historyMode HistoryMode
	version     uint64
}

// NewInstruction constructs an instruction directly from a definition. Most
// callers should use Registry.New or Registry.NewWithOptions.
func NewInstruction(definition InstructionDefinition, arguments []string, options ...Option) (*Instruction, error) {
	definition, err := normalizeInstructionDefinition(definition)
	if err != nil {
		return nil, err
	}
	defaults, err := newFlagValues(definition.Flags)
	if err != nil {
		return nil, fmt.Errorf("dockerfile: initialize %s defaults: %w", definition.Name, err)
	}
	config := InstructionOptions{
		extensions: make(map[string]any),
	}
	for _, option := range options {
		if option == nil {
			continue
		}
		if err := option(&config); err != nil {
			return nil, fmt.Errorf("dockerfile: configure %s: %w", definition.Name, err)
		}
	}

	instruction := &Instruction{
		definition:  definition,
		arguments:   append([]string(nil), arguments...),
		flags:       defaults.All(),
		flagAliases: cloneStringMap(defaults.aliases),
		extensions:  cloneValues(config.extensions),
		payload:     clonePayload(config.payload),
		buildKit:    cloneBuildKitOptions(config.buildKit),
		history:     definition.History,
		historyMode: definition.HistoryMode,
	}
	if config.history != nil {
		instruction.history = config.history
	}
	if config.historyMode != nil {
		instruction.historyMode = *config.historyMode
	}
	seenFlags := make(map[string]struct{}, len(config.flags))
	for name, value := range config.flags {
		index := findFlagIndex(definition.Flags, name)
		if index < 0 {
			return nil, fmt.Errorf("%w: flag --%s is not defined for %s", ErrInvalidInstruction, name, definition.Name)
		}
		canonical := NormalizeFlagName(definition.Flags[index].Name)
		if _, exists := seenFlags[canonical]; exists {
			return nil, fmt.Errorf("%w: flag --%s was configured more than once", ErrInvalidInstruction, canonical)
		}
		seenFlags[canonical] = struct{}{}
		if err := instruction.SetFlag(name, value); err != nil {
			return nil, err
		}
	}
	return instruction, nil
}

// New constructs an instruction from the current definition snapshot.
func (r *Registry) New(name string, arguments ...string) (*Instruction, error) {
	return r.NewWithOptions(name, arguments)
}

// NewWithOptions constructs an instruction and applies generic functional
// options. The options contract is shared by every current and future
// instruction; adding a new instruction flag does not change this signature.
func (r *Registry) NewWithOptions(name string, arguments []string, options ...Option) (*Instruction, error) {
	definition, err := r.GetInstruction(name)
	if err != nil {
		return nil, err
	}
	return NewInstruction(definition, arguments, options...)
}

// Builtin constructs an instruction from the standard Dockerfile definitions
// without requiring callers to create and manage a Registry.
func Builtin(name string, arguments []string, options ...Option) (*Instruction, error) {
	name = NormalizeInstructionName(name)
	for _, definition := range BuiltinInstructions() {
		if definition.Name == name {
			return NewInstruction(definition, arguments, options...)
		}
		for _, alias := range definition.Aliases {
			if alias == name {
				return NewInstruction(definition, arguments, options...)
			}
		}
	}
	return nil, fmt.Errorf("%w: built-in instruction %q", ErrNotFound, name)
}

// NewOnBuild creates an ONBUILD instruction containing a validated snapshot
// of its trigger. Trigger definitions must explicitly permit ONBUILD use.
func (r *Registry) NewOnBuild(trigger *Instruction, options ...Option) (*Instruction, error) {
	if trigger == nil {
		return nil, fmt.Errorf("%w: ONBUILD trigger is nil", ErrInvalidInstruction)
	}
	definition := trigger.Definition()
	if !definition.OnBuildable || definition.Class == InstructionStageInitializer || definition.Name == "ONBUILD" {
		return nil, fmt.Errorf("%w: %s cannot be used as an ONBUILD trigger", ErrInvalidInstruction, definition.Name)
	}
	if err := trigger.Validate(); err != nil {
		return nil, fmt.Errorf("invalid ONBUILD trigger %s: %w", definition.Name, err)
	}
	onBuildOptions := make([]Option, 0, len(options)+1)
	onBuildOptions = append(onBuildOptions, options...)
	onBuildOptions = append(onBuildOptions, WithPayload(trigger.Clone()))
	onBuild, err := r.NewWithOptions("ONBUILD", nil, onBuildOptions...)
	if err != nil {
		return nil, err
	}
	if err := onBuild.Validate(); err != nil {
		return nil, err
	}
	return onBuild, nil
}

// Definition returns a detached copy of the definition snapshot.
func (i *Instruction) Definition() InstructionDefinition {
	if i == nil {
		return InstructionDefinition{}
	}
	i.mu.RLock()
	definition := i.definition
	i.mu.RUnlock()
	return cloneInstructionDefinition(definition)
}

// Name returns the canonical instruction name.
func (i *Instruction) Name() string {
	if i == nil {
		return ""
	}
	i.mu.RLock()
	defer i.mu.RUnlock()
	return i.definition.Name
}

// Kind returns the built-in or custom instruction kind.
func (i *Instruction) Kind() Kind {
	if i == nil {
		return KindCustom
	}
	i.mu.RLock()
	defer i.mu.RUnlock()
	return i.definition.Kind
}

// Class returns the instruction placement class.
func (i *Instruction) Class() InstructionClass {
	if i == nil {
		return InstructionStandard
	}
	i.mu.RLock()
	defer i.mu.RUnlock()
	return i.definition.Class
}

// Arguments returns a copy of the positional arguments.
func (i *Instruction) Arguments() []string {
	if i == nil {
		return nil
	}
	i.mu.RLock()
	defer i.mu.RUnlock()
	return append([]string(nil), i.arguments...)
}

// PositionalArguments maps positional values to declared argument names. A
// variadic argument consumes all values except those required by later args.
func (i *Instruction) PositionalArguments() map[string][]string {
	if i == nil {
		return nil
	}
	i.mu.RLock()
	defer i.mu.RUnlock()

	result := make(map[string][]string, len(i.definition.Arguments))
	position := 0
	for index, definition := range i.definition.Arguments {
		remaining := len(i.arguments) - position
		if remaining <= 0 {
			result[definition.Name] = nil
			continue
		}
		if definition.Variadic {
			requiredAfter := 0
			for _, later := range i.definition.Arguments[index+1:] {
				if later.Required {
					requiredAfter++
				}
			}
			count := remaining - requiredAfter
			if count < 0 {
				count = 0
			}
			result[definition.Name] = append([]string(nil), i.arguments[position:position+count]...)
			position += count
			continue
		}
		result[definition.Name] = []string{i.arguments[position]}
		position++
	}
	return result
}

// SetArguments atomically replaces positional arguments after validation.
func (i *Instruction) SetArguments(arguments ...string) error {
	if i == nil {
		return fmt.Errorf("%w: nil instruction", ErrInvalidInstruction)
	}
	i.mu.Lock()
	defer i.mu.Unlock()
	if err := validateArguments(i.definition, arguments); err != nil {
		return err
	}
	i.arguments = append([]string(nil), arguments...)
	i.version++
	return nil
}

// SetFlag sets a flag value by canonical name or alias.
func (i *Instruction) SetFlag(name string, raw any) error {
	if i == nil {
		return fmt.Errorf("%w: nil instruction", ErrInvalidInstruction)
	}

	i.mu.RLock()
	index := findFlagIndex(i.definition.Flags, name)
	if index < 0 {
		instructionName := i.definition.Name
		i.mu.RUnlock()
		return fmt.Errorf("%w: flag --%s is not defined for %s", ErrInvalidInstruction, name, instructionName)
	}
	definition := i.definition.Flags[index]
	instructionName := i.definition.Name
	version := i.version
	i.mu.RUnlock()
	definition = cloneFlagDefinition(definition)

	value, err := parseFlagValue(definition, raw)
	if err != nil {
		return fmt.Errorf("%s --%s: %w", instructionName, definition.Name, err)
	}
	if err := validateFlagValue(definition, value); err != nil {
		return fmt.Errorf("%s --%s: %w", instructionName, definition.Name, err)
	}
	value = cloneFlagValue(value)

	i.mu.Lock()
	defer i.mu.Unlock()
	if i.version != version {
		return ErrInstructionConflict
	}
	i.flags[definition.Name] = value
	i.version++
	return nil
}


// AppendFlag appends one value to a repeatable string-slice flag.
func (i *Instruction) AppendFlag(name, raw string) error {
	if i == nil {
		return fmt.Errorf("%w: nil instruction", ErrInvalidInstruction)
	}

	i.mu.RLock()
	index := findFlagIndex(i.definition.Flags, name)
	if index < 0 {
		instructionName := i.definition.Name
		i.mu.RUnlock()
		return fmt.Errorf("%w: flag --%s is not defined for %s", ErrInvalidInstruction, name, instructionName)
	}
	definition := i.definition.Flags[index]
	instructionName := i.definition.Name
	version := i.version
	current, _ := i.flags[definition.Name].([]string)
	current = append([]string(nil), current...)
	i.mu.RUnlock()
	definition = cloneFlagDefinition(definition)

	if !definition.Repeatable || definition.Type != FlagStringSlice {
		return fmt.Errorf("%w: --%s is not repeatable", ErrInvalidInstruction, definition.Name)
	}
	next := append(current, raw)
	value, err := parseFlagValue(definition, next)
	if err != nil {
		return fmt.Errorf("%s --%s: %w", instructionName, definition.Name, err)
	}
	if err := validateFlagValue(definition, value); err != nil {
		return fmt.Errorf("%s --%s: %w", instructionName, definition.Name, err)
	}
	value = cloneFlagValue(value)

	i.mu.Lock()
	defer i.mu.Unlock()
	if i.version != version {
		return ErrInstructionConflict
	}
	i.flags[definition.Name] = value
	i.version++
	return nil
}


// DeleteFlagValue clears an instance's explicit value. It does not change the
// schema or remove the flag; ResetFlag restores its default.
func (i *Instruction) DeleteFlagValue(name string) error {
	if i == nil {
		return fmt.Errorf("%w: nil instruction", ErrInvalidInstruction)
	}
	i.mu.Lock()
	defer i.mu.Unlock()
	index := findFlagIndex(i.definition.Flags, name)
	if index < 0 {
		return fmt.Errorf("%w: flag --%s is not defined for %s", ErrInvalidInstruction, name, i.definition.Name)
	}
	delete(i.flags, i.definition.Flags[index].Name)
	i.version++
	return nil
}

// ResetFlag resets a flag to its declared default, or clears it when no default
// is declared.
func (i *Instruction) ResetFlag(name string) error {
	if i == nil {
		return fmt.Errorf("%w: nil instruction", ErrInvalidInstruction)
	}

	i.mu.RLock()
	index := findFlagIndex(i.definition.Flags, name)
	if index < 0 {
		instructionName := i.definition.Name
		i.mu.RUnlock()
		return fmt.Errorf("%w: flag --%s is not defined for %s", ErrInvalidInstruction, name, instructionName)
	}
	definition := i.definition.Flags[index]
	version := i.version
	i.mu.RUnlock()
	definition = cloneFlagDefinition(definition)

	if definition.Default == nil {
		i.mu.Lock()
		defer i.mu.Unlock()
		if i.version != version {
			return ErrInstructionConflict
		}
		delete(i.flags, definition.Name)
		i.version++
		return nil
	}

	value, err := parseFlagValue(definition, definition.Default)
	if err != nil {
		return err
	}
	if err := validateFlagValue(definition, value); err != nil {
		return err
	}
	value = cloneFlagValue(value)

	i.mu.Lock()
	defer i.mu.Unlock()
	if i.version != version {
		return ErrInstructionConflict
	}
	i.flags[definition.Name] = value
	i.version++
	return nil
}


// ResetFlags resets every flag to its definition default.
func (i *Instruction) ResetFlags() error {
	if i == nil {
		return fmt.Errorf("%w: nil instruction", ErrInvalidInstruction)
	}
	i.mu.RLock()
	definitions := append([]FlagDefinition(nil), i.definition.Flags...)
	version := i.version
	i.mu.RUnlock()

	values, err := newFlagValues(definitions)
	if err != nil {
		return err
	}
	resolved := values.All()

	i.mu.Lock()
	defer i.mu.Unlock()
	if i.version != version {
		return ErrInstructionConflict
	}
	i.flags = resolved
	i.version++
	return nil
}


// Flag returns one flag value and whether it is set.
func (i *Instruction) Flag(name string) (any, bool) {
	if i == nil {
		return nil, false
	}
	i.mu.RLock()
	index := findFlagIndex(i.definition.Flags, name)
	if index < 0 {
		i.mu.RUnlock()
		return nil, false
	}
	value, exists := i.flags[i.definition.Flags[index].Name]
	i.mu.RUnlock()
	return cloneFlagValue(value), exists
}


// Flags returns a read-only snapshot of all resolved flag values.
func (i *Instruction) Flags() FlagValues {
	if i == nil {
		return FlagValues{}
	}
	i.mu.RLock()
	values := shallowCopyValues(i.flags)
	aliases := cloneStringMap(i.flagAliases)
	i.mu.RUnlock()
	return FlagValues{values: cloneValues(values), aliases: aliases}
}


// SetExtension attaches application-specific structured data to the instance.
func (i *Instruction) SetExtension(name string, value any) error {
	if i == nil || name == "" {
		return fmt.Errorf("%w: instruction and extension name are required", ErrInvalidInstruction)
	}
	value = cloneFlagValue(value)
	i.mu.Lock()
	defer i.mu.Unlock()
	if i.extensions == nil {
		i.extensions = make(map[string]any)
	}
	i.extensions[name] = value
	i.version++
	return nil
}


// Extension retrieves an application-specific value.
func (i *Instruction) Extension(name string) (any, bool) {
	if i == nil {
		return nil, false
	}
	i.mu.RLock()
	value, exists := i.extensions[name]
	i.mu.RUnlock()
	return cloneFlagValue(value), exists
}


// Extensions returns a detached map of application-specific values.
func (i *Instruction) Extensions() map[string]any {
	if i == nil {
		return nil
	}
	i.mu.RLock()
	values := shallowCopyValues(i.extensions)
	i.mu.RUnlock()
	return cloneValues(values)
}


// SetPayload replaces the instruction-specific payload.
func (i *Instruction) SetPayload(payload any) error {
	if i == nil {
		return fmt.Errorf("%w: nil instruction", ErrInvalidInstruction)
	}
	payload = clonePayload(payload)
	i.mu.Lock()
	defer i.mu.Unlock()
	i.payload = payload
	i.version++
	return nil
}


// Payload returns the instruction payload.
func (i *Instruction) Payload() any {
	if i == nil {
		return nil
	}
	i.mu.RLock()
	payload := i.payload
	i.mu.RUnlock()
	return clonePayload(payload)
}


// UseBuildKitOptions appends native BuildKit options to the instruction.
func (i *Instruction) UseBuildKitOptions(options ...BuildKitOption) error {
	if i == nil {
		return fmt.Errorf("%w: nil instruction", ErrInvalidInstruction)
	}
	i.mu.RLock()
	buildKit := cloneBuildKitOptions(i.buildKit)
	version := i.version
	i.mu.RUnlock()

	for _, option := range options {
		if option != nil {
			option(&buildKit)
		}
	}
	buildKit = cloneBuildKitOptions(buildKit)

	i.mu.Lock()
	defer i.mu.Unlock()
	if i.version != version {
		return ErrInstructionConflict
	}
	i.buildKit = buildKit
	i.version++
	return nil
}


// BuildKitOptions returns a defensive slice snapshot of native LLB options.
func (i *Instruction) BuildKitOptions() BuildKitOptions {
	if i == nil {
		return BuildKitOptions{}
	}
	i.mu.RLock()
	defer i.mu.RUnlock()
	return cloneBuildKitOptions(i.buildKit)
}

// Validate checks arguments, required flags, flag value constraints, and the
// optional instruction-level validator.
func (i *Instruction) Validate() error {
	if i == nil {
		return fmt.Errorf("%w: nil instruction", ErrInvalidInstruction)
	}
	i.mu.RLock()
	rawDefinition := i.definition
	arguments := append([]string(nil), i.arguments...)
	rawFlags := shallowCopyValues(i.flags)
	aliases := cloneStringMap(i.flagAliases)
	payload := i.payload
	i.mu.RUnlock()

	definition := cloneInstructionDefinition(rawDefinition)
	flags := FlagValues{values: cloneValues(rawFlags), aliases: aliases}
	payload = clonePayload(payload)

	if err := validateArguments(definition, arguments); err != nil {
		return err
	}
	for _, flag := range definition.Flags {
		value, exists := flags.Get(flag.Name)
		if flag.Required && !exists {
			return fmt.Errorf("%w: %s requires --%s", ErrInvalidInstruction, definition.Name, flag.Name)
		}
		if exists {
			if err := validateFlagValue(flag, value); err != nil {
				return fmt.Errorf("%s --%s: %w", definition.Name, flag.Name, err)
			}
		}
	}
	if definition.Validate != nil {
		if err := definition.Validate(arguments, flags, payload); err != nil {
			return fmt.Errorf("%s: %w", definition.Name, err)
		}
	}
	return nil
}


// Clone creates an independent instruction snapshot.
func (i *Instruction) Clone() *Instruction {
	if i == nil {
		return nil
	}
	i.mu.RLock()
	rawDefinition := i.definition
	arguments := append([]string(nil), i.arguments...)
	rawFlags := shallowCopyValues(i.flags)
	aliases := cloneStringMap(i.flagAliases)
	rawExtensions := shallowCopyValues(i.extensions)
	payload := i.payload
	buildKit := cloneBuildKitOptions(i.buildKit)
	history := i.history
	historyMode := i.historyMode
	version := i.version
	i.mu.RUnlock()

	return &Instruction{
		definition:  cloneInstructionDefinition(rawDefinition),
		arguments:   arguments,
		flags:       cloneValues(rawFlags),
		flagAliases: aliases,
		extensions:  cloneValues(rawExtensions),
		payload:     clonePayload(payload),
		buildKit:    buildKit,
		history:     history,
		historyMode: historyMode,
		version:     version,
	}
}


func (i *Instruction) historyConfig() (HistoryFunc, HistoryMode) {
	i.mu.RLock()
	defer i.mu.RUnlock()
	return i.history, i.historyMode
}

func validateArguments(definition InstructionDefinition, arguments []string) error {
	minimum := 0
	variadic := false
	for _, argument := range definition.Arguments {
		if argument.Required {
			minimum++
		}
		if argument.Variadic {
			variadic = true
		}
	}
	if len(arguments) < minimum {
		return fmt.Errorf("%w: %s requires at least %d argument(s)", ErrInvalidInstruction, definition.Name, minimum)
	}
	if !variadic && len(arguments) > len(definition.Arguments) {
		return fmt.Errorf("%w: %s accepts at most %d argument(s)", ErrInvalidInstruction, definition.Name, len(definition.Arguments))
	}
	return nil
}

func shallowCopyValues(values map[string]any) map[string]any {
	if values == nil {
		return nil
	}
	result := make(map[string]any, len(values))
	for key, value := range values {
		result[key] = value
	}
	return result
}

func cloneValues(values map[string]any) map[string]any {
	result := make(map[string]any, len(values))
	for key, value := range values {
		result[key] = cloneFlagValue(value)
	}
	return result
}

func cloneStringMap(values map[string]string) map[string]string {
	result := make(map[string]string, len(values))
	for key, value := range values {
		result[key] = value
	}
	return result
}

func clonePayload(payload any) any {
	switch value := payload.(type) {
	case Source:
		return value.Clone()
	case *Instruction:
		return value.Clone()
	case []byte:
		return append([]byte(nil), value...)
	case []string:
		return append([]string(nil), value...)
	case Command:
		return value.Clone()
	case map[string]string:
		result := make(map[string]string, len(value))
		for key, item := range value {
			result[key] = item
		}
		return result
	case map[string]any:
		return cloneValues(value)
	case FlagValueCloner:
		return value.CloneDockerfileFlagValue()
	default:
		return value
	}
}

// ErrInvalidInstruction reports invalid instruction instances or values.
var ErrInvalidInstruction = errors.New("dockerfile: invalid instruction")

// ErrInstructionConflict reports a concurrent mutation while an instruction
// update was being prepared outside its lock.
var ErrInstructionConflict = errors.New("dockerfile: instruction changed concurrently")
