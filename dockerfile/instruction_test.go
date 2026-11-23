package dockerfile_test

import (
	"context"
	"errors"
	"fmt"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/ep0ll/lumet/dockerfile"
	"github.com/moby/buildkit/client/llb"
	"github.com/moby/buildkit/solver/pb"
	ocispecs "github.com/opencontainers/image-spec/specs-go/v1"
)

func TestDefaultRegistryAndBuiltinInstructionSet(t *testing.T) {
	registry, err := dockerfile.NewDefaultRegistry()
	if err != nil {
		t.Fatalf("NewDefaultRegistry() error = %v", err)
	}
	want := []string{
		"ADD", "ARG", "CMD", "COPY", "ENTRYPOINT", "ENV", "EXPOSE",
		"FROM", "HEALTHCHECK", "LABEL", "MAINTAINER", "ONBUILD", "RUN",
		"SHELL", "STOPSIGNAL", "USER", "VOLUME", "WORKDIR",
	}
	definitions := registry.ListInstructions()
	if len(definitions) != len(want) {
		t.Fatalf("instruction count = %d, want %d", len(definitions), len(want))
	}
	for index, name := range want {
		if definitions[index].Name != name {
			t.Errorf("instruction[%d] = %q, want %q", index, definitions[index].Name, name)
		}
	}

	for instruction, names := range map[string][]string{
		"FROM":        {"platform"},
		"RUN":         {"mount", "network", "security", "device"},
		"COPY":        {"from", "chown", "chmod", "link", "parents", "exclude"},
		"ADD":         {"chown", "chmod", "link", "exclude", "keep-git-dir", "checksum", "unpack"},
		"HEALTHCHECK": {"interval", "timeout", "start-period", "start-interval", "retries"},
	} {
		flags, err := registry.ListFlags(instruction)
		if err != nil {
			t.Fatalf("ListFlags(%s) error = %v", instruction, err)
		}
		for _, name := range names {
			if findFlag(flags, name) < 0 {
				t.Errorf("%s is missing --%s", instruction, name)
			}
		}
	}
}

