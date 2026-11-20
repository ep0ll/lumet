package maincontext

import "testing"

func TestCleanSubDir(t *testing.T) {
	tests := []struct {
		input string
		want  string
		ok    bool
	}{
		{input: "", want: "", ok: true},
		{input: ".", want: "", ok: true},
		{input: "src/./app", want: "src/app", ok: true},
		{input: "../outside", ok: false},
		{input: "src/../../outside", ok: false},
		{input: "/absolute", ok: false},
	}
	for _, test := range tests {
		got, err := cleanSubDir(test.input)
		if (err == nil) != test.ok {
			t.Fatalf("cleanSubDir(%q) error = %v, want success %v", test.input, err, test.ok)
		}
		if err == nil && got != test.want {
			t.Errorf("cleanSubDir(%q) = %q, want %q", test.input, got, test.want)
		}
	}
}
