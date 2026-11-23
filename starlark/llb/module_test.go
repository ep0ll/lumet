package llb

import (
	"context"
	"strings"
	"testing"

	"go.starlark.net/starlark"
	"go.starlark.net/syntax"
)

func TestExecBuildsAndMarshalsState(t *testing.T) {
	t.Parallel()

	source := `
load("llb", "Scratch", "Mkfile")

state = Scratch().file(Mkfile("/hello.txt", 0o644, b"hello"))
definition = state.marshal()
`

	globals, err := Exec(context.Background(), "build.star", source, nil)
	if err != nil {
		t.Fatalf("Exec() error = %v", err)
	}

	state, ok := globals["state"].(*StateValue)
	if !ok {
		t.Fatalf("state global has type %T, want *StateValue", globals["state"])
	}
	definition, err := state.Unwrap().Marshal(context.Background())
	if err != nil {
		t.Fatalf("State.Marshal() error = %v", err)
	}
	if len(definition.Def) == 0 {
		t.Fatal("State.Marshal() returned an empty definition")
	}

	definitionValue, ok := globals["definition"].(*DefinitionValue)
	if !ok {
		t.Fatalf("definition global has type %T, want *DefinitionValue", globals["definition"])
	}
	if definitionValue.Unwrap() == nil {
		t.Fatal("DefinitionValue.Unwrap() returned nil")
	}
}

func TestExecWithCustomModuleLoader(t *testing.T) {
	t.Parallel()

	const helperSource = `
def make_state():
    return llb.Scratch().file(llb.Mkfile("/from-module", 0o644, b"ok"))
`

	loader := func(thread *starlark.Thread, module string) (starlark.StringDict, error) {
		if module != "helpers.star" {
			return nil, nil
		}
		return starlark.ExecFileOptions(
			&syntax.FileOptions{},
			thread,
			module,
			helperSource,
			Predeclared(),
		)
	}

	globals, err := ExecWithOptions(
		context.Background(),
		"build.star",
		`
load("helpers.star", "make_state")
result = make_state()
`,
		nil,
		ExecOptions{Loader: loader},
	)
	if err != nil {
		t.Fatalf("ExecWithOptions() error = %v", err)
	}
	if _, ok := globals["result"].(*StateValue); !ok {
		t.Fatalf("result global has type %T, want *StateValue", globals["result"])
	}
}

func TestExecRejectsOptionFromWrongFamily(t *testing.T) {
	t.Parallel()

	_, err := Exec(
		context.Background(),
		"bad-option.star",
		`
load("llb", "Image", "WithParents")
result = Image("busybox:latest", WithParents(True))
`,
		nil,
	)
	if err == nil {
		t.Fatal("Exec() succeeded with a MkdirOption passed as an ImageOption")
	}
	if !strings.Contains(err.Error(), "cannot be used as ImageOption") {
		t.Fatalf("Exec() error = %v, want option-family mismatch", err)
	}
}

func TestPlatformIsAnImageConstraintOption(t *testing.T) {
	t.Parallel()

	globals, err := Exec(
		context.Background(),
		"platform.star",
		`
load("llb", "Image", "Platform")
result = Image("alpine:latest", Platform(os = "linux", architecture = "amd64"))
`,
		nil,
	)
	if err != nil {
		t.Fatalf("Exec() error = %v", err)
	}
	if _, ok := globals["result"].(*StateValue); !ok {
		t.Fatalf("result global has type %T, want *StateValue", globals["result"])
	}
}

func TestExecOptionsEnforceStepBudget(t *testing.T) {
	t.Parallel()

	_, err := ExecWithOptions(
		context.Background(),
		"budget.star",
		"x = 1 + 2\n",
		nil,
		ExecOptions{MaxSteps: 1},
	)
	if err == nil {
		t.Fatal("ExecWithOptions() succeeded despite a one-step execution budget")
	}
}


func TestStateAndConstraintOptions(t *testing.T) {
	t.Parallel()

	globals, err := Exec(
		context.Background(),
		"constraints.star",
		`
load("llb", "Scratch", "Reset", "AddEnv", "WithLinuxResources", "WithCustomNamef")

state = Scratch().with(Reset(Scratch()), AddEnv("MODE", "test"))
definition = state.marshal(
    WithLinuxResources(memory = 1073741824, cpu_shares = 512),
    WithCustomNamef("stage-%s", "test"),
)
`,
		nil,
	)
	if err != nil {
		t.Fatalf("Exec() error = %v", err)
	}
	state, ok := globals["state"].(*StateValue)
	if !ok {
		t.Fatalf("state global has type %T, want *StateValue", globals["state"])
	}
	definition, err := state.Unwrap().Marshal(context.Background())
	if err != nil {
		t.Fatalf("State.Marshal() error = %v", err)
	}
	if len(definition.Def) == 0 {
		t.Fatal("State.Marshal() returned an empty definition")
	}
	if _, ok := globals["definition"].(*DefinitionValue); !ok {
		t.Fatalf("definition global has type %T, want *DefinitionValue", globals["definition"])
	}
}
