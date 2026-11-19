package loader

import (
	"context"
	"strings"

	"github.com/ep0ll/lumet/frontend"
	"github.com/moby/buildkit/client/llb"
	dockerspec "github.com/moby/docker-image-spec/specs-go/v1"
	"github.com/pkg/errors"
)

// Request is the data shared by all concrete context loaders.
type Request struct {
	Input            string
	Name             string
	NameWithPlatform string
	Client           frontend.Client
	Opt              ContextOpt
	Depth            int
	Retry            func(newInput string) (*llb.State, *dockerspec.DockerOCIImage, error)
}

// Load dispatches a context source to its scheme-specific loader.
func Load(ctx context.Context, req Request) (*llb.State, *dockerspec.DockerOCIImage, error) {
	if req.Client == nil {
		return nil, nil, errors.New("context loader requires a frontend client")
	}
	req.Input = strings.TrimSpace(req.Input)
	scheme, _, err := splitScheme(req.Input)
	if err != nil {
		return nil, nil, errors.Wrapf(err, "invalid context specifier %q for %s", req.Input, req.NameWithPlatform)
	}
	// URI schemes are case-insensitive, while BuildKit's source parsers
	// expect canonical lowercase schemes. Preserve legacy git@ SSH syntax.
	if !strings.HasPrefix(req.Input, "git@") {
		if i := strings.IndexByte(req.Input, ':'); i >= 0 {
			req.Input = scheme + req.Input[i:]
		}
	}

	switch scheme {
	case "docker-image":
		_, rest, _ := splitScheme(req.Input)
		return loadImage(ctx, rest, req)
	case "oci-layout":
		_, rest, _ := splitScheme(req.Input)
		return loadOCI(ctx, rest, req)
	case "local":
		_, rest, _ := splitScheme(req.Input)
		return loadLocal(ctx, rest, req)
	case "input":
		_, rest, _ := splitScheme(req.Input)
		return loadInput(ctx, rest, req)
	case "git":
		return loadGit(req)
	case "http", "https":
		return loadHTTP(ctx, req.Input, req)
	default:
		return nil, nil, errors.Errorf("unsupported context source %q for %s", scheme, req.NameWithPlatform)
	}
}

// splitScheme accepts URI-style sources and the legacy git@host:path syntax.
func splitScheme(input string) (scheme, rest string, err error) {
	input = strings.TrimSpace(input)
	if input == "" {
		return "", "", errors.New("empty context specifier")
	}
	if strings.HasPrefix(input, "git@") {
		return "git", input, nil
	}

	i := strings.IndexByte(input, ':')
	if i <= 0 {
		return "", "", errors.New("missing source scheme")
	}
	scheme = strings.ToLower(input[:i])
	rest = input[i+1:]
	if strings.ContainsAny(scheme, "/\\") {
		return "", "", errors.Errorf("invalid source scheme %q", scheme)
	}
	if rest == "" {
		return "", "", errors.New("missing source after scheme")
	}
	return scheme, rest, nil
}

// label builds the conventional BuildKit custom name.
func label(nameWithPlatform, action string) string {
	return "[context " + nameWithPlatform + "] " + action
}
