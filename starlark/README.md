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
and test helpers, not production API, so they are intentionally excluded.
The runtime adapter keeps the same production file layout as BuildKit.
Small shared Starlark helpers (value wrappers, identifier normalization,
builtin inventory, and thread-context access) are folded into the existing
`starlark/llb/state.go` adapter instead of adding non-BuildKit files under
`starlark/llb`. The only parallel integration package is
`starlark/modules`, which owns Starlark module registration and execution.

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

Every Starlark-visible identifier is snake_case. This includes module globals,
builtin names, keyword arguments, object attributes, method names,
`dir()`/attribute listings, and Go-backed value type names. CamelCase aliases
are not accepted for attributes or keyword arguments. Go identifiers inside
the adapter remain idiomatic Go; they are not part of the Starlark surface.

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
root = scratch()
root = base.run(shlex("apk add --no-cache ca-certificates")).root

resolver_options = opt(
    image = resolve_image_opt(resolve_mode = "force-pull"),
)
# State methods use snake_case too:
# base.async(callback)
# base.run(...)
```

Use `load("llb", ...)` for high-level LLB functions and constants.
The `llb/imagemetaresolver`, `llb/llbbuild`, and `llb/sourceresolver`
submodules correspond to their respective BuildKit packages.


## Gateway client module

~~~text
BuildKit frontend/gateway/client             Lumet Starlark adapter
-------------------------------             -----------------------
Client.Solve                                  client.solve
Client.ResolveImageConfig                     client.resolve_image_config
Client.ResolveSourceMetadata (MetaResolver)    client.resolve_source_metadata
Client.BuildOpts                              client.build_opts
Client.Inputs                                 client.inputs
Client.NewContainer                           client.new_container
Client.Warn                                   client.warn
Result / Reference                            result / reference values
Container.Start / Release                     container.start / container.release
Container.ReadFile / StatFile / ReadDir        container.read_file / stat_file / read_dir
ContainerProcess.Wait / Resize / Signal        process.wait / resize / signal
Reference.ToState / Evaluate                  reference.to_state / evaluate
Reference.ReadFile / StatFile / ReadDir        reference.read_file / stat_file / read_dir
~~~

The adapter targets the BuildKit version pinned by this repository (v0.35.1).
Its source-of-truth is frontend/gateway/client/client.go and
frontend/gateway/client/attestation.go. The Starlark module is implemented
under starlark/gateway/client; it reuses the existing typed LLB values
instead of converting LLB states or definitions into untyped dictionaries.

gateway/client is an invocation-scoped module. A BuildKit gateway Client
belongs to one frontend solve, so it must not be installed in the process-wide
registry by modules.RegisterModule. The host binds that instance to one
Starlark evaluation:

~~~go
globals, err := modules.ExecWithOptions(
    ctx,
    "build.star",
    source,
    nil,
    modules.ExecOptions{GatewayClient: gatewayClient},
)
if err != nil {
    return err
}
~~~

The same integration is available explicitly through
modules.NewThreadWithGateway and modules.ExecWithGateway. These preserve
the regular llb module registry and the optional application ModuleLoader.
The dynamic gateway/client module is cached and frozen by the existing
per-thread loader, not shared between builds.

### Starlark API

~~~python
load("llb", "image")
load(
    "gateway/client",
    "client",
    "solve_request",
    "new_container_request",
    "start_request",
    "read_request",
    "warn_opts",
)

# LLB values remain native BuildKit values behind typed Starlark wrappers.
root = image("alpine:latest")
definition = root.marshal()

# SolveRequest accepts llb.definition or llb.definition_pb.
result = client.solve(solve_request(
    definition = definition,
    evaluate = True,
    frontend_opt = {"build-arg:MODE": "release"},
))

ref = result.single_ref()
if ref != None:
    state = ref.to_state()
    ref.evaluate()
    os_release = ref.read_file(read_request("/etc/os-release"))
    print(os_release)

    # The same Reference API exposes stat_file and read_dir.
    info = ref.stat_file(stat_request("/etc/os-release"))
    print(info.size)

# Inputs are a dictionary of names to llb.state values.
for name, state in client.inputs().items():
    print(name, state)

# BuildKit's container API uses explicit request records.
container = client.new_container(new_container_request(
    hostname = "starlark-worker",
    net_mode = net_mode_unset,
))
process_request = start_request(
    args = ["sh", "-c", "printf 'hello from BuildKit\\n'"],
    cwd = "/",
    env = ["LANG=C"],
)
process = container.start(process_request)
output = process.wait()
print(output.stdout)
container.release()

# Warnings preserve the source information/range/detail fields.
client.warn(
    digest = "sha256:...",
    message = "generated by Starlark",
    opts = warn_opts(level = 1, url = "https://example.invalid"),
)
~~~

gateway/client exports snake_case request constructors for
solve_request, cache_options_entry, mount, new_container_request,
start_request, file_range, read_request, read_container_request,
read_dir_request, read_dir_container_request, stat_request,
stat_container_request, win_size, and warn_opts. Protobuf-valued request
fields (for example platform, constraints, secret_opt, and source_info)
accept dictionaries using protobuf field names or serialized protobuf bytes.
attestation constructs typed attestation values; attestation_to_pb and
attestation_from_pb bridge serialized gateway protobuf messages.

All Starlark-visible names, keyword arguments, and object attributes use
snake_case. The corresponding Go adapter aliases retain BuildKit's native Go
types for Go callers. Native references, containers, processes, and clients
are unhashable because their identity is owned by the BuildKit solve.

### BuildKit callback integration

The module package can compile a Starlark source file into a
`gateway/client.BuildFunc` with `modules.BuildFunc`:

~~~go
build := modules.BuildFunc("build.star", source, modules.ExecOptions{})
return build(ctx, gatewayClient)
~~~

The file defines `build(gateway)`, receives the invocation-scoped
`gateway.client.client` value, and returns a `gateway.client.result` produced
by `gateway.solve()` or `new_result()`. The helper evaluates the source for
each BuildKit invocation, so both the Starlark thread and dynamic module are
bound to the active gateway client. Context cancellation interrupts execution
and gateway methods receive the same invocation context.

For applications that already have a compiled `starlark.Callable`,
`gatewayclient.BuildCallback(callable)` is the lower-level adapter. `BuildFunc`
is also re-exported as a Go alias to BuildKit's native callback type.

### Gateway API coverage and deliberate boundaries

The adapter exposes the complete callable client surface (Solve,
ResolveImageConfig, ResolveSourceMetadata, BuildOpts, Inputs,
NewContainer, and Warn) and the Reference, Container, and
ContainerProcess methods. It also maps the public request/response structures
and constants to typed records, values, and snake_case constructors. The Go
package provides aliases for the native gateway interfaces and request types,
plus NewResult, AttestationToPB, and AttestationFromPB.

StartRequest.stdin accepts bytes. stdout and stderr are captured in
thread-safe buffers and are returned by process.wait() as a
process_output struct. A live Go io.ReadCloser/io.WriteCloser is not
manufactured from an arbitrary Starlark value; applications that need custom
streaming should inject a Go-backed stream value at the embedding layer.
Likewise, gateway Reference values are obtained from solve results rather
than fabricated in Starlark.

### Adapter source map

| BuildKit gateway API | Starlark adapter |
| --- | --- |
| `frontend/gateway/client/client.go` | `starlark/gateway/client/client.go` |
| `frontend/gateway/client/attestation.go` | `starlark/gateway/client/attestation.go` |
| `solver/result.Result[Reference]` | `starlark/gateway/client/result.go` |
| `Reference`, `Container`, `ContainerProcess` | `starlark/gateway/client/container.go` |
| request/response structs and protobuf conversion | `starlark/gateway/client/types.go` |
| invocation-scoped module loading | `starlark/modules/runtime.go` |


### Minimal Starlark-Go embedding example

Starlark-Go extensions are Go values and builtins, not reflected Python objects.
A Go value that should support `x.field` implements `starlark.Value` and
`starlark.HasAttrs`; a Go function is exposed with `starlark.NewBuiltin`.
Modules are immutable globals returned by `Thread.Load`; `starlarkstruct.Module`
is useful when a Go host also wants a named, attribute-bearing module value:

~~~go
package main

import (
    "fmt"
    "go.starlark.net/starlark"
    "go.starlark.net/starlarkstruct"
)

func main() {
    hello := starlark.NewBuiltin("hello", func(
        _ *starlark.Thread, _ *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple,
    ) (starlark.Value, error) {
        var name string
        if err := starlark.UnpackArgs("hello", args, kwargs, "name", &name); err != nil {
            return nil, err
        }
        return starlark.String("hello, " + name), nil
    })

    module := &starlarkstruct.Module{
        Name: "demo",
        Members: starlark.StringDict{"hello": hello},
    }
    module.Freeze()

    thread := &starlark.Thread{
        Name: "lumet",
        Load: func(_ *starlark.Thread, name string) (starlark.StringDict, error) {
            if name == "demo" {
                return module.Members, nil
            }
            return nil, fmt.Errorf("unknown module %q", name)
        },
    }
    globals, err := starlark.ExecFile(thread, "build.star", `
