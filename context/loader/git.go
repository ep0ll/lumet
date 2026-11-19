package loader

import (
	"fmt"

	"github.com/moby/buildkit/client/llb"
	"github.com/moby/buildkit/frontend/dockerui"
	dockerspec "github.com/moby/docker-image-spec/specs-go/v1"
)

func loadGit(req Request) (*llb.State, *dockerspec.DockerOCIImage, error) {
	keepGit := true
	state, isGit, err := dockerui.DetectGitContext(req.Input, &keepGit)
	if err != nil {
		return nil, nil, fmt.Errorf("invalid git context %q: %w", req.Input, err)
	}
	if !isGit || state == nil {
		return nil, nil, fmt.Errorf("invalid git context %q", req.Input)
	}
	return state, nil, nil
}
