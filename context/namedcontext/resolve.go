package namedcontext

import (
	"github.com/ep0ll/lumet/frontend"
	"github.com/pkg/errors"
)

// Resolve looks up a previously registered context key
// ("context:<name>" or "context:<name>::<platform>").
func Resolve(name string, client frontend.Client, opt ContextOpt) (NamedContext, error) {
	canonical, err := Canonical(name)
	if err != nil {
		return nil, err
	}

	qualified := Qualified(canonical, opt.Platform)

	if s := fromKey(canonical, qualified, client, opt); s != nil {
		return s, nil
	}
	if s := fromKey(canonical, canonical, client, opt); s != nil {
		return s, nil
	}
	return nil, errors.Errorf("invalid context name %q", name)
}

// ResolveBase creates a source whose input is the name itself
// (base-image style contexts).
func ResolveBase(name string, client frontend.Client, opt ContextOpt) (NamedContext, error) {
	if c, err := Canonical(name); err == nil {
		name = c
	}

	qualified := Qualified(name, opt.Platform)

	if s := fromLiteral(name, qualified, client, opt); s != nil {
		return s, nil
	}
	if s := fromLiteral(name, name, client, opt); s != nil {
		return s, nil
	}
	return nil, errors.Errorf("invalid context name [%s]", name)
}

func fromKey(name, key string, client frontend.Client, opt ContextOpt) *Source {
	v, ok := client.BuildOpts().Opts["context:"+key]
	if !ok {
		return nil
	}
	return newSource(v, name, key, client, opt)
}

func fromLiteral(name, key string, client frontend.Client, opt ContextOpt) *Source {
	return newSource(name, name, key, client, opt)
}

func newSource(input, name, nameWithPlatform string, client frontend.Client, opt ContextOpt) *Source {
	return &Source{
		input:            input,
		name:             name,
		nameWithPlatform: nameWithPlatform,
		client:           client,
		opt:              opt,
	}
}
