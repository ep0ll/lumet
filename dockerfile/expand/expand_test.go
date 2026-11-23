package expand

import "testing"

func TestExpand(t *testing.T) {
	expander := New(Config{}, MapLookup(map[string]string{
		"NAME":  "lumet",
		"EMPTY": "",
	}))
	tests := map[string]string{
		"$NAME":                    "lumet",
		"$" + "{NAME}":             "lumet",
		"$" + "{MISSING:-fallback}": "fallback",
		"$" + "{NAME:+yes}":         "yes",
		"$$NAME":                   "$NAME",
		"\\$NAME":                  "$NAME",
	}
	for input, want := range tests {
		got, err := expander.Expand(input)
		if err != nil || got != want {
			t.Errorf("Expand(%q) = %q, %v; want %q", input, got, err, want)
		}
	}
}

func TestBacktickEscape(t *testing.T) {
	expander := New(Config{Escape: 96}, MapLookup(map[string]string{"N": "x"}))
	got, err := expander.Expand(string(rune(96)) + "$N")
	if err != nil || got != "$N" {
		t.Fatalf("Expand with backtick escape = %q, %v", got, err)
	}
}

func TestStrictUnset(t *testing.T) {
	if _, err := New(Config{Strict: true}, nil).Expand("$MISSING"); err == nil {
		t.Fatal("expected unset-variable error")
	}
}
