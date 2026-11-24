package dockerfile

import (
	"context"
	"strings"
	"testing"

	"github.com/moby/buildkit/client/llb"
)

func TestDockerfileExpandsGlobalArgInFromAndStageEnv(t *testing.T) {
	df := New()
	if err := df.AddGlobal(Arg("BASE", ArgDefault("alpine:3.21"))); err != nil { t.Fatal(err) }
	if err := df.AddStage(
		From(Image("$BASE"), As("build")),
		Env([]KeyValue{KV("APP", "service")}),
		Workdir("/opt/$APP"),
	); err != nil { t.Fatal(err) }
	result, err := df.Build(context.Background())
	if err != nil { t.Fatal(err) }
	if got := result.Target.Base.Ref; got != "alpine:3.21" { t.Fatalf("base ref = %q", got) }
	if got := result.Target.Image.Config.WorkingDir; got != "/opt/service" { t.Fatalf("working dir = %q", got) }
	if got := result.Target.Env["APP"]; got != "service" { t.Fatalf("APP = %q", got) }
}

func TestDockerfileOnBuildRunsInDerivedStageOnly(t *testing.T) {
	df := New()
	if err := df.AddStage(
		From(Scratch(), As("parent")),
		OnBuild(Env([]KeyValue{KV("TRIGGERED", "yes")})),
	); err != nil { t.Fatal(err) }
	if err := df.AddStage(
		From(StageOf("parent"), As("child")),
		Env([]KeyValue{KV("CHILD", "$TRIGGERED")}),
	); err != nil { t.Fatal(err) }
	result, err := df.Build(context.Background())
	if err != nil { t.Fatal(err) }
	parent := result.Stages["parent"]
	child := result.Stages["child"]
	if got := parent.Image.Config.OnBuild; len(got) != 1 || got[0] != "ENV TRIGGERED=yes" {
		t.Fatalf("parent ONBUILD = %#v", got)
	}
	if len(child.Image.Config.OnBuild) != 0 { t.Fatalf("child ONBUILD was not consumed: %#v", child.Image.Config.OnBuild) }
	if got := child.Env["TRIGGERED"]; got != "yes" { t.Fatalf("trigger did not run: %q", got) }
	if got := child.Env["CHILD"]; got != "yes" { t.Fatalf("child ENV did not see trigger environment: %q", got) }
}

func TestDockerfileResolvesCopyFromDependency(t *testing.T) {
	df := New()
	if err := df.AddStage(From(Scratch(), As("assets")), Workdir("/build")); err != nil { t.Fatal(err) }
	if err := df.AddStage(
		From(Scratch(), As("final")),
		Copy(Sources{"/out"}, "/app", CopyFrom(StageOf("assets"))),
	); err != nil { t.Fatal(err) }
	result, err := df.Build(context.Background())
	if err != nil { t.Fatal(err) }
	if _, ok := result.Stages["assets"]; !ok { t.Fatal("COPY --from dependency was not evaluated") }
	if result.Target.State.Output() == nil { t.Fatal("expected final LLB state output") }
}

func TestDockerfileRejectsStageDependencyCycle(t *testing.T) {
	df := New()
	if err := df.AddStage(From(StageOf("b"), As("a"))); err != nil { t.Fatal(err) }
	if err := df.AddStage(From(StageOf("a"), As("b"))); err != nil { t.Fatal(err) }
	_, err := df.Build(context.Background())
	if err == nil || !strings.Contains(err.Error(), "circular stage dependency") { t.Fatalf("expected cycle error, got %v", err) }
}

func TestFunctionalExpansionDoesNotTreatEscapeCharactersAsSyntax(t *testing.T) {
	df := New()
	if err := df.AddStage(
		From(Scratch(), As("test")),
		Env([]KeyValue{
			KV("VALUE", "expanded"),
			KV("SLASH", "C:\\\\$VALUE"),
			KV("TICK", "`$VALUE"),
		}),
	); err != nil { t.Fatal(err) }
	result, err := df.Build(context.Background())
	if err != nil { t.Fatal(err) }
	if got := result.Target.Env["SLASH"]; got != "C:\\\\expanded" { t.Fatalf("backslash was treated as expansion escape: %q", got) }
	if got := result.Target.Env["TICK"]; got != "`expanded" { t.Fatalf("backtick was treated as parser escape: %q", got) }
}

func TestDefaultShellAndEntrypointCmdSemantics(t *testing.T) {
	linux, err := commandArgs(Sh("echo ok"), nil, "linux")
	if err != nil { t.Fatal(err) }
	if len(linux) != 3 || linux[0] != "/bin/sh" || linux[1] != "-c" { t.Fatalf("linux shell = %#v", linux) }
	windows, err := commandArgs(Sh("echo ok"), nil, "windows")
	if err != nil { t.Fatal(err) }
	if len(windows) != 3 || windows[0] != "cmd" || windows[1] != "/S" || windows[2] != "/C" { t.Fatalf("windows shell = %#v", windows) }

	df := New()
	if err := df.AddStage(From(Scratch(), As("base")), Cmd(Exec("old"))); err != nil { t.Fatal(err) }
	if err := df.AddStage(From(StageOf("base")), Entrypoint(Exec("entry"))); err != nil { t.Fatal(err) }
	result, err := df.Build(context.Background())
	if err != nil { t.Fatal(err) }
	if len(result.Target.Image.Config.Cmd) != 0 { t.Fatalf("inherited CMD should be cleared by ENTRYPOINT: %#v", result.Target.Image.Config.Cmd) }
}

func TestRunAcceptsMountsAndExecutionOptions(t *testing.T) {
	df := New()
	if err := df.AddStage(
		From(Scratch()),
		Run(
			Sh("echo build"),
			Mounts(
				Bind("/src", nil, ".", false),
				Cache("/root/.cache", CacheID("golang"), CacheSharing(llb.CacheMountLocked)),
				Tmpfs("/tmp", 64<<20),
				Secret("token", "/run/secrets/token", SecretRequired()),
				SSH("default", ""),
			),
			Network(NetworkNone),
			Security(SecuritySandbox),
			Devices(Device("vendor.example/device=accelerator", true)),
			Ulimits(UlimitSpec{Name: llb.UlimitName("nofile"), Soft: 1024, Hard: 2048}),
		),
	); err != nil { t.Fatal(err) }
	if _, err := df.Build(context.Background()); err != nil { t.Fatal(err) }
}

type testCustomSpec struct {
	Common
	Value string
}
func (s testCustomSpec) String() string { return "CUSTOM " + s.Value }

func TestCustomInstructionRegistration(t *testing.T) {
	h := NewLLBHandlers()
	err := RegisterHandler(h, Kind("CUSTOM"), func(_ context.Context, call Call[testCustomSpec]) (Stage, error) {
		stage := call.Stage
		if stage.Image.Config.Labels == nil { stage.Image.Config.Labels = map[string]string{} }
		stage.Image.Config.Labels["custom"] = call.Spec.Value
		return stage, nil
	})
	if err != nil { t.Fatal(err) }
	in := NewTrigger(Kind("CUSTOM"), "CUSTOM", testCustomSpec{Value: "registered"})
	stage, err := Compose(context.Background(), h, NewRuntime(), NewStageInit(KindFrom, "FROM", FromSpec{
		Base: Scratch(), Name: "custom-stage",
	}), in)
	if err != nil { t.Fatal(err) }
	if got := stage.Image.Config.Labels["custom"]; got != "registered" { t.Fatalf("custom label = %q", got) }
}

var _ = llb.CacheMountShared
