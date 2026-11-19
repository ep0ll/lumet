package loader

import (
	"context"
	"fmt"

	"github.com/moby/buildkit/client/llb"
	"github.com/moby/buildkit/frontend/dockerui"
	dockerspec "github.com/moby/docker-image-spec/specs-go/v1"
)

func loadHTTP(ctx context.Context, fullURL string, req Request) (*llb.State, *dockerspec.DockerOCIImage, error) {
	// Some HTTPS URLs are Git remotes. Preserve BuildKit's Git interpretation
	// before falling back to a plain HTTP source.
	keepGit := true
	state, isGit, gitErr := dockerui.DetectGitContext(fullURL, &keepGit)
	if isGit {
		if gitErr != nil {
			return nil, nil, fmt.Errorf("invalid git context %q: %w", fullURL, gitErr)
		}
		return state, nil, nil
	}

	state, _, ok := dockerui.DetectHTTPContext(fullURL)
	if !ok || state == nil {
		if gitErr != nil {
			return nil, nil, fmt.Errorf("invalid HTTP context %q: %w", fullURL, gitErr)
		}
		return nil, nil, fmt.Errorf("invalid HTTP context %q", fullURL)
	}
	return state, nil, nil
}
