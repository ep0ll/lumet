package dockerfile

import (
	"context"
	"syscall"
	"testing"
	"time"

	"github.com/moby/buildkit/client/llb"
	dockerspec "github.com/moby/docker-image-spec/specs-go/v1"
	ocispecs "github.com/opencontainers/image-spec/specs-go/v1"
)

func TestInstructionPipelineMetadata(t *testing.T) {
	ctx := context.Background()
	createdAt := time.Date(2025, time.January, 2, 3, 4, 5, 0, time.UTC)
	build := New(
		WithPlatform(ocispecs.Platform{OS: "linux", Architecture: "amd64"}),
		WithBuildArgs(map[string]string{"VERSION": "2.4.1"}),
		WithCreatedAt(createdAt),
	)

	err := build.Apply(ctx,
		From("scratch", As("runtime")),
		Arg("VERSION", ArgDefault("0.0.0")),
		Env(KV("APP_VERSION", "$VERSION"), KV("APP_HOME", "/srv/app")),
		Label(KV("org.opencontainers.image.version", "$APP_VERSION")),
		Maintainer("Example Maintainer"),
		Shell("/bin/sh", "-e", "-c"),
		Cmd(Exec("server", "--port", "8080")),
		Entrypoint(Exec("server")),
		Expose("8080", "9000-9001/udp"),
		Volume("/var/lib/app"),
		User("10001:10001"),
		Workdir("/srv/app"),
		StopSignal(syscall.Signal(15)),
		Healthcheck(
			Exec("server", "health"),
			HealthInterval(30*time.Second),
			HealthTimeout(5*time.Second),
			HealthStartPeriod(10*time.Second),
			HealthRetries(3),
		),
		OnBuild("RUN server migrate"),
	)
	if err != nil {
		t.Fatal(err)
	}

	image := build.Image
	if image.Config.Env[0] != "APP_VERSION=2.4.1" {
		t.Fatalf("unexpected environment: %#v", image.Config.Env)
	}
	if image.Config.Labels["org.opencontainers.image.version"] != "2.4.1" {
		t.Fatalf("label was not expanded: %#v", image.Config.Labels)
	}
	if image.Author != "Example Maintainer" {
		t.Fatalf("unexpected author: %q", image.Author)
	}
	if got := image.Config.Cmd; len(got) != 3 || got[0] != "server" {
		t.Fatalf("unexpected CMD: %#v", got)
	}
	if got := image.Config.Entrypoint; len(got) != 1 || got[0] != "server" {
		t.Fatalf("unexpected ENTRYPOINT: %#v", got)
	}
	if _, ok := image.Config.ExposedPorts["8080/tcp"]; !ok {
		t.Fatalf("TCP port was not exposed: %#v", image.Config.ExposedPorts)
	}
	if _, ok := image.Config.ExposedPorts["9001/udp"]; !ok {
		t.Fatalf("UDP port range was not expanded: %#v", image.Config.ExposedPorts)
	}
	if _, ok := image.Config.Volumes["/var/lib/app"]; !ok {
		t.Fatalf("volume was not recorded: %#v", image.Config.Volumes)
	}
	if image.Config.User != "10001:10001" || image.Config.WorkingDir != "/srv/app" {
		t.Fatalf("unexpected runtime config: user=%q workdir=%q", image.Config.User, image.Config.WorkingDir)
	}
	if image.Config.StopSignal != "15" {
		t.Fatalf("expected numeric standard-library signal, got %q", image.Config.StopSignal)
	}
	if health := image.Config.Healthcheck; health == nil || health.Retries != 3 || health.Interval != 30*time.Second {
		t.Fatalf("unexpected healthcheck: %#v", health)
	}
	if len(image.Config.OnBuild) != 1 || image.Config.OnBuild[0] != "RUN server migrate" {
		t.Fatalf("unexpected ONBUILD triggers: %#v", image.Config.OnBuild)
	}
	if image.Created == nil || !image.Created.Equal(createdAt) {
		t.Fatalf("created timestamp was not set: %#v", image.Created)
	}
	if len(image.History) != 14 {
		t.Fatalf("expected history for all instructions after FROM, got %d", len(image.History))
	}
	if stage, ok := build.Stage("RUNTIME"); !ok || stage.Name != "runtime" {
		t.Fatalf("current named stage is unavailable: %#v, %v", stage, ok)
	}
}

func TestBaseImageMetadataResolver(t *testing.T) {
	ctx := context.Background()
	platform := ocispecs.Platform{OS: "linux", Architecture: "amd64"}
	build := New(
		WithPlatform(platform),
		WithImageResolver(func(_ context.Context, reference string, got ocispecs.Platform) (dockerspec.DockerOCIImage, error) {
			if reference != "example/base:latest" || got.OS != platform.OS {
				t.Fatalf("unexpected resolver input: %q %#v", reference, got)
			}
			image := emptyImage(platform)
			image.Config.Env = []string{"PATH=/usr/bin"}
			image.Config.WorkingDir = "/workspace"
			image.Config.Shell = []string{"/bin/bash", "-c"}
			return image, nil
		}),
	)
	if err := build.Apply(ctx, From("example/base:latest")); err != nil {
		t.Fatal(err)
	}
	if build.Image.Config.WorkingDir != "/workspace" || len(build.Image.Config.Shell) != 2 {
		t.Fatalf("base image config was not inherited: %#v", build.Image.Config)
	}
	if err := build.Apply(ctx, Env(KV("PATH", "/opt/bin:$PATH"))); err != nil {
		t.Fatal(err)
	}
	if got := build.Image.Config.Env[0]; got != "PATH=/opt/bin:/usr/bin" {
		t.Fatalf("base environment was not inherited and expanded: %q", got)
	}
}

