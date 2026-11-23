# starlarkllb

Starlark bindings for the full public surface of `github.com/moby/buildkit/client/llb`.

This is a **thin wrapper** layer. All logic lives in the real BuildKit LLB package;
Starlark only provides a convenient Python-like API.

## Layout (mirrors client/llb)

| File            | Corresponds to          | Status                          |
|-----------------|-------------------------|---------------------------------|
| `module.go`     | package entrypoint      | done                            |
| `source.go`     | source.go               | image, scratch, local, git, http |
| `state.go`      | state.go                | core State methods              |
| `exec.go`       | exec.go                 | ExecState + mounts              |
| `fileop.go`     | fileop.go               | FileAction + mkdir/mkfile/rm/copy/symlink |
| `merge.go`      | merge.go + diff.go      | merge, diff                     |
| `meta.go`       | meta.go                 | constraints helpers (partial)   |
| `definition.go` | definition.go           | planned                         |
| `options.go`    | various option types    | planned                         |

## Current public surface exposed to Starlark

```python
llb.image(ref)
llb.scratch()
llb.local(name)
llb.git(remote, ref="")
llb.http(url)

llb.mkdir(path, mode=0o755)
llb.mkfile(path, data, mode=0o644)
llb.rm(path)
llb.copy(input, src, dest)

llb.merge(state1, state2, ...)
llb.diff(lower, upper)

st.dir(path)
st.add_env(key, value)
st.del_env(key)
st.user(name)
st.hostname(name)
st.reset()
st.run(cmd)          # string or list → ExecState
st.file(action)      # FileAction → State
st.copy(src_state, dest)

exec.root()
exec.add_mount(target, source)
exec.get_mount(target)
```

## Host integration

```go
predeclared := starlark.StringDict{}
starlarkllb.Register(predeclared)

thread := &starlark.Thread{Name: "build"}
globals, err := starlark.ExecFile(thread, "build.star", nil, predeclared)

st, err := starlarkllb.AsState(globals["result"])
// st is a real llb.State
```

## Extending

Follow the existing pattern:

1. Add a new file that mirrors the original LLB file name when practical.
2. Create thin wrapper types that hold the real LLB value.
3. Implement `starlark.Value` + `starlark.HasAttrs`.
4. Register free functions in `module.go`.

The goal is complete coverage of every public identifier in `client/llb`.
