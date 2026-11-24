package dockerfile

import (
	"os"
	"strconv"
	"time"

	"github.com/moby/buildkit/client/llb"
	digest "github.com/opencontainers/go-digest"
	ocispecs "github.com/opencontainers/image-spec/specs-go/v1"
)

// CommonOption satisfies each typed option interface, so common metadata can
// be attached to any instruction without exposing unrelated instruction fields.
func (o CommonOption) applyFrom(s *FromSpec) { o(&s.Common) }
func (o CommonOption) applyReset(s *ResetSpec) { o(&s.FromSpec.Common) }
func (o CommonOption) applyCopy(s *CopySpec) { o(&s.Common) }
func (o CommonOption) applyAdd(s *AddSpec) { o(&s.Common) }
func (o CommonOption) applyRun(s *RunSpec) { o(&s.Common) }
func (o CommonOption) applyArg(s *ArgSpec) { o(&s.Common) }
func (o CommonOption) applyUser(s *UserSpec) { o(&s.Common) }
func (o CommonOption) applyHealthcheck(s *HealthcheckSpec) { o(&s.Common) }

// ---- FROM ----

type FromSpec struct {
	Common
	Base     Base
	Name     StageName
	Platform *ocispecs.Platform
	Options  []llb.ImageOption
}

type ResetSpec struct{ FromSpec }

type FromOption interface{ applyFrom(*FromSpec) }
type fromOption func(*FromSpec)

func (f fromOption) applyFrom(s *FromSpec) { f(s) }

func As(name StageName) FromOption {
	return fromOption(func(s *FromSpec) { s.Name = name })
}
func FromPlatform(platform ocispecs.Platform) FromOption {
	return fromOption(func(s *FromSpec) {
		p := platform
		p.OSFeatures = append([]string(nil), platform.OSFeatures...)
		s.Platform = &p
	})
}
func FromImageOptions(options ...llb.ImageOption) FromOption {
	return fromOption(func(s *FromSpec) { s.Options = append(s.Options, options...) })
}
func FromCommon(options ...CommonOption) FromOption {
	return fromOption(func(s *FromSpec) { for _, o := range options { if o != nil { o(&s.Common) } } })
}

func (s FromSpec) StageDependencies() []StageName {
	if ref, ok := s.Base.(StageRef); ok { return []StageName{ref.Name} }
	return nil
}

// ---- COPY and ADD file metadata ----

type FileOpts struct {
	Chown     *llb.ChownOpt
	ChownUser string
	Chmod     *llb.ChmodOpt
	Link      bool
	Exclude   []string
}

type FileOption func(*FileOpts)
func (f FileOption) applyCopy(s *CopySpec) { f(&s.FileOpts) }
func (f FileOption) applyAdd(s *AddSpec) { f(&s.FileOpts) }

