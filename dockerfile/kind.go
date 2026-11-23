package dockerfile

// Kind identifies a Dockerfile instruction.
type Kind uint8

const (
	KindFrom Kind = iota
	KindResetStage
	KindArg
	KindEnv
	KindLabel
	KindMaintainer
	KindRun
	KindCmd
	KindEntrypoint
	KindShell
	KindExpose
	KindVolume
	KindUser
	KindWorkdir
	KindCopy
	KindAdd
	KindStopSignal
	KindHealthcheck
	KindOnBuild
)

var kindNames = [...]string{
	"FROM", "FROM(reset)", "ARG", "ENV", "LABEL", "MAINTAINER", "RUN", "CMD",
	"ENTRYPOINT", "SHELL", "EXPOSE", "VOLUME", "USER", "WORKDIR", "COPY", "ADD",
	"STOPSIGNAL", "HEALTHCHECK", "ONBUILD",
}

func (k Kind) String() string {
	if int(k) < len(kindNames) {
		return kindNames[k]
	}
	return "UNKNOWN"
}

// HistoryMode tells a handler how an instruction should be recorded in image history.
type HistoryMode uint8

const (
	HistoryDefault    HistoryMode = iota // handler decides (per-kind default)
	HistoryLayer                         // record as a layer-producing entry
	HistoryEmptyLayer                    // record with EmptyLayer=true (metadata only)
	HistorySkip                          // do not record
)
