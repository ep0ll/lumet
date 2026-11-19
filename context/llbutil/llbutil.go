package llbutil

import (
	"context"
	stderrors "errors"
	"fmt"
	"io/fs"

	"github.com/containerd/containerd/v2/defaults"
	"github.com/containerd/errdefs"
	"github.com/moby/buildkit/client/llb"
	gwclient "github.com/moby/buildkit/frontend/gateway/client"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func WithInternalName(name string) llb.ConstraintsOpt {
	return llb.WithCustomName("[internal] " + name)
}

func MarshalOpts(c gwclient.BuildOpts) []llb.ConstraintsOpt {
	return []llb.ConstraintsOpt{llb.WithCaps(c.Caps)}
}

// ReadFile bounds source-file reads to the gateway's maximum receive size.
func ReadFile(ctx context.Context, ref gwclient.Reference, filename string) ([]byte, error) {
	if ref == nil {
		return nil, fmt.Errorf("cannot read %q from a nil reference", filename)
	}

	const maxFileSize = defaults.DefaultMaxRecvMsgSize
	if stat, err := ref.StatFile(ctx, gwclient.StatRequest{Path: filename}); err == nil && stat != nil && stat.Size > int64(maxFileSize) {
		return nil, fmt.Errorf("%s exceeds maximum allowed size of %d bytes", filename, maxFileSize)
	}

	data, err := ref.ReadFile(ctx, gwclient.ReadRequest{
		Filename: filename,
		Range:    &gwclient.FileRange{Length: maxFileSize + 1},
	})
	if err != nil {
		return nil, err
	}
	if len(data) > maxFileSize {
		return nil, fmt.Errorf("%s exceeds maximum allowed size of %d bytes", filename, maxFileSize)
	}
	return data, nil
}

// IsNotFound reports whether a gateway read failed because the path does not
// exist. Other errors, including permission, size-limit and transport errors,
// must not be treated as an absent optional file.
func IsNotFound(err error) bool {
	return err != nil && (
		stderrors.Is(err, fs.ErrNotExist) ||
			errdefs.IsNotFound(err) ||
			status.Code(err) == codes.NotFound
	)
}
