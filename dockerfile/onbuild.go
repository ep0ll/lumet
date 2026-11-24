package dockerfile

import (
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/moby/buildkit/client/llb"
)

// OnBuildExpression serializes a typed instruction into the image-config
// representation consumed by Docker/BuildKit when a derived stage starts.
// It intentionally rejects values that cannot be represented as an instruction.
type onBuildSerializable interface {
	OnBuildExpression() (string, error)
}

func (s ArgSpec) OnBuildExpression() (string, error) {
	if s.Name == "" { return "", errors.New("ARG name cannot be empty") }
	if s.Default == nil { return "ARG " + s.Name, nil }
	return "ARG " + s.Name + "=" + dockerWord(*s.Default), nil
}
func (s EnvSpec) OnBuildExpression() (string, error) {
	if len(s.Vars) == 0 { return "", errors.New("ENV requires values") }
	parts := []string{"ENV"}
	for _, item := range s.Vars {
		if item.Key == "" { return "", errors.New("ENV key cannot be empty") }
		parts = append(parts, item.Key+"="+dockerWord(item.Value))
	}
	return strings.Join(parts, " "), nil
}
func (s LabelSpec) OnBuildExpression() (string, error) {
	if len(s.Labels) == 0 { return "", errors.New("LABEL requires values") }
	parts := []string{"LABEL"}
	for _, item := range s.Labels {
		if item.Key == "" { return "", errors.New("LABEL key cannot be empty") }
		parts = append(parts, item.Key+"="+dockerWord(item.Value))
	}
	return strings.Join(parts, " "), nil
}
func (s MaintainerSpec) OnBuildExpression() (string, error) {
	if s.Name == "" { return "", errors.New("MAINTAINER cannot be empty") }
	return "MAINTAINER " + dockerWord(s.Name), nil
}
func (s RunSpec) OnBuildExpression() (string, error) {
	if len(s.Options) > 0 || len(s.Ulimits) > 0 || s.ReadOnlyRoot {
		return "", errors.New("ONBUILD RUN contains LLB-only options that cannot be serialized")
	}
	command, err := commandExpression(s.Command)
	if err != nil { return "", err }
	parts := []string{"RUN"}
	for _, mount := range s.Mounts {
		value, err := mountExpression(mount)
		if err != nil { return "", err }
		parts = append(parts, "--mount="+dockerFlag(value))
	}
	switch s.Network {
	case NetworkDefault:
	case NetworkNone: parts = append(parts, "--network=none")
	case NetworkHost: parts = append(parts, "--network=host")
	default: return "", fmt.Errorf("unsupported RUN network %d", s.Network)
	}
	switch s.Security {
	case SecuritySandbox:
	case SecurityInsecure: parts = append(parts, "--security=insecure")
	default: return "", fmt.Errorf("unsupported RUN security %d", s.Security)
	}
	for _, device := range s.Devices {
		if device.Name == "" { return "", errors.New("RUN device name cannot be empty") }
		value := device.Name
		if device.Optional { value += ",required=false" }
		parts = append(parts, "--device="+dockerFlag(value))
	}
	return strings.Join(append(parts, command), " "), nil
}
func (s CopySpec) OnBuildExpression() (string, error) {
	if len(s.Src) == 0 || s.Dest == "" { return "", errors.New("COPY requires sources and destination") }
	parts := []string{"COPY"}
	if s.From != nil {
		from, err := copySourceText(s.From)
		if err != nil { return "", err }
		parts = append(parts, "--from="+dockerFlag(from))
	}
	if s.Parents { parts = append(parts, "--parents") }
	parts = append(parts, fileFlags(s.FileOpts)...)
	for _, src := range s.Src { parts = append(parts, dockerWord(string(src))) }
	parts = append(parts, dockerWord(s.Dest))
	return strings.Join(parts, " "), nil
}
func (s AddSpec) OnBuildExpression() (string, error) {
	if len(s.Src) == 0 || s.Dest == "" { return "", errors.New("ADD requires sources and destination") }
	parts := []string{"ADD"}
	if s.KeepGitDir { parts = append(parts, "--keep-git-dir") }
	if s.Checksum != "" { parts = append(parts, "--checksum="+string(s.Checksum)) }
	if s.Unpack != nil { parts = append(parts, "--unpack="+strconv.FormatBool(*s.Unpack)) }
	parts = append(parts, fileFlags(s.FileOpts)...)
	for _, src := range s.Src {
		value, err := addSourceText(src)
		if err != nil { return "", err }
		parts = append(parts, dockerWord(value))
	}
	parts = append(parts, dockerWord(s.Dest))
	return strings.Join(parts, " "), nil
}
func (s CmdSpec) OnBuildExpression() (string, error) {
	value, err := commandExpression(s.Command); if err != nil { return "", err }
	return "CMD " + value, nil
}
func (s EntrypointSpec) OnBuildExpression() (string, error) {
	value, err := commandExpression(s.Command); if err != nil { return "", err }
	return "ENTRYPOINT " + value, nil
}
func (s ShellSpec) OnBuildExpression() (string, error) {
	if len(s.Shell) == 0 { return "", errors.New("SHELL requires an executable") }
	value, err := json.Marshal([]string(s.Shell)); if err != nil { return "", err }
	return "SHELL " + string(value), nil
}
func (s ExposeSpec) OnBuildExpression() (string, error) {
	if len(s.Ports) == 0 { return "", errors.New("EXPOSE requires ports") }
	parts := []string{"EXPOSE"}
	for _, value := range s.Ports {
		switch port := value.(type) {
		case Port:
			if port.Number == 0 { return "", errors.New("EXPOSE port cannot be zero") }
			parts = append(parts, portKey(port.Number, port.Proto))
		case PortRange:
			if port.From == 0 || port.To < port.From { return "", errors.New("invalid EXPOSE port range") }
			for n := uint32(port.From); n <= uint32(port.To); n++ { parts = append(parts, portKey(uint16(n), port.Proto)) }
		default: return "", fmt.Errorf("unsupported exposed port %T", value)
		}
	}
	return strings.Join(parts, " "), nil
}
func (s VolumeSpec) OnBuildExpression() (string, error) {
	if len(s.Paths) == 0 { return "", errors.New("VOLUME requires paths") }
	value, err := json.Marshal(s.Paths); if err != nil { return "", err }
	return "VOLUME " + string(value), nil
}
func (s UserSpec) OnBuildExpression() (string, error) {
	value := principalString(s.User)
	if s.Group != nil { value += ":" + principalString(*s.Group) }
	if value == "" { return "", errors.New("USER cannot be empty") }
	return "USER " + dockerWord(value), nil
}
func (s WorkdirSpec) OnBuildExpression() (string, error) {
	if s.Path == "" { return "", errors.New("WORKDIR cannot be empty") }
	return "WORKDIR " + dockerWord(s.Path), nil
}
func (s StopSignalSpec) OnBuildExpression() (string, error) {
	if s.Signal == "" { return "", errors.New("STOPSIGNAL cannot be empty") }
	return "STOPSIGNAL " + string(s.Signal), nil
}
func (s HealthcheckSpec) OnBuildExpression() (string, error) {
	if s.None { return "HEALTHCHECK NONE", nil }
	command, err := commandExpression(s.Command); if err != nil { return "", err }
	parts := []string{"HEALTHCHECK"}
	if s.Interval > 0 { parts = append(parts, "--interval="+s.Interval.String()) }
	if s.Timeout > 0 { parts = append(parts, "--timeout="+s.Timeout.String()) }
	if s.StartPeriod > 0 { parts = append(parts, "--start-period="+s.StartPeriod.String()) }
	if s.StartInterval > 0 { parts = append(parts, "--start-interval="+s.StartInterval.String()) }
	if s.Retries > 0 { parts = append(parts, "--retries="+strconv.Itoa(s.Retries)) }
	return strings.Join(append(parts, command), " "), nil
}