load("demo", "hello")
message = hello(name = "BuildKit")
`, nil)
    if err != nil {
        panic(err)
    }
    fmt.Println(globals["message"])
}
~~~

The official `starlark/example_test.go` is the reference for minimal embedding,
custom builtins, and sequential/parallel `load()` caching; Lumet adds
thread-local context, cycle detection, execution limits, and dynamic gateway
client scoping on top of those primitives.

## Starlark-Go embedding notes

The adapter intentionally uses only the public go.starlark.net packages.
The upstream tree can be read in functional layers:

- syntax/ contains scanning, parsing, AST walking, quoting, and file options.
- resolve/ performs lexical binding and static name resolution.
- starlark/ provides the evaluator, interpreter, core values, argument
  unpacking, builtins, thread execution, profiling, and debugging.
- starlarkstruct/ provides immutable attribute-bearing structs and modules.
- starlarktest/ provides testing helpers and the assert module.
- starlarkjson/, lib/json/, lib/math/, lib/proto/, and lib/time/
  provide optional library integrations.
- repl/ and cmd/starlark/ implement the interactive and command-line hosts.
  internal/compile/, internal/chunkedfile/, and internal/spell/ are
  implementation details and are not imported by Lumet.

The repository's source tree also includes .star language fixtures,
*_test.go conformance tests, benchmarks, generated HTML docs, and build
scripts. Those are valuable for learning semantics and expected errors, but
they are not extension points for an embedding application. The examples in
starlark/example_test.go are the shortest runnable reference for
starlark.ExecFile, starlark.Thread.Load, sequential/parallel module loads,
and custom builtins.

The main embedding contracts used here are:

- Implement starlark.Value for Go-backed values (String, Type, Freeze,
  Truth, and Hash); implement starlark.HasAttrs for typed attribute access
  and deterministic AttrNames.
- Use starlark.NewBuiltin and starlark.UnpackArgs for typed native calls.
  Avoid reflection-based exposure of arbitrary Go methods.
- Return immutable starlark.StringDict module globals. Wrap them with
  starlarkstruct.Module when a module value is useful, and serve load()
  through starlark.Thread.Load.
- Use a per-thread module cache and cycle detection. The same import must
  return the same frozen globals during a thread's lifetime.
- Configure language features with syntax.FileOptions, and execute with
  starlark.ExecFileOptions. Lumet additionally applies execution-step limits,
  print hooks, thread-local context, and context cancellation.
- Keep evaluator globals and module caches per evaluation. Gateway clients
  are per solve and must not leak into a global registry.

These boundaries keep the language runtime separate from BuildKit's domain
objects: Starlark-Go owns parsing/evaluation and the core value model; Lumet
owns the typed BuildKit bridge, module lifetime, and conversion of Go
identifiers to snake_case.
