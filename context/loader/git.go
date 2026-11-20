package loader

import (
	"fmt"
	"net/url"
	"path"
	"strconv"
	"strings"

	"github.com/ep0ll/lumet/context/llbutil"
	"github.com/ep0ll/lumet/frontend"
	"github.com/moby/buildkit/client/llb"
	"github.com/moby/buildkit/frontend/dockerui"
	"github.com/moby/buildkit/util/gitutil"
	dockerspec "github.com/moby/docker-image-spec/specs-go/v1"
)

type gitReference struct {
	remote        string
	ref           string
	subDir        string
	checksum      string
	keepGitDir    bool
	submodules    *bool
	mtime         string
	fetchByCommit bool
}

// ResolveGit builds the LLB state for a Git source using BuildKit's transport
// parser and the frontend's Git-related build arguments.
func ResolveGit(client frontend.Client, reference string, keepGitDir bool) (*llb.State, error) {
	ref, err := parseGitReference(reference, keepGitDir)
	if err != nil {
		return nil, fmt.Errorf("invalid Git source %q: %w", reference, err)
	}

	gitOpts := []llb.GitOption{
		llb.GitRef(ref.ref),
		llbutil.WithInternalName("load git source " + reference),
	}
	if ref.keepGitDir {
		gitOpts = append(gitOpts, llb.KeepGitDir())
	}
	if ref.subDir != "" {
		gitOpts = append(gitOpts, llb.GitSubDir(ref.subDir))
	}
	if ref.checksum != "" {
		gitOpts = append(gitOpts, llb.GitChecksum(ref.checksum))
	}
	if ref.submodules != nil && !*ref.submodules {
		gitOpts = append(gitOpts, llb.GitSkipSubmodules())
	}
	if ref.mtime != "" {
		gitOpts = append(gitOpts, llb.GitMTime(ref.mtime))
	}
	if ref.fetchByCommit {
		gitOpts = append(gitOpts, llb.GitFetchByCommit())
	}
	if client.BuildOpts().Opts["build-arg:SOURCE_DATE_EPOCH"] != "" {
		gitOpts = append(gitOpts, llb.GitMTimeCommit())
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
		value := client.BuildOpts().Opts[option.key]
		if value == "" {
			continue
		}
		enabled, err := strconv.ParseBool(value)
		if err != nil {
			return nil, fmt.Errorf("invalid %s value %q: %w", option.key, value, err)
		}
		if enabled {
			gitOpts = append(gitOpts, option.set())
		}
	}

	state := llb.Git(ref.remote, "", gitOpts...)
	return &state, nil
}

// parseGitReference uses BuildKit's transport-level Git URL parser rather than
// Dockerfile frontend parsing. It supports fragment refs/subdirectories,
// query overrides, SSH SCP-style URLs, and the GitHub shorthand.
func parseGitReference(value string, keepGitDir bool) (*gitReference, error) {
	value = strings.TrimSpace(value)
	if strings.HasPrefix(value, "github.com/") {
		value = "https://" + value
	}

	parsed, err := gitutil.ParseURL(value)
	if err != nil {
		return nil, err
	}

	result := &gitReference{
		remote:     parsed.Remote,
		keepGitDir: keepGitDir,
	}
	if parsed.Opts != nil {
		result.ref = parsed.Opts.Ref
		result.subDir = parsed.Opts.Subdir
	}
	if err := result.loadQuery(parsed.Query); err != nil {
		return nil, err
	}
	return result, nil
}

func (r *gitReference) loadQuery(query url.Values) error {
	for key, values := range query {
		value, err := singleQueryValue(key, values)
		if err != nil {
			return err
		}

		switch key {
		case "ref":
			if r.ref != "" && r.ref != value {
				return fmt.Errorf("git ref conflicts: %q vs %q", r.ref, value)
			}
			r.ref = value
		case "tag":
			tag := value
			if tag != "" && !strings.HasPrefix(tag, "refs/tags/") {
				tag = "refs/tags/" + tag
			}
			if r.ref != "" && r.ref != tag {
				return fmt.Errorf("git ref conflicts: %q vs %q", r.ref, tag)
			}
			r.ref = tag
		case "branch":
			branch := value
			if branch != "" && !strings.HasPrefix(branch, "refs/heads/") {
				branch = "refs/heads/" + branch
			}
			if strings.HasPrefix(r.ref, "refs/tags/") {
				return fmt.Errorf("git branch conflicts with tag %q", r.ref)
			}
			if r.ref != "" && r.ref != branch {
				return fmt.Errorf("git ref conflicts: %q vs %q", r.ref, branch)
			}
			r.ref = branch
		case "subdir":
			subDir := cleanGitSubDir(value)
			if r.subDir != "" && subDir != r.subDir {
				return fmt.Errorf("git subdirectory conflicts: %q vs %q", r.subDir, subDir)
			}
			r.subDir = subDir
		case "checksum", "commit":
			if r.checksum != "" && r.checksum != value {
				return fmt.Errorf("git checksum conflicts: %q vs %q", r.checksum, value)
			}
			r.checksum = value
		case "keep-git-dir":
			enabled, err := parseGitBool(key, value)
			if err != nil {
				return err
			}
			r.keepGitDir = r.keepGitDir || enabled
		case "submodules":
			enabled, err := parseGitBool(key, value)
			if err != nil {
				return err
			}
			r.submodules = &enabled
		case "mtime":
			switch value {
			case "checkout", "commit":
				r.mtime = value
			default:
				return fmt.Errorf("invalid git mtime value %q (must be checkout or commit)", value)
			}
		case "fetch-by-commit":
			enabled, err := parseGitBool(key, value)
			if err != nil {
				return err
			}
			r.fetchByCommit = enabled
		default:
			return fmt.Errorf("unexpected Git URL query %q", key)
		}
	}
	return nil
}

func singleQueryValue(key string, values []string) (string, error) {
	if len(values) == 0 {
		switch key {
		case "keep-git-dir", "submodules", "fetch-by-commit":
			return "", nil
		default:
			return "", fmt.Errorf("git query %q has no value", key)
		}
	}
	if len(values) != 1 {
		return "", fmt.Errorf("git query %q must have exactly one value", key)
	}
	value := values[0]
	if value == "" {
		switch key {
		case "keep-git-dir", "submodules", "fetch-by-commit":
			return value, nil
		default:
			return "", fmt.Errorf("git query %q has no value", key)
		}
	}
	return value, nil
}

func parseGitBool(key, value string) (bool, error) {
	if value == "" {
		return true, nil
	}
	enabled, err := strconv.ParseBool(value)
	if err != nil {
		return false, fmt.Errorf("invalid Git URL query %q value %q: %w", key, value, err)
	}
	return enabled, nil
}

func cleanGitSubDir(value string) string {
	value = strings.ReplaceAll(value, "\\", "/")
	clean := path.Clean("/" + value)
	return strings.TrimPrefix(clean, "/")
}

// Keep legacy named-context parsing unchanged.
func loadGit(req Request) (*llb.State, *dockerspec.DockerOCIImage, error) {
	True := true
	st, _, err := dockerui.DetectGitContext(req.Input, &True)
	if err != nil {
		return nil, nil, fmt.Errorf("invalid git context %s: %w", req.Input, err)
	}
	return st, nil, nil
}
