package dockerfile

import (
	"os"
	"time"

	"github.com/moby/buildkit/client/llb"
	"github.com/moby/buildkit/solver/pb"
	"github.com/opencontainers/go-digest"
	ocispecs "github.com/opencontainers/image-spec/specs-go/v1"
)

// ---- Common (valid on every instruction) ----

type Common struct {
	History     HistoryFunc
	HistoryMode HistoryMode
	Constraints []llb.ConstraintsOpt // llb.WithCustomName, llb.ProgressGroup, ...
}

func (c Common) commonOf() Common { return c }

// CommonOption is accepted by every instruction, and satisfies every
// instruction-specific option interface below.
type CommonOption func(*Common)

func WithHistory(fn HistoryFunc) CommonOption { return func(c *Common) { c.History = fn } }
func WithHistoryMode(m HistoryMode) CommonOption {
	return func(c *Common) { c.HistoryMode = m }
}
func WithConstraints(o ...llb.ConstraintsOpt) CommonOption {
	return func(c *Common) { c.Constraints = append(c.Constraints, o...) }
}

func (o CommonOption) applyFrom(s *FromSpec)               { o(&s.Common) }
func (o CommonOption) applyCopy(s *CopySpec)               { o(&s.Common) }
func (o CommonOption) applyAdd(s *AddSpec)                 { o(&s.Common) }
func (o CommonOption) applyRun(s *RunSpec)                 { o(&s.Common) }
func (o CommonOption) applyArg(s *ArgSpec)                 { o(&s.Common) }
func (o CommonOption) applyUser(s *UserSpec)               { o(&s.Common) }
func (o CommonOption) applyHealthcheck(s *HealthcheckSpec) { o(&s.Common) }

func collect(opts []CommonOption) Common {
	var c Common
	for _, o := range opts {
		o(&c)
	}
	return c
}

// ---- FROM ----

type FromSpec struct {
	Common
	Base     Base
	Name     StageName
	Platform *ocispecs.Platform
}
type ResetSpec struct{ FromSpec }

type FromOption interface{ applyFrom(*FromSpec) }
type fromOption func(*FromSpec)

func (f fromOption) applyFrom(s *FromSpec) { f(s) }

func As(name StageName) FromOption { return fromOption(func(s *FromSpec) { s.Name = name }) }
func FromPlatform(p ocispecs.Platform) FromOption {
	return fromOption(func(s *FromSpec) { s.Platform = &p })
}

// ---- shared file options (COPY + ADD) ----

type FileOpts struct {
	Chown   *llb.ChownOpt
	Chmod   *llb.ChmodOpt
	Link    bool
	Exclude []string
}

type FileOption func(*FileOpts)

func (f FileOption) applyCopy(s *CopySpec) { f(&s.FileOpts) }
func (f FileOption) applyAdd(s *AddSpec)   { f(&s.FileOpts) }

func Chown(user string) FileOption { // "user", "uid", "user:group", "uid:gid"
	o := llb.WithUser(user).(llb.ChownOpt)
	return func(f *FileOpts) { f.Chown = &o }
}
func ChownIDs(uid, gid int) FileOption {
	o := llb.WithUIDGID(uid, gid).(llb.ChownOpt)
	return func(f *FileOpts) { f.Chown = &o }
}
func Chmod(mode os.FileMode) FileOption {
	return func(f *FileOpts) { f.Chmod = &llb.ChmodOpt{Mode: mode} }
}
func ChmodSymbolic(s string) FileOption { // e.g. "u+x,g=u"
	return func(f *FileOpts) { f.Chmod = &llb.ChmodOpt{ModeStr: s} }
}
func Link() FileOption { return func(f *FileOpts) { f.Link = true } }
func Exclude(patterns ...string) FileOption {
	return func(f *FileOpts) { f.Exclude = append(f.Exclude, patterns...) }
}

// ---- COPY ----

type Sources []string

type CopySpec struct {
	Common
	FileOpts
	Src     Sources
	Dest    string
	From    CopySource
	Parents bool
}
type CopyOption interface{ applyCopy(*CopySpec) }
type copyOption func(*CopySpec)

