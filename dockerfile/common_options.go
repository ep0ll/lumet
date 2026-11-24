package dockerfile

import "github.com/moby/buildkit/client/llb"

func WithHistory(fn HistoryFunc) CommonOption {
	return func(c *Common) { c.History = fn }
}
func WithHistoryMode(mode HistoryMode) CommonOption {
	return func(c *Common) { c.HistoryMode = mode }
}
func WithConstraints(options ...llb.ConstraintsOpt) CommonOption {
	return func(c *Common) { c.Constraints = append(c.Constraints, options...) }
}
func WithCustomName(name string) CommonOption {
	return func(c *Common) { c.CustomName = name }
}
func WithProgressGroup(name string) CommonOption {
	return func(c *Common) { c.ProgressGroup = name }
}
func WithProgressTitle(title string) CommonOption {
	return func(c *Common) { c.ProgressTitle = title }
}
func DependsOn(stages ...StageName) CommonOption {
	return func(c *Common) { c.DependsOn = append(c.DependsOn, stages...) }
}
func collectCommon(options []CommonOption) Common {
	var result Common
	for _, option := range options {
		if option != nil { option(&result) }
	}
	return result
}

func WithInstructionHooks(hooks InstructionHooks) CommonOption {
	return func(c *Common) { c.Hooks = hooks }
}
