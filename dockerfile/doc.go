// Package dockerfile builds BuildKit LLB graphs from typed, functional
// instruction values. It is not a Dockerfile text parser.
//
// The package separates four scopes: instruction specifications carry
// instruction-local options; StagePlan owns stage-local bindings and hooks;
// Dockerfile owns global ARG scope, contexts, the stage DAG and target;
// dockerfile/expand and dockerfile/progress provide independent contracts.
//
// Instructions are immutable descriptions. RegisterHandler binds a kind to
// a typed handler, and NewLLBHandlers provides standard BuildKit handlers.
// Applications can replace a binding or register new instruction kinds.
//
// No Dockerfile source escape token is interpreted here. Backslashes and
// backticks are ordinary variable-expansion input. dockerfile/expand supports
// dollar-prefixed names, braced variables, default/alternate/required
// operators and double-dollar for a literal dollar sign.
//
// Example:
//
//	df := dockerfile.New(
//	    dockerfile.WithBuildArgs(map[string]string{"GO_VERSION": "1.24"}),
//	    dockerfile.WithImageResolver(resolveImageConfig),
//	)
//	err := df.AddStage(
//	    dockerfile.From(dockerfile.Image("golang:$GO_VERSION"), dockerfile.As("build")),
//	    dockerfile.Workdir("/src"),
//	    dockerfile.Copy(dockerfile.Sources{"."}, "."),
//	    dockerfile.Run(
//	        dockerfile.Sh("go build ./..."),
//	        dockerfile.Mounts(
//	            dockerfile.Cache("/go/pkg/mod", dockerfile.CacheID("gomod")),
//	            dockerfile.Secret("private-token", "/run/secrets/token"),
//	            dockerfile.SSH("default", ""),
//	            dockerfile.Tmpfs("/tmp", 64<<20),
//	        ),
//	    ),
//	)
//	if err != nil { return err }
//	err = df.AddStage(
//	    dockerfile.From(dockerfile.Image("alpine:3.21"), dockerfile.As("runtime")),
//	    dockerfile.Copy(
//	        dockerfile.Sources{"/src/app"}, "/usr/local/bin/app",
//	        dockerfile.CopyFrom(dockerfile.StageOf("build")),
//	    ),
//	    dockerfile.Entrypoint(dockerfile.Exec("/usr/local/bin/app")),
//	)
//	if err != nil { return err }
//	result, err := df.Build(ctx)
//	if err != nil { return err }
//	_ = result.Target.State
//	_ = result.Target.Image // OCI image config: Env, Cmd, Entrypoint, Labels...
//
// ONBUILD triggers are typed. They are stored on the parent image and replayed
// when a derived stage starts, after FROM resolution and before its own body.
// For an external base image containing raw Config.OnBuild strings, configure
// WithOnBuildDecoder to turn them into typed Trigger values.
package dockerfile
