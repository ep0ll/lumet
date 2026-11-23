package dockerfile

// InstructionRegistry names the schema registry by its role. Registry remains
// the concise type name.
type InstructionRegistry = Registry

// NewInstructionRegistry creates an empty instruction definition registry.
func NewInstructionRegistry() *InstructionRegistry { return NewRegistry() }

// HandlerRegistry names the instruction execution registry by its role.
type HandlerRegistry = Handlers

// NewHandlerRegistry creates an empty handler registry.
func NewHandlerRegistry() *HandlerRegistry { return NewHandlers() }
