package context

import (
	"bytes"
	"context"
	"fmt"
	"path"
	"strconv"
	"strings"

	"github.com/ep0ll/lumet/context/definition"
	"github.com/ep0ll/lumet/context/llbutil"
	"github.com/ep0ll/lumet/context/namedcontext"
	"github.com/ep0ll/lumet/frontend"
	"github.com/moby/buildkit/client/llb"
	gwclient "github.com/moby/buildkit/frontend/gateway/client"
	gwpb "github.com/moby/buildkit/frontend/gateway/pb"
	"github.com/moby/patternmatcher/ignorefile"
	"github.com/moby/buildkit/solver/pb"
)

// buildContextState is the immutable, resolved view of the main context and
// the build-definition source for one frontend invocation.
type buildContextState struct {
	mainState       *llb.State
	definitionState *llb.State

	mainLocalName       string
	definitionLocalName string
	filename            string

	mainSubDir       string
	definitionSubDir string
}

func (b *buildContext) initContext(ctx context.Context) (*buildContextState, error) {
	buildOpts := b.client.BuildOpts()
	values := buildOpts.Opts

	mainName := b.spec.DefaultContextLocalName
	if value := values[b.spec.OptKeys.ContextLocalName]; strings.TrimSpace(value) != "" {
		mainName = value
	}

	filename := strings.TrimSpace(b.spec.DefaultEntrypoint(b.client.Config().Frontend))
	if filename == "" {
		return nil, errorsForContext("frontend spec returned an empty default entrypoint")
	}
	if value := strings.TrimSpace(values[b.spec.OptKeys.Filename]); value != "" {
		filename = value
	}

	entrypointName := b.spec.EntrypointLocalNames[len(b.spec.EntrypointLocalNames)-1]
	forceLocalDefinition := false
	for _, key := range b.spec.OptKeys.EntrypointLocalNames {
		if value := strings.TrimSpace(values[key]); value != "" {
			entrypointName = value
			forceLocalDefinition = true
			break
		}
	}

	mainName = strings.TrimSpace(mainName)
	entrypointName = strings.TrimSpace(entrypointName)
	mainLocalName, forceLocalMain := stripLocalPrefix(mainName)
	definitionLocalName, forceLocalPrefix := stripLocalPrefix(entrypointName)
	forceLocalDefinition = forceLocalDefinition || forceLocalPrefix
	if mainLocalName == "" {
		return nil, errorsForContext("main context local name cannot be empty")
	}
	if definitionLocalName == "" {
		return nil, errorsForContext("build definition local name cannot be empty")
	}

	if _, err := cleanRelativePath(filename); err != nil {
		return nil, fmt.Errorf("invalid build definition filename %q: %w", filename, err)
	}
	mainSubDir, err := cleanSubDir(values[b.spec.OptKeys.ContextSubDir])
	if err != nil {
		return nil, fmt.Errorf("invalid main context subdirectory: %w", err)
	}
	definitionSubDir, err := cleanSubDir(values[b.spec.OptKeys.EntrypointSubDir])
	if err != nil {
		return nil, fmt.Errorf("invalid build definition subdirectory: %w", err)
	}

	keepGitDir := false
	if value := strings.TrimSpace(values[b.spec.OptKeys.KeepGitDir]); value != "" {
		keepGitDir, err = strconv.ParseBool(value)
		if err != nil {
			return nil, fmt.Errorf("invalid boolean value for %s: %w", b.spec.OptKeys.KeepGitDir, err)
		}
	}

	mainState, mainFilename, err := b.resolveName(ctx, mainLocalName, forceLocalMain, keepGitDir)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve main build context source %q: %w", mainName, err)
	}

	definitionState, definitionFilename, err := b.resolveName(ctx, definitionLocalName, forceLocalDefinition, keepGitDir)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve build definition source %q: %w", entrypointName, err)
	}

	// When a remote main context is selected and no separate definition
	// source was selected, the entrypoint is expected to live in that context.
	if definitionState == nil && !forceLocalDefinition && mainState != nil {
		definitionState = mainState
		definitionFilename = mainFilename
	}

	if mainState != nil {
		mainState = scopeToSubDir(mainState, mainSubDir)
		mainSubDir = ""
	}
	if definitionState != nil {
		definitionState = scopeToSubDir(definitionState, definitionSubDir)
		definitionSubDir = ""
	}

	if definitionFilename != "" {
		filename = definitionFilename
	}

	return &buildContextState{
		mainState:           mainState,
		definitionState:     definitionState,
		mainLocalName:       mainLocalName,
		definitionLocalName: definitionLocalName,
		filename:            filename,
		mainSubDir:          mainSubDir,
		definitionSubDir:    definitionSubDir,
	}, nil
}

