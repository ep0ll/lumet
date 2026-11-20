package llbutil

import (
	stderrors "errors"
	"io/fs"

	"github.com/containerd/errdefs"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// IsNotFound reports whether a gateway file operation failed because the
// requested path does not exist. Permission, size-limit, transport and other
// failures must not be mistaken for an absent optional file.
func IsNotFound(err error) bool {
	return err != nil && (
		stderrors.Is(err, fs.ErrNotExist) ||
			errdefs.IsNotFound(err) ||
			status.Code(err) == codes.NotFound
	)
}
