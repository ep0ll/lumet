package dockerfile

import (
	"context"
	"errors"
	"fmt"
)

// OnBuild records a typed instruction on the image. It does not execute the
// trigger in the declaring stage; a derived stage executes it immediately
// after its FROM closure resolves.
func OnBuild(trigger Instruction) Instruction {
	return NewInstruction("ONBUILD", func(_ context.Context, state *State, _ *Runtime, _ InstructionMetadata) error {
		if trigger.action == nil {
			return errors.New("ONBUILD trigger has no action")
		}
		switch trigger.name {
		case "FROM", "MAINTAINER", "ONBUILD":
			return fmt.Errorf("%s cannot be used as an ONBUILD trigger", trigger.name)
		}
		expression, err := trigger.OnBuildExpression()
		if err != nil {
			return err
		}
		state.Triggers = append(state.Triggers, trigger)
		state.Image.Config.OnBuild = append(state.Image.Config.OnBuild, expression)
		return nil
	}, WithDescription("ONBUILD "+trigger.String()), WithHistoryMode(HistorySkip), WithDeferredDependencies())
	return instruction.With(DependsOn(trigger.Dependencies()...))
}
