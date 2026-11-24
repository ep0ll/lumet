// Package instruction contains the built-in state-transition instruction
// constructors. It depends on the Dockerfile evaluator but is not imported by
// the evaluator, keeping the dependency direction one-way.
package instruction

import dockerfile "github.com/ep0ll/lumet/dockerfile"

type (
	State               = dockerfile.State
	Runtime             = dockerfile.Runtime
	Instruction         = dockerfile.Instruction
	InstructionMetadata = dockerfile.InstructionMetadata
	InstructionOption   = dockerfile.InstructionOption
	StageName           = dockerfile.StageName
	CopySource          = dockerfile.CopySource
	StageRef            = dockerfile.StageRef
	ContextRef          = dockerfile.ContextRef
	ImageRef            = dockerfile.ImageRef
	HistoryMode         = dockerfile.HistoryMode
)

const HistorySkip = dockerfile.HistorySkip

var (
	WithDescription       = dockerfile.WithDescription
	WithOnBuildExpression = dockerfile.WithOnBuildExpression
	DependsOn             = dockerfile.DependsOn
	WithGlobalARG         = dockerfile.WithGlobalARG
	WithDeferredDependencies = dockerfile.WithDeferredDependencies
	WithHistoryMode         = dockerfile.WithHistoryMode
	RuntimeConstraints    = dockerfile.RuntimeConstraints
	OperationConstraints  = dockerfile.OperationConstraints
	BuildKitCustomName    = dockerfile.BuildKitCustomName
)
