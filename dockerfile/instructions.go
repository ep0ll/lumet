package dockerfile

import (
	"fmt"
	"strings"
)

func From(base Base, options ...FromOption) StageInit {
	spec := FromSpec{Base: base}
	for _, option := range options {
		if option != nil { option.applyFrom(&spec) }
	}
	return NewStageInit(KindFrom, "FROM", spec)
}

// ResetStage starts from a new base while explicitly discarding inherited stage
// environment and metadata. It is useful for intentionally isolated stages.
func ResetStage(base Base, options ...FromOption) StageInit {
	spec := FromSpec{Base: base}
	for _, option := range options {
		if option != nil { option.applyFrom(&spec) }
	}
	return NewStageInit(KindResetStage, "FROM(reset)", ResetSpec{FromSpec: spec})
}

func Arg(name string, options ...ArgOption) Trigger {
	spec := ArgSpec{Name: name}
	for _, option := range options {
		if option != nil { option.applyArg(&spec) }
	}
	return NewTrigger(KindArg, "ARG", spec)
}
func Env(vars []KeyValue, options ...CommonOption) Trigger {
	return NewTrigger(KindEnv, "ENV", EnvSpec{Common: collectCommon(options), Vars: append([]KeyValue(nil), vars...)})
}
func Label(labels []KeyValue, options ...CommonOption) Trigger {
	return NewTrigger(KindLabel, "LABEL", LabelSpec{Common: collectCommon(options), Labels: append([]KeyValue(nil), labels...)})
}
func Maintainer(name string, options ...CommonOption) Instruction {
	return NewInstruction(KindMaintainer, "MAINTAINER", MaintainerSpec{Common: collectCommon(options), Name: name})
}
func Run(command Command, options ...RunOption) Trigger {
	spec := RunSpec{Command: command}
	for _, option := range options {
		if option != nil { option.applyRun(&spec) }
	}
	return NewTrigger(KindRun, "RUN", spec)
}
func Cmd(command Command, options ...CommonOption) Trigger {
	return NewTrigger(KindCmd, "CMD", CmdSpec{Common: collectCommon(options), Command: command})
}
func Entrypoint(command Command, options ...CommonOption) Trigger {
	return NewTrigger(KindEntrypoint, "ENTRYPOINT", EntrypointSpec{Common: collectCommon(options), Command: command})
}
func Shell(shell ExecForm, options ...CommonOption) Trigger {
	return NewTrigger(KindShell, "SHELL", ShellSpec{Common: collectCommon(options), Shell: append(ExecForm(nil), shell...)})
}
func Expose(ports []ExposedPort, options ...CommonOption) Trigger {
	return NewTrigger(KindExpose, "EXPOSE", ExposeSpec{Common: collectCommon(options), Ports: append([]ExposedPort(nil), ports...)})
}
func Volume(paths []string, options ...CommonOption) Trigger {
	return NewTrigger(KindVolume, "VOLUME", VolumeSpec{Common: collectCommon(options), Paths: append([]string(nil), paths...)})
}
func User(user Principal, options ...UserOption) Trigger {
	spec := UserSpec{User: user}
	for _, option := range options {
		if option != nil { option.applyUser(&spec) }
	}
	return NewTrigger(KindUser, "USER", spec)
}
func Workdir(path string, options ...CommonOption) Trigger {
	return NewTrigger(KindWorkdir, "WORKDIR", WorkdirSpec{Common: collectCommon(options), Path: path})
}
func Copy(src Sources, dest string, options ...CopyOption) Trigger {
	spec := CopySpec{Src: append(Sources(nil), src...), Dest: dest}
	for _, option := range options {
		if option != nil { option.applyCopy(&spec) }
	}
	return NewTrigger(KindCopy, "COPY", spec)
}
func Add(src []AddSource, dest string, options ...AddOption) Trigger {
	spec := AddSpec{Src: append([]AddSource(nil), src...), Dest: dest}
	for _, option := range options {
		if option != nil { option.applyAdd(&spec) }
	}
	return NewTrigger(KindAdd, "ADD", spec)
}
func StopSignal(signal Signal, options ...CommonOption) Trigger {
	return NewTrigger(KindStopSignal, "STOPSIGNAL", StopSignalSpec{Common: collectCommon(options), Signal: signal})
}
func Healthcheck(command Command, options ...HealthcheckOption) Trigger {
	spec := HealthcheckSpec{Command: command}
	for _, option := range options {
		if option != nil { option.applyHealthcheck(&spec) }
	}
	return NewTrigger(KindHealthcheck, "HEALTHCHECK", spec)
}
func HealthcheckNone(options ...CommonOption) Trigger {
	return NewTrigger(KindHealthcheck, "HEALTHCHECK", HealthcheckSpec{
		Common: collectCommon(options), None: true,
	})
}

