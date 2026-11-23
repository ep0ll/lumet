package dockerfile

// Kind identifies a built-in Dockerfile instruction. Custom instructions use
// KindCustom and are distinguished by their registered name.
type Kind uint16

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

	// KindCustom is shared by all user-defined instruction names. The actual
	// identity is InstructionDefinition.Name, never a new enum constant.
	KindCustom
)

func (k Kind) String() string {
	switch k {
	case KindFrom:
		return "FROM"
	case KindArg:
		return "ARG"
	case KindEnv:
		return "ENV"
	case KindLabel:
		return "LABEL"
	case KindMaintainer:
		return "MAINTAINER"
	case KindRun:
		return "RUN"
	case KindCmd:
		return "CMD"
	case KindEntrypoint:
		return "ENTRYPOINT"
	case KindShell:
		return "SHELL"
	case KindExpose:
		return "EXPOSE"
	case KindVolume:
		return "VOLUME"
	case KindUser:
		return "USER"
	case KindWorkdir:
		return "WORKDIR"
	case KindCopy:
		return "COPY"
	case KindAdd:
		return "ADD"
	case KindStopSignal:
		return "STOPSIGNAL"
	case KindHealthcheck:
		return "HEALTHCHECK"
	case KindOnBuild:
		return "ONBUILD"
	case KindCustom:
		return "CUSTOM"
	case KindResetStage:
		return "FROM(reset)"
	default:
		return "UNKNOWN"
	}
}

// HistoryMode controls the image-history behavior after an instruction handler
// succeeds.
type HistoryMode uint8

const (
	// HistoryDefault leaves history management to the instruction handler.
	HistoryDefault HistoryMode = iota
	// HistoryLayer appends an entry with EmptyLayer=false.
	HistoryLayer
	// HistoryEmptyLayer appends a metadata-only history entry.
	HistoryEmptyLayer
	// HistorySkip suppresses automatic history handling.
	HistorySkip
)
