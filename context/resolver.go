package context

import (
	"context"
	"fmt"
	"net/url"
	"path"
	"strconv"
	"strings"

	"github.com/ep0ll/lumet/context/archive"
	"github.com/ep0ll/lumet/context/llbutil"
	"github.com/ep0ll/lumet/frontend"
	"github.com/moby/buildkit/client/llb"
	"github.com/moby/buildkit/frontend/dockerfile/dfgitutil"
	gwclient "github.com/moby/buildkit/frontend/gateway/client"
)

// BuildKitSourceResolver implements the common BuildKit Git and HTTP source
// conventions. A frontend can replace it with WithSourceResolver when it has
// additional source schemes or different resolution policies.
type BuildKitSourceResolver struct{}

// Resolve returns nil for local names and source kinds it does not recognize.
func (BuildKitSourceResolver) Resolve(
	ctx context.Context,
	input string,
	opt ResolveOptions,
	client frontend.Client,
) (*ResolvedSource, error) {
	input = strings.TrimSpace(input)
	if input == "" || strings.HasPrefix(input, "local:") || strings.HasPrefix(input, "input:") {
		return nil, nil
	}

	if gitRef, isGit, err := dfgitutil.ParseGitRef(input); isGit {
		if err != nil {
			return nil, err
		}
		gitOpts, err := gitOptions(client, gitRef, opt.KeepGitDir)
		if err != nil {
			return nil, err
		}
		state := llb.Git(gitRef.Remote, "", gitOpts...)
		return &ResolvedSource{State: &state}, nil
	}

	parsed, err := url.Parse(input)
	if err != nil {
		return nil, err
	}
	if !strings.EqualFold(parsed.Scheme, "http") && !strings.EqualFold(parsed.Scheme, "https") {
		return nil, nil
	}
	if parsed.Host == "" {
		return nil, nil
	}

	const filename = "context"
	state := llb.HTTP(input,
		llb.Filename(filename),
		llbutil.WithInternalName("load remote build context"),
	)
	definition, err := state.Marshal(ctx, llbutil.MarshalOpts(client.BuildOpts())...)
	if err != nil {
		return nil, err
	}
	result, err := client.Solve(ctx, gwclient.SolveRequest{Definition: definition.ToPB()})
	if err != nil {
		return nil, err
	}
	ref, err := result.SingleRef()
	if err != nil {
		return nil, err
	}
	header, err := ref.ReadFile(ctx, gwclient.ReadRequest{
		Filename: filename,
		Range:    &gwclient.FileRange{Length: 1024},
	})
	if err != nil {
		return nil, err
	}
	if archive.IsArchive(header) {
		unpacked := llb.Scratch().File(llb.Copy(state, path.Join("/", filename), "/", &llb.CopyInfo{
			AttemptUnpack: true,
		}))
		return &ResolvedSource{State: &unpacked}, nil
	}

	return &ResolvedSource{State: &state, Filename: filename}, nil
}

func gitOptions(client frontend.Client, gitRef *dfgitutil.GitRef, keepGitDir bool) ([]llb.GitOption, error) {
	values := client.BuildOpts().Opts
	options := []llb.GitOption{
		llb.GitRef(gitRef.Ref),
		llbutil.WithInternalName("load git source " + gitRef.Remote),
	}

	if (gitRef.KeepGitDir != nil && *gitRef.KeepGitDir) || keepGitDir {
		options = append(options, llb.KeepGitDir())
	}
	if gitRef.SubDir != "" {
		options = append(options, llb.GitSubDir(gitRef.SubDir))
	}
	if gitRef.Checksum != "" {
		options = append(options, llb.GitChecksum(gitRef.Checksum))
	}
	if gitRef.Submodules != nil && !*gitRef.Submodules {
		options = append(options, llb.GitSkipSubmodules())
	}
	if gitRef.MTime != "" {
		options = append(options, llb.GitMTime(gitRef.MTime))
	}
	if gitRef.FetchByCommit {
		options = append(options, llb.GitFetchByCommit())
	}
	if values["build-arg:SOURCE_DATE_EPOCH"] != "" {
		options = append(options, llb.GitMTimeCommit())
	}

	for _, option := range []struct {
		key string
		set func() llb.GitOption
	}{
		{
			key: "build-arg:BUILDKIT_GIT_ADVICE",
			set: func() llb.GitOption { return llb.GitAdvice(true) },
		},
		{
			key: "build-arg:BUILDKIT_GIT_DEBUG_COMMANDS",
			set: func() llb.GitOption { return llb.GitDebugCommands(true) },
		},
	} {
		raw := values[option.key]
		if raw == "" {
			continue
		}
		enabled, err := strconv.ParseBool(raw)
		if err != nil {
			return nil, fmt.Errorf("invalid %s value %q: %w", option.key, raw, err)
		}
		if enabled {
			options = append(options, option.set())
		}
	}
	return options, nil
}
