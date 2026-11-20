package llbutil

import (
	"errors"
	"io/fs"

	"github.com/containerd/errdefs"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// IsNotFound reports whether err indicates that a requested path does not exist.
// It deliberately ignores permission, size-limit, transport and other failures
// so that optional files are not treated as missing when a different error occurred.
func IsNotFound(err error) bool {
	if err == nil {
		return false
	}

	return errors.Is(err, fs.ErrNotExist) ||
		errdefs.IsNotFound(err) ||
		status.Code(err) == codes.NotFound
}
