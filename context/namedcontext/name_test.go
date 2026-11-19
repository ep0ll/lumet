package namedcontext

import (
	"testing"

	"github.com/containerd/platforms"
	ocispecs "github.com/opencontainers/image-spec/specs-go/v1"
)

func TestCanonical(t *testing.T) {
	t.Parallel()

	tests := []struct {
		input string
		want  string
	}{
		{input: "alpine:latest", want: "alpine"},
		{input: "alpine", want: "alpine"},
		{input: "docker.io/library/alpine:3.20", want: "alpine:3.20"},
		{input: "  alpine:latest  ", want: "alpine"},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			t.Parallel()
			got, err := Canonical(tt.input)
			if err != nil {
				t.Fatalf("Canonical(%q): %v", tt.input, err)
			}
			if got != tt.want {
				t.Fatalf("Canonical(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestCanonicalRejectsEmptyName(t *testing.T) {
	t.Parallel()

	for _, input := range []string{"", " ", "\t"} {
		if _, err := Canonical(input); err == nil {
			t.Errorf("Canonical(%q succeeded, want error", input)
		}
	}
}

func TestResolveRejectsNilClient(t *testing.T) {
	if _, err := Resolve("alpine", nil, ContextOpt{}); err == nil {
		t.Fatal("Resolve with nil client succeeded, want error")
	}
}

func TestResolveBaseRejectsNilClientAndEmptyName(t *testing.T) {
	if _, err := ResolveBase("alpine", nil, ContextOpt{}); err == nil {
		t.Fatal("ResolveBase with nil client succeeded, want error")
	}
	for _, input := range []string{"", " "} {
		if _, err := ResolveBase(input, nil, ContextOpt{}); err == nil {
			t.Errorf("ResolveBase(%q) succeeded, want error", input)
		}
	}
}

func TestQualifiedNormalizesPlatform(t *t) {
	t.Parallel()

	platform := &ocispecs.Platform{OS: "linux", Architecture: "amd64"}
	want := "alpine::" + platforms.FormatAll(platforms.Normalize(*platform))
	if got := Qualified("alpine", platform); got != want {
		t.Fatalf("Qualified() = %q, want %q", got, want)
	}
}
