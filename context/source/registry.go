package source

import (
	"context"
	"fmt"
	"strings"
	"sync"

	"github.com/ep0ll/lumet/frontend"
)

// Registry dispatches source resolution to independently replaceable handlers.
type Registry struct {
	mu       sync.RWMutex
	handlers map[Kind]Handler
}

// NewRegistry creates a registry containing the built-in BuildKit source
// handlers. A handler can be replaced with Register.
func NewRegistry() *Registry {
	return &Registry{
		handlers: map[Kind]Handler{
			KindLocal:     HandlerFunc(resolveLocal),
			KindGit:       HandlerFunc(resolveGit),
			KindHTTP:      HandlerFunc(resolveHTTP),
			KindInput:     HandlerFunc(resolveInput),
			KindImage:     HandlerFunc(resolveImage),
			KindOCILayout: HandlerFunc(resolveOCI),
		},
	}
}

// Register installs or replaces a handler for kind.
func (r *Registry) Register(kind Kind, handler Handler) error {
	if r == nil {
		return fmt.Errorf("source registry is nil")
	}
	if kind == "" {
		return fmt.Errorf("source kind is required")
	}
	if handler == nil {
		return fmt.Errorf("source handler for %q is nil", kind)
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.handlers == nil {
		r.handlers = make(map[Kind]Handler)
	}
	r.handlers[kind] = handler
	return nil
}

// Resolve implements Resolver.
func (r *Registry) Resolve(ctx context.Context, client frontend.Client, spec Spec, opts Options) (*Resolved, error) {
	if r == nil {
		return nil, fmt.Errorf("source registry is nil")
	}
	spec.Reference = strings.TrimSpace(spec.Reference)
	if err := spec.Validate(); err != nil {
		return nil, err
	}

	r.mu.RLock()
	handler := r.handlers[spec.Kind]
	r.mu.RUnlock()
	if handler == nil {
		return nil, fmt.Errorf("no source handler registered for %q", spec.Kind)
	}

	resolved, err := handler.Resolve(ctx, client, spec, opts.Clone())
	if err != nil {
		return nil, fmt.Errorf("resolve %s: %w", spec.String(), err)
	}
	if resolved == nil {
		return nil, fmt.Errorf("source handler for %q returned a nil result", spec.Kind)
	}
	if resolved.State == nil {
		return nil, fmt.Errorf("source handler for %q returned a nil LLB state", spec.Kind)
	}
	if resolved.Spec.Kind == "" {
		resolved.Spec = spec
	}
	return resolved, nil
}

// DefaultResolver returns a fresh registry with all built-in handlers.
func DefaultResolver() Resolver { return NewRegistry() }
