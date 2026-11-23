# Dockerfile instruction model

The `dockerfile` package uses one data-driven instruction model. Built-in and
application-defined instructions share the same definition registry, flag
schema, generic options, validation, handler registry, and interceptor chain.

The important extension rule is that instruction behavior is selected by name,
not by adding a new public field, enum value, constructor option interface, or
sealed marker interface. A future Dockerfile flag is a definition change. A
future instruction is a definition plus a handler.

## Create instructions

Use a registry for applications that want to customize instruction schemas:

```go
registry, err := dockerfile.NewDefaultRegistry()
if err != nil {
    return err
}

// Definitions validate argument and flag schemas when registered. Instruction
// instances can be assembled in stages and validated before execution.
run, err := registry.NewWithOptions(
    "RUN",
    []string{"go", "test", "./..."},
    dockerfile.WithFlag("network", "none"),
    dockerfile.WithFlag("mount", []string{
        "type=cache,target=/root/.cache/go-build",
    }),
)
if err != nil {
    return err
}
```

An instruction instance can be configured in stages. Required arguments
and flags are checked by `Validate` and again by `Handlers.Apply` before
execution; option values and definition errors are rejected during creation.
This allows callers to create an instance, populate required flags, and then
validate it:

```go
instruction, err := registry.New("PUBLISH")
if err != nil {
    return err
}
if err := instruction.SetArguments("registry.example/app:latest"); err != nil {
    return err
}
if err := instruction.SetFlag("replicas", 3); err != nil {
    return err
}
if err := instruction.Validate(); err != nil {
    return err
}
```

For an unmodified built-in definition, use the shortcut:

```go
copy, err := dockerfile.Builtin(
    "COPY",
    []string{"src/", "/app/"},
    dockerfile.WithFlag("chown", "1000:1000"),
    dockerfile.WithFlag("link", true),
)
```

`ParseTokens` parses already-tokenized instruction operands. It supports
`--flag=value`, `--flag value`, aliases, boolean `--no-flag`, repeatable flags,
and `--`. It deliberately does not implement Dockerfile lexical parsing:
quotes, continuations, heredocs, JSON forms, and shell expansion belong in a
parser layer.

## Add a flag to an existing instruction

A flag definition describes parsing, defaulting, validation, aliases, and
repeatability. It can be created, read, updated, listed, and deleted:

```go
err := registry.CreateFlag("RUN", dockerfile.FlagDefinition{
    Name:         "cache-scope",
    Type:         dockerfile.FlagEnum,
    Description:  "Select the cache namespace.",
    AllowedValues: []string{"shared", "private"},
    Default:      "shared",
})
if err != nil {
    return err
}

run, err := registry.New("RUN", "go", "test", "./...")
if err != nil {
    return err
}
err = run.SetFlag("cache-scope", "private")

// Flag CRUD:
flag, err := registry.GetFlag("RUN", "cache-scope")
flags, err := registry.ListFlags("RUN")
err = registry.UpdateFlag("RUN", "cache-scope", func(flag *dockerfile.FlagDefinition) error {
    flag.Description = "Choose an isolated or shared cache."
    return nil
})
err = registry.DeleteFlag("RUN", "cache-scope")
```

A registry update affects instructions created afterwards. Existing instances
retain the exact definition snapshot they were created with; a build already
in progress cannot silently change meaning because a schema is edited.

Supported flag types include strings, booleans, signed/unsigned integers,
durations, repeatable string slices, enums, and custom application-defined
types. A custom type supplies a parser and optional validator:

```go
dockerfile.FlagDefinition{
    Name: "resources",
    Type: dockerfile.FlagCustom,
    Parse: func(raw string) (any, error) {
        return parseResources(raw)
    },
    Validate: func(value any) error {
        return validateResources(value)
    },
}
```

Mutable custom flag values should implement
`dockerfile.FlagValueCloner` so instruction snapshots own independent values.

Flag schema operations change the contract for instructions created after
the update. Instance operations manage only one instruction's resolved
values:

```go
healthcheck, err := registry.New(
    "HEALTHCHECK",
    "CMD", "curl", "-f", "http://localhost",
)
if err != nil {
    return err
}

// Set, read, and reset one value.
err = healthcheck.SetFlag("retries", 5)
value, present := healthcheck.Flags().Int("retries")
err = healthcheck.ResetFlag("retries")

// Clear one value, or restore every value to its schema default.
err = healthcheck.DeleteFlagValue("retries")
err = healthcheck.ResetFlags()
```

## Add an instruction

Create a definition; no new Go constructor or enum is necessary:

```go
err := registry.CreateInstruction(dockerfile.InstructionDefinition{
    Name:        "GENERATE",
    Aliases:     []string{"GEN"},
    Kind:        dockerfile.KindCustom,
    Description: "Generate release artifacts.",
    Arguments: []dockerfile.ArgumentDefinition{
        {Name: "output", Required: true},
    },
    Flags: []dockerfile.FlagDefinition{
        {
            Name: "format",
            Type: dockerfile.FlagEnum,
            AllowedValues: []string{"json", "yaml"},
            Default: "json",
        },
    },
})
if err != nil {
    return err
}
```