// resolveName first consults the pluggable source resolver, then the
// BuildKit frontend-input map, and finally leaves the name as a local session.
func (b *buildContext) resolveName(
	ctx context.Context,
	name string,
	forceLocal bool,
	keepGitDir bool,
) (*llb.State, string, error) {
	if forceLocal {
		return nil, "", nil
	}

	if inputName, explicitInput := strings.CutPrefix(name, "input:"); explicitInput {
		state, found, err := b.frontendInput(ctx, inputName, true)
		if err != nil {
			return nil, "", err
		}
		if !found {
			return nil, "", fmt.Errorf("frontend input %q does not exist", inputName)
		}
		return state, "", nil
	}

	resolved, err := b.resolver.Resolve(ctx, name, ResolveOptions{KeepGitDir: keepGitDir}, b.client)
	if err != nil {
		return nil, "", err
	}
	if resolved != nil {
		if resolved.State == nil {
			return nil, "", fmt.Errorf("source resolver returned an empty state for %q", name)
		}
		return resolved.State, resolved.Filename, nil
	}

	state, found, err := b.frontendInput(ctx, name, false)
	if err != nil {
		return nil, "", err
	}
	if found {
		return state, "", nil
	}
	return nil, "", nil
}

func (b *buildContext) frontendInput(ctx context.Context, name string, explicit bool) (*llb.State, bool, error) {
	if !explicit && b.client.BuildOpts().Caps.Supports(gwpb.CapFrontendInputs) != nil {
		return nil, false, nil
	}
	if explicit && b.client.BuildOpts().Caps.Supports(gwpb.CapFrontendInputs) != nil {
		return nil, false, fmt.Errorf("frontend does not support gateway inputs")
	}

	inputs, err := b.client.Inputs(ctx)
	if err != nil {
		return nil, false, fmt.Errorf("failed to get frontend inputs: %w", err)
	}
	state, ok := inputs[name]
	if !ok {
		return nil, false, nil
	}
	return &state, true, nil
}

// BuildDefinition resolves and reads the frontend's build-definition file.
func (b *buildContext) BuildDefinition(ctx context.Context, opts ...llb.LocalOption) (definition.Source, error) {
	bctx, err := b.buildContext(ctx)
	if err != nil {
		return nil, err
	}

	definitionIgnoreFiles := b.ignoreFiles(b.spec.DefinitionIgnoreFiles, bctx.filename)

	state := bctx.definitionState
	if state == nil {
		follow := []string{bctx.filename}
		follow = append(follow, b.entrypointFallbacks(bctx.filename)...)
		follow = append(follow, definitionIgnoreFiles...)
		state = b.localState(
			bctx.definitionLocalName,
			bctx.definitionSubDir,
			follow,
			"load build definition from "+bctx.filename,
			opts...,
		)
	}

	llbDef, err := state.Marshal(ctx, llbutil.MarshalOpts(b.client.BuildOpts())...)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal build definition source: %w", err)
	}
	vertex, err := llbDef.Head()
	if err != nil {
		return nil, fmt.Errorf("failed to resolve build definition vertex: %w", err)
	}

	result, err := b.client.Solve(ctx, gwclient.SolveRequest{Definition: llbDef.ToPB()})
	if err != nil {
		return nil, fmt.Errorf("failed to resolve build definition: %w", err)
	}
	ref, err := result.SingleRef()
	if err != nil {
		return nil, fmt.Errorf("failed to get build definition reference: %w", err)
	}

	data, err := readDefinition(ctx, ref, bctx.filename, b.entrypointFallbacks(bctx.filename))
	if err != nil {
		return nil, fmt.Errorf("failed to read build definition %q: %w", bctx.filename, err)
	}

	sourceMap := llb.NewSourceMap(state, bctx.filename, b.spec.Name, data)
	sourceMap.Definition = llbDef

	warn := func(ctx context.Context, message string, opt gwclient.WarnOpts) {
		if opt.Level == 0 {
			opt.Level = 1
		}
		if opt.SourceInfo == nil {
			opt.SourceInfo = sourceInfo(sourceMap)
		}
		_ = b.client.Warn(ctx, vertex, message, opt)
	}

	ignore := func(ctx context.Context) ([]string, error) {
		return b.ignorePatterns(ctx, bctx, ref)
	}

	return definition.NewSource(sourceMap, ignore, warn), nil
}

