package dockerfile

import (
	"fmt"
	"time"
)

// BuiltinInstructions returns the definitions for all standard Dockerfile
// instructions supported by this package. The returned values are independent
// copies and can be registered, modified, or extended by an application.
func BuiltinInstructions() []InstructionDefinition {
	stringFlag := func(name, description string) FlagDefinition {
		return FlagDefinition{Name: name, Type: FlagString, Description: description}
	}
	boolFlag := func(name, description string) FlagDefinition {
		return FlagDefinition{Name: name, Type: FlagBool, Description: description}
	}
	repeatable := func(name, description string) FlagDefinition {
		return FlagDefinition{
			Name: name, Type: FlagStringSlice, Description: description, Repeatable: true,
		}
	}
	enumFlag := func(name, description string, values ...string) FlagDefinition {
		return FlagDefinition{Name: name, Type: FlagEnum, Description: description, AllowedValues: values}
	}
	durationFlag := func(name, description, value string) FlagDefinition {
		return FlagDefinition{Name: name, Type: FlagDuration, Description: description, Default: value}
	}
	retries := int64(3)
	minimumRetry := int64(1)

	required := func(name, description string) []ArgumentDefinition {
		return []ArgumentDefinition{{Name: name, Description: description, Required: true}}
	}
	variadic := func(name, description string) []ArgumentDefinition {
		return []ArgumentDefinition{{Name: name, Description: description, Required: true, Variadic: true}}
	}
	sourcesAndDestination := func() []ArgumentDefinition {
		return []ArgumentDefinition{
			{Name: "source", Description: "One or more source paths", Required: true, Variadic: true},
			{Name: "destination", Description: "Destination path", Required: true},
		}
	}
	metadata := HistoryEmptyLayer
	layer := HistoryLayer

	return []InstructionDefinition{
		{
			Name: "FROM", Kind: KindFrom, Class: InstructionStageInitializer,
			Description: "Create a new build stage from a base image or previous stage.",
			Arguments: required("image", "Base image, stage, named context, or scratch"),
			Flags: []FlagDefinition{
				stringFlag("platform", "Set the target platform for the base image."),
			},
			HistoryMode: metadata,
		},
		{
			Name: "ARG", Kind: KindArg, Description: "Declare a build-time variable.",
			Arguments: required("name[=value]", "Argument name and optional default"),
			OnBuildable: true, HistoryMode: metadata,
		},
		{
			Name: "ENV", Kind: KindEnv, Description: "Set environment variables for subsequent instructions.",
			Arguments: variadic("key=value", "One or more environment assignments"),
			OnBuildable: true, HistoryMode: metadata,
		},
		{
			Name: "LABEL", Kind: KindLabel, Description: "Add metadata to the image.",
			Arguments: variadic("key=value", "One or more label assignments"),
			OnBuildable: true, HistoryMode: metadata,
		},
		{
			Name: "MAINTAINER", Kind: KindMaintainer, Description: "Set the deprecated image maintainer field.",
			Arguments: variadic("name", "Maintainer name"),
			Deprecated: "Use LABEL org.opencontainers.image.authors instead.",
			HistoryMode: metadata,
		},
		{
			Name: "RUN", Kind: KindRun, Description: "Execute commands in a new build layer.",
			Arguments: variadic("command", "Shell-form command or exec-form arguments"),
			OnBuildable: true, HistoryMode: layer,
			Flags: []FlagDefinition{
				repeatable("mount", "Mount a build context, stage, cache, secret, or SSH socket."),
				enumFlag("network", "Set the RUN networking mode.", "default", "none", "host"),
				enumFlag("security", "Set the RUN security mode.", "sandbox", "insecure"),
				repeatable("device", "Expose a CDI device to the RUN process."),
			},
		},
		{
			Name: "CMD", Kind: KindCmd, Description: "Specify the default command for the container.",
			Arguments: variadic("command", "Shell-form command or exec-form arguments"),
			OnBuildable: true, HistoryMode: metadata,
		},
		{
			Name: "ENTRYPOINT", Kind: KindEntrypoint, Description: "Specify the container executable.",
			Arguments: variadic("command", "Shell-form command or exec-form arguments"),
			OnBuildable: true, HistoryMode: metadata,
		},
		{
			Name: "SHELL", Kind: KindShell, Description: "Set the default shell for shell-form instructions.",
			Arguments: variadic("shell", "Exec-form shell arguments"),
			OnBuildable: true, HistoryMode: metadata,
		},
		{
			Name: "EXPOSE", Kind: KindExpose, Description: "Document the ports used by the application.",
			Arguments: variadic("port[/protocol]", "One or more port declarations"),
			OnBuildable: true, HistoryMode: metadata,
		},
		{
			Name: "VOLUME", Kind: KindVolume, Description: "Declare volume mount points.",
			Arguments: variadic("path", "One or more volume paths"),
			OnBuildable: true, HistoryMode: metadata,
		},
		{
			Name: "USER", Kind: KindUser, Description: "Set the user and optional group for subsequent instructions.",
			Arguments: required("user[:group]", "User name or numeric identity"),
			OnBuildable: true, HistoryMode: metadata,
		},
		{
			Name: "WORKDIR", Kind: KindWorkdir, Description: "Set the working directory for subsequent instructions.",
			Arguments: required("path", "Working directory path"),
			OnBuildable: true, HistoryMode: metadata,
		},
		{
			Name: "COPY", Kind: KindCopy, Description: "Copy files and directories into the image.",
			Arguments: sourcesAndDestination(), OnBuildable: true, HistoryMode: layer,
			Flags: []FlagDefinition{
				stringFlag("from", "Copy from a build stage or named context."),
				stringFlag("chown", "Set ownership of copied files."),
				stringFlag("chmod", "Set permissions of copied files."),
				boolFlag("link", "Perform the copy using an independent layer."),
				boolFlag("parents", "Preserve source parent directories."),
				repeatable("exclude", "Exclude paths matching the supplied pattern."),
			},
		},
		{
			Name: "ADD", Kind: KindAdd, Description: "Copy local files, remote URLs, or Git repositories into the image.",
			Arguments: sourcesAndDestination(), OnBuildable: true, HistoryMode: layer,
			Flags: []FlagDefinition{
				stringFlag("chown", "Set ownership of added files."),
				stringFlag("chmod", "Set permissions of added files."),
				boolFlag("link", "Perform the add using an independent layer."),
				repeatable("exclude", "Exclude paths matching the supplied pattern."),
				boolFlag("keep-git-dir", "Retain the .git directory when adding a Git repository."),
				stringFlag("checksum", "Verify a remote source against the supplied checksum."),
				boolFlag("unpack", "Control automatic unpacking of supported archives."),
			},
		},
		{
			Name: "STOPSIGNAL", Kind: KindStopSignal, Description: "Set the signal used to stop the container.",
			Arguments: required("signal", "Signal name or number"),
			OnBuildable: true, HistoryMode: metadata,
		},
		{
			Name: "HEALTHCHECK", Kind: KindHealthcheck, Description: "Configure the command used to check container health.",
			Arguments: variadic("command", "CMD and command, or NONE"),
			OnBuildable: true, HistoryMode: metadata,
			Flags: []FlagDefinition{
				durationFlag("interval", "Time between health checks.", "30s"),
				durationFlag("timeout", "Maximum duration of one health check.", "30s"),
				durationFlag("start-period", "Grace period before failures count toward the retry limit.", "0s"),
				durationFlag("start-interval", "Time between health checks during the start period.", "5s"),
				{Name: "retries", Type: FlagInt, Description: "Consecutive failures required to report unhealthy.", Default: retries, MinInt: &minimumRetry},
			},
		},
		{
			Name: "ONBUILD", Kind: KindOnBuild, Description: "Register an instruction to execute when this image is used as a base.",
			HistoryMode: metadata,
			Validate: func(_ []string, _ FlagValues, payload any) error {
				trigger, ok := payload.(*Instruction)
				if !ok || trigger == nil {
					return fmt.Errorf("ONBUILD requires a trigger instruction payload")
				}
				definition := trigger.Definition()
				if !definition.OnBuildable || definition.Class == InstructionStageInitializer || definition.Name == "ONBUILD" {
					return fmt.Errorf("%s cannot be used as an ONBUILD trigger", definition.Name)
				}
				return trigger.Validate()
			},
		},
	}
}

// DefaultHealthcheckIntervals returns the Dockerfile healthcheck defaults.
func DefaultHealthcheckIntervals() (interval, timeout, startPeriod, startInterval time.Duration, retries int) {
	return 30 * time.Second, 30 * time.Second, 0, 5 * time.Second, 3
}
