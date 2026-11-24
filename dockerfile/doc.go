// Package dockerfile evaluates typed build instructions into BuildKit LLB
// and OCI/Docker image configuration. It is a functional build API, not a
// Dockerfile source parser.
//
// An instruction action is a state transition:
//
//	type InstructionAction func(context.Context, State, *Runtime, InstructionMetadata) (State, error)
//
// NewInstruction captures that closure with instruction metadata. The
// evaluator passes a detached State containing both llb.State and image
// configuration. The action returns the desired successor state; an error
// leaves the previously committed state unchanged.
//
// Built-in instruction constructors are in the dockerfile/instruction package.
// This keeps the evaluator independent of individual Docker instruction
// implementations:
//
//	import (
//	    "github.com/ep0ll/lumet/dockerfile"
//	    "github.com/ep0ll/lumet/dockerfile/instruction"
//	)
//
//	df := dockerfile.New(
//	    dockerfile.WithBuildArgs(map[string]string{"GO_VERSION": "1.24"}),
//	    dockerfile.WithCacheIDNamespace("example"),
//	    dockerfile.WithImageResolver(resolveImageConfig),
//	)
//	if err := df.AddStage(
//	    dockerfile.From(dockerfile.Image("golang:$GO_VERSION"), dockerfile.As("build")),
//	    instruction.Workdir("/src"),
//	    instruction.Copy(instruction.Sources{"."}, "."),
//	    instruction.Run(
//	        instruction.Sh("go build -o /out/app ./..."),
//	        instruction.Mounts(
//	            instruction.Cache("/go/pkg/mod", instruction.CacheID("go-mod")),
//	            instruction.Cache("/root/.cache/go-build", instruction.CacheID("go-build")),
//	            instruction.Secret("private-token", "/run/secrets/token"),
//	            instruction.SSH("default", ""),
//	            instruction.Tmpfs("/tmp", 64<<20),
//	        ),
//	        instruction.Network(instruction.NetworkDefault),
//	        instruction.Security(instruction.SecuritySandbox),
//	    ),
//	); err != nil {
//	    return err
//	}
//	if err := df.AddStage(
//	    dockerfile.From(dockerfile.Image("alpine:3.21"), dockerfile.As("runtime")),
//	    instruction.Copy(
//	        instruction.Sources{"/out/app"}, "/usr/local/bin/app",
//	        instruction.CopyFrom(dockerfile.StageOf("build")),
//	    ),
//	    instruction.Entrypoint(instruction.Exec("/usr/local/bin/app")),
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
//	_ = result.Target.Image
//
// Custom instructions use the same closure model and need no registration:
//
//	audit := dockerfile.NewInstruction("AUDIT",
//	    func(ctx context.Context, current dockerfile.State, runtime *dockerfile.Runtime, metadata dockerfile.InstructionMetadata) (dockerfile.State, error) {
//	        next := current
//	        if next.Image.Config.Labels == nil {
//	            next.Image.Config.Labels = make(map[string]string)
//	        }
//	        next.Image.Config.Labels["example.com/audit"] = "enabled"
//	        next.LLB = next.LLB.AddEnv("AUDIT_ENABLED", "1")
//	        if next.Env == nil {
//	            next.Env = make(map[string]string)
//	        }
//	        next.Env["AUDIT_ENABLED"] = "1"
//	        return next, nil
//	    },
//	    dockerfile.WithDescription("AUDIT enable build metadata"),
//	)
//
// StageFrom is a closure that initializes one stage. StagePlan attaches the
// stage body, constraints and stage-scoped hooks. Dockerfile owns global ARG
// scope, named contexts, target selection and dependency-DAG evaluation.
// Only stages reachable from the selected target are evaluated. Dependencies
// in ONBUILD triggers are resolved lazily when the trigger fires.
//
// The dockerfile/instruction package groups built-in constructors by
// instruction concern: arguments and environment, filesystem operations,
// command execution and mounts, image configuration, and ONBUILD triggers.
// dockerfile/expand and dockerfile/progress are independent services used by
// the evaluator.
//
// Variable expansion is not Dockerfile lexing. It has no source escape-rune
// option; backslashes and backticks are ordinary characters. ONBUILD can only
// persist instructions with a typed serializer. Raw Config.OnBuild strings on
// external images require a caller-supplied WithOnBuildDecoder.
package dockerfile
