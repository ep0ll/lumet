package loader

import "testing"

func TestSplitScheme(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		input      string
		wantScheme string
		wantRest   string
		wantErr    bool
	}{
		{name: "HTTP URL", input: "https://example.com/context", wantScheme: "https", wantRest: "//example.com/context"},
		{name: "docker image", input: "docker-image://alpine:latest", wantScheme: "docker-image", wantRest: "//alpine:latest"},
		{name: "OCI layout", input: "oci-layout://store/image@sha256:abc", wantScheme: "oci-layout", wantRest: "//store/image@sha256:abc"},
		{name: "legacy SSH Git", input: "git@github.com:owner/repo.git", wantScheme: "git", wantRest: "git@github.com:owner/repo.git"},
		{name: "uppercase URI scheme", input: "HTTPS://example.com/context", wantScheme: "https", wantRest: "//example.com/context"},
		{name: "trailing whitespace", input: " input:build ", wantScheme: "input", wantRest: "build"},
		{name: "missing scheme", input: "owner/repo", wantErr: true},
		{name: "empty source", input: "http:", wantErr: true},
		{name: "empty input", input: "", wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			scheme, rest, err := splitScheme(tt.input)
			if (err != nil) != tt.wantErr {
				t.Fatalf("splitScheme(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
			}
			if err != nil {
				return
			}
			if scheme != tt.wantScheme || rest != tt.wantRest {
				t.Fatalf("splitScheme(%q) = (%q, %q), want (%q, %q)", tt.input, scheme, rest, tt.wantScheme, tt.wantRest)
			}
		})
	}
}
