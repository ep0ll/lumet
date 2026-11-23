# starlarkllb

Starlark (Python-like) bindings for BuildKit's LLB API.

This package is a **thin wrapper** around the real Go package
`github.com/moby/buildkit/client/llb`. It does not re-implement any of the
LLB logic; it only makes the most common constructors and methods available
from Starlark scripts.

## Status

Initial foundation (created by Grok):

- `llb.image(ref)`
- `llb.scratch()`
- `State.dir(path)`
- `State.add_env(key, value)`
- `State.user(name)`
- `State.reset()`
- `State.run(cmd)` → `ExecState`
- `ExecState.root()` → `State`

Many more public APIs from `client/llb` still need to be bound
(Local, Git, HTTP, File ops, Merge, Diff, platforms, constraints, mounts, etc.).
The existing code shows the exact pattern to follow.

## Host usage

```go
package main

import (
	"fmt"
	"os"

	"github.com/ep0ll/lumet/starlarkllb"
	"go.starlark.net/starlark"
)

func main() {
	predeclared := starlark.StringDict{}
	starlarkllb.Register(predeclared)

	thread := &starlark.Thread{Name: "main"}
	globals, err := starlark.ExecFile(thread, "build.star", nil, predeclared)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	st, err := starlarkllb.AsState(globals["result"])
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	// Now you have a real llb.State that you can marshal / solve
	_ = st
}
```

## Adding new bindings

1. For a free function (e.g. `llb.local(...)`) add a `*Builtin` in `module.go`
   and implement it in the appropriate file.
2. For a method on `State` / `ExecState`:
   - add the name to `AttrNames()`
   - add a case in `Attr()` that returns a `NewBuiltin`
   - implement the method that unpacks Starlark args and calls the real
     `llb.State` / `llb.ExecState` method.

Keep the wrappers thin. Prefer returning new wrapper values rather than
mutating in place (LLB states are immutable).
