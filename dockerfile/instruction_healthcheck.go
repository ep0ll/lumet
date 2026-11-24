package dockerfile

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	dockerspec "github.com/moby/docker-image-spec/specs-go/v1"
)

// HealthcheckConfig is the configuration applied by a HEALTHCHECK instruction.
type HealthcheckConfig struct {
	None          bool
	Command       Command
	Interval      time.Duration
	Timeout       time.Duration
	StartPeriod   time.Duration
	StartInterval time.Duration
	Retries       int
}

type HealthcheckOption func(*HealthcheckConfig)

func HealthcheckNone() HealthcheckOption {
	return func(config *HealthcheckConfig) { config.None = true }
}
func HealthcheckInterval(value time.Duration) HealthcheckOption {
	return func(config *HealthcheckConfig) { config.Interval = value }
}
func HealthcheckTimeout(value time.Duration) HealthcheckOption {
	return func(config *HealthcheckConfig) { config.Timeout = value }
}
func HealthcheckStartPeriod(value time.Duration) HealthcheckOption {
	return func(config *HealthcheckConfig) { config.StartPeriod = value }
}
func HealthcheckStartInterval(value time.Duration) HealthcheckOption {
	return func(config *HealthcheckConfig) { config.StartInterval = value }
}
func HealthcheckRetries(value int) HealthcheckOption {
	return func(config *HealthcheckConfig) { config.Retries = value }
}

// Healthcheck modifies image metadata only; it does not create an LLB vertex.
func Healthcheck(command Command, options ...HealthcheckOption) Instruction {
	config := HealthcheckConfig{Command: cloneCommand(command)}
	for _, option := range options {
		if option != nil {
			option(&config)
		}
	}
	return NewInstruction("HEALTHCHECK", func(_ context.Context, state *State, _ *Runtime, _ InstructionMetadata) error {
		if config.None {
			state.Image.Config.Healthcheck = &dockerspec.HealthcheckConfig{Test: []string{"NONE"}}
			return nil
		}
		test, err := healthcheckTest(config.Command)
		if err != nil {
			return err
		}
		if config.Interval < 0 || config.Timeout < 0 || config.StartPeriod < 0 ||
			config.StartInterval < 0 || config.Retries < 0 {
			return errors.New("HEALTHCHECK durations and retries cannot be negative")
		}
		interval := config.Interval
		if interval == 0 {
			interval = 30 * time.Second
		}
		timeout := config.Timeout
		if timeout == 0 {
			timeout = 30 * time.Second
		}
		startInterval := config.StartInterval
		if startInterval == 0 {
			startInterval = 5 * time.Second
		}
		retries := config.Retries
		if retries == 0 {
			retries = 3
		}
		if interval < time.Millisecond || timeout < time.Millisecond ||
			startInterval < time.Millisecond ||
			(config.StartPeriod > 0 && config.StartPeriod < time.Millisecond) {
			return errors.New("HEALTHCHECK non-zero durations must be at least 1ms")
		}
		state.Image.Config.Healthcheck = &dockerspec.HealthcheckConfig{
			Test: test, Interval: interval, Timeout: timeout,
			StartPeriod: config.StartPeriod, StartInterval: startInterval, Retries: retries,
		}
		return nil
	}, WithDescription("HEALTHCHECK "+formatCommand(config.Command)),
		WithOnBuildExpression(func() (string, error) {
			if config.None {
				return "HEALTHCHECK NONE", nil
			}
			commandText, err := commandExpression(config.Command)
			if err != nil {
				return "", err
			}
			parts := []string{"HEALTHCHECK"}
			if config.Interval > 0 {
				parts = append(parts, "--interval="+config.Interval.String())
			}
			if config.Timeout > 0 {
				parts = append(parts, "--timeout="+config.Timeout.String())
			}
			if config.StartPeriod > 0 {
				parts = append(parts, "--start-period="+config.StartPeriod.String())
			}
			if config.StartInterval > 0 {
				parts = append(parts, "--start-interval="+config.StartInterval.String())
			}
			if config.Retries > 0 {
				parts = append(parts, "--retries="+strconv.Itoa(config.Retries))
			}
			return strings.Join(append(parts, commandText), " "), nil
		}))
}

func healthcheckTest(command Command) ([]string, error) {
	switch value := command.(type) {
	case ExecForm:
		if len(value) == 0 {
			return nil, errors.New("HEALTHCHECK exec command cannot be empty")
		}
		return append([]string{"CMD"}, []string(value)...), nil
	case ShellForm:
		if strings.TrimSpace(string(value)) == "" {
			return nil, errors.New("HEALTHCHECK shell command cannot be empty")
		}
		return []string{"CMD-SHELL", string(value)}, nil
	default:
		return nil, fmt.Errorf("unsupported HEALTHCHECK command form %T", command)
	}
}
