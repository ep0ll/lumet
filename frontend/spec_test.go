package frontend

import (
	"reflect"
	"testing"
)

func TestSpecValidateMigratesLegacyOptionKeys(t *testing.T) {
	spec := Spec{
		Name: "custom",
		DefaultEntrypoint: func(string) string {
			return "Buildfile"
		},
		OptKeys: ContextOptKeys{
			Filename:             "custom-filename",
			DockerfileLocalNames: []string{"custom-entrypoint"},
			DockerfileSubDir:     "custom-entrypoint-dir",
		},
	}
	if err := spec.Validate(); err != nil {
		t.Fatal(err)
	}

	if spec.OptKeys.Filename != "custom-filename" {
		t.Fatalf("filename key was overwritten: %q", spec.OptKeys.Filename)
	}
	if !reflect.DeepEqual(spec.OptKeys.EntrypointLocalNames, []string{"custom-entrypoint"}) {
		t.Fatalf("legacy entrypoint keys were not migrated: %#v", spec.OptKeys.EntrypointLocalNames)
	}
	if spec.OptKeys.EntrypointSubDir != "custom-entrypoint-dir" {
		t.Fatalf("legacy subdirectory key was not migrated: %q", spec.OptKeys.EntrypointSubDir)
	}
	if spec.OptKeys.ContextLocalName != "contextkey" {
		t.Fatalf("missing context option key was not defaulted: %q", spec.OptKeys.ContextLocalName)
	}
}

func TestSpecValidateDoesNotAliasCallerSlices(t *testing.T) {
	localNames := []string{"entrypoint"}
	optionKeys := []string{"entrypointkey"}
	spec := Spec{
		Name: "custom",
		DefaultEntrypoint: func(string) string {
			return "Buildfile"
		},
		EntrypointLocalNames: localNames,
		OptKeys: ContextOptKeys{
			EntrypointLocalNames: optionKeys,
		},
	}
	if err := spec.Validate(); err != nil {
		t.Fatal(err)
	}

	spec.EntrypointLocalNames[0] = "changed"
	spec.OptKeys.EntrypointLocalNames[0] = "changed"

	if localNames[0] != "entrypoint" {
		t.Fatalf("Spec.Validate modified caller local names: %#v", localNames)
	}
	if optionKeys[0] != "entrypointkey" {
		t.Fatalf("Spec.Validate modified caller option keys: %#v", optionKeys)
	}
}

func TestEntrypointAndIgnoreHelpers(t *testing.T) {
	fallback := LowerBaseFallback("Dockerfile")("src/Dockerfile")
	if !reflect.DeepEqual(fallback, []string{"src/dockerfile"}) {
		t.Fatalf("unexpected fallback paths: %#v", fallback)
	}

	ignore := SuffixIgnore(".dexnore")("Dexfile")
	if !reflect.DeepEqual(ignore, []string{"Dexfile.dexnore"}) {
		t.Fatalf("unexpected ignore paths: %#v", ignore)
	}
}
