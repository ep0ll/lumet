package loader

import (
	"context"
	"strings"

	"github.com/ep0ll/lumet/frontend"
	"github.com/moby/buildkit/client/llb"
	dockerspec "github.com/moby/docker-image-spec/specs-go/v1"
	"github.com/pkg/errors"
)

// Request is the single data bag passed to every concrete loader.
type Request struct {
	Input            string
	Name             string
	NameWithPlatform string
	Client           frontend.Client
	Opt              ContextOpt
	Depth            int
	Retry            func(newInput string) (*llb.State, *dockerspec.DockerOCIImage, error)
}

// Load is the only entry point of the loader subpackage.
func Load(ctx context.Context, req Request) (*llb.State, *dockerspec.DockerOCIImage, error) {
	scheme, rest, err := splitScheme(req.Input)
	if err != nil {
		return nil, nil, errors.Wrapf(err, "invalid context specifier %q for %s", req.Input, req.NameWithPlatform)
	}

	// Backwards compatibility: git@host:... 
	if strings.HasPrefix(scheme, "git@") {
		scheme = "git"
		rest = req.Input
	}

	switch scheme {
	case "docker-image":
		return loadImage(ctx, rest, req)
	case "oci-layout":
		return loadOCI(ctx, rest, req)
	case "local":
		return loadLocal(ctx, rest, req)
	case "input":
		return loadInput(ctx, rest, req)
	case "git":
		return loadGit(req)
	case "http", "https":
		return loadHTTP(ctx, req.Input, req)
	default:
		return nil, nil, errors.Errorf("unsupported context source %q for %s", scheme, req.NameWithPlatform)
	}
}

func splitScheme(s string) (scheme, rest string, err error) {
	i := strings.IndexByte(s, ':')
	if i < 0 {
		return "", "", errors.New("missing scheme")
	}
	return s[:i], s[i+1:], nil
}

// label builds the conventional BuildKit custom name.
func label(nameWithPlatform, action string) string {
	return "[context " + nameWithPlatform + "] " + action
}
