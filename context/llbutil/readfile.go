package llbutil

import (
	"context"
	"fmt"

	"github.com/containerd/containerd/v2/defaults"
	"github.com/moby/buildkit/frontend/gateway/client"
	"github.com/pkg/errors"
)

// maxFileSize bounds files that the builtin frontend loads into daemon memory.
// A frontend running over the gateway cannot read more than one gRPC message in
// a single request anyway, so the builtin frontend is held to the same size.
const maxFileSize = defaults.DefaultMaxRecvMsgSize

type fileTooLargeError struct {
	filename string
}

func (e *fileTooLargeError) Error() string {
	return fmt.Sprintf("%s exceeds maximum allowed size of %d bytes", e.filename, maxFileSize)
}

func isFileTooLarge(err error) bool {
	var e *fileTooLargeError
	return errors.As(err, &e)
}

// ReadFile reads filename from ref, refusing files larger than maxFileSize.
// Oversized files are rejected on their reported size so that nothing is read
// from them, and the read itself carries a range so that no more than
// maxFileSize+1 bytes are ever loaded even when the size could not be checked.
func ReadFile(ctx context.Context, ref client.Reference, filename string) ([]byte, error) {
	// stat failures are left to the read, which reports them properly
	if st, err := ref.StatFile(ctx, client.StatRequest{Path: filename}); err == nil && st.Size > maxFileSize {
		return nil, errors.WithStack(&fileTooLargeError{filename: filename})
	}
	dt, err := ref.ReadFile(ctx, client.ReadRequest{
		Filename: filename,
		Range: &client.FileRange{
			Length: maxFileSize + 1,
		},
	})
	if err != nil {
		return nil, err
	}
	if len(dt) > maxFileSize {
		return nil, errors.WithStack(&fileTooLargeError{filename: filename})
	}
	return dt, nil
}

// HeaderSize is enough bytes to sniff archive magic numbers and a tar header.
const HeaderSize = 1024

// ReadFileHead reads at most n bytes from the start of filename in ref.
// Unlike ReadFile it never rejects large files: only the requested range is
// loaded, so it is safe for sniffing the type of arbitrarily large files.
// n is clamped to [1, maxFileSize] so a bad value cannot exceed the gRPC limit.
func ReadFileHead(ctx context.Context, ref client.Reference, filename string, n int) ([]byte, error) {
	if n < 1 {
		n = 1
	}
	if n > maxFileSize {
		n = maxFileSize
	}
	dt, err := ref.ReadFile(ctx, client.ReadRequest{
		Filename: filename,
		Range:    &client.FileRange{Offset: 0, Length: n},
	})
	if err != nil {
		return nil, errors.Wrapf(err, "failed to read head of %s", filename)
	}
	return dt, nil
}
