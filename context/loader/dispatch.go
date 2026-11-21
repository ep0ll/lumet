package loader

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/ep0ll/lumet/frontend"
	"github.com/moby/buildkit/client/llb"
	gwpb "github.com/moby/buildkit/frontend/gateway/pb"
	dockerspec "github.com/moby/docker-image-spec/specs-go/v1"
)

type RequestRetryFunc func(newInput string) (*llb.State, *dockerspec.DockerOCIImage, error)

type RequestRetry interface {
	Retry(newInput string) (*llb.State, *dockerspec.DockerOCIImage, error)
}

func (r RequestRetryFunc) Retry(newInput string) (*llb.State, *dockerspec.DockerOCIImage, error) {
	return r(newInput)
}

// Request is the single data bag passed to every concrete loader.
type Request struct {
	Input            string
	Name             string
	NameWithPlatform string
	Client           frontend.Client
	Opt              ContextOpt
	Depth            int
	RetryHandler     RequestRetry
}

// Load is the only entry point of the loader subpackage.
func Load(ctx context.Context, req Request) (*llb.State, *dockerspec.DockerOCIImage, error) {
	scheme, rest, err := splitScheme(req.Input)
	if err != nil {
		st, desc, serr := SourceType(ctx, req)
		if serr != nil {
			return nil, nil, errors.Join(
				fmt.Errorf("invalid context specifier %q for %s: %w", req.Input, req.NameWithPlatform, err),
				serr,
			)
		}
		return st, desc, nil
	}

	// Backwards compatibility: git@host:...
	if strings.HasPrefix(scheme, "git@") {
		scheme, rest = "git", req.Input
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
		st, desc, serr := SourceType(ctx, req)
		if serr != nil {
			return nil, nil, errors.Join(
				fmt.Errorf("unsupported context source %q for %s", scheme, req.NameWithPlatform),
				serr,
			)
		}
		return st, desc, nil
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

// SourceType infers the source for inputs without a known scheme by trying
// each loader in order: git, http(s), frontend input (if supported), local.
func SourceType(ctx context.Context, req Request) (*llb.State, *dockerspec.DockerOCIImage, error) {
	if st, desc, err := loadGit(req); err == nil {
		return st, desc, nil
	}

	if st, desc, err := loadHTTP(ctx, req.Input, req); err == nil {
		return st, desc, nil
	}

	opts := req.Client.BuildOpts() // copy: Supports has a pointer receiver
	if opts.Caps.Supports(gwpb.CapFrontendInputs) == nil {
		if st, desc, err := loadInput(ctx, req.Input, req); err == nil {
			return st, desc, nil
		}
	}

	return loadLocal(ctx, req.Input, req)
}
