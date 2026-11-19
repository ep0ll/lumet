package lumet

import (
	"context"
	stderrors "errors"
	"testing"
)

func TestSourceIgnorePatternsCachesCopiesAndRetriesErrors(t *testing.T) {
	t.Parallel()

	calls := 0
	source := NewSource(nil, func(context.Context) ([]string, error) {
		calls++
		if calls == 1 {
			return nil, stderrors.New("transient read failure")
		}
		return []string{"node_modules", "dist"}, nil
	}, nil)

	if _, err := source.IgnorePatterns(context.Background()); err == nil {
		t.Fatal("first IgnorePatterns() error = nil, want transient error")
	}

	patterns, err := source.IgnorePatterns(context.Background())
	if err != nil {
		t.Fatalf("second IgnorePatterns(): %v", err)
	}
	patterns[0] = "mutated"

	cached, err := source.IgnorePatterns(context.Background())
	if err != nil {
		t.Fatalf("third IgnorePatterns(): %v", err)
	}
	if calls != 2 {
		t.Fatalf("ignore callback called %d times, want 2", calls)
	}
	if cached[0] != "node_modules" {
		t.Fatalf("cached patterns were mutated: %#v", cached)
	}
}
