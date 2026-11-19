package namedcontext

import (
	"strings"

	"github.com/ep0ll/lumet/frontend"
	"github.com/pkg/errors"
)

// Resolve looks up a registered context, preferring the platform-qualified
// option over the unqualified context option.
func Resolve(name string, client frontend.Client, opt ContextOpt) (NamedContext, error) {
	if client == nil {
		return nil, errors.New("cannot resolve a named context without a frontend client")
	}
	canonical, err := Canonical(name)
	if err != nil {
		return nil, err
	}

	qualified := Qualified(canonical, opt.Platform)
	if source := fromKey(canonical, qualified, client, opt); source != nil {
		return source, nil
	}
	if source := fromKey(canonical, canonical, client, opt); source != nil {
		return source, nil
	}
	return nil, errors.Errorf("invalid context name %q", name)
}

// ResolveBase creates a source whose input is the name itself. Bare image
// references are treated as docker-image sources; explicitly qualified
// sources (local:, input:, git:, HTTP and OCI-layout) retain their scheme.
func ResolveBase(name string, client frontend.Client, opt ContextOpt) (NamedContext, error) {
	if client == nil {
		return nil, errors.New("cannot resolve a base context without a frontend client")
	}
	if strings.TrimSpace(name) == "" {
		return nil, errors.New("base context name cannot be empty")
	}
	if canonical, err := Canonical(name); err == nil {
		name = canonical
	}

	qualified := Qualified(name, opt.Platform)
	return fromLiteral(name, qualified, client, opt), nil
}

func fromKey(name, key string, client frontend.Client, opt ContextOpt) *Source {
	value, ok := client.BuildOpts().Opts["context:"+key]
	if !ok {
		return nil
	}
	return newSource(value, name, key, client, opt)
}

func fromLiteral(name, key string, client frontend.Client, opt ContextOpt) *Source {
	input := name
	switch {
	case strings.HasPrefix(name, "docker-image:"),
		strings.HasPrefix(name, "oci-layout:"),
		strings.HasPrefix(name, "local:"),
		strings.HasPrefix(name, "input:"),
		strings.HasPrefix(name, "git:"),
		strings.HasPrefix(name, "http:"),
		strings.HasPrefix(name, "https:"),
		strings.HasPrefix(name, "git@"):
		// Already a source specifier.
	default:
		input = "docker-image://" + name
	}
	return newSource(input, name, key, client, opt)
}

func newSource(input, name, nameWithPlatform string, client frontend.Client, opt ContextOpt) *Source {
	if opt.Platform != nil {
		platform := *opt.Platform
		opt.Platform = &platform
	}
	return &Source{
		input:            input,
		name:             name,
		nameWithPlatform: nameWithPlatform,
		client:           client,
		opt:              opt,
	}
}