func TestStagesAndCopyFrom(t *testing.T) {
	build := New(
		WithPlatform(ocispecs.Platform{OS: "linux", Architecture: "amd64"}),
		WithBuildContext(llb.Local("source-context")),
	)
	err := build.Apply(context.Background(),
		From("scratch", As("build")),
		Workdir("/out"),
		From("build", As("runtime")),
		Copy([]string{"bin/app"}, "/usr/local/bin/", CopyFrom("BUILD")),
	)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := build.Stage("build"); !ok {
		t.Fatal("completed build stage was not retained")
	}
	if _, ok := build.Stage("runtime"); !ok {
		t.Fatal("current runtime stage was not exposed")
	}
}

func TestRunCopyAndAddInstructionsCompose(t *testing.T) {
	build := New(
		WithPlatform(ocispecs.Platform{OS: "linux", Architecture: "amd64"}),
		WithBuildContext(llb.Local("context")),
	)
	err := build.Apply(context.Background(),
		From("scratch"),
		Copy([]string{"./src/main.go"}, "/app/", Parents(), Chmod(0o644)),
		Copy([]string{"assets"}, "/app/", Link(), Chown("1000:1000")),
		Copy([]string{"/etc/os-release"}, "/etc/", CopyFrom("busybox:latest")),
		Add([]AddSource{
			LocalSource("archive.tar"),
			RemoteSource("https://example.com/assets.tar"),
			GitSource("https://github.com/example/assets.git", "main", llb.KeepGitDir()),
		}, "/opt/assets/", Unpack(true)),
		Run(Sh("echo ready")),
		Run(Exec("true"), CustomName("health probe")),
	)
	if err != nil {
		t.Fatal(err)
	}
	if len(build.Image.History) != 6 {
		t.Fatalf("expected three COPY operations, ADD and two RUN history entries after FROM, got %d", len(build.Image.History))
	}
	definition, err := build.State.Marshal(context.Background())
	if err != nil {
		t.Fatalf("marshal LLB definition: %v", err)
	}
	if len(definition.Def) == 0 {
		t.Fatal("expected a non-empty LLB definition")
	}
}

func TestInvalidInstructionsReturnErrors(t *testing.T) {
	build := New()
	if err := build.Apply(context.Background(), Env(KV("A", "b"))); err == nil {
		t.Fatal("ENV before FROM should fail")
	}
	if err := build.Apply(context.Background(), From("scratch"), Expose("70000/tcp")); err == nil {
		t.Fatal("invalid port should fail")
	}
	if err := build.Apply(context.Background(), OnBuild("FROM alpine")); err == nil {
		t.Fatal("ONBUILD FROM should fail")
	}
	if err := build.Apply(context.Background(), Healthcheck(Exec())); err == nil {
		t.Fatal("empty healthcheck exec form should fail")
	}
}

func TestInstructionOptionsCanBeExtended(t *testing.T) {
	custom := func(options *InstructionOptions) {
		options.CustomName = "custom-progress-name"
		options.RunOptions = append(options.RunOptions, llb.WithCustomName("custom-progress-name"))
	}
	build := New()
	if err := build.Apply(context.Background(),
		From("scratch"),
		Run(Sh("true"), custom),
	); err != nil {
		t.Fatal(err)
	}
}


func TestArgScopeAcrossStages(t *testing.T) {
	ctx := context.Background()
	platform := ocispecs.Platform{OS: "linux", Architecture: "amd64"}
	var resolvedReference string
	build := New(
		WithPlatform(platform),
		WithBuildArgs(map[string]string{
			"BASE_IMAGE": "example/base:1.0",
		}),
		WithImageResolver(func(_ context.Context, reference string, _ ocispecs.Platform) (dockerspec.DockerOCIImage, error) {
			resolvedReference = reference
			return emptyImage(platform), nil
		}),
	)

	err := build.Apply(ctx,
		Arg("BASE_IMAGE", ArgDefault("example/fallback:latest")),
		From("$BASE_IMAGE", As("build")),
		Arg("BUILD_ONLY", ArgDefault("secret")),
		Env(KV("PUBLIC_VALUE", "visible")),
		From("build", As("runtime")),
	)
	if err != nil {
		t.Fatal(err)
	}
	if resolvedReference != "example/base:1.0" {
		t.Fatalf("FROM did not use global ARG: %q", resolvedReference)
	}
	if _, ok, err := build.State.GetEnv(ctx, "BUILD_ONLY"); err != nil {
		t.Fatal(err)
	} else if ok {
		t.Fatal("stage-local ARG leaked into the next stage")
	}
	if value, ok, err := build.State.GetEnv(ctx, "PUBLIC_VALUE"); err != nil {
		t.Fatal(err)
	} else if !ok || value != "visible" {
		t.Fatalf("ENV should survive FROM from a previous stage: value=%q present=%v", value, ok)
	}
}