// MainContext returns an external context state when one was supplied.
// Otherwise it creates a local source with the ignore patterns from the
// frontend's specification.
func (b *buildContext) MainContext(ctx context.Context, opts ...llb.LocalOption) (*llb.State, error) {
	bctx, err := b.buildContext(ctx)
	if err != nil {
		return nil, err
	}
	if bctx.mainState != nil {
		state := *bctx.mainState
		return &state, nil
	}

	excludes, err := b.ignorePatterns(ctx, bctx, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to load build-context ignore patterns: %w", err)
	}

	localOpts := []llb.LocalOption{
		llb.SessionID(b.localSessionID(bctx.mainLocalName)),
		llb.SharedKeyHint(bctx.mainLocalName),
		llbutil.WithInternalName("load build context"),
	}
	localOpts = append(localOpts, opts...)

	if bctx.mainSubDir != "" {
		local := llb.Local(bctx.mainLocalName, localOpts...)
		scoped := llb.Scratch().File(llb.Copy(local, "/"+bctx.mainSubDir, "/", &llb.CopyInfo{
			CopyDirContentsOnly: true,
			ExcludePatterns:     excludes,
		}))
		return &scoped, nil
	}

	localOpts = append(localOpts, llb.ExcludePatterns(excludes))
	local := llb.Local(bctx.mainLocalName, localOpts...)
	return &local, nil
}

// NamedContext resolves a context registered through the BuildKit named
// context options.
func (b *buildContext) NamedContext(name string, opt ContextOpt) (NamedContext, error) {
	if b.initErr != nil {
		return nil, b.initErr
	}
	return namedcontext.Resolve(name, b.client, opt)
}

// BaseContext resolves a named or literal source with base-image semantics.
func (b *buildContext) BaseContext(name string, opt ContextOpt) (NamedContext, error) {
	if b.initErr != nil {
		return nil, b.initErr
	}
	return namedcontext.ResolveBase(name, b.client, opt)
}

func (b *buildContext) localState(
	name string,
	subDir string,
	followPaths []string,
	description string,
	opts ...llb.LocalOption,
) *llb.State {
	localOpts := []llb.LocalOption{
		llb.SessionID(b.localSessionID(name)),
		llb.SharedKeyHint(name),
		llbutil.WithInternalName(description),
		llb.Differ(llb.DiffNone, false),
	}
	if len(followPaths) != 0 {
		paths := make([]string, 0, len(followPaths))
		for _, candidate := range followPaths {
			if clean, err := cleanRelativePath(candidate); err == nil {
				paths = append(paths, joinSubDir(subDir, clean))
			}
		}
		localOpts = append(localOpts, llb.FollowPaths(uniqueStrings(paths)))
	}
	localOpts = append(localOpts, opts...)
	state := llb.Local(name, localOpts...)
	return scopeToSubDir(&state, subDir)
}

func (b *buildContext) localSessionID(name string) string {
	sessionID, _ := frontend.GetLocalSession(b.client, name)
	return sessionID
}

func (b *buildContext) readIgnoreFiles(
	ctx context.Context,
	state *llb.State,
	localName string,
	subDir string,
	names []string,
	ref gwclient.Reference,
) ([]string, bool, error) {
	names = uniqueStrings(names)
	if len(names) == 0 {
		return nil, false, nil
	}

	if ref == nil {
		probe := state
		if probe == nil {
			probe = b.localState(
				localName,
				subDir,
				names,
				"load build-context ignore files",
			)
		}

		definition, err := probe.Marshal(ctx, llbutil.MarshalOpts(b.client.BuildOpts())...)
		if err != nil {
			return nil, false, fmt.Errorf("failed to marshal ignore-file probe: %w", err)
		}
		result, err := b.client.Solve(ctx, gwclient.SolveRequest{
			Evaluate:   true,
			Definition: definition.ToPB(),
		})
		if err != nil {
			return nil, false, fmt.Errorf("failed to solve ignore-file probe: %w", err)
		}
		ref, err = result.SingleRef()
		if err != nil {
			return nil, false, fmt.Errorf("failed to resolve ignore-file reference: %w", err)
		}
	}

	for _, name := range names {
		data, err := llbutil.ReadFile(ctx, ref, name)
		if llbutil.IsNotFound(err) {
			continue
		}
		if err != nil {
			return nil, false, fmt.Errorf("failed to read ignore file %q: %w", name, err)
		}
		patterns, err := b.parseIgnoreFile(name, data)
		if err != nil {
			return nil, false, fmt.Errorf("failed to parse ignore file %q: %w", name, err)
		}
		return patterns, true, nil
	}
	return nil, false, nil
}


