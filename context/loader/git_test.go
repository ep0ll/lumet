package loader

import (
	"reflect"
	"testing"
)

func TestParseGitReference(t *testing.T) {
	tests := []struct {
		name       string
		input      string
		keepGitDir bool
		want       gitReference
	}{
		{
			name:  "fragment branch and subdirectory",
			input: "https://github.com/acme/app.git#main:src",
			want: gitReference{
				remote: "https://github.com/acme/app.git",
				ref:    "main",
				subDir: "src",
			},
		},
		{
			name:  "query branch and keep git directory",
			input: "https://github.com/acme/app.git?branch=release&keep-git-dir",
			want: gitReference{
				remote:     "https://github.com/acme/app.git",
				ref:        "refs/heads/release",
				keepGitDir: true,
			},
		},
		{
			name:  "checksum and submodules",
			input: "https://github.com/acme/app.git?checksum=0123456789abcdef&submodules=false&mtime=commit",
			want: gitReference{
				remote:     "https://github.com/acme/app.git",
				checksum:   "0123456789abcdef",
				submodules: boolPointer(false),
				mtime:      "commit",
			},
		},
		{
			name:  "github shorthand",
			input: "github.com/acme/app",
			want: gitReference{
				remote: "https://github.com/acme/app",
			},
		},
		{
			name:       "caller keeps git directory",
			input:      "https://github.com/acme/app.git#main",
			keepGitDir: true,
			want: gitReference{
				remote:     "https://github.com/acme/app.git",
				ref:        "main",
				keepGitDir: true,
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, err := parseGitReference(test.input, test.keepGitDir)
			if err != nil {
				t.Fatal(err)
			}
			if got.keepGitDir != test.want.keepGitDir ||
				got.remote != test.want.remote ||
				got.ref != test.want.ref ||
				got.subDir != test.want.subDir ||
				got.checksum != test.want.checksum ||
				got.mtime != test.want.mtime ||
				got.fetchByCommit != test.want.fetchByCommit {
				t.Fatalf("parseGitReference() = %#v, want %#v", got, test.want)
			}
			if !reflect.DeepEqual(got.submodules, test.want.submodules) {
				t.Fatalf("submodules = %#v, want %#v", got.submodules, test.want.submodules)
			}
		})
	}
}

func TestParseGitReferenceRejectsInvalidQueries(t *testing.T) {
	tests := []string{
		"https://github.com/acme/app.git?unknown=value",
		"https://github.com/acme/app.git?ref=",
		"https://github.com/acme/app.git?keep-git-dir=not-a-bool",
		"https://github.com/acme/app.git?mtime=invalid",
		"https://github.com/acme/app.git?subdir=other#main:src",
	}
	for _, input := range tests {
		t.Run(input, func(t *testing.T) {
			if _, err := parseGitReference(input, false); err == nil {
				t.Fatalf("expected invalid Git reference %q to fail", input)
			}
		})
	}
}

func boolPointer(value bool) *bool {
	return &value
}
