package dockerfile

// Kind is a stable identifier for an instruction. It is a string rather than
// an enum so applications can register new instruction kinds without editing
// this package.
type Kind string

const (
	KindFrom        Kind = "FROM"
	KindResetStage  Kind = "FROM(reset)"
	KindArg         Kind = "ARG"
	KindEnv         Kind = "ENV"
	KindLabel       Kind = "LABEL"
	KindMaintainer  Kind = "MAINTAINER"
	KindRun         Kind = "RUN"
	KindCmd         Kind = "CMD"
	KindEntrypoint  Kind = "ENTRYPOINT"
	KindShell       Kind = "SHELL"
	KindExpose      Kind = "EXPOSE"
	KindVolume      Kind = "VOLUME"
	KindUser        Kind = "USER"
	KindWorkdir     Kind = "WORKDIR"
	KindCopy        Kind = "COPY"
	KindAdd         Kind = "ADD"
	KindStopSignal  Kind = "STOPSIGNAL"
	KindHealthcheck Kind = "HEALTHCHECK"
	KindOnBuild     Kind = "ONBUILD"
)

// HistoryMode controls how the common instruction pipeline records image history.
type HistoryMode uint8

const (
	HistoryDefault HistoryMode = iota
	HistoryLayer
	HistoryEmptyLayer
	HistorySkip
)
