package definition

import "testing"

func TestCleanPath(t *testing.T) {
	tests := []struct {
		input string
		want  string
		ok    bool
	}{
		{input: "Buildfile", want: "Buildfile", ok: true},
		{input: "src/./Buildfile", want: "src/Buildfile", ok: true},
		{input: "../outside", ok: false},
		{input: "src/../../outside", ok: false},
		{input: "/absolute", ok: false},
		{input: ".", ok: false},
		{input: "", ok: false},
	}
	for _, test := range tests {
		got, err := cleanPath(test.input)
		if (err == nil) != test.ok {
			t.Fatalf("cleanPath(%q) error = %v, want success %v", test.input, err, test.ok)
		}
		if err == nil && got != test.want {
			t.Errorf("cleanPath(%q) = %q, want %q", test.input, got, test.want)
		}
	}
}