Then register its handler by name:

```go
handlers := dockerfile.NewHandlers()
err := handlers.RegisterHandler("GENERATE", func(
    ctx context.Context,
    call dockerfile.Call,
) (dockerfile.Stage, error) {
    format, _ := call.Flags.String("format")
    output := call.Positionals["output"][0]

    // BuildKit lowering and OCI image config mutation live here.
    _ = format
    _ = output
    return call.Stage, nil
})
```

Definitions and handlers are independent. The definition registry supports
instruction CRUD; the handler registry supports handler CRUD. Adding a
custom instruction does not require a new field in `Handlers`.

```go
definition, err := registry.GetInstruction("GENERATE")
definitions := registry.ListInstructions()
err = registry.UpdateInstruction("GENERATE", func(def *dockerfile.InstructionDefinition) error {
    def.Description = "Generate deterministic release artifacts."
    return nil
})
err = registry.DeleteInstruction("GENERATE")
```

Updates are transactional. Concurrent updates use optimistic revisions and
may return `ErrRevisionConflict`; reload and retry instead of overwriting an
update made by another goroutine.

## One option type for every instruction

Every instruction accepts the same `dockerfile.Option`:

```go
func WithBuildPolicy(policy string) dockerfile.Option {
    return func(options *dockerfile.InstructionOptions) error {
        options.SetExtension("example.com/build-policy", policy)
        return nil
    }
}

run, err := registry.NewWithOptions(
    "RUN",
    []string{"go", "test", "./..."},
    dockerfile.WithFlag("network", "none"),
    dockerfile.WithPayload(myTypedPayload),
    WithBuildPolicy("isolated"),
)
```

A future flag does not need a new option interface or a constructor signature
change. It can be supplied generically with `WithFlag`, or a convenience
function can be added as an additive wrapper around it.

For instructions whose primary value is structured rather than positional,
use the typed payload helpers:

```go
dockerfile.WithCommand(dockerfile.Exec("echo", "hello"))
dockerfile.WithSource(dockerfile.Image("alpine:3.21"))
```


Extensions are namespaced application-owned values. They are intended for
metadata and behavior that should not be promoted into a shared flag schema.

## BuildKit integration

BuildKit's own option types are accepted directly; Lumet does not mirror every
new BuildKit option in its public API:

```go
run, err := registry.NewWithOptions(
    "RUN",
    []string{"go", "test"},
    dockerfile.WithLLBRunOptions(
        llb.AddMount("/go/pkg/mod", llb.Scratch()),
    ),
    dockerfile.WithLLBNetworkMode(pb.NetMode_NONE),
    dockerfile.WithLLBConstraints(llb.WithCustomName("RUN go test")),
)
```

The option bundle exposes native `llb.ImageOption`, `llb.RunOption`,
`llb.CopyOption`, and `llb.ConstraintsOpt` collections. Network and
security are represented separately because BuildKit models them on
`llb.State`, not on `llb.RunOption`. Constraints are applied as state marshal
defaults, while RUN options are supplied to `State.Run`:

```go
func lowerRun(call dockerfile.Call, commandArgs []string) llb.State {
    buildOptions := call.LLB
    state := buildOptions.ApplyToState(call.Stage.State)
    state = state.SetMarshalDefaults(buildOptions.Constraints()...)
    runOptions := append(
        []llb.RunOption{llb.Args(commandArgs)},
        buildOptions.RunOptions()...,
    )
    return state.Run(runOptions...).Root()
}
```

For a FROM handler, pass `ImageOptions()` to `llb.Image`. For COPY and ADD
handlers, pass `CopyOptions()` to `llb.Copy` or the appropriate file action.
Handlers are responsible for mapping validated Dockerfile flags to the
appropriate LLB operation and for updating OCI image configuration.

## Handler lifecycle, interceptors, and history

Handlers support create, update, get, list, and delete. Named interceptors
execute in registration order and support the same lifecycle. This keeps
logging, tracing, policy checks, and custom instrumentation out of instruction
implementations.

Definitions can specify a history mode or custom history callback. Per-instance
`WithHistory` and `WithHistoryMode` options override the definition for one
instruction. `HistoryDefault` leaves history management to the handler.

## ONBUILD

Set `OnBuildable: true` on a definition to make it eligible as an ONBUILD
trigger. Stage initializers, ONBUILD itself, and definitions not marked
ONBUILD-able are rejected:

```go
trigger, err := registry.New("GENERATE", "dist/")
if err != nil {
    return err
}
onBuild, err := registry.NewOnBuild(trigger)
```

The ONBUILD handler receives the nested instruction in `Call.Payload` and can
execute it through the same handler registry.

## Standard instruction definitions

`NewDefaultRegistry` registers ADD, ARG, CMD, COPY, ENTRYPOINT, ENV, EXPOSE,
FROM, HEALTHCHECK, LABEL, MAINTAINER, ONBUILD, RUN, SHELL, STOPSIGNAL, USER,
VOLUME, and WORKDIR. Their flags and positional arguments are ordinary
definitions and can be queried, extended, or overridden with the same CRUD
methods as application-defined instructions.
