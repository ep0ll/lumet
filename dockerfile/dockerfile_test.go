package dockerfile_test

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/moby/buildkit/client/llb"
	dockerspec "github.com/moby/docker-image-spec/specs-go/v1"
	ocispecs "github.com/opencontainers/image-spec/specs-go/v1"

	"github.com/ep0ll/lumet/dockerfile"
	"github.com/ep0ll/lumet/dockerfile/instruction"
	"github.com/ep0ll/lumet/dockerfile/progress"
)

func TestGlobalArgExpansionAndImageStateTransitions(t *testing.T) {
	df := dockerfile.New(
		dockerfile.WithPlatform(testPlatform()),
		dockerfile.WithImageResolver(func(_ context.Context, reference string, _ ocispecs.Platform) (dockerspec.DockerOCIImage, error) {
			if reference != "alpine:3.21" {
				t.Fatalf("image reference = %q", reference)
			}
			return testImage(), nil
		}),
	)
	if err := df.AddGlobal(instruction.Arg("BASE", instruction.ArgDefault("alpine:3.21"))); err != nil {
		t.Fatal(err)
	}
	if err := df.AddStage(
		dockerfile.From(dockerfile.Image("$BASE"), dockerfile.As("build")),
		instruction.Arg("APP_VERSION", instruction.ArgDefault("1.0")),
		instruction.Env([]instruction.KeyValue{instruction.KV("APP", "service")}),
		instruction.Env([]instruction.KeyValue{instruction.KV("APP_HOME", "/opt/$APP")}),
		instruction.Workdir("$APP_HOME"),
		instruction.Expose(instruction.TCP(8080), instruction.UDP(5353)),
		instruction.Volume("/var/lib/$APP"),
		instruction.Label([]instruction.KeyValue{instruction.KV("org.example.version", "$APP_VERSION")}),
	); err != nil {
		t.Fatal(err)
	}

	result, err := df.Build(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if got := result.Target.BaseRef; got != "alpine:3.21" {
		t.Fatalf("base ref = %q", got)
	}
	if got := result.Target.Image.Config.WorkingDir; got != "/opt/service" {
		t.Fatalf("working directory = %q", got)
	}
	if got := result.Target.Env["APP"]; got != "service" {
		t.Fatalf("APP = %q", got)
	}
	if got := result.Target.Image.Config.Labels["org.example.version"]; got != "1.0" {
		t.Fatalf("label version = %q", got)
	}
	if _, ok := result.Target.Image.Config.ExposedPorts["8080/tcp"]; !ok {
		t.Fatal("TCP port missing from image config")
	}
	if _, ok := result.Target.Image.Config.ExposedPorts["5353/udp"]; !ok {
		t.Fatal("UDP port missing from image config")
	}
	if _, ok := result.Target.Image.Config.Volumes["/var/lib/service"]; !ok {
		t.Fatal("volume missing from image config")
	}
}

func TestEnvPairsExpandAgainstPreInstructionEnvironment(t *testing.T) {
	df := dockerfile.New()
	if err := df.AddStage(
		dockerfile.From(dockerfile.Scratch(), dockerfile.As("env")),
		instruction.Env([]instruction.KeyValue{
			instruction.KV("A", "first"),
			instruction.KV("B", "$A"),
		}),
	); err != nil {
		t.Fatal(err)
	}
	result, err := df.Build(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if got := result.Target.Env["B"]; got != "" {
		t.Fatalf("B should not see another assignment from the same ENV instruction, got %q", got)
	}
}

func TestOnBuildIsRecordedThenExecutedInDerivedStage(t *testing.T) {
	df := dockerfile.New()
	if err := df.AddStage(
		dockerfile.From(dockerfile.Scratch(), dockerfile.As("parent")),
		instruction.OnBuild(instruction.Env([]instruction.KeyValue{instruction.KV("TRIGGERED", "yes")})),
	); err != nil {
		t.Fatal(err)
	}
	if err := df.AddStage(
		dockerfile.From(dockerfile.StageOf("parent"), dockerfile.As("child")),
		instruction.Env([]instruction.KeyValue{instruction.KV("CHILD", "$TRIGGERED")}),
	); err != nil {
		t.Fatal(err)
	}

	result, err := df.Build(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	parent := result.Stages["parent"]
	child := result.Stages["child"]
	if got := parent.Image.Config.OnBuild; len(got) != 1 || got[0] != "ENV TRIGGERED=yes" {
		t.Fatalf("parent ONBUILD metadata = %#v", got)
	}
	if len(child.Image.Config.OnBuild) != 0 {
		t.Fatalf("derived stage did not consume ONBUILD metadata: %#v", child.Image.Config.OnBuild)
	}
	if got := child.Env["TRIGGERED"]; got != "yes" {
		t.Fatalf("ONBUILD trigger did not execute: %q", got)
	}
	if got := child.Env["CHILD"]; got != "yes" {
		t.Fatalf("child ENV did not see trigger environment: %q", got)
	}
}

func TestOnBuildStageDependenciesResolveWhenTriggered(t *testing.T) {
	df := dockerfile.New()
	if err := df.AddStage(
		dockerfile.From(dockerfile.Scratch(), dockerfile.As("parent")),
		instruction.OnBuild(instruction.Copy(
			instruction.Sources{"/artifact"}, "/opt/artifact",
			instruction.CopyFrom(dockerfile.StageOf("assets")),
		)),
	); err != nil {
		t.Fatal(err)
	}
	if err := df.AddStage(
		dockerfile.From(dockerfile.Scratch(), dockerfile.As("assets")),
		instruction.Workdir("/build"),
	); err != nil {
		t.Fatal(err)
	}
	if err := df.AddStage(
		dockerfile.From(dockerfile.StageOf("parent"), dockerfile.As("child")),
	); err != nil {
		t.Fatal(err)
	}

	result, err := df.Build(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := result.Stages["assets"]; !ok {
		t.Fatal("stage referenced by ONBUILD was not evaluated")
	}
}

func TestCopyFromCreatesStageDependency(t *testing.T) {
	df := dockerfile.New()
	if err := df.AddStage(
		dockerfile.From(dockerfile.Scratch(), dockerfile.As("assets")),
		instruction.Workdir("/build"),
	); err != nil {
		t.Fatal(err)
	}
	if err := df.AddStage(
		dockerfile.From(dockerfile.Scratch(), dockerfile.As("final")),
		instruction.Copy(instruction.Sources{"/out"}, "/app",
			instruction.CopyFrom(dockerfile.StageOf("assets"))),
	); err != nil {
		t.Fatal(err)
	}
	result, err := df.Build(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := result.Stages["assets"]; !ok {
		t.Fatal("COPY --from dependency was not evaluated")
	}
}

func TestFromStageReferenceCanExpandGlobalArg(t *testing.T) {
	df := dockerfile.New()
	if err := df.AddGlobal(instruction.Arg("BASE_STAGE", instruction.ArgDefault("assets"))); err != nil {
		t.Fatal(err)
	}
	if err := df.AddStage(
		dockerfile.From(dockerfile.Scratch(), dockerfile.As("assets")),
		instruction.Workdir("/assets"),
	); err != nil {
		t.Fatal(err)
	}
	if err := df.AddStage(
		dockerfile.From(dockerfile.StageOf("$BASE_STAGE"), dockerfile.As("final")),
	); err != nil {
		t.Fatal(err)
	}
	result, err := df.Build(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := result.Stages["assets"]; !ok {
		t.Fatal("stage selected by expanded FROM reference was not resolved")
	}
	if result.Target.BaseRef != "assets" {
		t.Fatalf("base ref = %q, want assets", result.Target.BaseRef)
	}
}

func TestFromNamedContextReferenceCanExpandGlobalArg(t *testing.T) {
	df := dockerfile.New(
		dockerfile.WithNamedContext("source", llb.Scratch()),
	)
	if err := df.AddGlobal(instruction.Arg("CONTEXT_NAME", instruction.ArgDefault("source"))); err != nil {
		t.Fatal(err)
	}
	if err := df.AddStage(
		dockerfile.From(dockerfile.Context("$CONTEXT_NAME"), dockerfile.As("final")),
		instruction.Workdir("/workspace"),
	); err != nil {
		t.Fatal(err)
	}
	result, err := df.Build(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if result.Target.BaseRef != "context:source" {
		t.Fatalf("base ref = %q, want context:source", result.Target.BaseRef)
	}
}

func TestCopyFromStageNameCanExpandFromArg(t *testing.T) {
	df := dockerfile.New()
	if err := df.AddStage(
		dockerfile.From(dockerfile.Scratch(), dockerfile.As("assets")),
		instruction.Workdir("/assets"),
	); err != nil {
		t.Fatal(err)
	}
	if err := df.AddStage(
		dockerfile.From(dockerfile.Scratch(), dockerfile.As("final")),
		instruction.Arg("ASSET_STAGE", instruction.ArgDefault("assets")),
		instruction.Copy(instruction.Sources{"/bundle"}, "/opt/bundle",
			instruction.CopyFrom(dockerfile.StageOf("$ASSET_STAGE"))),
	); err != nil {
		t.Fatal(err)
	}
	result, err := df.Build(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := result.Stages["assets"]; !ok {
		t.Fatal("stage selected by expanded COPY --from was not resolved")
	}
}

func TestImageResolverSuppliesOCIConfigAndShell(t *testing.T) {
	image := testImage()
	image.Config.Env = []string{"BASE_SETTING=present"}
	image.Config.WorkingDir = "/workspace"
	image.Config.User = "1000:1000"
	image.Config.Shell = []string{"/bin/bash", "-e", "-c"}

	df := dockerfile.New(
		dockerfile.WithPlatform(testPlatform()),
		dockerfile.WithImageResolver(func(_ context.Context, reference string, _ ocispecs.Platform) (dockerspec.DockerOCIImage, error) {
			if reference != "example/base:latest" {
				t.Fatalf("resolved image reference = %q", reference)
			}
			return image, nil
		}),
	)
	if err := df.AddStage(
		dockerfile.From(dockerfile.Image("example/base:latest"), dockerfile.As("image")),
		instruction.Run(instruction.Sh("echo $BASE_SETTING")),
		instruction.Cmd(instruction.Sh("echo $BASE_SETTING")),
	); err != nil {
		t.Fatal(err)
	}
	result, err := df.Build(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if got := result.Target.Env["BASE_SETTING"]; got != "present" {
		t.Fatalf("image ENV = %q", got)
	}
	if got := result.Target.Image.Config.WorkingDir; got != "/workspace" {
		t.Fatalf("image WORKDIR = %q", got)
	}
	want := []string{"/bin/bash", "-e", "-c", "echo $BASE_SETTING"}
	if strings.Join(result.Target.Image.Config.Cmd, "|") != strings.Join(want, "|") {
		t.Fatalf("CMD = %#v, want %#v", result.Target.Image.Config.Cmd, want)
	}
}

func TestRejectsStageDependencyCycles(t *testing.T) {
	df := dockerfile.New()
	if err := df.AddStage(dockerfile.From(dockerfile.StageOf("b"), dockerfile.As("a"))); err != nil {
		t.Fatal(err)
	}
	if err := df.AddStage(dockerfile.From(dockerfile.StageOf("a"), dockerfile.As("b"))); err != nil {
		t.Fatal(err)
	}
	_, err := df.Build(context.Background())
	if err == nil || !strings.Contains(err.Error(), "circular stage dependency") {
		t.Fatalf("expected dependency cycle error, got %v", err)
	}
}

func TestVariableExpansionDoesNotConsumeBackslashesOrBackticks(t *testing.T) {
	df := dockerfile.New()
	if err := df.AddStage(
		dockerfile.From(dockerfile.Scratch(), dockerfile.As("escape")),
		instruction.Env([]instruction.KeyValue{
			instruction.KV("VALUE", "expanded"),
			instruction.KV("WINDOWS_PATH", "C:\\$VALUE"),
			instruction.KV("BACKTICK", "`$VALUE"),
		}),
	); err != nil {
		t.Fatal(err)
	}
	result, err := df.Build(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if got := result.Target.Env["WINDOWS_PATH"]; got != "C:\\expanded" {
		t.Fatalf("backslash was interpreted as an expansion escape: %q", got)
	}
	if got := result.Target.Env["BACKTICK"]; got != "`expanded" {
		t.Fatalf("backtick was interpreted as parser syntax: %q", got)
	}
}

func TestEntrypointClearsInheritedCmd(t *testing.T) {
	df := dockerfile.New()
	if err := df.AddStage(
		dockerfile.From(dockerfile.Scratch(), dockerfile.As("base")),
		instruction.Cmd(instruction.Exec("old")),
	); err != nil {
		t.Fatal(err)
	}
	if err := df.AddStage(
		dockerfile.From(dockerfile.StageOf("base"), dockerfile.As("derived")),
		instruction.Entrypoint(instruction.Exec("entry")),
	); err != nil {
		t.Fatal(err)
	}
	result, err := df.Build(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Target.Image.Config.Cmd) != 0 {
		t.Fatalf("ENTRYPOINT should clear inherited CMD: %#v", result.Target.Image.Config.Cmd)
	}
}

func TestRunMountsAndExecutionOptions(t *testing.T) {
	df := dockerfile.New()
	if err := df.AddStage(
		dockerfile.From(dockerfile.Scratch()),
		instruction.Run(
			instruction.Sh("go test ./..."),
			instruction.Mounts(
				instruction.Bind("/src", nil, ".", false),
				instruction.Cache("/root/.cache", instruction.CacheID("go-build"), instruction.CacheSharing(llb.CacheMountLocked)),
				instruction.Cache("/go/pkg/mod", instruction.CacheID("go-mod"), instruction.CacheOwnership(1000, 1000, 0o755)),
				instruction.Tmpfs("/tmp", 64<<20),
				instruction.Secret("token", "/run/secrets/token", instruction.SecretRequired()),
				instruction.Secret("registry-password", "", instruction.SecretEnv("REGISTRY_PASSWORD"), instruction.SecretOptional()),
				instruction.Secret("dual-secret", "/run/secrets/dual", instruction.SecretEnv("DUAL_SECRET")),
				instruction.SSH("default", "", instruction.SSHSocket(1000, 1000, 0o600)),
			),
			instruction.Network(instruction.NetworkNone),
			instruction.Security(instruction.SecurityInsecure),
			instruction.ReadOnlyRootFS(),
			instruction.Devices(instruction.Device("vendor.example/device=accelerator", true)),
			instruction.Ulimits(instruction.Ulimit{Name: llb.UlimitName("nofile"), Soft: 1024, Hard: 2048}),
		),
	); err != nil {
		t.Fatal(err)
	}
	if _, err := df.Build(context.Background()); err != nil {
		t.Fatal(err)
	}
}

func TestInstructionClosureUpdatesLLBAndImageConfig(t *testing.T) {
	custom := dockerfile.NewInstruction("ANNOTATE", func(_ context.Context, state dockerfile.State, _ *dockerfile.Runtime, _ dockerfile.InstructionMetadata) (dockerfile.State, error) {
		state.LLB = state.LLB.AddEnv("CUSTOM_VALUE", "present")
		if state.Env == nil {
			state.Env = make(map[string]string)
		}
		state.Env["CUSTOM_VALUE"] = "present"
		if state.Image.Config.Labels == nil {
			state.Image.Config.Labels = make(map[string]string)
		}
		state.Image.Config.Labels["example.com/custom"] = "present"
		return state, nil
	}, dockerfile.WithDescription("ANNOTATE custom metadata"), dockerfile.WithProgressGroup("custom"))
	df := dockerfile.New()
	if err := df.AddStage(dockerfile.From(dockerfile.Scratch(), dockerfile.As("custom")), custom); err != nil {
		t.Fatal(err)
	}
	result, err := df.Build(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if got := result.Target.Env["CUSTOM_VALUE"]; got != "present" {
		t.Fatalf("closure did not update state environment: %q", got)
	}
	if got := result.Target.Image.Config.Labels["example.com/custom"]; got != "present" {
		t.Fatalf("closure did not update OCI image config: %q", got)
	}
}

func TestInstructionApplyRollsBackStateOnError(t *testing.T) {
	wantErr := errors.New("deliberate failure")
	action := dockerfile.NewInstruction("FAIL", func(_ context.Context, state dockerfile.State, _ *dockerfile.Runtime, _ dockerfile.InstructionMetadata) (dockerfile.State, error) {
		state.Env["LEAK"] = "no"
		if state.Image.Config.Labels == nil {
			state.Image.Config.Labels = make(map[string]string)
		}
		state.Image.Config.Labels["leak"] = "no"
		state.LLB = state.LLB.AddEnv("LEAK", "no")
		return state, wantErr
	})
	initial := dockerfile.State{
		LLB: llb.Scratch(), Image: testImage(),
		Env: map[string]string{"KEEP": "yes"},
	}
	next, err := action.Apply(context.Background(), initial, nil)
	if !errors.Is(err, wantErr) {
		t.Fatalf("error = %v, want %v", err, wantErr)
	}
	if _, ok := initial.Env["LEAK"]; ok {
		t.Fatal("failed instruction mutated caller's state")
	}
	if _, ok := next.Env["LEAK"]; ok {
		t.Fatal("failed instruction returned partially-mutated state")
	}
	if got := next.Env["KEEP"]; got != "yes" {
		t.Fatalf("existing state was not preserved: %q", got)
	}
}

func TestInstructionAndStageHooksAreScoped(t *testing.T) {
	var sequence []string
	action := dockerfile.NewInstruction("MARK", func(_ context.Context, state dockerfile.State, _ *dockerfile.Runtime, _ dockerfile.InstructionMetadata) (dockerfile.State, error) {
		state.Image.Author = "test"
		return state, nil
	}, dockerfile.WithInstructionHooks(dockerfile.InstructionHooks{
		Before: func(context.Context, dockerfile.InstructionEvent) error {
			sequence = append(sequence, "instruction-before")
			return nil
		},
		After: func(context.Context, dockerfile.InstructionEvent) error {
			sequence = append(sequence, "instruction-after")
			return nil
		},
	}))
	df := dockerfile.New()
	if err := df.AddStagePlan(dockerfile.Plan(
		dockerfile.From(dockerfile.Scratch(), dockerfile.As("hooks")),
		[]dockerfile.Instruction{action},
		dockerfile.WithStageHooks(dockerfile.StageHooks{
			Before: func(context.Context, dockerfile.StageEvent) error {
				sequence = append(sequence, "stage-before")
				return nil
			},
			After: func(context.Context, dockerfile.StageEvent) error {
				sequence = append(sequence, "stage-after")
				return nil
			},
		}),
	)); err != nil {
		t.Fatal(err)
	}
	if _, err := df.Build(context.Background()); err != nil {
		t.Fatal(err)
	}
	want := []string{"stage-before", "instruction-before", "instruction-after", "stage-after"}
	if strings.Join(sequence, ",") != strings.Join(want, ",") {
		t.Fatalf("hook order = %v, want %v", sequence, want)
	}
}

func TestProgressIncludesStageAndInstructionTotals(t *testing.T) {
	var events []progress.Event
	df := dockerfile.New(dockerfile.WithProgressReporter(progress.ReporterFunc(func(_ context.Context, event progress.Event) error {
		events = append(events, event)
		return nil
	})))
	if err := df.AddStage(
		dockerfile.From(dockerfile.Scratch(), dockerfile.As("progress")),
		instruction.Env([]instruction.KeyValue{instruction.KV("A", "1")}),
		instruction.Run(instruction.Exec("true")),
	); err != nil {
		t.Fatal(err)
	}
	if _, err := df.Build(context.Background()); err != nil {
		t.Fatal(err)
	}
	started, completed := 0, 0
	for _, event := range events {
		if event.StageName != "progress" {
			continue
		}
		if event.Phase == progress.Started {
			started++
			if event.StageTotal != 1 {
				t.Fatalf("stage total = %d", event.StageTotal)
			}
		}
		if event.Phase == progress.Completed {
			completed++
			if event.InstructionTotal < 3 {
				t.Fatalf("instruction total = %d, want FROM + ENV + RUN", event.InstructionTotal)
			}
		}
	}
	if started == 0 || completed == 0 {
		t.Fatalf("expected started and completed events; got %d and %d", started, completed)
	}
}

func testPlatform() ocispecs.Platform {
	return ocispecs.Platform{OS: "linux", Architecture: "amd64"}
}

func testImage() dockerspec.DockerOCIImage {
	image := dockerspec.DockerOCIImage{}
	image.OS = "linux"
	image.Architecture = "amd64"
	image.RootFS.Type = "layers"
	image.Config.WorkingDir = "/"
	return image
}
