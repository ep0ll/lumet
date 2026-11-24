# Dockerfile API architecture

This package builds a BuildKit LLB graph and an OCI/Docker image configuration from typed Go values. It is not a Dockerfile lexer or parser. A value such as `Sh("go test ./...")` is already a shell-form command; no parser escape character, quoting grammar, or continuation rules are involved.

## State-transition instructions

The core contract is:

```go
type InstructionAction func(
    context.Context,
    State,
    *Runtime,
    InstructionMetadata,
) (State, error)
```

An instruction constructor captures its arguments and options in a closure. Evaluation supplies the current state and the closure returns the desired next state. `State` carries both:

- `LLB llb.State`: BuildKit's persistent filesystem and execution graph.
- `Image dockerspec.DockerOCIImage`: the OCI/Docker config and image history that are not represented by the LLB state alone.

The evaluator clones mutable state before invoking an action, and commits the returned state only when the action succeeds. That gives instruction execution transactional error behavior: an action that returns an error cannot leak a partial image-config or map mutation into the previously committed state.

A custom operation is just another closure:

```go
audit := dockerfile.NewInstruction(
    "AUDIT",
    func(
        _ context.Context,
        current dockerfile.State,
        _ *dockerfile.Runtime,
        _ dockerfile.InstructionMetadata,
    ) (dockerfile.State, error) {
        next := current
        if next.Image.Config.Labels == nil {
            next.Image.Config.Labels = make(map[string]string)
        }
        next.Image.Config.Labels["example.com/audit"] = "enabled"
        next.LLB = next.LLB.AddEnv("AUDIT_ENABLED", "1")
        if next.Env == nil {
            next.Env = make(map[string]string)
        }
        next.Env["AUDIT_ENABLED"] = "1"
        return next, nil
    },
    dockerfile.WithDescription("AUDIT enable metadata"),
)
```

The contract is deliberately concrete and fixed to `State`; a generic handler registry and runtime type inspection would add indirection without making this one-operation transition more extensible. Built-ins use a private mutation adapter to keep their assignments concise, but callers implement the public input/output closure.

### Benefits

- Arguments are captured at construction time and execution dependencies are supplied explicitly.
- The next state is visible in the function signature rather than hidden behind a mutable receiver.
- Instructions are independently composable and custom instructions need no registration.
- The evaluator owns transaction boundaries, lifecycle hooks, history, progress, and DAG scheduling.

### Costs and constraints

- A closure is not comparable, introspectable, or serializable as a Go value. The API therefore carries explicit descriptive metadata and requires an explicit serializer for persisted `ONBUILD` triggers.
- Captured values can still point at mutable caller-owned memory. Constructors copy slices/maps they own; custom closures should capture immutable values or copy their inputs.
- Cloning the mutable OCI config has a cost. LLB itself is persistent, so it is copied by value rather than rebuilt.

## Functional options

Functional options are used where callers need to add orthogonal configuration without a long positional parameter list. Each option type is scoped to its constructor: `RunOption`, `CopyOption`, `AddOption`, `StageFromOption`, `StageOption`, and `InstructionOption`.

Instruction metadata options are separate from operation arguments. For example, a RUN mount belongs to the RUN instruction, while history behavior, progress grouping, BuildKit constraints, dependencies, and hooks decorate the instruction.

Options are composable and support defaults, but they also have tradeoffs: order can matter when two options set the same field, and validation may need to occur when the instruction executes. The implementation copies option-owned slices and validates values before creating an LLB operation.

## Interfaces and closed variants

Interfaces are used only where a value genuinely has multiple implementations:

- `progress.Reporter` is a small consumer-facing interface.
- `Base` and `CopySource` are sealed source variants.
- `Command`, `Mount`, and `AddSource` are sealed instruction argument variants.

The source/command/mount interfaces have unexported marker methods. This intentionally makes them closed sum types: the evaluator can switch over a finite set of supported BuildKit source or mount forms. External extensibility is provided at the instruction-closure boundary, not by permitting arbitrary values that the evaluator cannot translate.

The reporter is an ordinary narrow interface because applications can supply their own progress sink. Image resolution, ONBUILD decoding, and variable expansion are function dependencies because the evaluator needs one operation from each. This follows Go's structural interfaces and the general preference for small interfaces defined near their consumers.

## Functional DAG evaluation

A `Dockerfile` owns build-wide bindings and stage plans. A `StageFrom` closure creates the initial state of one stage. A `StagePlan` binds that initializer to its instruction sequence, stage constraints, and stage hooks.

The evaluator discovers dependencies from FROM stage references, COPY --from, and RUN bind/cache mounts. It walks the graph from the selected target, memoizes completed stages, and detects cycles. Dynamic stage references can be resolved at execution time; dependencies inside ONBUILD triggers are deferred until the trigger fires.

This separates three scopes:

- instruction scope: operation arguments, history, LLB constraints, progress metadata, and instruction hooks;
- stage scope: initializer, body, stage hooks, and stage constraints;
- build scope: global ARGs, named contexts, image resolver, target, progress reporter, and dependency DAG.

## ONBUILD and serialization

A Go closure cannot be reconstructed from an OCI image config. A built-in trigger therefore provides an explicit `OnBuildExpression` serializer. The parent image stores that representation in `Config.OnBuild` and retains the typed trigger in the in-memory stage state. A derived stage executes typed triggers after its FROM state is resolved and before its body.

Raw `Config.OnBuild` strings from an external image require a caller-supplied `WithOnBuildDecoder`. Opaque BuildKit-only RUN options cannot be serialized and are rejected when used in an ONBUILD trigger instead of being silently lost.

## Package boundaries

- `dockerfile`: State, instruction action contract, runtime, stage plans, build definition, DAG evaluator, image history, and BuildKit constraint composition.
- `dockerfile/instruction`: built-in typed instruction constructors, split by operation concern.
- `dockerfile/expand`: source-independent variable expansion. Backslashes and backticks are ordinary characters; there is no parser escape setting.
- `dockerfile/progress`: UI-independent lifecycle events and reporter interface.

The dependency direction is one-way: the built-in instruction package depends on the evaluator's contracts; the evaluator does not import the built-in package. This keeps custom instruction support independent of the built-in catalogue.

## Go design references

The choices above use ordinary Go composition rather than imposing a large pattern framework:

- [Effective Go](https://go.dev/doc/effective_go) — functions, closures, interfaces, and composition.
- [The Go language specification](https://go.dev/ref/spec) — structural interface implementation and function types.
- [Google Go Style Guide: Best Practices](https://google.github.io/styleguide/go/best-practices.html) — simplicity, interfaces, and avoiding unnecessary abstractions.
- [Google Go Decisions: Interfaces](https://google.github.io/go-cloud/guides/decisions.html#interfaces) — small consumer-owned interfaces and API tradeoffs.
- [Uber Go Style Guide: Functional Options](https://github.com/uber-go/guide/blob/master/src/functional-options.md) — functional-option tradeoffs and when not to use them.