func TestInstructionAndFlagCRUD(t *testing.T) {
	registry := dockerfile.NewRegistry()
	minimum, maximum := int64(1), int64(16)
	if err := registry.CreateInstruction(dockerfile.InstructionDefinition{
		Name:        "BUILD",
		Aliases:     []string{"COMPILE"},
		Kind:        dockerfile.KindCustom,
		Description: "Build an artifact.",
		Flags: []dockerfile.FlagDefinition{
			{Name: "jobs", Aliases: []string{"j"}, Type: dockerfile.FlagInt, Default: 4, MinInt: &minimum, MaxInt: &maximum},
			{Name: "tag", Type: dockerfile.FlagStringSlice, Repeatable: true},
			{Name: "cache", Type: dockerfile.FlagBool, Default: true},
		},
	}); err != nil {
		t.Fatalf("CreateInstruction() error = %v", err)
	}

	definition, err := registry.GetInstruction("compile")
	if err != nil {
		t.Fatalf("GetInstruction(alias) error = %v", err)
	}
	if definition.Name != "BUILD" {
		t.Fatalf("alias resolved to %q, want BUILD", definition.Name)
	}
	instance, err := registry.New("BUILD")
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	if value, ok := instance.Flags().Int("j"); !ok || value != 4 {
		t.Fatalf("default --jobs = %d, present = %v; want 4, true", value, ok)
	}
	aliasConfigured, err := registry.NewWithOptions("BUILD", nil, dockerfile.WithFlag("j", 6))
	if err != nil {
		t.Fatalf("NewWithOptions(alias flag) error = %v", err)
	}
	if value, ok := aliasConfigured.Flags().Int("jobs"); !ok || value != 6 {
		t.Fatalf("option alias override = %d, present = %v; want 6, true", value, ok)
	}

	arguments, err := instance.ParseTokens(
		"--j=8",
		"--tag", "release",
		"--tag=stable",
		"--no-cache",
	)
	if err != nil {
		t.Fatalf("ParseTokens() error = %v", err)
	}
	if len(arguments) != 0 {
		t.Fatalf("arguments = %v, want none", arguments)
	}
	if value, ok := instance.Flags().Int("jobs"); !ok || value != 8 {
		t.Fatalf("--jobs = %d, present = %v; want 8, true", value, ok)
	}
	if value, ok := instance.Flags().Strings("tag"); !ok || fmt.Sprint(value) != "[release stable]" {
		t.Fatalf("--tag = %v, present = %v", value, ok)
	}
	if value, ok := instance.Flags().Bool("cache"); !ok || value {
		t.Fatalf("--cache = %v, present = %v; want false, true", value, ok)
	}

	oldGeneration := registry.Generation()
	if err := registry.UpdateFlag("BUILD", "jobs", func(flag *dockerfile.FlagDefinition) error {
		flag.Name = "parallelism"
		flag.Aliases = []string{"j"}
		limit := int64(8)
		flag.MaxInt = &limit
		return nil
	}); err != nil {
		t.Fatalf("UpdateFlag() error = %v", err)
	}
	if registry.Generation() <= oldGeneration {
		t.Fatal("registry generation did not advance")
	}

	// An existing instruction retains its schema snapshot.
	if value, ok := instance.Flags().Int("jobs"); !ok || value != 8 {
		t.Fatalf("existing instruction lost old flag: %d, %v", value, ok)
	}
	updated, err := registry.New("COMPILE")
	if err != nil {
		t.Fatalf("New() after update error = %v", err)
	}
	if err := updated.SetFlag("parallelism", 9); err == nil {
		t.Fatal("SetFlag() accepted a value above the new maximum")
	}
	if err := updated.SetFlag("j", 8); err != nil {
		t.Fatalf("SetFlag(alias) error = %v", err)
	}

	if _, err := registry.GetFlag("BUILD", "parallelism"); err != nil {
		t.Fatalf("GetFlag() error = %v", err)
	}
	if err := registry.DeleteFlag("BUILD", "j"); err != nil {
		t.Fatalf("DeleteFlag(alias) error = %v", err)
	}
	if _, err := registry.GetFlag("BUILD", "parallelism"); !errors.Is(err, dockerfile.ErrNotFound) {
		t.Fatalf("GetFlag() after delete error = %v, want ErrNotFound", err)
	}

	if err := registry.UpdateInstruction("BUILD", func(def *dockerfile.InstructionDefinition) error {
		def.Name = "COMPILE_TASK"
		def.Description = "Renamed operation."
		return nil
	}); err != nil {
		t.Fatalf("UpdateInstruction(rename) error = %v", err)
	}
	if _, err := registry.GetInstruction("BUILD"); !errors.Is(err, dockerfile.ErrNotFound) {
		t.Fatalf("old name error = %v, want ErrNotFound", err)
	}
	if definition, err := registry.GetInstruction("COMPILE_TASK"); err != nil || definition.Description != "Renamed operation." {
		t.Fatalf("renamed definition = %#v, error = %v", definition, err)
	}
	if err := registry.DeleteInstruction("COMPILE"); err != nil {
		t.Fatalf("DeleteInstruction(alias) error = %v", err)
	}
	if _, err := registry.GetInstruction("COMPILE_TASK"); !errors.Is(err, dockerfile.ErrNotFound) {
		t.Fatalf("GetInstruction() after delete error = %v, want ErrNotFound", err)
	}
}

func TestInstructionFlagValueLifecycle(t *testing.T) {
	registry, err := dockerfile.NewDefaultRegistry()
	if err != nil {
		t.Fatal(err)
	}
	instruction, err := registry.New("HEALTHCHECK", "CMD", "true")
	if err != nil {
		t.Fatal(err)
	}

	if err := instruction.SetFlag("retries", 5); err != nil {
		t.Fatalf("SetFlag() error = %v", err)
	}
	if value, ok := instruction.Flags().Int("retries"); !ok || value != 5 {
		t.Fatalf("retries = %d, present = %v; want 5, true", value, ok)
	}
	if err := instruction.DeleteFlagValue("retries"); err != nil {
		t.Fatalf("DeleteFlagValue() error = %v", err)
	}
	if _, ok := instruction.Flags().Int("retries"); ok {
		t.Fatal("retries value remains after DeleteFlagValue")
	}
	if err := instruction.ResetFlag("retries"); err != nil {
		t.Fatalf("ResetFlag() error = %v", err)
	}
	if value, ok := instruction.Flags().Int("retries"); !ok || value != 3 {
		t.Fatalf("reset retries = %d, present = %v; want default 3, true", value, ok)
	}
	if err := instruction.SetFlag("retries", 6); err != nil {
		t.Fatal(err)
	}
	if err := instruction.ResetFlags(); err != nil {
		t.Fatalf("ResetFlags() error = %v", err)
	}
	if value, ok := instruction.Flags().Int("retries"); !ok || value != 3 {
		t.Fatalf("ResetFlags retries = %d, present = %v; want 3, true", value, ok)
	}
}

