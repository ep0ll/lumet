package namedcontext

import (
	"github.com/ep0ll/lumet/context/loader"
	"github.com/ep0ll/lumet/frontend"
	"github.com/pkg/errors"
)

const (
	contextPrefix = "context:"
	sharedKeyPrefix = "sharedkey:localdir:"
)

// Resolve looks up a previously registered context key, preferring the
// platform-qualified option and falling back to the unqualified option.
func Resolve(name string, client frontend.Client, opt loader.ContextOpt) (NamedContext, error) {
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

// ResolveBase creates a source whose input is the name itself
// (base-image style contexts).
func ResolveBase(name string, client frontend.Client, opt loader.ContextOpt) (NamedContext, error) {
	if canonical, err := Canonical(name); err == nil {
		name = canonical
	}

	qualified := Qualified(name, opt.Platform)
	if source := fromLiteral(name, qualified, client, opt); source != nil {
		return source, nil
	}
	if source := fromLiteral(name, name, client, opt); source != nil {
		return source, nil
	}
	return nil, errors.Errorf("invalid context name [%s]", name)
}

func fromKey(name, key string, client frontend.Client, opt loader.ContextOpt) *Source {
	opts := client.BuildOpts().Opts
	input, ok := opts[contextPrefix+key]
	if !ok {
		return nil
	}
	opt.SharedKey = opts[sharedKeyPrefix+key]
	return newSource(input, name, key, client, opt)
}

func fromLiteral(name, key string, client frontend.Client, opt loader.ContextOpt) *Source {
	opt.SharedKey = client.BuildOpts().Opts[sharedKeyPrefix+key]
	return newSource(name, name, key, client, opt)
}

func newSource(input, name, nameWithPlatform string, client frontend.Client, opt loader.ContextOpt) *Source {
	return &Source{
		input: input,
		name: name,
		nameWithPlatform: nameWithPlatform,
		client: client,
		opt: opt,
	}
}