func (f copyOption) applyCopy(s *CopySpec) { f(s) }

func CopyFrom(src CopySource) CopyOption { return copyOption(func(s *CopySpec) { s.From = src }) }
func Parents() CopyOption                { return copyOption(func(s *CopySpec) { s.Parents = true }) }

// ---- ADD ----

type AddSource interface{ addSource() }
type LocalPath string
type URL string
type GitRepo struct{ URL, Ref string }

func (LocalPath) addSource() {}
func (URL) addSource()       {}
func (GitRepo) addSource()   {}

type AddSpec struct {
	Common
	FileOpts
	Src        []AddSource
	Dest       string
	KeepGitDir bool
	Checksum   digest.Digest
	Unpack     *bool
}
type AddOption interface{ applyAdd(*AddSpec) }
type addOption func(*AddSpec)

func (f addOption) applyAdd(s *AddSpec) { f(s) }

func KeepGitDir() AddOption { return addOption(func(s *AddSpec) { s.KeepGitDir = true }) }
func Checksum(d digest.Digest) AddOption {
	return addOption(func(s *AddSpec) { s.Checksum = d })
}
func Unpack(v bool) AddOption { return addOption(func(s *AddSpec) { s.Unpack = &v }) }

// ---- RUN / CMD / ENTRYPOINT command forms ----

type Command interface{ command() }
type ShellForm string
type ExecForm []string

func (ShellForm) command() {}
func (ExecForm) command()  {}

func Sh(cmd string) ShellForm       { return ShellForm(cmd) }
func Exec(args ...string) ExecForm { return ExecForm(args) }

type NetworkMode uint8

const (
	NetworkDefault NetworkMode = iota
	NetworkNone
	NetworkHost
)

type SecurityMode uint8

const (
	SecuritySandbox SecurityMode = iota
	SecurityInsecure
)

// Sealed mount types for RUN --mount.
type Mount interface{ mount() }

type BindMount struct {
	Target, Source string
	From           CopySource
	ReadWrite      bool
}
type CacheMount struct {
	Target, ID, Source string
	From               CopySource
	Sharing            llb.CacheMountSharingMode
	ReadOnly           bool
	Mode               os.FileMode
	UID, GID           int
}
type TmpfsMount struct {
	Target string
	Size   int64
}
type SecretMount struct {
	ID, Target, Env string
	Required        bool
	Mode            os.FileMode
	UID, GID        int
}
type SSHMount struct {
	ID, Target string
	Required   bool
	Mode       os.FileMode
	UID, GID   int
}

func (BindMount) mount()   {}
func (CacheMount) mount()  {}
func (TmpfsMount) mount()  {}
func (SecretMount) mount() {}
func (SSHMount) mount()    {}

type RunSpec struct {
	Common
	Command  Command
	Mounts   []Mount
	Network  NetworkMode
	Security SecurityMode
	Devices  []llb.CDIDeviceInfo // RUN --device
}
type RunOption interface{ applyRun(*RunSpec) }
type runOption func(*RunSpec)

func (f runOption) applyRun(s *RunSpec) { f(s) }

func Mounts(m ...Mount) RunOption { return runOption(func(s *RunSpec) { s.Mounts = append(s.Mounts, m...) }) }
func Network(n NetworkMode) RunOption   { return runOption(func(s *RunSpec) { s.Network = n }) }
func Security(m SecurityMode) RunOption { return runOption(func(s *RunSpec) { s.Security = m }) }
func Devices(d ...llb.CDIDeviceInfo) RunOption {
	return runOption(func(s *RunSpec) { s.Devices = append(s.Devices, d...) })
}

var (
	_ = pb.NetMode_UNSET // network/security map onto llb.NetMode*/SecurityMode* in your handler
)

// ---- ARG / ENV / LABEL ----

type KeyValue struct{ Key, Value string }

func KV(k, v string) KeyValue { return KeyValue{k, v} }

type ArgSpec struct {
	Common
	Name    string
	Default *string
}
type ArgOption interface{ applyArg(*ArgSpec) }
type argOption func(*ArgSpec)