func TestAddingFlagToExistingInstructionIsAdditive(t *testing.T) {
	registry, err := dockerfile.NewDefaultRegistry()
	if err != nil {
		t.Fatal(err)
	}
	oldInstruction, err := registry.New("RUN", "go", "test", "./...")
	if err != nil {
		t.Fatal(err)
	}
	if err := registry.CreateFlag("RUN", dockerfile.FlagDefinition{
		Name: "cache-scope", Type: dockerfile.FlagEnum,
		AllowedValues: []string{"shared", "private"}, Default: "shared",
	}); err != nil {
		t.Fatalf("CreateFlag() error = %v", err)
	}

	// Existing instances keep their original definition snapshot.
	if err := oldInstruction.SetFlag("cache-scope", "private"); err == nil {
		t.Fatal("old instruction unexpectedly acquired a newly-added flag")
	}
	newInstruction, err := registry.New("RUN", "go", "test", "./...")
	if err != nil {
		t.Fatal(err)
	}
	if err := newInstruction.SetFlag("cache-scope", "private"); err != nil {
		t.Fatalf("new flag not available on new instruction: %v", err)
	}
	if value, ok := newInstruction.Flags().String("cache-scope"); !ok || value != "private" {
		t.Fatalf("new flag value = %q, present = %v", value, ok)
	}
}

func TestGenericOptionsAndNativeBuildKitIntegration(t *testing.T) {
	registry, err := dockerfile.NewDefaultRegistry()
	if err != nil {
		t.Fatal(err)
	}

	customOption := func(value string) dockerfile.Option {
		return func(options *dockerfile.InstructionOptions) error {
			options.SetExtension("example.com/cache-policy", value)
			options.SetPayload(map[string]any{"owner": "builder"})
			options.AddBuildKitOption(dockerfile.LLBConstraints(llb.WithCustomName("RUN go test")))
			return nil
		}
	}

	instruction, err := registry.NewWithOptions("RUN", []string{"go", "test", "./..."},
		dockerfile.WithFlag("network", "none"),
		dockerfile.WithFlag("mount", []string{"type=cache,target=/root/.cache"}),
		dockerfile.WithExtension("example.com/retries", 2),
		dockerfile.WithLLBRunOptions(llb.Args([]string{"go", "test", "./..."})),
		dockerfile.WithLLBNetworkMode(pb.NetMode_NONE),
		customOption("locked"),
	)
	if err != nil {
		t.Fatalf("NewWithOptions() error = %v", err)
	}
	if value, ok := instruction.Flags().String("network"); !ok || value != "none" {
		t.Fatalf("--network = %q, present = %v", value, ok)
	}
	if value, ok := instruction.Flags().Strings("mount"); !ok || len(value) != 1 {
		t.Fatalf("--mount = %v, present = %v", value, ok)
	}
	if value, ok := instruction.Extension("example.com/cache-policy"); !ok || value != "locked" {
		t.Fatalf("extension = %v, present = %v", value, ok)
	}
	buildOptions := instruction.BuildKitOptions()
	network, hasNetwork := buildOptions.NetworkMode()
	if len(buildOptions.RunOptions()) != 1 || !hasNetwork || network != pb.NetMode_NONE {
		t.Fatalf("BuildKit options = %#v", buildOptions)
	}
}

