package definition

import (
	"context"
	"errors"
	"testing"
)

func TestIgnorePatternsAreCachedAndCopied(t *testing.T) {
	calls := 0
	source := NewSource(nil, func(context.Context) ([]string, error) {
		calls++
		return []string{"vendor/**"}, nil
	}, nil)

	first, err := source.IgnorePatterns(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	first[0] = "mutated"

	second, err := source.IgnorePatterns(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if calls != 1 {
		t.Fatalf("ignore function called %d times, want 1", calls)
	}
	if len(second) != 1 || second[0] != "vendor/**" {
		t.Fatalf("cached patterns were aliased by caller: %#v", second)
	}
}

func TestIgnorePatternsRetryAfterFailure(t *testing.T) {
	calls := 0
	source := NewSource(nil, func(context.Context) ([]string, error) {
		calls++
		if calls == 1 {
			return nil, errors.New("transient read failure")
		}
		return []string{"tmp"}, nil
	}, nil)

	if _, err := source.IgnorePatterns(context.Background()); err == nil {
		t.Fatal("first call should return the transient error")
	}
	patterns, err := source.IgnorePatterns(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if calls != 2 {
		t.Fatalf("ignore function called %d times, want 2", calls)
	}
	if len(patterns) != 1 || patterns[0] != "tmp" {
		t.Fatalf("unexpected patterns: %#v", patterns)
	}
}
