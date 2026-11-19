package loader

import (
	"context"
	"fmt"

	"github.com/moby/buildkit/client/llb"
	"github.com/moby/buildkit/frontend/dockerui"
	dockerspec "github.com/moby/docker-image-spec/specs-go/v1"
)

func loadHTTP(ctx context.Context, fullURL string, req Request) (*llb.State, *dockerspec.DockerOCIImage, error) {
	// Prefer git semantics when possible.
	True := true
	st, _, err := dockerui.DetectGitContext(req.Input, &True)
	if err == nil {
		return st, nil, nil
	}

	st, _, ok := dockerui.DetectHTTPContext(req.Input)
	if !ok {
		return nil, nil, fmt.Errorf("invalid http context %s", req.Input)
	}
	return st, nil, nil
}