func TestParseTokensStopsAtFirstPositionalAndPrefersExactNoPrefixFlag(t *testing.T) {
	registry := dockerfile.NewRegistry()
	if err := registry.CreateInstruction(dockerfile.InstructionDefinition{
		Name: "OPTIONS",
		Kind: dockerfile.KindCustom,
		Arguments: []dockerfile.ArgumentDefinition{{Name: "command", Required: true, Variadic: true}},
		Flags: []dockerfile.FlagDefinition{
			{Name: "cache", Type: dockerfile.FlagBool, Default: true},
			{Name: "no-cache", Type: dockerfile.FlagBool},
		},
	}); err != nil {
		t.Fatal(err)
	}
	instruction, err := registry.New("OPTIONS")
	if err != nil {
		t.Fatal(err)
	}
	arguments, err := instruction.ParseTokens("--no-cache", "echo", "--version")
	if err != nil {
		t.Fatalf("ParseTokens() error = %v", err)
	}
	if fmt.Sprint(arguments) != "[echo --version]" {
		t.Fatalf("arguments = %v, want [echo --version]", arguments)
	}
	if value, ok := instruction.Flags().Bool("no-cache"); !ok || !value {
		t.Fatalf("--no-cache exact flag = %v, present = %v", value, ok)
	}
	if value, ok := instruction.Flags().Bool("cache"); !ok || !value {
		t.Fatalf("cache flag = %v, present = %v; want default true", value, ok)
	}
}

func TestParseTokensAllowsCommandOptionsAndFlaglessArguments(t *testing.T) {
	registry, err := dockerfile.NewDefaultRegistry()
	if err != nil {
		t.Fatal(err)
	}

	run, err := registry.New("RUN")
	if err != nil {
		t.Fatal(err)
	}
	arguments, err := run.ParseTokens("-v", "go", "test", "--version")
	if err != nil {
		t.Fatalf("ParseTokens(RUN) error = %v", err)
	}
	if fmt.Sprint(arguments) != "[-v go test --version]" {
		t.Fatalf("RUN arguments = %v, want [-v go test --version]", arguments)
	}

	cmd, err := registry.New("CMD")
	if err != nil {
		t.Fatal(err)
	}
	arguments, err = cmd.ParseTokens("--version", "-v")
	if err != nil {
		t.Fatalf("ParseTokens(CMD) error = %v", err)
	}
	if fmt.Sprint(arguments) != "[--version -v]" {
		t.Fatalf("CMD arguments = %v, want [--version -v]", arguments)
	}
}

func TestOptionFactoriesSnapshotCallerValues(t *testing.T) {
	registry, err := dockerfile.NewDefaultRegistry()
	if err != nil {
		t.Fatal(err)
	}

	mounts := []string{"type=cache,target=/cache/original"}
	flagValues := map[string]any{"mount": mounts}
	flagsOption := dockerfile.WithFlags(flagValues)
	mounts[0] = "type=cache,target=/cache/mutated"
	flagValues["mount"] = []string{"type=cache,target=/cache/replaced"}

	run, err := registry.NewWithOptions("RUN", nil, flagsOption)
	if err != nil {
		t.Fatalf("NewWithOptions(RUN) error = %v", err)
	}
	gotMounts, ok := run.Flags().Strings("mount")
	if !ok || fmt.Sprint(gotMounts) != "[type=cache,target=/cache/original]" {
		t.Fatalf("mount snapshot = %v, present = %v", gotMounts, ok)
	}

	args := []string{"echo", "original"}
	command := dockerfile.Command{Form: dockerfile.CommandExec, Args: args}
	commandOption := dockerfile.WithCommand(command)
	args[1] = "mutated"
	command.Args[0] = "replaced"

	cmd, err := registry.NewWithOptions("CMD", nil, commandOption)
	if err != nil {
		t.Fatalf("NewWithOptions(CMD) error = %v", err)
	}
	payload, ok := cmd.Payload().(dockerfile.Command)
	if !ok || fmt.Sprint(payload.Args) != "[echo original]" {
		t.Fatalf("command payload = %#v", cmd.Payload())
	}

	source := dockerfile.Context("original-context")
	sourceOption := dockerfile.WithSource(source)
	source.Reference = "mutated-context"
	from, err := registry.NewWithOptions("FROM", nil, sourceOption)
	if err != nil {
		t.Fatalf("NewWithOptions(FROM) error = %v", err)
	}
	sourcePayload, ok := from.Payload().(dockerfile.Source)
	if !ok || sourcePayload.Reference != "original-context" {
		t.Fatalf("source payload = %#v", from.Payload())
	}
}

