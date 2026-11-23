# Starlark LLB

Lumet exposes BuildKit's `client/llb` API through typed Starlark values and
builtins. Adapter files mirror BuildKit's production source layout so public
API work can be reviewed file-by-file against the upstream package.

## Source layout and BuildKit parity

| BuildKit source | Starlark adapter |
| --- | --- |
| `client/llb/async.go` | `starlark/llb/async.go` |
| `client/llb/definition.go` | `starlark/llb/definition.go` |
| `client/llb/diff.go` | `starlark/llb/diff.go` |
| `client/llb/exec.go` | `starlark/llb/exec.go` |
| `client/llb/fileop.go` | `starlark/llb/fileop.go` |
| `client/llb/marshal.go` | `starlark/llb/marshal.go` |
| `client/llb/merge.go` | `starlark/llb/merge.go` |
| `client/llb/meta.go` | `starlark/llb/meta.go` |
| `client/llb/passthrough.go` | `starlark/llb/passthrough.go` |
| `client/llb/resolver.go` | `starlark/llb/resolver.go` |
| `client/llb/source.go` | `starlark/llb/source.go` |
| `client/llb/sourcemap.go` | `starlark/llb/sourcemap.go` |
| `client/llb/state.go` | `starlark/llb/state.go` |
| `client/llb/imagemetaresolver/resolver.go` | `starlark/llb/imagemetaresolver/resolver.go` |
| `client/llb/llbbuild/llbbuild.go` | `starlark/llb/llbbuild/llbbuild.go` |
| `client/llb/sourceresolver/imageresolver.go` | `starlark/llb/sourceresolver/imageresolver.go` |
| `client/llb/sourceresolver/types.go` | `starlark/llb/sourceresolver/types.go` |

BuildKit's `_test.go` files and `client/llb/llbtest` package are test code
and test helpers, not production API. They are intentionally not copied into
the runtime adapter. The additional local files under `starlark/llb`
(`names.go`, `value.go`, `builtins.go`, and `context.go`) implement the
Starlark bridge rather than mirror upstream source.

## Starlark-Go embedding model

The bridge uses the extension points from `go.starlark.net`:

- `starlark.Value` is the contract for Go-backed values: `String`,
  `Type`, `Freeze`, `Truth`, and `Hash`. Native BuildKit handles are
  represented by typed wrappers; handles without stable value identity are
  deliberately unhashable.
- `starlark.HasAttrs` implements attribute lookup and introspection. Each
  adapter normalizes incoming names in `Attr` and reports its public
  snake_case surface through `AttrNames`.
- `starlark.NewBuiltin` exposes Go functions to Starlark. The common
  `UnpackArgs` bridge normalizes callable names, parameter specifications,
  and keyword names before delegating argument validation to Starlark-Go.
- `starlark.StringDict` holds module globals. `starlarkstruct.Module`
  packages those globals into frozen modules.
- `starlark.Thread.Load` resolves `load()` statements. The host package
  owns module aliases, caching, and cycle detection.
- `starlark.ExecFileOptions` evaluates source using explicit syntax options,
  execution-step limits, print handling, an application loader, and context
  cancellation.

Starlark-Go supplies the language evaluator, core values, and standard
builtins. Lumet adds BuildKit-specific functions and typed values. Module
construction, freezing, registration, aliases, thread loaders, and evaluation
belong exclusively to `starlark/modules`. Adapter subpackages expose
`Members()` dictionaries rather than competing `Module`, `Load`, or
registration implementations.

## Identifier rules

Go-exported identifiers exposed to Starlark are converted to snake_case.
This includes module globals, callable names, keyword arguments, object
attributes, `dir()`/attribute listings, and Go-backed value type names.

| Go identifier | Starlark name |
| --- | --- |
| `SourceOp` | `source_op` |
| `Opt` | `opt` |
| `ResolveImageOpt` | `resolve_image_opt` |
| `NewImageMetaResolver` | `new_image_meta_resolver` |
| `ReadonlyRootFS` | `readonly_root_fs` |
| `CDIDeviceName` | `cdi_device_name` |
| `ResolveHTTPChecksumRequest` | `resolve_http_checksum_request` |

Normalization rejects collisions instead of silently allowing one Go
identifier to shadow another. Attribute listings are sorted and de-duplicated
after normalization.

## Go usage

```go
package main

import (
    "context"
    "fmt"

    "github.com/ep0ll/lumet/starlark/modules"
    "go.starlark.net/starlark"
)

func evaluate(source string) error {
    globals, err := modules.Exec(context.Background(), "build.star", source, nil)
    if err != nil {
        return err
    }
    fmt.Println(globals.Keys())
    return nil
}

var _ = starlark.String("the application can also build custom globals")
```

A nil predeclared dictionary installs the root `llb` module. Callers may also
use `modules.Predeclared()`, `modules.NewThread(ctx)`, or
`modules.ExecWithOptions(...)` when explicit evaluation control is needed.

Application-owned modules use the same central registry:

```go
err := modules.RegisterModule("app", starlark.StringDict{
    "build_tag": starlark.String("dev"),
})
```

The optional `ModuleLoader` passed to `NewThreadWithLoader` resolves modules
not found in the registry. Returned globals are normalized, frozen, and
cached for the lifetime of the thread.

## Starlark usage

```python
load("llb", "image")
load("llb/sourceresolver.star", "opt", "resolve_image_opt")
load("llb/sourceresolver", "new_image_meta_resolver")

base = image("alpine:latest")
resolver_options = opt(
    image = resolve_image_opt(resolve_mode = "force-pull"),
)
```

Use `load("llb", ...)` for high-level LLB functions and constants.
The `llb/imagemetaresolver`, `llb/llbbuild`, and `llb/sourceresolver`
submodules correspond to their respective BuildKit packages.
