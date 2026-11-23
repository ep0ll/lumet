package starlarkllb

import (
	"fmt"

	"github.com/moby/buildkit/client/llb"
	"go.starlark.net/starlark"
)

// imageBuiltin implements llb.image(ref, **opts)
func imageBuiltin(thread *starlark.Thread, fn *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	var ref string
	if err := starlark.UnpackArgs(fn.Name(), args, kwargs, "ref", &ref); err != nil {
		return nil, err
	}

	// TODO: full ImageOption support (platform, resolve_digest, meta_resolver, etc.)
	st := llb.Image(ref)
	return &State{state: st}, nil
}

// scratchBuiltin implements llb.scratch()
func scratchBuiltin(thread *starlark.Thread, fn *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	if err := starlark.UnpackArgs(fn.Name(), args, kwargs); err != nil {
		return nil, err
	}
	return &State{state: llb.Scratch()}, nil
}

// localBuiltin implements llb.local(name, **opts)
func localBuiltin(thread *starlark.Thread, fn *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	var name string
	if err := starlark.UnpackArgs(fn.Name(), args, kwargs, "name", &name); err != nil {
		return nil, err
	}

	// TODO: LocalOption support (include_patterns, exclude_patterns, follow_paths, session_id, ...)
	st := llb.Local(name)
	return &State{state: st}, nil
}

// gitBuiltin implements llb.git(remote, ref, **opts)
func gitBuiltin(thread *starlark.Thread, fn *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	var remote, ref string
	if err := starlark.UnpackArgs(fn.Name(), args, kwargs, "remote", &remote, "ref?", &ref); err != nil {
		return nil, err
	}

	var opts []llb.GitOption
	if ref != "" {
		opts = append(opts, llb.GitRef(ref))
	}

	// TODO: full GitOption surface (KeepGitDir, AuthTokenSecret, MountSSHSock, etc.)
	st := llb.Git(remote, ref, opts...)
	return &State{state: st}, nil
}

// httpBuiltin implements llb.http(url, **opts)
func httpBuiltin(thread *starlark.Thread, fn *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	var url string
	if err := starlark.UnpackArgs(fn.Name(), args, kwargs, "url", &url); err != nil {
		return nil, err
	}

	// TODO: HTTPOption support (Checksum, Header, ...)
	st := llb.HTTP(url)
	return &State{state: st}, nil
}

// shlexBuiltin is a convenience helper.
func shlexBuiltin(thread *starlark.Thread, fn *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	var s string
	if err := starlark.UnpackArgs(fn.Name(), args, kwargs, "s", &s); err != nil {
		return nil, err
	}
	// Real implementation can use github.com/google/shlex or buildkit's internal helper.
	list := starlark.NewList([]starlark.Value{starlark.String(s)})
	return list, nil
}

// AsState converts a Starlark value back to llb.State.
func AsState(v starlark.Value) (llb.State, error) {
	s, ok := v.(*State)
	if !ok {
		return llb.State{}, fmt.Errorf("expected llb.State, got %s", v.Type())
	}
	return s.state, nil
}
