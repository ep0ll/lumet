// Package instruction contains the built-in state-transition instruction
// constructors. It depends on the Dockerfile evaluator but is not imported by
// the evaluator, keeping the dependency direction one-way.
package instruction

import dockerfile "github.com/ep0ll/lumet/dockerfile"

type (
	State               = dockerfile.State
	Runtime             = dockerfile.Runtime
	Instruction         = dockerfile.Instruction
	InstructionAction   = dockerfile.InstructionAction
	InstructionMetadata = dockerfile.InstructionMetadata
	InstructionOption   = dockerfile.InstructionOption
	InstructionHooks    = dockerfile.InstructionHooks
	InstructionEvent    = dockerfile.InstructionEvent
	HistoryFunc         = dockerfile.HistoryFunc
	HistoryEvent        = dockerfile.HistoryEvent
	StageName           = dockerfile.StageName
	CopySource          = dockerfile.CopySource
	StageRef            = dockerfile.StageRef
	ContextRef          = dockerfile.ContextRef
	ImageRef            = dockerfile.ImageRef
	HistoryMode         = dockerfile.HistoryMode
)

const (
	HistoryDefault    = dockerfile.HistoryDefault
	HistoryLayer      = dockerfile.HistoryLayer
	HistoryEmptyLayer = dockerfile.HistoryEmptyLayer
	HistorySkip       = dockerfile.HistorySkip
)

var (
	WithDescription          = dockerfile.WithDescription
	WithProgressGroup        = dockerfile.WithProgressGroup
	WithProgressTitle        = dockerfile.WithProgressTitle
	WithCustomName            = dockerfile.WithCustomName
	WithInstructionConstraints = dockerfile.WithInstructionConstraints
	WithInstructionHooks      = dockerfile.WithInstructionHooks
	WithHistory               = dockerfile.WithHistory
	WithOnBuildExpression    = dockerfile.WithOnBuildExpression
	DependsOn             = dockerfile.DependsOn
	WithGlobalARG         = dockerfile.WithGlobalARG
	WithDeferredDependencies = dockerfile.WithDeferredDependencies
	WithHistoryMode         = dockerfile.WithHistoryMode
	RuntimeConstraints    = dockerfile.RuntimeConstraints
	OperationConstraints  = dockerfile.OperationConstraints
	BuildKitCustomName    = dockerfile.BuildKitCustomName
)