func TestCustomFlagParserAndTransactionalParsing(t *testing.T) {
	type resourceLimit struct{ CPUMilli int }

	registry := dockerfile.NewRegistry()
	if err := registry.CreateInstruction(dockerfile.InstructionDefinition{
		Name: "TUNE", Kind: dockerfile.KindCustom,
		Flags: []dockerfile.FlagDefinition{
			{
				Name: "cpu", Type: dockerfile.FlagCustom,
				Parse: func(raw string) (any, error) {
					var milli int
					if _, err := fmt.Sscanf(raw, "%dm", &milli); err != nil {
						return nil, err
					}
					return resourceLimit{CPUMilli: milli}, nil
				},
				Validate: func(value any) error {
					limit, ok := value.(resourceLimit)
					if !ok {
						return fmt.Errorf("unexpected value type %T", value)
					}
					if limit.CPUMilli < 100 {
						return fmt.Errorf("CPU limit must be at least 100m")
					}
					return nil
				},
			},
			{Name: "cache", Type: dockerfile.FlagBool},
		},
	}); err != nil {
		t.Fatal(err)
	}
	instruction, err := registry.New("TUNE")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := instruction.ParseTokens("--cache", "--unknown"); err == nil {
		t.Fatal("ParseTokens accepted an unknown flag")
	}
	if _, exists := instruction.Flag("cache"); exists {
		t.Fatal("ParseTokens left a partial flag mutation after failure")
	}
	if _, err := instruction.ParseTokens("--cache", "--cpu=500m"); err != nil {
		t.Fatalf("ParseTokens() error = %v", err)
	}
	value, ok := instruction.Flags().Get("cpu")
	if !ok {
		t.Fatal("custom flag value missing")
	}
	limit, ok := value.(resourceLimit)
	if !ok || limit.CPUMilli != 500 {
		t.Fatalf("custom flag value = %#v", value)
	}
}

func TestIncompleteInstructionCanBeConfiguredBeforeValidation(t *testing.T) {
	registry := dockerfile.NewRegistry()
	minimum := int64(1)
	if err := registry.CreateInstruction(dockerfile.InstructionDefinition{
		Name: "PUBLISH",
		Kind: dockerfile.KindCustom,
		Arguments: []dockerfile.ArgumentDefinition{{Name: "image", Required: true}},
		Flags: []dockerfile.FlagDefinition{{
			Name: "replicas", Type: dockerfile.FlagInt, Required: true, MinInt: &minimum,
		}},
	}); err != nil {
		t.Fatal(err)
	}

	instruction, err := registry.New("PUBLISH")
	if err != nil {
		t.Fatalf("New() should create an incomplete instance: %v", err)
	}
	if err := instruction.Validate(); err == nil {
		t.Fatal("Validate() accepted missing arguments and required flags")
	}
	if err := instruction.SetArguments("registry.example/app:latest"); err != nil {
		t.Fatalf("SetArguments() error = %v", err)
	}
	if err := instruction.SetFlag("replicas", 3); err != nil {
		t.Fatalf("SetFlag() error = %v", err)
	}
	if err := instruction.Validate(); err != nil {
		t.Fatalf("Validate() after configuration error = %v", err)
	}
}

func TestCopyPositionalArgumentsAndInstructionClasses(t *testing.T) {
	registry, err := dockerfile.NewDefaultRegistry()
	if err != nil {
		t.Fatal(err)
	}
	copyInstruction, err := registry.New("COPY", "src/a", "src/b", "/app/")
	if err != nil {
		t.Fatalf("New(COPY) error = %v", err)
	}
	positionals := copyInstruction.PositionalArguments()
	if fmt.Sprint(positionals["source"]) != "[src/a src/b]" || fmt.Sprint(positionals["destination"]) != "[/app/]" {
		t.Fatalf("COPY positionals = %#v", positionals)
	}

	from, err := registry.New("FROM", "alpine")
	if err != nil {
		t.Fatal(err)
	}
	run, err := registry.New("RUN", "echo", "hello")
	if err != nil {
		t.Fatal(err)
	}
	if from.Class() != dockerfile.InstructionStageInitializer {
		t.Fatal("FROM is not marked as a stage initializer")
	}
	if run.Class() != dockerfile.InstructionStandard {
		t.Fatal("RUN is not a standard instruction")
	}
}

