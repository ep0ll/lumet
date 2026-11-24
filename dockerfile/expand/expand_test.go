package expand

import "testing"

func TestExpandVariableForms(t *testing.T) {
	values := map[string]string{"SET": "value", "EMPTY": ""}
	lookup := MapLookup(values)
	expander := New(Config{}, lookup)
	tests := []struct{ input, want string }{
		{"$SET", "value"},
		{"${SET}", "value"},
		{"${UNSET-default}", "default"},
		{"${EMPTY:-default}", "default"},
		{"${SET+alternate}", "alternate"},
		{"${EMPTY:+alternate}", ""},
		{"$${SET}", "${SET}"},
		{"C:\\\\$SET", "C:\\\\value"},
		{"`$SET", "`value"},
	}
	for _, test := range tests {
		got, err := expander.Expand(test.input)
		if err != nil { t.Fatalf("Expand(%q): %v", test.input, err) }
		if got != test.want { t.Errorf("Expand(%q) = %q, want %q", test.input, got, test.want) }
	}
}

func TestExpandRequiredAndStrictUnset(t *testing.T) {
	expander := New(Config{}, MapLookup(map[string]string{}))
	if _, err := expander.Expand("${TOKEN:?token is required}"); err == nil { t.Fatal("expected required-variable error") }
	strict := New(Config{Strict: true}, MapLookup(map[string]string{}))
	if _, err := strict.Expand("$MISSING"); err == nil { t.Fatal("expected unset-variable error") }
}

func TestExpandPreserveUnset(t *testing.T) {
	expander := New(Config{PreserveUnset: true}, MapLookup(map[string]string{}))
	got, err := expander.Expand("left-$MISSING-${OTHER}")
	if err != nil { t.Fatal(err) }
	if got != "left-$MISSING-${OTHER}" { t.Fatalf("got %q", got) }
}