// OnBuild stores a typed trigger on the image. Only Trigger values are accepted,
// so FROM, ONBUILD and MAINTAINER cannot be nested.
func OnBuild(trigger Trigger, options ...CommonOption) Instruction {
	return NewInstruction(KindOnBuild, "ONBUILD", OnBuildSpec{
		Common: collectCommon(options), Trigger: trigger,
	})
}

// String methods provide stable, source-independent command descriptions for
// history, ONBUILD metadata and progress. They are not Dockerfile serializers.
func (s FromSpec) String() string { return "FROM " + baseString(s.Base) }
func (s ResetSpec) String() string { return "FROM(reset) " + baseString(s.Base) }
func (s ArgSpec) String() string {
	if s.Default == nil { return "ARG " + s.Name }
	return "ARG " + s.Name + "=" + *s.Default
}
func (s EnvSpec) String() string { return "ENV " + formatKeyValues(s.Vars) }
func (s LabelSpec) String() string { return "LABEL " + formatKeyValues(s.Labels) }
func (s MaintainerSpec) String() string { return "MAINTAINER " + s.Name }
func (s RunSpec) String() string { return "RUN " + formatCommand(s.Command) }
func (s CmdSpec) String() string { return "CMD " + formatCommand(s.Command) }
func (s EntrypointSpec) String() string { return "ENTRYPOINT " + formatCommand(s.Command) }
func (s ShellSpec) String() string { return "SHELL " + strings.Join(s.Shell, " ") }
func (s ExposeSpec) String() string { return fmt.Sprintf("EXPOSE %v", s.Ports) }
func (s VolumeSpec) String() string { return "VOLUME " + strings.Join(s.Paths, " ") }
func (s UserSpec) String() string {
	value := principalString(s.User)
	if s.Group != nil { value += ":" + principalString(*s.Group) }
	return "USER " + value
}
func (s WorkdirSpec) String() string { return "WORKDIR " + s.Path }
func (s CopySpec) String() string { return "COPY " + strings.Join(s.Src, " ") + " " + s.Dest }
func (s AddSpec) String() string { return fmt.Sprintf("ADD %v %s", s.Src, s.Dest) }
func (s StopSignalSpec) String() string { return "STOPSIGNAL " + string(s.Signal) }
func (s HealthcheckSpec) String() string {
	if s.None { return "HEALTHCHECK NONE" }
	return "HEALTHCHECK " + formatCommand(s.Command)
}
func (s OnBuildSpec) String() string {
	if s.Trigger == nil { return "ONBUILD <nil>" }
	return "ONBUILD " + s.Trigger.String()
}

func baseString(base Base) string {
	if base == nil { return "<nil>" }
	if value, ok := base.(fmt.Stringer); ok { return value.String() }
	return fmt.Sprintf("%T", base)
}
func formatKeyValues(values []KeyValue) string {
	items := make([]string, 0, len(values))
	for _, value := range values { items = append(items, value.Key+"="+value.Value) }
	return strings.Join(items, " ")
}
func formatCommand(command Command) string {
	switch value := command.(type) {
	case ShellForm:
		return string(value)
	case ExecForm:
		return "[" + strings.Join([]string(value), " ") + "]"
	default:
		return fmt.Sprintf("%v", command)
	}
}
func principalString(user Principal) string {
	if user.ByID { return fmt.Sprint(user.ID) }
	return user.Name
}
