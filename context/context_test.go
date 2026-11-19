package context

import "testing"

func TestLowerCaseFallback(t *testing.T) {
	t.Parallel()

	tests := []struct {
		filename string
		want     string
	}{
		{filename: "Dockerfile", want: "dockerfile"},
		{filename: "subdir/Dockerfile", want: "subdir/dockerfile"},
		{filename: "Dockerfile.dev", want: ""},
		{filename: "dockerfile", want: ""},
	}
	for _, tt := range tests {
		t.Run(tt.filename, func(t *testing.T) {
			t.Parallel()
			if got := lowerCaseFallback(tt.filename, "Dockerfile"); got != tt.want {
				t.Fatalf("lowerCaseFallback(%q) = %q, want %q", tt.filename, got, tt.want)
			}
		})
	}
}