func commandExpression(command Command) (string, error) {
	switch value := command.(type) {
	case ShellForm:
		if strings.TrimSpace(string(value)) == "" { return "", errors.New("shell command cannot be empty") }
		return string(value), nil
	case ExecForm:
		if len(value) == 0 { return "", errors.New("exec command cannot be empty") }
		encoded, err := json.Marshal([]string(value)); return string(encoded), err
	default: return "", fmt.Errorf("unsupported command form %T", command)
	}
}
func fileFlags(options FileOpts) []string {
	var parts []string
	if options.ChownUser != "" { parts = append(parts, "--chown="+dockerFlag(options.ChownUser)) } else if options.Chown != nil {
		value := chownText(*options.Chown)
		parts = append(parts, "--chown="+dockerFlag(value))
	}
	if options.Chmod != nil {
		value := options.Chmod.ModeStr
		if value == "" { value = fmt.Sprintf("%#o", options.Chmod.Mode.Perm()) }
		parts = append(parts, "--chmod="+dockerFlag(value))
	}
	if options.Link { parts = append(parts, "--link") }
	for _, pattern := range options.Exclude { parts = append(parts, "--exclude="+dockerFlag(pattern)) }
	return parts
}
func chownText(value llb.ChownOpt) string {
	user, group := userText(value.User), userText(value.Group)
	if group == "" { return user }
	return user + ":" + group
}
func userText(value *llb.UserOpt) string {
	if value == nil { return "" }
	if value.Name != "" { return value.Name }
	return strconv.Itoa(value.UID)
}
func mountExpression(mount Mount) (string, error) {
	var fields []string
	add := func(key, value string) error {
		if value == "" { return nil }
		if strings.ContainsAny(value, ",\r\n") { return fmt.Errorf("mount %s contains an unescaped comma or newline", key) }
		fields = append(fields, key+"="+value); return nil
	}
	switch value := mount.(type) {
	case BindMount:
		fields = append(fields, "type=bind")
		if value.From != nil { from, err := copySourceText(value.From); if err != nil { return "", err }; if err = add("from", from); err != nil { return "", err } }
		if err := add("source", value.Source); err != nil { return "", err }
		if err := add("target", value.Target); err != nil { return "", err }
		if value.ReadWrite { fields = append(fields, "rw") } else { fields = append(fields, "ro") }
	case CacheMount:
		fields = append(fields, "type=cache")
		for _, field := range []struct{ key, value string }{
			{"id", value.ID}, {"source", value.Source}, {"target", value.Target},
		} {
			if err := add(field.key, field.value); err != nil { return "", err }
		}
		if value.From != nil { from, err := copySourceText(value.From); if err != nil { return "", err }; if err = add("from", from); err != nil { return "", err } }
		switch value.Sharing { case llb.CacheMountShared: fields=append(fields,"sharing=shared"); case llb.CacheMountPrivate: fields=append(fields,"sharing=private"); case llb.CacheMountLocked: fields=append(fields,"sharing=locked"); default: return "", fmt.Errorf("unknown cache sharing mode %d", value.Sharing) }
		if value.ReadOnly { fields=append(fields,"ro") }
		if value.Mode != 0 { fields=append(fields,"mode="+strconv.FormatUint(uint64(value.Mode.Perm()),8)) }
		if value.UID != 0 { fields=append(fields,"uid="+strconv.Itoa(value.UID)) }
		if value.GID != 0 { fields=append(fields,"gid="+strconv.Itoa(value.GID)) }
	case TmpfsMount:
		fields=append(fields,"type=tmpfs")
		if err := add("target",value.Target); err != nil { return "",err }
		if value.Size>0 { fields=append(fields,"size="+strconv.FormatInt(value.Size,10)) }
	case SecretMount:
		fields=append(fields,"type=secret")
		if err:=add("id",value.ID);err!=nil{return "",err}; if err:=add("target",value.Target);err!=nil{return "",err}; if err:=add("env",value.Env);err!=nil{return "",err}
		fields=append(fields,"required="+strconv.FormatBool(value.Required))
		mode:=value.Mode;if mode==0{mode=0o400};fields=append(fields,"mode="+strconv.FormatUint(uint64(mode.Perm()),8))
		if value.UID!=0{fields=append(fields,"uid="+strconv.Itoa(value.UID))};if value.GID!=0{fields=append(fields,"gid="+strconv.Itoa(value.GID))}
	case SSHMount:
		fields=append(fields,"type=ssh")
		if err:=add("id",value.ID);err!=nil{return "",err};if err:=add("target",value.Target);err!=nil{return "",err}
		fields=append(fields,"required="+strconv.FormatBool(value.Required))
		mode:=value.Mode;if mode==0{mode=0o600};fields=append(fields,"mode="+strconv.FormatUint(uint64(mode.Perm()),8))
		if value.UID!=0{fields=append(fields,"uid="+strconv.Itoa(value.UID))};if value.GID!=0{fields=append(fields,"gid="+strconv.Itoa(value.GID))}
	default: return "", fmt.Errorf("unsupported RUN mount %T", mount)
	}
	return strings.Join(fields, ","), nil
}
func copySourceText(source CopySource) (string,error) {
	switch value:=source.(type) {
	case StageRef:return string(value.Name),nil
	case ContextRef:return value.Name,nil
	case ImageRef:return value.Ref,nil
	default:return "",fmt.Errorf("%T cannot be serialized as --from",source)
	}
}
func addSourceText(source AddSource)(string,error) {
	switch value:=source.(type) {
	case LocalPath:return string(value),nil
	case URL:return string(value),nil
	case GitRepo:
		result:=value.URL;if value.Ref!=""{result+="#"+value.Ref};if value.Subdir!=""{result+=":"+strings.TrimPrefix(value.Subdir,"/")};return result,nil
	default:return "",fmt.Errorf("unsupported ADD source %T",source)
	}
}
func dockerWord(value string) string {
	if strings.ContainsAny(value, " \t\r\n\"'") { return strconv.Quote(value) }
	return value
}
func dockerFlag(value string) string { return dockerWord(value) }

