load(
    "llb",
    "Image",
    "Platform",
    "Shlex",
    "WithCustomName",
    "Scratch",
    "Mkfile",
    "Mkdir",
    "WithParents",
)

def go_builder(image):
    return Image(
        image,
        Platform(os = "linux", architecture = "amd64"),
        WithCustomName("go-builder"),
    )

builder = go_builder("golang:1.26")
compiled = builder.run(Shlex("go build -o /out/app ./...")).root()

# File actions are composable values, not mutable callbacks.
result = Scratch().file(
    Mkdir("/metadata", 0o755, WithParents(True)).mkfile(
        "/metadata/build.txt",
        0o644,
        b"graph generated from Starlark",
    )
)

# 'result' is the Go-visible *llb.State wrapper.
