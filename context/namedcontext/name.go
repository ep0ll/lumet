package namedcontext

import (
	"strings"

	"github.com/containerd/platforms"
	"github.com/distribution/reference"
	"github.com/pkg/errors"
	ocispecs "github.com/opencontainers/image-spec/specs-go/v1"
)

// Canonical normalises a context name and removes the conventional :latest tag.
func Canonical(raw string) (string, error) {
	if strings.TrimSpace(raw) == "" {
		return "", errors.New("context name cannot be empty")
	}
	named, err := reference.ParseNormalizedNamed(strings.TrimSpace(raw))
	if err != nil {
		return "", errors.Wrapf(err, "invalid context name %q", raw)
	}
	return strings.TrimSuffix(reference.FamiliarString(named), ":latest"), nil
}

// Qualified returns the platform-qualified key name::os/arch[/variant].
func Qualified(name string, platform *ocispecs.Platform) string {
	spec := platforms.DefaultSpec()
	if platform != nil {
		spec = *platform
	}
	return name + "::" + platforms.FormatAll(platforms.Normalize(spec))
}
