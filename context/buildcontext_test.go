package context

import (
	"reflect"
	"testing"
)

func TestCleanRelativePath(t *testing.T) {
	tests := []struct {
		input string
		want  string
		ok    bool
	}{
		{input: "Dockerfile", want: "Dockerfile", ok: true},
		{input: "src/./Buildfile", want: "src/Buildfile", ok: true},
		{input: "../outside", ok: false},
		{input: "src/../../outside", ok: false},
		{input: "/absolute", ok: false},
		{input: ".", ok: false},
		{input: "", ok: false},
	}
	for _, test := range tests {
		got, err := cleanRelativePath(test.input)
		if (err == nil) != test.ok {
			t.Fatalf("cleanRelativePath(%q) error = %v, want success %v", test.input, err, test.ok)
		}
		if err == nil && got != test.want {
			t.Errorf("cleanRelativePath(%q) = %q, want %q", test.input, got, test.want)
		}
	}
}

func TestUniqueStringsCleansAndDeduplicates(t *testing.T) {
	got := uniqueStrings([]string{
		"Dockerfile",
		"src/../.dockerignore",
		".dockerignore",
		"Dockerfile",
		"../outside",
	})
	want := []string{"Dockerfile", ".dockerignore"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("uniqueStrings() = %#v, want %#v", got, want)
	}
}
