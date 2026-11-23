// Package dockerfile builds Dockerfile-like instruction sequences directly
// as BuildKit LLB graphs and Docker OCI image metadata.
//
// The package is intentionally a builder API, not a Dockerfile text parser.
// Each instruction constructor returns an Instruction, a function that receives
// a context and the mutable build coordinator:
//
//	build := dockerfile.New(
//		dockerfile.WithPlatform(platform),
//		dockerfile.WithBuildContext(llb.Local("context")),
//		dockerfile.WithImageResolver(resolveBaseImageConfig),
//	)
//	err := build.Apply(ctx,
//		dockerfile.From("golang:1.26", dockerfile.As("builder")),
//		dockerfile.Workdir("/src"),
//		dockerfile.Copy([]string{"go.mod", "go.sum"}, "."),
//		dockerfile.Run(dockerfile.Sh("go mod download")),
//		dockerfile.Copy([]string{"."}, "."),
//		dockerfile.Run(dockerfile.Exec("go", "build", "-o", "/out/app", ".")),
//		dockerfile.From("scratch", dockerfile.As("runtime")),
//		dockerfile.Copy([]string{"/out/app"}, "/app", dockerfile.CopyFrom("builder")),
//		dockerfile.Entrypoint(dockerfile.Exec("/app")),
//	)
//	if err != nil {
//		return err
//	}
//	result := build.Result()
//
// A Build owns the current LLB state, image config, declared build arguments,
// named contexts and completed stages. Apply executes instructions in order.
// The LLB State itself is immutable; each instruction replaces it with the
// next state graph. A Build must not be mutated concurrently.
//
// Base image filesystem resolution is still represented by llb.Image. To
// populate Build.Image with the base image's Docker-specific config (Env,
// WorkingDir, Shell, Cmd, Entrypoint, labels, health check and other metadata),
// install an ImageResolver. The resolver should return the config for the same
// reference and target platform that BuildKit will resolve. Without it, the
// LLB graph can still resolve the image, but Build.Image starts with empty
// metadata for that external base image.
//
// BuildKit-specific behavior is deliberately exposed instead of being copied
// into this package. RunOptions forwards llb.RunOption values (mounts, network,
// security, secrets, SSH, ulimits and devices); CopyOptions and LLBConstraints
// forward the corresponding BuildKit extension points. InstructionOptions is
// public so callers can define reusable options and future instruction flags
// can be added without changing the instruction constructor signatures.
//
// ONBUILD writes trigger expressions to the image config for a downstream
// Dockerfile frontend to consume. This package does not parse Dockerfile text,
// so it does not interpret or execute pre-existing ONBUILD trigger strings.
//
// StopSignal accepts os.Signal. On Unix, syscall.Signal values are stored as
// their numeric signal numbers; other os.Signal implementations use String().
package dockerfile
