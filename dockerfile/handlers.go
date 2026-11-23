package dockerfile

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"sync"

	ocispecs "github.com/opencontainers/image-spec/specs-go/v1"
)

// Call is the complete, immutable execution input for an instruction handler.
type Call struct {
	Stage       Stage
	Instruction *Instruction
	Definition  InstructionDefinition
	Arguments   []string
	Positionals map[string][]string
	Flags       FlagValues
	Extensions  map[string]any
	Payload     any
	LLB         BuildKitOptions
	Handlers    *Handlers
	_           struct{}
}

// Handler lowers one instruction into a new immutable build Stage.
type Handler func(ctx context.Context, call Call) (Stage, error)

// Interceptor wraps a Handler. Interceptors execute in the order in which
// they were registered.
type Interceptor func(ctx context.Context, call Call, next Handler) (Stage, error)

// HistoryEvent is passed to a per-instruction history callback.
type HistoryEvent struct {
	Instruction *Instruction
	Before      Stage
	After       Stage
	_           struct{}
}

// HistoryFunc creates the OCI history entry for one instruction.
type HistoryFunc func(ctx context.Context, event HistoryEvent) (ocispecs.History, error)

// Handlers is a concurrency-safe registry of instruction handlers and
// interceptors. New instructions do not require adding struct fields: register
// their implementation by name.
type Handlers struct {
	mu           sync.RWMutex
	handlers     map[string]Handler
	interceptors map[string]Interceptor
	order        []string
}

// NewHandlers returns an empty, ready-to-use handler registry.
func NewHandlers() *Handlers { return &Handlers{} }

