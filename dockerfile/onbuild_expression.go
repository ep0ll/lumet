package dockerfile

import (
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/moby/buildkit/client/llb"
)

func runOnBuildExpression(config runConfig) (string, error) {
	if len(config.options) > 0 || len(config.ulimits) > 0 || config.readOnlyRoot {
		return "", errors.New("ONBUILD RUN cannot serialize opaque LLB-only options")
	}
	command, err := commandExpression(config.command)
	if err != nil {
		return "", err
	}
	parts := []string{"RUN"}
	for _, mount := range config.mounts {
		value, err := mountExpression(mount)
		if err != nil {
			return "", err
		}
		parts = append(parts, "--mount="+dockerWord(value))
	}
	switch config.network {
	case NetworkDefault:
	case NetworkNone:
		parts = append(parts, "--network=none")
	case NetworkHost:
		parts = append(parts, "--network=host")
	default:
		return "", fmt.Errorf("unsupported RUN network mode %d", config.network)
	}
	switch config.security {
	case SecuritySandbox:
	case SecurityInsecure:
		parts = append(parts, "--security=insecure")
	default:
		return "", fmt.Errorf("unsupported RUN security mode %d", config.security)
	}
	for _, device := range config.devices {
		if device.Name == "" {
			return "", errors.New("ONBUILD RUN device name cannot be empty")
		}
		value := device.Name
		if device.Optional {
			value += ",required=false"
		}
		parts = append(parts, "--device="+dockerWord(value))
	}
	return strings.Join(append(parts, command), " "), nil
}

func copyOnBuildExpression(sources Sources, destination string, config copyConfig) (string, error) {
	if len(sources) == 0 || destination == "" {
		return "", errors.New("COPY requires source paths and destination")
	}
	parts := []string{"COPY"}
	if config.from != nil {
		from, err := copySourceText(config.from)
		if err != nil {
			return "", err
		}
		parts = append(parts, "--from="+dockerWord(from))
	}
	if config.parents {
		parts = append(parts, "--parents")
	}
	parts = append(parts, fileOptionExpressions(config.fileOptions)...)
	for _, source := range sources {
		parts = append(parts, dockerWord(string(source)))
	}
	parts = append(parts, dockerWord(destination))
	return strings.Join(parts, " "), nil
}

func addOnBuildExpression(sources []AddSource, destination string, config addConfig) (string, error) {
	if len(sources) == 0 || destination == "" {
		return "", errors.New("ADD requires sources and destination")
	}
	parts := []string{"ADD"}
	if config.keepGitDir {
		parts = append(parts, "--keep-git-dir")
	}
	if config.checksum != "" {
		parts = append(parts, "--checksum="+string(config.checksum))
	}
	if config.unpack != nil {
		parts = append(parts, "--unpack="+strconv.FormatBool(*config.unpack))
	}
	parts = append(parts, fileOptionExpressions(config.fileOptions)...)
	for _, source := range sources {
		value, err := addSourceText(source)
		if err != nil {
			return "", err
		}
		parts = append(parts, dockerWord(value))
	}
	parts = append(parts, dockerWord(destination))
	return strings.Join(parts, " "), nil
}

func commandExpression(command Command) (string, error) {
	switch value := command.(type) {
	case ShellForm:
		if strings.TrimSpace(string(value)) == "" {
			return "", errors.New("shell command cannot be empty")
		}
		return string(value), nil
	case ExecForm:
		if len(value) == 0 {
			return "", errors.New("exec command cannot be empty")
		}
		encoded, err := jsonMarshal([]string(value))
		return encoded, err
	default:
		return "", fmt.Errorf("unsupported command form %T", command)
	}
}

func fileOptionExpressions(options FileOptions) []string {
	var parts []string
	if options.ChownUser != "" {
		parts = append(parts, "--chown="+dockerWord(options.ChownUser))
	} else if options.Chown != nil {
		parts = append(parts, "--chown="+dockerWord(chownText(*options.Chown)))
	}
	if options.Chmod != nil {
		value := options.Chmod.ModeStr
		if value == "" {
			value = fmt.Sprintf("%#o", options.Chmod.Mode.Perm())
		}
		parts = append(parts, "--chmod="+dockerWord(value))
	}
	if options.Link {
		parts = append(parts, "--link")
	}
	for _, pattern := range options.Exclude {
		parts = append(parts, "--exclude="+dockerWord(pattern))
	}
	return parts
}

func chownText(value llb.ChownOpt) string {
	user := userOptionText(value.User)
	group := userOptionText(value.Group)
	if group == "" {
		return user
	}
	return user + ":" + group
}

func userOptionText(value *llb.UserOpt) string {
	if value == nil {
		return ""
	}
	if value.Name != "" {
		return value.Name
	}
	return strconv.Itoa(value.UID)
}