func TestHandlersInterceptorsHistoryAndCustomInstruction(t *testing.T) {
	registry, err := dockerfile.NewDefaultRegistry()
	if err != nil {
		t.Fatal(err)
	}
	if err := registry.CreateInstruction(dockerfile.InstructionDefinition{
		Name: "ANNOTATE", Kind: dockerfile.KindCustom, OnBuildable: true,
		Arguments: []dockerfile.ArgumentDefinition{{Name: "value", Required: true}},
		HistoryMode: dockerfile.HistoryEmptyLayer,
	}); err != nil {
		t.Fatal(err)
	}
	trigger, err := registry.New("ANNOTATE", "release")
	if err != nil {
		t.Fatal(err)
	}
	onBuild, err := registry.NewOnBuild(trigger)
	if err != nil {
		t.Fatal(err)
	}

	handlers := dockerfile.NewHandlers()
	var order []string
	if err := handlers.RegisterHandler("FROM", func(_ context.Context, call dockerfile.Call) (dockerfile.Stage, error) {
		return dockerfile.Stage{Name: dockerfile.StageName(call.Arguments[0])}, nil
	}); err != nil {
		t.Fatal(err)
	}
	if err := handlers.RegisterHandler("ANNOTATE", func(_ context.Context, call dockerfile.Call) (dockerfile.Stage, error) {
		stage := call.Stage
		stage.Name += dockerfile.StageName("-" + call.Arguments[0])
		return stage, nil
	}); err != nil {
		t.Fatal(err)
	}
	if err := handlers.RegisterHandler("ONBUILD", func(ctx context.Context, call dockerfile.Call) (dockerfile.Stage, error) {
		nested, ok := call.Payload.(*dockerfile.Instruction)
		if !ok {
			return call.Stage, fmt.Errorf("unexpected ONBUILD payload %T", call.Payload)
		}
		return call.Handlers.Apply(ctx, call.Stage, nested)
	}); err != nil {
		t.Fatal(err)
	}
	if err := handlers.RegisterInterceptor("trace", func(ctx context.Context, call dockerfile.Call, next dockerfile.Handler) (dockerfile.Stage, error) {
		order = append(order, "before:"+call.Definition.Name)
		stage, err := next(ctx, call)
		order = append(order, "after:"+call.Definition.Name)
		return stage, err
	}); err != nil {
		t.Fatal(err)
	}

	from, err := registry.New("FROM", "alpine")
	if err != nil {
		t.Fatal(err)
	}
	stage, err := dockerfile.Compose(context.Background(), handlers, from, onBuild)
	if err != nil {
		t.Fatalf("Compose() error = %v", err)
	}
	if stage.Name != "alpine-release" {
		t.Fatalf("stage name = %q", stage.Name)
	}
	if len(stage.Image.History) != 3 || !stage.Image.History[1].EmptyLayer || !stage.Image.History[2].EmptyLayer {
		t.Fatalf("history = %#v", stage.Image.History)
	}
	if fmt.Sprint(order) != "[before:FROM after:FROM before:ONBUILD before:ANNOTATE after:ANNOTATE after:ONBUILD]" {
		t.Fatalf("interceptor order = %v", order)
	}

	if err := handlers.UpdateHandler("ANNOTATE", func(_ context.Context, call dockerfile.Call) (dockerfile.Stage, error) {
		return call.Stage, nil
	}); err != nil {
		t.Fatalf("UpdateHandler() error = %v", err)
	}
	if err := handlers.DeleteHandler("ANNOTATE"); err != nil {
		t.Fatalf("DeleteHandler() error = %v", err)
	}
	if _, ok := handlers.GetHandler("ANNOTATE"); ok {
		t.Fatal("deleted handler remains registered")
	}
}

