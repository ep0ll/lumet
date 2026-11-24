package instruction

import (
	"context"
	"errors"
	"strconv"
)

// Signal is the signal name or numeric signal identifier written to image config.
type Signal string

const (
	SIGHUP   Signal = "SIGHUP"
	SIGINT   Signal = "SIGINT"
	SIGQUIT  Signal = "SIGQUIT"
	SIGKILL  Signal = "SIGKILL"
	SIGUSR1  Signal = "SIGUSR1"
	SIGUSR2  Signal = "SIGUSR2"
	SIGTERM  Signal = "SIGTERM"
	SIGWINCH Signal = "SIGWINCH"
)

func SignalNumber(number uint8) Signal { return Signal(strconv.Itoa(int(number))) }

// StopSignal updates the image's stop signal without changing LLB filesystem state.
func StopSignal(signal Signal) Instruction {
	return newMutationInstruction("STOPSIGNAL", func(_ context.Context, state *State, _ *Runtime, _ InstructionMetadata) error {
		if signal == "" {
			return errors.New("STOPSIGNAL cannot be empty")
		}
		state.Image.Config.StopSignal = string(signal)
		return nil
	}, WithDescription("STOPSIGNAL "+string(signal)),
		WithOnBuildExpression(func() (string, error) {
			if signal == "" {
				return "", errors.New("STOPSIGNAL cannot be empty")
			}
			return "STOPSIGNAL " + string(signal), nil
		}))
}
