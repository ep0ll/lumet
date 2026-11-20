package source

import (
	"context"
	"testing"

	"github.com/ep0ll/lumet/frontend"
	"github.com/moby/buildkit/client/llb"
)

func TestParseSourceSpec(t *testing.T) {
	tests := []struct {
		input string
		kind  Kind
		ref   string
	}{
		{input: "local:context", kind: KindLocal, ref: "context"},
		{input: "local://dockerfile", kind: KindLocal, ref: "dockerfile"},
		{input: "input:build-definition", kind: KindInput, ref: "build-definition"},
		{input: "input://build-context", kind: KindInput, ref: "build-context"},
		{input: "https://example.com/context.tar.gz", kind: KindHTTP, ref: "https://example.com/context.tar.gz"},
		{input: "https://github.com/acme/project.git#main", kind: KindGit, ref: "https://github.com/acme/project.git#main"},
		{input: "github.com/acme/project", kind: KindGit, ref: "github.com/acme/project"},
		{input: "git://github.com/example/project.git#main", kind: KindGit, ref: "git://github.com/example/project.git#main"},
		{input: "ssh://github.com/example/project.git#main", kind: KindGit, ref: "ssh://github.com/example/project.git#main"},
		{input: "docker-image:alpine:latest", kind: KindImage, ref: "alpine:latest"},
		{input: "oci-layout:store/image@sha256:abcd", kind: KindOCILayout, ref: "store/image@sha256:abcd"},
		{input: "git@github.com:example/project.git", kind: KindGit, ref: "git@github.com:example/project.git"},
	}

	for _, test := range tests {
		t.Run(test.input, func(t *testing.T) {
			got, err := Parse(test.input)
			if err != nil {
				t.Fatal(err)
			}
			if got.Kind != test.kind || got.Reference != test.ref {
				t.Fatalf("Parse(%q) = %#v, want kind %q ref %q", test.input, got, test.kind, test.ref)
			}
		})
	}
}

func TestParseRejectsBareLocalNames(t *testing.T) {
	for _, input := range []string{"context", "local:", "input:"} {
		if _, err := Parse(input); err == nil {
			t.Fatalf("expected source %q to be rejected", input)
		}
	}
}

func TestRegistrySupportsCustomSourceKinds(t *testing.T) {
	registry := NewRegistry()
	err := registry.Register("custom", HandlerFunc(func(
		_ context.Context,
		_ frontend.Client,
		spec Spec,
		_ Options,
	) (*Resolved, error) {
		state := llb.Scratch()
		return &Resolved{Spec: spec, State: &state}, nil
	}))
	if err != nil {
		t.Fatal(err)
	}

	resolved, err := registry.Resolve(context.Background(), nil, Spec{
		Kind:      "custom",
		Reference: "artifact://example/build",
	}, Options{})
	if err != nil {
		t.Fatal(err)
	}
	if resolved.Spec.Kind != "custom" || resolved.State == nil {
		t.Fatalf("unexpected custom resolution: %#v", resolved)
	}
}