func (f argOption) applyArg(s *ArgSpec) { f(s) }
func ArgDefault(v string) ArgOption     { return argOption(func(s *ArgSpec) { s.Default = &v }) }

type EnvSpec struct {
	Common
	Vars []KeyValue
}
type LabelSpec struct {
	Common
	Labels []KeyValue
}
type MaintainerSpec struct {
	Common
	Name string
}

// ---- CMD / ENTRYPOINT / SHELL ----

type CmdSpec struct {
	Common
	Command Command
}
type EntrypointSpec struct {
	Common
	Command Command
}
type ShellSpec struct {
	Common
	Shell ExecForm
}

// ---- EXPOSE / VOLUME / WORKDIR ----

type Protocol uint8

const (
	ProtoTCP Protocol = iota
	ProtoUDP
)

type ExposedPort interface{ exposed() }
type Port struct {
	Number uint16
	Proto  Protocol
}
type PortRange struct {
	From, To uint16
	Proto    Protocol
}

func (Port) exposed()      {}
func (PortRange) exposed() {}

func TCP(n uint16) Port                  { return Port{n, ProtoTCP} }
func UDP(n uint16) Port                  { return Port{n, ProtoUDP} }
func TCPRange(from, to uint16) PortRange { return PortRange{from, to, ProtoTCP} }
func UDPRange(from, to uint16) PortRange { return PortRange{from, to, ProtoUDP} }

type ExposeSpec struct {
	Common
	Ports []ExposedPort
}
type VolumeSpec struct {
	Common
	Paths []string
}
type WorkdirSpec struct {
	Common
	Path string
}

// ---- USER ----

type Principal struct {
	Name string
	ID   uint32
	ByID bool
}

func Named(name string) Principal { return Principal{Name: name} }
func Numeric(id uint32) Principal { return Principal{ID: id, ByID: true} }

type UserSpec struct {
	Common
	User  Principal
	Group *Principal
}
type UserOption interface{ applyUser(*UserSpec) }
type userOption func(*UserSpec)

func (f userOption) applyUser(s *UserSpec) { f(s) }
func WithGroup(g Principal) UserOption     { return userOption(func(s *UserSpec) { s.Group = &g }) }

// ---- STOPSIGNAL ----

type Signal string

const (
	SIGHUP  Signal = "SIGHUP"
	SIGINT  Signal = "SIGINT"
	SIGQUIT Signal = "SIGQUIT"
	SIGKILL Signal = "SIGKILL"
	SIGUSR1 Signal = "SIGUSR1"
	SIGUSR2 Signal = "SIGUSR2"
	SIGTERM Signal = "SIGTERM"
	SIGWINCH Signal = "SIGWINCH"
)

func SignalNumber(n uint8) Signal { return Signal(string(rune('0'+n/10)) + string(rune('0'+n%10))) }

type StopSignalSpec struct {
	Common
	Signal Signal
}

// ---- HEALTHCHECK ----

type HealthcheckSpec struct {
	Common
	None          bool
	Command       Command
	Interval      time.Duration
	Timeout       time.Duration
	StartPeriod   time.Duration
	StartInterval time.Duration
	Retries       int
}
type HealthcheckOption interface{ applyHealthcheck(*HealthcheckSpec) }
type healthOption func(*HealthcheckSpec)

func (f healthOption) applyHealthcheck(s *HealthcheckSpec) { f(s) }

func Interval(d time.Duration) HealthcheckOption {
	return healthOption(func(s *HealthcheckSpec) { s.Interval = d })
}
func Timeout(d time.Duration) HealthcheckOption {
	return healthOption(func(s *HealthcheckSpec) { s.Timeout = d })
}
func StartPeriod(d time.Duration) HealthcheckOption {
	return healthOption(func(s *HealthcheckSpec) { s.StartPeriod = d })
}
func StartInterval(d time.Duration) HealthcheckOption {
	return healthOption(func(s *HealthcheckSpec) { s.StartInterval = d })
}
func Retries(n int) HealthcheckOption {
	return healthOption(func(s *HealthcheckSpec) { s.Retries = n })
}

// ---- ONBUILD ----

type OnBuildSpec struct {
	Common
	Trigger Trigger
}