func mountExpression(mount Mount) (string, error) {
	var fields []string
	add := func(key, value string) error {
		if value == "" {
			return nil
		}
		if strings.ContainsAny(value, ",\r\n") {
			return fmt.Errorf("RUN mount field %q contains a comma or newline", key)
		}
		fields = append(fields, key+"="+value)
		return nil
	}
	switch value := mount.(type) {
	case BindMount:
		fields = append(fields, "type=bind")
		if value.From != nil {
			from, err := copySourceText(value.From)
			if err != nil {
				return "", err
			}
			if err := add("from", from); err != nil {
				return "", err
			}
		}
		if err := add("source", value.Source); err != nil {
			return "", err
		}
		if err := add("target", value.Target); err != nil {
			return "", err
		}
		if value.ReadWrite {
			fields = append(fields, "rw")
		} else {
			fields = append(fields, "ro")
		}

	case CacheMount:
		fields = append(fields, "type=cache")
		if err := add("id", value.ID); err != nil {
			return "", err
		}
		if value.From != nil {
			from, err := copySourceText(value.From)
			if err != nil {
				return "", err
			}
			if err := add("from", from); err != nil {
				return "", err
			}
		}
		if err := add("source", value.Source); err != nil {
			return "", err
		}
		if err := add("target", value.Target); err != nil {
			return "", err
		}
		switch value.Sharing {
		case llb.CacheMountShared:
			fields = append(fields, "sharing=shared")
		case llb.CacheMountPrivate:
			fields = append(fields, "sharing=private")
		case llb.CacheMountLocked:
			fields = append(fields, "sharing=locked")
		default:
			return "", fmt.Errorf("unsupported cache sharing mode %d", value.Sharing)
		}
		if value.ReadOnly {
			fields = append(fields, "ro")
		}
		if value.Mode != 0 {
			fields = append(fields, "mode="+strconv.FormatUint(uint64(value.Mode.Perm()), 8))
		}
		if value.UID != 0 {
			fields = append(fields, "uid="+strconv.Itoa(value.UID))
		}
		if value.GID != 0 {
			fields = append(fields, "gid="+strconv.Itoa(value.GID))
		}

	case TmpfsMount:
		fields = append(fields, "type=tmpfs")
		if err := add("target", value.Target); err != nil {
			return "", err
		}
		if value.Size > 0 {
			fields = append(fields, "size="+strconv.FormatInt(value.Size, 10))
		}

	case SecretMount:
		fields = append(fields, "type=secret")
		if err := add("id", value.ID); err != nil {
			return "", err
		}
		if err := add("target", value.Target); err != nil {
			return "", err
		}
		if err := add("env", value.Env); err != nil {
			return "", err
		}
		if !value.Required {
			fields = append(fields, "required=false")
		}
		mode := value.Mode
		if mode == 0 {
			mode = 0o400
		}
		fields = append(fields, "mode="+strconv.FormatUint(uint64(mode.Perm()), 8))
		if value.UID != 0 {
			fields = append(fields, "uid="+strconv.Itoa(value.UID))
		}
		if value.GID != 0 {
			fields = append(fields, "gid="+strconv.Itoa(value.GID))
		}

	case SSHMount:
		fields = append(fields, "type=ssh")
		if err := add("id", value.ID); err != nil {
			return "", err
		}
		if err := add("target", value.Target); err != nil {
			return "", err
		}
		if !value.Required {
			fields = append(fields, "required=false")
		}
		mode := value.Mode
		if mode == 0 {
			mode = 0o600
		}
		fields = append(fields, "mode="+strconv.FormatUint(uint64(mode.Perm()), 8))
		if value.UID != 0 {
			fields = append(fields, "uid="+strconv.Itoa(value.UID))
		}
		if value.GID != 0 {
			fields = append(fields, "gid="+strconv.Itoa(value.GID))
		}

	default:
		return "", fmt.Errorf("unsupported RUN mount type %T", mount)
	}
	return strings.Join(fields, ","), nil
}

func copySourceText(source CopySource) (string, error) {
	switch value := source.(type) {
	case StageRef:
		return string(value.Name), nil
	case ContextRef:
		return value.Name, nil
	case ImageRef:
		return value.Ref, nil
	default:
		return "", fmt.Errorf("%T cannot be serialized as a COPY source", source)
	}
}

func addSourceText(source AddSource) (string, error) {
	switch value := source.(type) {
	case LocalPath:
		return string(value), nil
	case URL:
		return string(value), nil
	case GitRepo:
		result := value.URL
		if value.Ref != "" {
			result += "#" + value.Ref
		}
		if value.Subdir != "" {
			result += ":" + strings.TrimPrefix(value.Subdir, "/")
		}
		return result, nil
	default:
		return "", fmt.Errorf("unsupported ADD source type %T", source)
	}
}
