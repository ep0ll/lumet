package source

import (
	"context"
	"fmt"

	"github.com/ep0ll/lumet/context/llbutil"
)

// ReadFirstFile returns the first candidate file present in a solved source.
// Missing candidates are skipped; other errors are returned immediately.
func ReadFirstFile(ctx context.Context, solved *Solved, candidates []string) ([]byte, string, error) {
	var missing error
	for _, candidate := range candidates {
		data, err := solved.ReadFile(ctx, candidate)
		if err == nil {
			return data, candidate, nil
		}
		if llbutil.IsNotFound(err) {
			missing = err
			continue
		}
		return nil, "", err
	}
	if missing != nil {
		return nil, "", missing
	}
	return nil, "", fmt.Errorf("no candidate paths were provided")
}
