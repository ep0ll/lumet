package llbutil

import (
	"errors"
	"io/fs"
	"testing"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestIsNotFound(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		err  error
		want bool
	}{
		{name: "nil", err: nil, want: false},
		{name: "filesystem not found", err: fs.ErrNotExist, want: true},
		{name: "gRPC not found", err: status.Error(codes.NotFound, "missing"), want: true},
		{name: "permission denied", err: status.Error(codes.PermissionDenied, "denied"), want: false},
		{name: "other error", err: errors.New("unexpected"), want: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := IsNotFound(tt.err); got != tt.want {
				t.Fatalf("IsNotFound(%v) = %v, want %v", tt.err, got, tt.want)
			}
		})
	}
}
