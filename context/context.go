package context

import (
	"context"
	"fmt"
	"slices"

	"github.com/ep0ll/lumet/context/lumet"
	"github.com/ep0ll/lumet/context/namedcontext"
	"github.com/ep0ll/lumet/frontend"
	"github.com/moby/buildkit/client/llb"
	gwclient "github.com/moby/buildkit/frontend/gateway/client"
	"github.com/moby/buildkit/util/flightcontrol"
)

// buildContext coordinates build-definition, main-context and named-context
// resolution for one frontend invocation.
type buildContext struct {
	group   flightcontrol.CachedGroup[*buildContextState]
	spec    frontend.Spec
	client  frontend.Client
	initErr error
}

// New creates a context client for a frontend specification.
//
// Validation is deferred to the first operation to preserve the original
// constructor signature. Invalid specifications and nil clients are reported
// by the operations instead of causing a panic.
func New(spec frontend.Spec, client frontend.Client) BuildClient {
	specCopy := spec
	specCopy.EntrypointLocalNames = slices.Clone(spec.EntrypointLocalNames)
	specCopy.OptKeys.EntrypointLocalNames = slices.Clone(spec.OptKeys.EntrypointLocalNames)
	err := specCopy.Validate()
	if client == nil && err == nil {
		err = fmt.Errorf("frontend client is nil")
	}
	return &buildContext{spec: specCopy, client: client, initErr: err}
}

// BuildDefinition resolves and reads the frontend's build-definition file.
func (b *buildContext) BuildDefinition(ctx context.Context, opts ...llb.LocalOption) (lumet.Source, error) {
	bctx, err := b.buildContext(ctx)
	if err != nil {
		return nil, err
	}

	state := bctx.definition
	if state == nil || bctx.forceLocalDefinition {
		files := []string{bctx.filename}
		if fallback := lowerCaseFallback(bctx.filename, bctx.defaultEntrypoint); fallback != "" {
			files = append(files, fallback)
		}
		if b.spec.IgnoreFiles != nil {
			files = append(files, b.spec.IgnoreFiles(bctx.filename)...)
		}

		sessionID, _ := b.client.GetLocalSession(bctx.definitionLocalName)
		localOpts := []llb.LocalOption{
			llb.FollowPaths(uniqueStrings(files)),
			llb.SessionID(sessionID),
			llb.SharedKeyHint(bctx.definitionLocalName),
			llb.WithCustomName("[internal] load build definition from " + bctx.filename),
			llb.Differ(llb.DiffNone, false),
		}
		localOpts = append(localOpts, opts...)
		local := llb.Local(bctx.definitionLocalName, localOpts...)
		state = &local
	}

	definition, err := state.Marshal(ctx, marshalOpts(b.client)...)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal build definition source: %w", err)
	}
	vertex, err := definition.Head()
	if err != nil {
		return nil, fmt.Errorf("failed to resolve build definition vertex: %w", err)
	}

	res, err := b.client.Solve(ctx, solveRequest(definition))
	if err != nil {
		return nil, fmt.Errorf("failed to resolve build definition: %w", err)
	}
	ref, err := res.SingleRef()
	if err != nil {
		return nil, fmt.Errorf("failed to get build definition reference: %w", err)
	}

	data, err := readDefinition(ctx, ref, bctx.filename, bctx.defaultEntrypoint)
	if err != nil {
		return nil, fmt.Errorf("failed to read build definition %q: %w", bctx.filename, err)
	}

	sourceMap := llb.NewSourceMap(state, bctx.filename, b.spec.Name, data)
	sourceMap.Definition = definition

	var ignoreFiles []string
	if b.spec.IgnoreFiles != nil {
		ignoreFiles = uniqueStrings(b.spec.IgnoreFiles(bctx.filename))
	}
	warn := func(ctx context.Context, message string, opt gwclient.WarnOpts) {
		if opt.Level == 0 {
			opt.Level = 1
		}
		if opt.SourceInfo == nil {
			opt.SourceInfo = sourceInfo(sourceMap)
		}
		_ = b.client.Warn(ctx, vertex, message, opt)
	}

	return lumet.NewSource(
		sourceMap,
		func(ctx context.Context) ([]string, error) {
			return b.readIgnorePatterns(ctx, bctx, ignoreFiles)
		},
		warn,
	), nil
}

// MainContext returns a remote or frontend-input context when one was
// supplied. Otherwise it constructs the local context with the ignore
// patterns declared by the frontend specification.
func (b *buildContext) MainContext(ctx context.Context, opts ...llb.LocalOption) (*llb.State, error) {
	bctx, err := b.buildContext(ctx)
	if err != nil {
		return nil, err
	}
	if bctx.main != nil {
		state := *bctx.main
		return &state, nil
	}

	source, err := b.BuildDefinition(ctx)
	if err != nil {
		return nil, err
	}
	excludes, err := source.IgnorePatterns(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to read build-context ignore patterns: %w", err)
	}

	sessionID, _ := b.client.GetLocalSession(bctx.mainLocalName)
	localOpts := []llb.LocalOption{
		llb.SessionID(sessionID),
		llb.ExcludePatterns(excludes),
		llb.SharedKeyHint(bctx.mainLocalName),
		llb.WithCustomName("[internal] load build context"),
	}
	localOpts = append(localOpts, opts...)
	state := llb.Local(bctx.mainLocalName, localOpts...)
	return &state, nil
}

// NamedContext resolves a registered named context.
func (b *buildContext) NamedContext(name string, opt ContextOpt) (NamedContext, error) {
	if b.initErr != nil {
		return nil, b.initErr
	}
	return namedcontext.Resolve(name, b.client, opt)
}

// BaseContext resolves a named or literal source using base-image semantics.
func (b *buildContext) BaseContext(name string, opt ContextOpt) (NamedContext, error) {
	if b.initErr != nil {
		return nil, b.initErr
	}
	return namedcontext.ResolveBase(name, b.client, opt)
}