// RegisterHandler creates a handler for an instruction name.
func (h *Handlers) RegisterHandler(name string, handler Handler) error {
	if h == nil || handler == nil {
		return fmt.Errorf("%w: handlers and handler are required", ErrInvalidDefinition)
	}
	name = NormalizeInstructionName(name)
	if !validInstructionName(name) {
		return fmt.Errorf("%w: invalid handler name %q", ErrInvalidDefinition, name)
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.handlers == nil {
		h.handlers = make(map[string]Handler)
	}
	if _, exists := h.handlers[name]; exists {
		return fmt.Errorf("%w: handler %s", ErrAlreadyExists, name)
	}
	h.handlers[name] = handler
	return nil
}

// UpdateHandler replaces an existing handler without changing the registry
// key. It is safe for concurrent execution.
func (h *Handlers) UpdateHandler(name string, handler Handler) error {
	if h == nil || handler == nil {
		return fmt.Errorf("%w: handlers and handler are required", ErrInvalidDefinition)
	}
	name = NormalizeInstructionName(name)
	h.mu.Lock()
	defer h.mu.Unlock()
	if _, exists := h.handlers[name]; !exists {
		return fmt.Errorf("%w: handler %s", ErrNotFound, name)
	}
	h.handlers[name] = handler
	return nil
}

// DeleteHandler removes a handler.
func (h *Handlers) DeleteHandler(name string) error {
	if h == nil {
		return fmt.Errorf("%w: nil handlers", ErrNotFound)
	}
	name = NormalizeInstructionName(name)
	h.mu.Lock()
	defer h.mu.Unlock()
	if _, exists := h.handlers[name]; !exists {
		return fmt.Errorf("%w: handler %s", ErrNotFound, name)
	}
	delete(h.handlers, name)
	return nil
}

// GetHandler returns a handler by canonical name.
func (h *Handlers) GetHandler(name string) (Handler, bool) {
	if h == nil {
		return nil, false
	}
	h.mu.RLock()
	defer h.mu.RUnlock()
	handler, exists := h.handlers[NormalizeInstructionName(name)]
	return handler, exists
}

// ListHandlers returns handler names in deterministic order.
func (h *Handlers) ListHandlers() []string {
	if h == nil {
		return nil
	}
	h.mu.RLock()
	defer h.mu.RUnlock()
	names := make([]string, 0, len(h.handlers))
	for name := range h.handlers {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

// RegisterInterceptor creates a named interceptor and appends it to the
// execution chain.
func (h *Handlers) RegisterInterceptor(name string, interceptor Interceptor) error {
	if h == nil || interceptor == nil || strings.TrimSpace(name) == "" {
		return fmt.Errorf("%w: handlers, name, and interceptor are required", ErrInvalidDefinition)
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.interceptors == nil {
		h.interceptors = make(map[string]Interceptor)
	}
	if _, exists := h.interceptors[name]; exists {
		return fmt.Errorf("%w: interceptor %s", ErrAlreadyExists, name)
	}
	h.interceptors[name] = interceptor
	h.order = append(h.order, name)
	return nil
}

// UpdateInterceptor replaces an interceptor without changing its position.
func (h *Handlers) UpdateInterceptor(name string, interceptor Interceptor) error {
	if h == nil || interceptor == nil {
		return fmt.Errorf("%w: handlers and interceptor are required", ErrInvalidDefinition)
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	if _, exists := h.interceptors[name]; !exists {
		return fmt.Errorf("%w: interceptor %s", ErrNotFound, name)
	}
	h.interceptors[name] = interceptor
	return nil
}

// DeleteInterceptor removes a named interceptor and its ordering entry.
func (h *Handlers) DeleteInterceptor(name string) error {
	if h == nil {
		return fmt.Errorf("%w: nil handlers", ErrNotFound)
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	if _, exists := h.interceptors[name]; !exists {
		return fmt.Errorf("%w: interceptor %s", ErrNotFound, name)
	}
	delete(h.interceptors, name)
	for index, current := range h.order {
		if current == name {
			h.order = append(h.order[:index], h.order[index+1:]...)
			break
		}
	}
	return nil
}

// GetInterceptor returns a named interceptor.
func (h *Handlers) GetInterceptor(name string) (Interceptor, bool) {
	if h == nil {
		return nil, false
	}
	h.mu.RLock()
	defer h.mu.RUnlock()
	interceptor, exists := h.interceptors[name]
	return interceptor, exists
}

// ListInterceptors returns interceptor names in execution order.
func (h *Handlers) ListInterceptors() []string {
	if h == nil {
		return nil
	}
	h.mu.RLock()
	defer h.mu.RUnlock()
	return append([]string(nil), h.order...)
}

// Apply validates and executes an instruction through the handler and
// interceptor registries.
func (h *Handlers) Apply(ctx context.Context, stage Stage, instruction *Instruction) (Stage, error) {
	if instruction == nil {
		return stage, fmt.Errorf("%w: nil instruction", ErrInvalidInstruction)
	}
	if h == nil {
		return stage, &ErrNoHandler{Name: instruction.Name()}
	}

	snapshot := instruction.Clone()
	if err := snapshot.Validate(); err != nil {
		return stage, err
	}
	definition := snapshot.Definition()
	handler, exists := h.lookupHandler(definition)
	if !exists {
		return stage, &ErrNoHandler{Name: definition.Name}
	}

	call := Call{
		Stage:       stage,
		Instruction: snapshot.Clone(),
		Definition:  definition,
		Arguments:   snapshot.Arguments(),
		Positionals: snapshot.PositionalArguments(),
		Flags:       snapshot.Flags(),
		Extensions:  snapshot.Extensions(),
		Payload:     snapshot.Payload(),
		LLB:         snapshot.BuildKitOptions(),
		Handlers:    h,
	}

	h.mu.RLock()
	interceptors := make([]Interceptor, 0, len(h.order))
	for _, name := range h.order {
		if interceptor := h.interceptors[name]; interceptor != nil {
			interceptors = append(interceptors, interceptor)
		}
	}
	h.mu.RUnlock()

	next := handler
	for index := len(interceptors) - 1; index >= 0; index-- {
		interceptor := interceptors[index]
		downstream := next
		next = func(ctx context.Context, call Call) (Stage, error) {
			return interceptor(ctx, call, downstream)
		}
	}
	result, err := next(ctx, call)
	if err != nil {
		return result, err
	}

	history, mode := snapshot.historyConfig()
	if mode == HistorySkip {
		return result, nil
	}
	if history != nil {
		entry, err := history(ctx, HistoryEvent{
			Instruction: snapshot.Clone(),
			Before:      stage,
			After:       result,
		})
		if err != nil {
			return result, err
		}
		result.Image.History = append(result.Image.History, entry)
		return result, nil
	}
	switch mode {
	case HistoryLayer, HistoryEmptyLayer:
		createdBy := definition.Name
		if len(call.Arguments) != 0 {
			createdBy += " " + strings.Join(call.Arguments, " ")
		}
		result.Image.History = append(result.Image.History, ocispecs.History{
			CreatedBy:  createdBy,
			EmptyLayer: mode == HistoryEmptyLayer,
		})
	}
	return result, nil
}

func (h *Handlers) lookupHandler(definition InstructionDefinition) (Handler, bool) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	if handler, exists := h.handlers[definition.Name]; exists {
		return handler, true
	}
	for _, alias := range definition.Aliases {
		if handler, exists := h.handlers[alias]; exists {
			return handler, true
		}
	}
	return nil, false
}

// Apply executes this instruction using the supplied handler registry.
func (i *Instruction) Apply(ctx context.Context, stage Stage, handlers *Handlers) (Stage, error) {
	if handlers == nil {
		return stage, &ErrNoHandler{Name: i.Name()}
	}
	return handlers.Apply(ctx, stage, i)
}

// Compose executes a stage initializer followed by regular instructions.
func Compose(ctx context.Context, handlers *Handlers, first *Instruction, body ...*Instruction) (Stage, error) {
	if first == nil {
		return Stage{}, fmt.Errorf("%w: stage initializer is nil", ErrInvalidInstruction)
	}
	if first.Class() != InstructionStageInitializer {
		return Stage{}, fmt.Errorf("%w: first instruction %s is not a stage initializer", ErrInvalidInstruction, first.Name())
	}
	if handlers == nil {
		return Stage{}, &ErrNoHandler{Name: first.Name()}
	}
	stage, err := handlers.Apply(ctx, Stage{}, first)
	if err != nil {
		return stage, err
	}
	for _, instruction := range body {
		if instruction == nil {
			return stage, fmt.Errorf("%w: nil instruction", ErrInvalidInstruction)
		}
		if instruction.Class() == InstructionStageInitializer {
			return stage, fmt.Errorf("%w: stage initializer %s cannot appear in a stage body", ErrInvalidInstruction, instruction.Name())
		}
		stage, err = handlers.Apply(ctx, stage, instruction)
		if err != nil {
			return stage, err
		}
	}
	return stage, nil
}

// ErrNoHandler reports an instruction with no registered implementation.
type ErrNoHandler struct {
	Name string
	_    struct{}
}

func (e *ErrNoHandler) Error() string {
	return fmt.Sprintf("dockerfile: no handler registered for instruction %s", e.Name)
}
