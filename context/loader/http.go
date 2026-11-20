package loader

import (
	"context"
	"fmt"
	"path"

	"github.com/ep0ll/lumet/context/archive"
	"github.com/ep0ll/lumet/context/llbutil"
	"github.com/moby/buildkit/client/llb"
	"github.com/moby/buildkit/frontend/dockerui"
	gwclient "github.com/moby/buildkit/frontend/gateway/client"
	dockerspec "github.com/moby/docker-image-spec/specs-go/v1"
)

func loadHTTP(ctx context.Context, fullURL string, req Request) (*llb.State, *dockerspec.DockerOCIImage, error) {
	keepGit := true
	if gst, _, err := dockerui.DetectGitContext(fullURL, &keepGit); err == nil {
		return gst, nil, nil
	}

	st, filename, ok := dockerui.DetectHTTPContext(fullURL)
	if !ok {
		return nil, nil, fmt.Errorf("invalid http context %q for %s", fullURL, req.NameWithPlatform)
	}

	// Download, then sniff the header to see whether it is an archive.
	def, err := st.Marshal(ctx, llbutil.MarshalOpts(req.Client.BuildOpts())...)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to marshal http context %q: %w", fullURL, err)
	}

	res, err := req.Client.Solve(ctx, gwclient.SolveRequest{Definition: def.ToPB()})
	if err != nil {
		return nil, nil, fmt.Errorf("failed to resolve http context %q: %w", fullURL, err)
	}

	ref, err := res.SingleRef()
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get http context ref %q: %w", fullURL, err)
	}

	header, err := llbutil.ReadFileHead(ctx, ref, filename, llbutil.HeaderSize)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to read downloaded http context %q: %w", fullURL, err)
	}

	if archive.IsArchive(header) {
		bc := llb.Scratch().File(
			llb.Copy(*st, path.Join("/", filename), "/", &llb.CopyInfo{AttemptUnpack: true}),
			llbutil.WithInternalName("unpack remote build context"),
		)
		return &bc, nil, nil
	}
	
	return st, nil, nil
}