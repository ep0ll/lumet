package namedcontext

import (
	"strings"

	"github.com/containerd/platforms"
	"github.com/distribution/reference"
	"github.com/pkg/errors"
	ocispecs "github.com/opencontainers/image-spec/specs-go/v1"
)

// Canonical returns the normalised context name (strips conventional ":latest").
func Canonical(raw string) (string, error) {
	named, err := reference.ParseNormalizedNamed(raw)
	if err != nil {
		return "", errors.Wrapf(err, "invalid context name %q", raw)
	}
	return strings.TrimSuffix(reference.FamiliarString(named), ":latest"), nil
}

// Qualified returns the platform-qualified key "name::os/arch[/variant]".
func Qualified(name string, p *ocispecs.Platform) string {
	spec := platforms.DefaultSpec()
	if p != nil {
		spec = *p
	}
	return name + "::" + platforms.FormatAll(platforms.Normalize(spec))
}
