package loader

import (
	"fmt"

	"github.com/moby/buildkit/client/llb"
	"github.com/moby/buildkit/frontend/dockerui"
	dockerspec "github.com/moby/docker-image-spec/specs-go/v1"
)

func loadGit(req Request) (*llb.State, *dockerspec.DockerOCIImage, error) {
	True := true
	st, _, err := dockerui.DetectGitContext(req.Input, &True)
	if err != nil {
		return nil, nil, fmt.Errorf("invalid git context %s: %w", req.Input, err)
	}
	return st, nil, nil
}