func TestHistoryOverridesAndBuildKitNetworkParsers(t *testing.T) {
	registry, err := dockerfile.NewDefaultRegistry()
	if err != nil {
		t.Fatal(err)
	}
	run, err := registry.NewWithOptions("RUN", []string{"true"},
		dockerfile.WithHistory(func(_ context.Context, _ dockerfile.HistoryEvent) (ocispecs.History, error) {
			return ocispecs.History{CreatedBy: "custom"}, nil
		}),
	)
	if err != nil {
		t.Fatal(err)
	}
	handlers := dockerfile.NewHandlers()
	if err := handlers.RegisterHandler("RUN", func(_ context.Context, call dockerfile.Call) (dockerfile.Stage, error) {
		return call.Stage, nil
	}); err != nil {
		t.Fatal(err)
	}
	stage, err := handlers.Apply(context.Background(), dockerfile.Stage{}, run)
	if err != nil {
		t.Fatal(err)
	}
	if len(stage.Image.History) != 1 || stage.Image.History[0].CreatedBy != "custom" {
		t.Fatalf("history = %#v", stage.Image.History)
	}

	network, err := dockerfile.ParseLLBNetworkMode("none")
	if err != nil || network != pb.NetMode_NONE {
		t.Fatalf("ParseLLBNetworkMode() = %v, %v", network, err)
	}
	security, err := dockerfile.ParseLLBSecurityMode("insecure")
	if err != nil || security != pb.SecurityMode_INSECURE {
		t.Fatalf("ParseLLBSecurityMode() = %v, %v", security, err)
	}
}

func TestZeroValueAndConcurrentRegistryUse(t *testing.T) {
	var registry dockerfile.Registry
	if err := registry.CreateInstruction(dockerfile.InstructionDefinition{Name: "TASK", Kind: dockerfile.KindCustom}); err != nil {
		t.Fatalf("zero-value Registry.CreateInstruction() error = %v", err)
	}
	var handlers dockerfile.Handlers
	if err := handlers.RegisterHandler("TASK", func(_ context.Context, call dockerfile.Call) (dockerfile.Stage, error) {
		return call.Stage, nil
	}); err != nil {
		t.Fatalf("zero-value Handlers.RegisterHandler() error = %v", err)
	}

	if err := registry.CreateFlag("TASK", dockerfile.FlagDefinition{Name: "workers", Type: dockerfile.FlagInt, Default: 1}); err != nil {
		t.Fatal(err)
	}
	var group sync.WaitGroup
	for worker := 0; worker < 8; worker++ {
		group.Add(1)
		go func(worker int) {
			defer group.Done()
			for iteration := 0; iteration < 25; iteration++ {
				_, _ = registry.GetInstruction("TASK")
				_, _ = registry.GetFlag("TASK", "workers")
				_ = registry.UpdateFlag("TASK", "workers", func(flag *dockerfile.FlagDefinition) error {
					flag.Description = fmt.Sprintf("worker-%d", worker)
					return nil
				})
			}
		}(worker)
	}
	group.Wait()
	if _, err := registry.New("TASK"); err != nil {
		t.Fatalf("Registry.New() after concurrent updates: %v", err)
	}
}

func TestHealthcheckDefaultsAndOSSignalPayload(t *testing.T) {
	registry, err := dockerfile.NewDefaultRegistry()
	if err != nil {
		t.Fatal(err)
	}
	healthcheck, err := registry.New("HEALTHCHECK", "CMD", "curl", "-f", "http://localhost")
	if err != nil {
		t.Fatal(err)
	}
	if value, ok := healthcheck.Flags().Duration("interval"); !ok || value != 30*time.Second {
		t.Fatalf("healthcheck interval = %v, present = %v", value, ok)
	}
	instruction, err := registry.NewWithOptions("STOPSIGNAL", []string{"SIGTERM"}, dockerfile.WithOSSignal(os.Interrupt))
	if err != nil {
		t.Fatal(err)
	}
	if signal, ok := instruction.Payload().(os.Signal); !ok || signal == nil {
		t.Fatalf("STOPSIGNAL payload = %#v; want os.Signal", instruction.Payload())
	}
}

func findFlag(flags []dockerfile.FlagDefinition, name string) int {
	for index, flag := range flags {
		if flag.Name == name {
			return index
		}
	}
	return -1
}
