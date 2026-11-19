package frontend

import "testing"

func TestSpecValidateFillsMissingOptionKeysWithoutReplacingCustomKeys(t *testing.T) {
	t.Parallel()

	spec := Spec{
		Name:              "Lumet",
		DefaultEntrypoint: func(string) string { return "Lumetfile" },
		OptKeys: ContextOptKeys{
			Filename:             "lumetfilename",
			ContextLocalName:     "lumetcontext",
			EntrypointLocalNames: []string{"lumetkey"},
		},
	}
	if err := spec.Validate(); err != nil {
		t.Fatalf("Validate(): %v", err)
	}

	if spec.OptKeys.Filename != "lumetfilename" {
		t.Fatalf("Filename key = %q, want custom key", spec.OptKeys.Filename)
	}
	if spec.OptKeys.ContextLocalName != "lumetcontext" {
		t.Fatalf("ContextLocalName key = %q, want custom key", spec.OptKeys.ContextLocalName)
	}
	if len(spec.OptKeys.EntrypointLocalNames) != 1 || spec.OptKeys.EntrypointLocalNames[0] != "lumetkey" {
		t.Fatalf("EntrypointLocalNames = %#v, want [lumetkey]", spec.OptKeys.EntrypointLocalNames)
	}
	if spec.OptKeys.ContextSubDir != "contextsubdir" {
		t.Fatalf("ContextSubDir = %q, want default", spec.OptKeys.ContextSubDir)
	}
	if spec.DefaultContextLocalName != "context" {
		t.Fatalf("DefaultContextLocalName = %q, want context", spec.DefaultContextLocalName)
	}
}

func TestDefaultContextOptKeys(t *testing.T) {
	t.Parallel()

	keys := DefaultContextOptKeys()
	if keys.Filename != "filename" {
		t.Fatalf("Filename = %q, want filename", keys.Filename)
	}
	if keys.ContextLocalName != "contextkey" {
		t.Fatalf("ContextLocalName = %q, want contextkey", keys.ContextLocalName)
	}
}

func TestSpecValidateRejectsEmptyDefaultEntrypoint(t *testing.T) {
	t.Parallel()

	spec := Spec{
		Name:              "Lumet",
		DefaultEntrypoint: func(string) string { return "" },
	}
	if err := spec.Validate(); err == nil {
		t.Fatal("Validate() succeeded for empty default entrypoint, want error")
	}
}