func (b *buildContext) ignorePatterns(
	ctx context.Context,
	bctx *buildContextState,
	definitionRef gwclient.Reference,
) ([]string, error) {
	definitionIgnoreFiles := b.ignoreFiles(b.spec.DefinitionIgnoreFiles, bctx.filename)
	if len(definitionIgnoreFiles) != 0 {
		patterns, found, err := b.readIgnoreFiles(
			ctx, bctx.definitionState, bctx.definitionLocalName, bctx.definitionSubDir,
			definitionIgnoreFiles, definitionRef,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to read entrypoint ignore files: %w", err)
		}
		if found {
			return patterns, nil
		}
	}

	contextIgnoreFiles := b.ignoreFiles(b.spec.IgnoreFiles, bctx.filename)
	patterns, _, err := b.readIgnoreFiles(
		ctx, bctx.mainState, bctx.mainLocalName, bctx.mainSubDir,
		contextIgnoreFiles, nil,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to read build-context ignore files: %w", err)
	}
	return patterns, nil
}

func (b *buildContext) parseIgnoreFile(filename string, data []byte) ([]string, error) {
	if b.spec.IgnoreParser != nil {
		return b.spec.IgnoreParser(filename, data)
	}
	return ignorefile.ReadAll(bytes.NewReader(data))
}

func (b *buildContext) ignoreFiles(getter func(string) []string, filename string) []string {
	if getter == nil {
		return nil
	}
	return uniqueStrings(getter(filename))
}

func (b *buildContext) entrypointFallbacks(filename string) []string {
	if b.spec.EntrypointFallback == nil {
		return nil
	}
	return uniqueStrings(b.spec.EntrypointFallback(filename))
}

func readDefinition(
	ctx context.Context,
	ref gwclient.Reference,
	filename string,
	fallbacks []string,
) ([]byte, error) {
	candidates := uniqueStrings(append([]string{filename}, fallbacks...))
	var lastErr error
	for _, candidate := range candidates {
		data, err := llbutil.ReadFile(ctx, ref, candidate)
		if err == nil {
			return data, nil
		}
		if !llbutil.IsNotFound(err) {
			return nil, err
		}
		lastErr = err
	}
	if lastErr != nil {
		return nil, lastErr
	}
	return nil, fmt.Errorf("no build definition filename was provided")
}

func sourceInfo(source *llb.SourceMap) *pb.SourceInfo {
	info := &pb.SourceInfo{
		Data:     source.Data,
		Filename: source.Filename,
		Language: source.Language,
	}
	if source.Definition != nil {
		info.Definition = source.Definition.ToPB()
	}
	return info
}

func stripLocalPrefix(name string) (string, bool) {
	if value, ok := strings.CutPrefix(name, "local:"); ok {
		return value, true
	}
	return name, false
}

func cleanRelativePath(value string) (string, error) {
	value = strings.TrimSpace(strings.ReplaceAll(value, "\\", "/"))
	if value == "" || path.IsAbs(value) {
		return "", fmt.Errorf("path must be a non-empty relative path")
	}
	clean := path.Clean(value)
	if clean == "." || clean == ".." || strings.HasPrefix(clean, "../") {
		return "", fmt.Errorf("path must remain inside its source root")
	}
	return clean, nil
}

func cleanSubDir(value string) (string, error) {
	value = strings.TrimSpace(strings.ReplaceAll(value, "\\", "/"))
	if value == "" || value == "." {
		return "", nil
	}
	return cleanRelativePath(value)
}

func joinSubDir(subDir, filename string) string {
	if subDir == "" {
		return filename
	}
	return path.Join(subDir, filename)
}

func scopeToSubDir(state *llb.State, dir string) *llb.State {
	if state == nil || dir == "" {
		return state
	}
	scoped := llb.Scratch().File(llb.Copy(*state, "/"+dir, "/", &llb.CopyInfo{
		CopyDirContentsOnly: true,
	}))
	return &scoped
}

func uniqueStrings(values []string) []string {
	seen := make(map[string]struct{}, len(values))
	out := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(strings.ReplaceAll(value, "\\", "/"))
		if value == "" {
			continue
		}
		clean, err := cleanRelativePath(value)
		if err != nil {
			continue
		}
		if _, ok := seen[clean]; ok {
			continue
		}
		seen[clean] = struct{}{}
		out = append(out, clean)
	}
	return out
}

func errorsForContext(message string) error {
	return fmt.Errorf("invalid build context: %s", message)
}