func Chown(user string) FileOption {
	return func(options *FileOpts) { options.ChownUser = user }
}
func ChownIDs(uid, gid int) FileOption {
	value := llb.WithUIDGID(uid, gid).(llb.ChownOpt)
	return func(options *FileOpts) { options.Chown = &value }
}
func Chmod(mode os.FileMode) FileOption {
	return func(options *FileOpts) { options.Chmod = &llb.ChmodOpt{Mode: mode} }
}
func ChmodSymbolic(mode string) FileOption {
	return func(options *FileOpts) { options.Chmod = &llb.ChmodOpt{ModeStr: mode} }
}
func Link() FileOption { return func(options *FileOpts) { options.Link = true } }
func Exclude(patterns ...string) FileOption {
	return func(options *FileOpts) { options.Exclude = append(options.Exclude, patterns...) }
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

func CopyFrom(source CopySource) CopyOption {
	return copyOption(func(s *CopySpec) { s.From = source })
}
func Parents() CopyOption { return copyOption(func(s *CopySpec) { s.Parents = true }) }
func CopyCommon(options ...CommonOption) CopyOption {
	return copyOption(func(s *CopySpec) { for _, o := range options { if o != nil { o(&s.Common) } } })
}
func (s CopySpec) StageDependencies() []StageName {
	if ref, ok := s.From.(StageRef); ok { return []StageName{ref.Name} }
	return nil
}

// ---- ADD ----

type AddSource interface{ addSource() }
type LocalPath string
type URL string
type GitRepo struct{ URL, Ref, Subdir string }

func (LocalPath) addSource() {}
func (URL) addSource() {}
func (GitRepo) addSource() {}

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
func Checksum(value digest.Digest) AddOption { return addOption(func(s *AddSpec) { s.Checksum = value }) }
func Unpack(enabled bool) AddOption { return addOption(func(s *AddSpec) { s.Unpack = &enabled }) }
func AddCommon(options ...CommonOption) AddOption {
	return addOption(func(s *AddSpec) { for _, o := range options { if o != nil { o(&s.Common) } } })
}

// ---- RUN command forms, mounts and execution controls ----

type Command interface{ command() }
type ShellForm string
type ExecForm []string

func (ShellForm) command() {}
func (ExecForm) command() {}
func Sh(command string) ShellForm { return ShellForm(command) }
func Exec(args ...string) ExecForm { return ExecForm(append([]string(nil), args...)) }

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

type Mount interface{ mount() }

type BindMount struct {
	Target, Source string
	From CopySource
	ReadWrite bool
}
type CacheMount struct {
	Target, ID, Source string
	From CopySource
	Sharing llb.CacheMountSharingMode
	ReadOnly bool
	Mode os.FileMode
	UID, GID int
}
type TmpfsMount struct { Target string; Size int64 }
type SecretMount struct {
	ID, Target, Env string
	Required bool
	Mode os.FileMode
	UID, GID int
}
type SSHMount struct {
	ID, Target string
	Required bool
	Mode os.FileMode
	UID, GID int
}

func (BindMount) mount() {}
func (CacheMount) mount() {}
func (TmpfsMount) mount() {}
func (SecretMount) mount() {}
func (SSHMount) mount() {}

func Bind(target string, source CopySource, sourcePath string, readWrite bool) Mount {
	return BindMount{Target: target, From: source, Source: sourcePath, ReadWrite: readWrite}
}
func Cache(target string, options ...CacheMountOption) Mount {
	m := CacheMount{Target: target, Sharing: llb.CacheMountShared}
	for _, option := range options { if option != nil { option(&m) } }
	return m
}
type CacheMountOption func(*CacheMount)
func CacheID(id string) CacheMountOption { return func(m *CacheMount) { m.ID = id } }
func CacheFrom(source CopySource, sourcePath string) CacheMountOption {
	return func(m *CacheMount) { m.From, m.Source = source, sourcePath }
}
func CacheSharing(mode llb.CacheMountSharingMode) CacheMountOption {
	return func(m *CacheMount) { m.Sharing = mode }
}
func CacheReadOnly() CacheMountOption { return func(m *CacheMount) { m.ReadOnly = true } }
func CacheOwnership(uid, gid int, mode os.FileMode) CacheMountOption {
	return func(m *CacheMount) { m.UID, m.GID, m.Mode = uid, gid, mode }
}
func Tmpfs(target string, size int64) Mount { return TmpfsMount{Target: target, Size: size} }
func Secret(id, target string, options ...SecretMountOption) Mount {
	m := SecretMount{ID: id, Target: target, Required: true, Mode: 0o400}
	for _, option := range options { if option != nil { option(&m) } }
	return m
}
type SecretMountOption func(*SecretMount)
func SecretOptional() SecretMountOption { return func(m *SecretMount) { m.Required = false } }
func SecretRequired() SecretMountOption { return func(m *SecretMount) { m.Required = true } }
func SecretEnv(name string) SecretMountOption { return func(m *SecretMount) { m.Env = name } }
func SecretFile(uid, gid int, mode os.FileMode) SecretMountOption {
	return func(m *SecretMount) { m.UID, m.GID, m.Mode = uid, gid, mode }
}
func SSH(id, target string, options ...SSHMountOption) Mount {
	m := SSHMount{ID: id, Target: target, Mode: 0o600}
	for _, option := range options { if option != nil { option(&m) } }
	return m
}
type SSHMountOption func(*SSHMount)
func SSHRequired() SSHMountOption { return func(m *SSHMount) { m.Required = true } }
func SSHSocket(uid, gid int, mode os.FileMode) SSHMountOption {
	return func(m *SSHMount) { m.UID, m.GID, m.Mode = uid, gid, mode }
}

type UlimitSpec struct {
	Name llb.UlimitName
	Soft int64
	Hard int64
}

type RunSpec struct {
	Common
	Command Command
	Mounts []Mount
	Network NetworkMode
	Security SecurityMode
	ReadOnlyRoot bool
	Devices []llb.CDIDeviceInfo
	Ulimits []UlimitSpec
	Options []llb.RunOption
}
type RunOption interface{ applyRun(*RunSpec) }
type runOption func(*RunSpec)
func (f runOption) applyRun(s *RunSpec) { f(s) }

func Mounts(mounts ...Mount) RunOption {
	return runOption(func(s *RunSpec) { s.Mounts = append(s.Mounts, mounts...) })
}
func Network(mode NetworkMode) RunOption {
	return runOption(func(s *RunSpec) { s.Network = mode })
}
func Security(mode SecurityMode) RunOption {
	return runOption(func(s *RunSpec) { s.Security = mode })
}
func ReadOnlyRootFS() RunOption {
	return runOption(func(s *RunSpec) { s.ReadOnlyRoot = true })
}
func Devices(devices ...llb.CDIDeviceInfo) RunOption {
	return runOption(func(s *RunSpec) { s.Devices = append(s.Devices, devices...) })
}
func Ulimits(ulimits ...UlimitSpec) RunOption {
	return runOption(func(s *RunSpec) { s.Ulimits = append(s.Ulimits, ulimits...) })
}
func Device(name string, optional bool) llb.CDIDeviceInfo {
	return llb.CDIDeviceInfo{Name: name, Optional: optional}
}
func LLBRunOptions(options ...llb.RunOption) RunOption {
	return runOption(func(s *RunSpec) { s.Options = append(s.Options, options...) })
}
func RunCommon(options ...CommonOption) RunOption {
	return runOption(func(s *RunSpec) { for _, o := range options { if o != nil { o(&s.Common) } } })
}
func (s RunSpec) StageDependencies() []StageName {
	var dependencies []StageName
	for _, mount := range s.Mounts {
		switch value := mount.(type) {
		case BindMount:
			if ref, ok := value.From.(StageRef); ok { dependencies = append(dependencies, ref.Name) }
		case CacheMount:
			if ref, ok := value.From.(StageRef); ok { dependencies = append(dependencies, ref.Name) }
		}
	}
	return dependencies
}

// ---- ARG / ENV / LABEL ----

type KeyValue struct{ Key, Value string }
func KV(key, value string) KeyValue { return KeyValue{Key: key, Value: value} }

type ArgSpec struct { Common; Name string; Default *string }
type ArgOption interface{ applyArg(*ArgSpec) }
type argOption func(*ArgSpec)
func (f argOption) applyArg(s *ArgSpec) { f(s) }
func ArgDefault(value string) ArgOption { return argOption(func(s *ArgSpec) { v := value; s.Default = &v }) }
func ArgCommon(options ...CommonOption) ArgOption {
	return argOption(func(s *ArgSpec) { for _, o := range options { if o != nil { o(&s.Common) } } })
}

type EnvSpec struct { Common; Vars []KeyValue }
type LabelSpec struct { Common; Labels []KeyValue }
type MaintainerSpec struct { Common; Name string }

// ---- CMD / ENTRYPOINT / SHELL ----

type CmdSpec struct { Common; Command Command }
type EntrypointSpec struct { Common; Command Command }
type ShellSpec struct { Common; Shell ExecForm }

// ---- EXPOSE / VOLUME / WORKDIR ----

type Protocol uint8
const (
	ProtoTCP Protocol = iota
	ProtoUDP
)
type ExposedPort interface{ exposed() }
type Port struct { Number uint16; Proto Protocol }
type PortRange struct { From, To uint16; Proto Protocol }
func (Port) exposed() {}
func (PortRange) exposed() {}
func TCP(number uint16) Port { return Port{Number: number, Proto: ProtoTCP} }
func UDP(number uint16) Port { return Port{Number: number, Proto: ProtoUDP} }
func TCPRange(from, to uint16) PortRange { return PortRange{From: from, To: to, Proto: ProtoTCP} }
func UDPRange(from, to uint16) PortRange { return PortRange{From: from, To: to, Proto: ProtoUDP} }

type ExposeSpec struct { Common; Ports []ExposedPort }
type VolumeSpec struct { Common; Paths []string }
type WorkdirSpec struct { Common; Path string }

// ---- USER ----

type Principal struct { Name string; ID uint32; ByID bool }
func Named(name string) Principal { return Principal{Name: name} }
func Numeric(id uint32) Principal { return Principal{ID: id, ByID: true} }
type UserSpec struct { Common; User Principal; Group *Principal }
type UserOption interface{ applyUser(*UserSpec) }
type userOption func(*UserSpec)
func (f userOption) applyUser(s *UserSpec) { f(s) }
func WithGroup(group Principal) UserOption { return userOption(func(s *UserSpec) { g := group; s.Group = &g }) }
func UserCommon(options ...CommonOption) UserOption {
	return userOption(func(s *UserSpec) { for _, o := range options { if o != nil { o(&s.Common) } } })
}

// ---- STOPSIGNAL ----

type Signal string
const (
	SIGHUP Signal = "SIGHUP"
	SIGINT Signal = "SIGINT"
	SIGQUIT Signal = "SIGQUIT"
	SIGKILL Signal = "SIGKILL"
	SIGUSR1 Signal = "SIGUSR1"
	SIGUSR2 Signal = "SIGUSR2"
	SIGTERM Signal = "SIGTERM"
	SIGWINCH Signal = "SIGWINCH"
)
func SignalNumber(number uint8) Signal { return Signal(strconv.Itoa(int(number))) }
type StopSignalSpec struct { Common; Signal Signal }

// ---- HEALTHCHECK ----

type HealthcheckSpec struct {
	Common
	None bool
	Command Command
	Interval, Timeout, StartPeriod, StartInterval time.Duration
	Retries int
}
type HealthcheckOption interface{ applyHealthcheck(*HealthcheckSpec) }
type healthOption func(*HealthcheckSpec)
func (f healthOption) applyHealthcheck(s *HealthcheckSpec) { f(s) }
func Interval(value time.Duration) HealthcheckOption { return healthOption(func(s *HealthcheckSpec) { s.Interval = value }) }
func Timeout(value time.Duration) HealthcheckOption { return healthOption(func(s *HealthcheckSpec) { s.Timeout = value }) }
func StartPeriod(value time.Duration) HealthcheckOption { return healthOption(func(s *HealthcheckSpec) { s.StartPeriod = value }) }
func StartInterval(value time.Duration) HealthcheckOption { return healthOption(func(s *HealthcheckSpec) { s.StartInterval = value }) }
func Retries(value int) HealthcheckOption { return healthOption(func(s *HealthcheckSpec) { s.Retries = value }) }
func HealthcheckCommon(options ...CommonOption) HealthcheckOption {
	return healthOption(func(s *HealthcheckSpec) { for _, o := range options { if o != nil { o(&s.Common) } } })
}

// ---- ONBUILD ----

type OnBuildSpec struct { Common; Trigger Trigger }
func (s OnBuildSpec) StageDependencies() []StageName {
	if s.Trigger == nil { return nil }
	return s.Trigger.Dependencies()
}

// Specification interface assertions.
var (
	_ Spec = FromSpec{}
	_ Spec = RunSpec{}
	_ Spec = CopySpec{}
	_ Spec = AddSpec{}
	_ Spec = OnBuildSpec{}
)
