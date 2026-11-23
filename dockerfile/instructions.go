package dockerfile

import ocispecs "github.com/opencontainers/image-spec/specs-go/v1"

var _ ocispecs.Image // image config is manipulated by handlers

// From starts a stage: From(Image("golang:1.23"), As("build")).
func From(base Base, opts ...FromOption) StageInit {
	s := FromSpec{Base: base}
	for _, o := range opts {
		o.applyFrom(&s)
	}
	return stageInit[FromSpec]{instruction[FromSpec]{KindFrom, s, func(h *Handlers) Handler[FromSpec] { return h.From }}}
}

// ResetStage is the "reset" flavour of FROM, kept as its own instruction so
// From stays small and its handler independent.
func ResetStage(base Base, opts ...FromOption) StageInit {
	s := ResetSpec{FromSpec{Base: base}}
	for _, o := range opts {
		o.applyFrom(&s.FromSpec)
	}
	return stageInit[ResetSpec]{instruction[ResetSpec]{KindResetStage, s, func(h *Handlers) Handler[ResetSpec] { return h.ResetStage }}}
}

func Arg(name string, opts ...ArgOption) Trigger {
	s := ArgSpec{Name: name}
	for _, o := range opts {
		o.applyArg(&s)
	}
	return trigger[ArgSpec]{instruction[ArgSpec]{KindArg, s, func(h *Handlers) Handler[ArgSpec] { return h.Arg }}}
}

func Env(vars []KeyValue, opts ...CommonOption) Trigger {
	return trigger[EnvSpec]{instruction[EnvSpec]{KindEnv, EnvSpec{collect(opts), vars}, func(h *Handlers) Handler[EnvSpec] { return h.Env }}}
}

func Label(labels []KeyValue, opts ...CommonOption) Trigger {
	return trigger[LabelSpec]{instruction[LabelSpec]{KindLabel, LabelSpec{collect(opts), labels}, func(h *Handlers) Handler[LabelSpec] { return h.Label }}}
}

// Deprecated: use Label(KV("org.opencontainers.image.authors", ...)).
func Maintainer(name string, opts ...CommonOption) Instruction {
	return instruction[MaintainerSpec]{KindMaintainer, MaintainerSpec{collect(opts), name}, func(h *Handlers) Handler[MaintainerSpec] { return h.Maintainer }}
}

func Run(cmd Command, opts ...RunOption) Trigger {
	s := RunSpec{Command: cmd}
	for _, o := range opts {
		o.applyRun(&s)
	}
	return trigger[RunSpec]{instruction[RunSpec]{KindRun, s, func(h *Handlers) Handler[RunSpec] { return h.Run }}}
}

func Cmd(cmd Command, opts ...CommonOption) Trigger {
	return trigger[CmdSpec]{instruction[CmdSpec]{KindCmd, CmdSpec{collect(opts), cmd}, func(h *Handlers) Handler[CmdSpec] { return h.Cmd }}}
}

func Entrypoint(cmd Command, opts ...CommonOption) Trigger {
	return trigger[EntrypointSpec]{instruction[EntrypointSpec]{KindEntrypoint, EntrypointSpec{collect(opts), cmd}, func(h *Handlers) Handler[EntrypointSpec] { return h.Entrypoint }}}
}

func Shell(shell ExecForm, opts ...CommonOption) Trigger {
	return trigger[ShellSpec]{instruction[ShellSpec]{KindShell, ShellSpec{collect(opts), shell}, func(h *Handlers) Handler[ShellSpec] { return h.Shell }}}
}

func Expose(ports []ExposedPort, opts ...CommonOption) Trigger {
	return trigger[ExposeSpec]{instruction[ExposeSpec]{KindExpose, ExposeSpec{collect(opts), ports}, func(h *Handlers) Handler[ExposeSpec] { return h.Expose }}}
}

func Volume(paths []string, opts ...CommonOption) Trigger {
	return trigger[VolumeSpec]{instruction[VolumeSpec]{KindVolume, VolumeSpec{collect(opts), paths}, func(h *Handlers) Handler[VolumeSpec] { return h.Volume }}}
}

func User(user Principal, opts ...UserOption) Trigger {
	s := UserSpec{User: user}
	for _, o := range opts {
		o.applyUser(&s)
	}
	return trigger[UserSpec]{instruction[UserSpec]{KindUser, s, func(h *Handlers) Handler[UserSpec] { return h.User }}}
}

func Workdir(path string, opts ...CommonOption) Trigger {
	return trigger[WorkdirSpec]{instruction[WorkdirSpec]{KindWorkdir, WorkdirSpec{collect(opts), path}, func(h *Handlers) Handler[WorkdirSpec] { return h.Workdir }}}
}

func Copy(src Sources, dest string, opts ...CopyOption) Trigger {
	s := CopySpec{Src: src, Dest: dest}
	for _, o := range opts {
		o.applyCopy(&s)
	}
	return trigger[CopySpec]{instruction[CopySpec]{KindCopy, s, func(h *Handlers) Handler[CopySpec] { return h.Copy }}}
}

func Add(src []AddSource, dest string, opts ...AddOption) Trigger {
	s := AddSpec{Src: src, Dest: dest}
	for _, o := range opts {
		o.applyAdd(&s)
	}
	return trigger[AddSpec]{instruction[AddSpec]{KindAdd, s, func(h *Handlers) Handler[AddSpec] { return h.Add }}}
}

func StopSignal(sig Signal, opts ...CommonOption) Trigger {
	return trigger[StopSignalSpec]{instruction[StopSignalSpec]{KindStopSignal, StopSignalSpec{collect(opts), sig}, func(h *Handlers) Handler[StopSignalSpec] { return h.StopSignal }}}
}

func Healthcheck(cmd Command, opts ...HealthcheckOption) Trigger {
	s := HealthcheckSpec{Command: cmd}
	for _, o := range opts {
		o.applyHealthcheck(&s)
	}
	return trigger[HealthcheckSpec]{instruction[HealthcheckSpec]{KindHealthcheck, s, func(h *Handlers) Handler[HealthcheckSpec] { return h.Healthcheck }}}
}

func HealthcheckNone(opts ...CommonOption) Trigger {
	s := HealthcheckSpec{Common: collect(opts), None: true}
	return trigger[HealthcheckSpec]{instruction[HealthcheckSpec]{KindHealthcheck, s, func(h *Handlers) Handler[HealthcheckSpec] { return h.Healthcheck }}}
}

// OnBuild only accepts Triggers: From/OnBuild/Maintainer inside it is a compile error.
func OnBuild(t Trigger, opts ...CommonOption) Instruction {
	return instruction[OnBuildSpec]{KindOnBuild, OnBuildSpec{collect(opts), t}, func(h *Handlers) Handler[OnBuildSpec] { return h.OnBuild }}
}
