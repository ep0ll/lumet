package context

import (
	"bytes"
	"context"
	"path"
	"strconv"
	"strings"

	"github.com/ep0ll/lumet/context/archive"
	"github.com/ep0ll/lumet/context/llbutil"
	"github.com/ep0ll/lumet/frontend"
	"github.com/moby/buildkit/client/llb"
	gwclient "github.com/moby/buildkit/frontend/gateway/client"
	"github.com/moby/buildkit/frontend/dockerui"
	gwpb "github.com/moby/buildkit/frontend/gateway/pb"
	"github.com/moby/buildkit/solver/pb"
	"github.com/moby/patternmatcher/ignorefile"
	"github.com/pkg/errors"
)

type buildContextState struct {
	main                 *llb.State
	definition           *llb.State
	mainLocalName        string
	definitionLocalName  string
	filename             string
	defaultEntrypoint    string
	forceLocalDefinition bool
}

func (b *buildContext) buildContext(ctx context.Context) (*buildContextState, error) {
	if b.initErr != nil {
		return nil, b.initErr
	}
	return b.group.Do(ctx, "initcontext", func(ctx context.Context) (*buildContextState, error) {
		return b.initContext(ctx)
	})
}

func (b *buildContext) initContext(ctx context.Context) (*buildContextState, error) {
	buildOpts := b.client.BuildOpts()
	values := buildOpts.Opts

	mainName := b.spec.DefaultContextLocalName
	if key := b.spec.OptKeys.ContextLocalName; key != "" {
		if value := values[key]; value != "" {
			mainName = value
		}
	}

	defaultEntrypoint := strings.TrimSpace(b.spec.DefaultEntrypoint(""))
	if defaultEntrypoint == "" {
		return nil, errors.New("frontend spec returned an empty default entrypoint")
	}
	filename := defaultEntrypoint
	if key := b.spec.OptKeys.Filename; key != "" {
		if value := strings.TrimSpace(values[key]); value != "" {
			filename = value
		}
	}

	if len(b.spec.EntrypointLocalNames) == 0 {
		return nil, errors.New("frontend spec has no entrypoint local name")
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

	keepGit := false
	if key := b.spec.OptKeys.KeepGitDir; key != "" && values[key] != "" {
		var err error
		keepGit, err = strconv.ParseBool(values[key])
		if err != nil {
			return nil, errors.Wrapf(err, "invalid boolean value for %s", key)
		}
	}

	mainName = strings.TrimSpace(mainName)
	entrypointName = strings.TrimSpace(entrypointName)
	if mainName == "" || stripLocalPrefix(mainName) == "" {
		return nil, errors.New("main context local name cannot be empty")
	}
	if entrypointName == "" || stripLocalPrefix(entrypointName) == "" {
		return nil, errors.New("build definition local name cannot be empty")
	}

	forceLocalMain := strings.HasPrefix(mainName, "local:")
	forceLocalDefinition = forceLocalDefinition || strings.HasPrefix(entrypointName, "local:")
	mainName = stripLocalPrefix(mainName)
	entrypointName = stripLocalPrefix(entrypointName)

	main, mainFilename, err := b.resolveSource(ctx, mainName, keepGit)
	if err != nil {
		return nil, errors.Wrap(err, "failed to resolve main build context")
	}

	var definition *llb.State
	var definitionFilename string
	if entrypointName == mainName {
		definition, definitionFilename = main, mainFilename
	} else {
		definition, definitionFilename, err = b.resolveSource(ctx, entrypointName, keepGit)
		if err != nil {
			return nil, errors.Wrap(err, "failed to resolve build definition source")
		}
	}

	// Gateway inputs are preferred over local-session sources unless a local:
	// prefix or an explicit frontend option forces a local source.
	if buildOpts.Caps.Supports(gwpb.CapFrontendInputs) == nil {
		inputs, err := b.client.Inputs(ctx)
		if err != nil {
			return nil, errors.Wrap(err, "failed to get frontend inputs")
		}
		if main == nil && !forceLocalMain {
			if input, ok := inputs[mainName]; ok {
				state := input
				main = &state
			}
		}
		if definition == nil && !forceLocalDefinition {
			if input, ok := inputs[entrypointName]; ok {
				state := input
				definition = &state
			}
		}
	}

	// A remote main context also contains the build definition unless a
	// separate definition source was explicitly selected.
	if definition == nil && main != nil && !forceLocalDefinition {
		definition = main
		definitionFilename = mainFilename
	}
	if definitionFilename != "" {
		filename = definitionFilename
	}

	if main != nil {
		if subdir := values[b.spec.OptKeys.ContextSubDir]; subdir != "" {
			main = scopeToSubDir(main, subdir)
		}
	}
	if definition != nil {
		if subdir := values[b.spec.OptKeys.EntrypointSubDir]; subdir != "" {
			definition = scopeToSubDir(definition, subdir)
		}
	}

	return &buildContextState{
		main:                 main,
		definition:           definition,
		mainLocalName:        mainName,
		definitionLocalName:  entrypointName,
		filename:             filename,
		defaultEntrypoint:    defaultEntrypoint,
		forceLocalDefinition: forceLocalDefinition,
	}, nil
}

// resolveSource recognises Git and HTTP sources. A local: prefix explicitly
// forces BuildKit to use the named local session.
func (b *buildContext) resolveSource(ctx context.Context, input string, keepGit bool) (*llb.State, string, error) {
	if input == "" || strings.HasPrefix(input, "local:") {
		return nil, "", nil
	}

	gitOptions, err := b.gitSourceOptions()
	if err != nil {
		return nil, "", err
	}
	state, isGit, err := dockerui.DetectGitContext(input, &keepGit, gitOptions...)
	if isGit {
		if err != nil {
			return nil, "", errors.Wrapf(err, "invalid Git source %q", input)
		}
		return state, "", nil
	}
	// A non-Git local name commonly produces a parse error from Git URL
	// detection. It is not an error unless the source was identified as Git.
	httpState, filename, ok := dockerui.DetectHTTPContext(input)
	if !ok {
		return nil, "", nil
	}
	resolved, resolvedFilename, err := b.resolveHTTP(ctx, httpState, filename)
	return resolved, resolvedFilename, err
}

func (b *buildContext) gitSourceOptions() ([]llb.GitOption, error) {
	values := b.client.BuildOpts().Opts
	var options []llb.GitOption

	if values["build-arg:SOURCE_DATE_EPOCH"] != "" {
		options = append(options, llb.GitMTimeCommit())
	}

	for _, option := range []struct {
		key string
		set func() llb.GitOption
	}{
		{key: "build-arg:BUILDKIT_GIT_ADVICE", set: func() llb.GitOption { return llb.GitAdvice(true) }},
		{key: "build-arg:BUILDKIT_GIT_DEBUG_COMMANDS", set: func() llb.GitOption { return llb.GitDebugCommands(true) }},
	} {
		raw := values[option.key]
		if raw == "" {
			continue
		}
		enabled, err := strconv.ParseBool(raw)
		if err != nil {
			return nil, errors.Wrapf(err, "invalid boolean value for %s", option.key)
		}
		if enabled {
			options = append(options, option.set())
		}
	}
	return options, nil
}

func (b *buildContext) resolveHTTP(ctx context.Context, state *llb.State, filename string) (*llb.State, string, error) {
	definition, err := state.Marshal(ctx, marshalOpts(b.client)...)
	if err != nil {
		return nil, "", errors.Wrap(err, "failed to marshal remote HTTP context")
	}
	res, err := b.client.Solve(ctx, gwclient.SolveRequest{Definition: definition.ToPB()})
	if err != nil {
		return nil, "", errors.Wrap(err, "failed to resolve remote HTTP context")
	}
	ref, err := res.SingleRef()
	if err != nil {
		return nil, "", errors.Wrap(err, "failed to get remote HTTP context reference")
	}
	header, err := ref.ReadFile(ctx, gwclient.ReadRequest{
		Filename: filename,
		Range:    &gwclient.FileRange{Length: 1024},
	})
	if err != nil {
		return nil, "", errors.Wrap(err, "failed to inspect downloaded HTTP context")
	}

	if archive.IsArchive(header) {
		unpacked := llb.Scratch().File(llb.Copy(*state, path.Join("/", filename), "/", &llb.CopyInfo{
			AttemptUnpack: true,
		}))
		return &unpacked, "", nil
	}
	return state, filename, nil
}

func (b *buildContext) readIgnorePatterns(ctx context.Context, bctx *buildContextState, names []string) ([]string, error) {
	if bctx.main != nil || len(names) == 0 {
		return nil, nil
	}

	sessionID, _ := b.client.GetLocalSession(bctx.mainLocalName)
	probe := llb.Local(
		bctx.mainLocalName,
		llb.SessionID(sessionID),
		llb.FollowPaths(names),
		llb.SharedKeyHint("context:"+bctx.mainLocalName+"-"+strings.Join(names, "-")),
		llb.WithCustomName("[internal] load build-context ignore files"),
		llb.Differ(llb.DiffNone, false),
	)
	definition, err := probe.Marshal(ctx, marshalOpts(b.client)...)
	if err != nil {
		return nil, errors.Wrap(err, "failed to marshal ignore-file probe")
	}
	res, err := b.client.Solve(ctx, gwclient.SolveRequest{
		Evaluate:   true,
		Definition: definition.ToPB(),
	})
	if err != nil {
		return nil, errors.Wrap(err, "failed to solve ignore-file probe")
	}
	ref, err := res.SingleRef()
	if err != nil {
		return nil, errors.Wrap(err, "failed to get ignore-file reference")
	}

	for _, name := range names {
		data, err := llbutil.ReadFile(ctx, ref, name)
		if llbutil.IsNotFound(err) {
			continue
		}
		if err != nil {
			return nil, errors.Wrapf(err, "failed to read ignore file %q", name)
		}
		patterns, err := ignorefile.ReadAll(bytes.NewReader(data))
		if err != nil {
			return nil, errors.Wrapf(err, "failed to parse ignore file %q", name)
		}
		return patterns, nil
	}
	return nil, nil
}

func readDefinition(ctx context.Context, ref gwclient.Reference, filename, defaultEntrypoint string) ([]byte, error) {
	data, err := llbutil.ReadFile(ctx, ref, filename)
	if err == nil {
		return data, nil
	}

	fallback := lowerCaseFallback(filename, defaultEntrypoint)
	if fallback == "" || !llbutil.IsNotFound(err) {
		return nil, err
	}
	data, fallbackErr := llbutil.ReadFile(ctx, ref, fallback)
	if fallbackErr != nil {
		return nil, fallbackErr
	}
	return data, nil
}

func lowerCaseFallback(filename, defaultEntrypoint string) string {
	if path.Base(filename) != path.Base(defaultEntrypoint) {
		return ""
	}
	fallback := path.Join(path.Dir(filename), strings.ToLower(path.Base(filename)))
	if fallback == filename {
		return ""
	}
	return fallback
}

func stripLocalPrefix(name string) string {
	return strings.TrimPrefix(name, "local:")
}

func scopeToSubDir(state *llb.State, dir string) *llb.State {
	if state == nil || dir == "" {
		return state
	}
	clean := path.Clean("/" + dir)
	scoped := llb.Scratch().File(llb.Copy(*state, clean, "/", &llb.CopyInfo{
		CopyDirContentsOnly: true,
	}))
	return &scoped
}

func uniqueStrings(values []string) []string {
	seen := make(map[string]struct{}, len(values))
	out := make([]string, 0, len(values))
	for _, value := range values {
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		out = append(out, value)
	}
	return out
}

func marshalOpts(client frontend.Client) []llb.ConstraintsOpt {
	return llbutil.MarshalOpts(client.BuildOpts())
}

func solveRequest(definition *llb.Definition) gwclient.SolveRequest {
	return gwclient.SolveRequest{Definition: definition.ToPB()}
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
