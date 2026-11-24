// Package dockerfile creates BuildKit LLB graphs and OCI/Docker image
// configuration from typed, state-transforming instruction closures. It does
// not parse Dockerfile source text.
//
// Every instruction is a closure:
//
//	 type InstructionAction func(context.Context, *State, *Runtime, InstructionMetadata) error
//
// The constructor captures instruction arguments and options. When the
// Dockerfile is evaluated, the closure receives the current State, which holds
// both the persistent llb.State graph and the mutable OCI image configuration.
// The closure changes only the parts of the state owned by its instruction.
//
// This is intentionally different from a parser or a registry of
// instruction-specific structs. A custom instruction does not need to add a
// new handler type, register a handler, or edit the evaluator:
//
//	audit := dockerfile.NewInstruction("AUDIT",
//	    func(ctx context.Context, state *dockerfile.State, runtime *dockerfile.Runtime, metadata dockerfile.InstructionMetadata) error {
//	        if state.Image.Config.Labels == nil {
//	            state.Image.Config.Labels = make(map[string]string)
//	        }
//	        state.Image.Config.Labels["example.com/audit"] = "enabled"
//	        state.LLB = state.LLB.AddEnv("AUDIT_ENABLED", "1")
//	        state.Env["AUDIT_ENABLED"] = "1"
//	        return nil
//	    },
//	    dockerfile.WithDescription("AUDIT enable build metadata"),
//	)
//
// Instructions may capture their own options with functional options or use
// Instruction.With to attach common metadata, dependencies, history behavior,
// progress labels and lifecycle hooks.
//
// StageFrom is also a closure. StagePlan binds one FROM closure to a stage
// body, stage-local hooks and LLB constraints. Dockerfile owns build-global
// arguments, named contexts, target selection and the dependency DAG. Build
// evaluates only stages reachable from the target and resolves each stage
// once. Stage references in ONBUILD triggers are resolved lazily when those
// triggers actually execute.
//
// Example:
//
//	df := dockerfile.New(
//	    dockerfile.WithBuildArgs(map[string]string{"GO_VERSION": "1.24"}),
//	    dockerfile.WithCacheIDNamespace("my-app"),
//	    dockerfile.WithImageResolver(resolveImageConfig),
//	)
//	if err := df.AddStage(
//	    dockerfile.From(dockerfile.Image("golang:$GO_VERSION"), dockerfile.As("build")),
//	    dockerfile.Workdir("/src"),
//	    dockerfile.Copy(dockerfile.Sources{"."}, "."),
//	    dockerfile.Run(
//	        dockerfile.Sh("go build -o /out/app ./..."),
//	        dockerfile.Mounts(
//	            dockerfile.Cache("/go/pkg/mod", dockerfile.CacheID("go-mod")),
//	            dockerfile.Cache("/root/.cache/go-build", dockerfile.CacheID("go-build")),
//	            dockerfile.Secret("private-token", "/run/secrets/token"),
//	            dockerfile.SSH("default", ""),
//	            dockerfile.Tmpfs("/tmp", 64<<20),
//	        ),
//	        dockerfile.Network(dockerfile.NetworkDefault),
//	        dockerfile.Security(dockerfile.SecuritySandbox),
//	    ),
//	); err != nil {
//	    return err
//	}
//	if err := df.AddStage(
//	    dockerfile.From(dockerfile.Image("alpine:3.21"), dockerfile.As("runtime")),
//	    dockerfile.Copy(
//	        dockerfile.Sources{"/out/app"}, "/usr/local/bin/app",
//	        dockerfile.CopyFrom(dockerfile.StageOf("build")),
//	    ),
//	    dockerfile.Entrypoint(dockerfile.Exec("/usr/local/bin/app")),
//	); err != nil {
//	    return err
//	}
//	result, err := df.Build(ctx)
//	if err != nil {
//	    return err
//	}
//	definition, err := result.Target.LLB.Marshal(ctx)
//	if err != nil {
//	    return err
//	}
//	_ = definition
//	_ = result.Target.Image // OCI config: Env, Cmd, Entrypoint, Healthcheck, Labels...
//
// RUN supports bind, cache, tmpfs, secret and SSH mounts; network and security
// modes; CDI devices; ulimits; a read-only root filesystem; and passthrough
// BuildKit RunOptions. Cache IDs may be namespaced at build scope.
//
// Variable expansion is provided by dockerfile/expand. It is not a Dockerfile
// lexer and deliberately has no escape-rune option: backslashes and backticks
// are ordinary input characters. The expander handles variable references,
// braced default/alternate/required forms and $$ for a literal dollar.
//
// ONBUILD records a serializable typed trigger in the parent image's
// Config.OnBuild. A derived stage replays it after FROM resolution and before
// the derived stage body. Raw Config.OnBuild strings received from an external
// base image require WithOnBuildDecoder; arbitrary Go closures cannot be
// recovered from an OCI image configuration.
//
// The progress package provides a reporter interface and lifecycle events.
// The image resolver and ONBUILD decoder are function types because the
// evaluator only needs one operation from each dependency.
package dockerfile
