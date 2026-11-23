package dockerfile

import (
	"context"
	"fmt"
	"strings"

	"github.com/moby/buildkit/client/llb"
)

// Run executes a shell-form or exec-form command against the current root
// filesystem. RunOptions forwards additional BuildKit controls such as mounts,
// networking, security, secrets, SSH sockets, ulimits and CDI devices.
func Run(command Command, opts ...Option) Instruction {
	options := collectOptions(opts)
	return func(ctx context.Context, b *Build) error {
		if err := b.requireStage("RUN"); err != nil {
			return err
		}
		args, err := commandArgs(command, b.Image.Config.Shell, b.Platform.OS)
		if err != nil {
			return fmt.Errorf("RUN: %w", err)
		}

		runOptions := make([]llb.RunOption, 0, len(options.RunOptions)+2)
		runOptions = append(runOptions, llb.Args(args))
		name := options.CustomName
		if name == "" {
			name = "RUN " + formatCommand(command)
		}
		runOptions = append(runOptions, llb.WithCustomName(name))
		runOptions = append(runOptions, options.RunOptions...)

		b.State = b.State.Run(runOptions...).Root()
		return b.recordHistory(ctx, "RUN "+strings.Join(args, " "), false)
	}
}
