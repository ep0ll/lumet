# BuildKit frontend context loading

The context package composes three independent operations:

- BuildDefinition loads a frontend's language/build definition and returns
  source-map diagnostics plus lazy access to its associated ignore patterns.
- IgnorePatterns either reads patterns from an already-loaded definition or
  loads an ignore file from a completely independent source.
- MainContext loads the filesystem used as the main build context. It never
  loads or parses ignore files implicitly.

There is no shared BuildContext initialization step. Each operation receives
its own options, source, and loader, so changing one source does not implicitly
change the others. Common BuildKit settings are shared through
`source.Options`; use `definition.WithSourceOptions`,
`ignore.WithSourceOptions`, or `maincontext.WithSourceOptions` with
`source.WithName`, `source.WithPlatform`, `source.WithLocalOptions`, and
related options rather than maintaining separate settings for each operation.

## Source kinds

The context/source package provides constructors for:

~~~go
source.Local("context")
source.Git("https://github.com/acme/project.git#main")
source.HTTP("https://example.com/context.tar.gz")
source.Input("build-context")
source.Image("alpine:latest")
source.OCI("my-store/image@sha256:...")
~~~

The default resolver handles local BuildKit sessions, Git references (including
refs/checksums/subdirectories and BuildKit query options), HTTP(S) files and
archives, frontend inputs, container images, and OCI layout references.
source.Registry allows handlers for individual source kinds to be replaced
or added without replacing other handlers. A complete source.Resolver can
also be supplied per operation.

## Dexfile-style use

The definition source, its adjacent .dexnore, and the main context are
configured independently:

~~~go
ctxClient, err := context.New(frontendClient)
if err != nil {
    return err
}

buildDefinition, err := ctxClient.BuildDefinition(ctx,
    definition.WithSource(source.Git("https://github.com/acme/project.git#main")),
    definition.WithFilename("Dexfile"),
    definition.WithLanguage("dexfile"),
    definition.WithFallbacks("dexfile"),
    definition.WithIgnoreFiles("Dexfile.dexnore"),
)
if err != nil {
    return err
}

patterns, err := buildDefinition.IgnorePatterns(ctx)
if err != nil {
    return err
}

main, err := ctxClient.MainContext(ctx,
    maincontext.WithSource(source.Local("context")),
    maincontext.WithIgnorePatterns(patterns...),
)
if err != nil {
    return err
}
_ = main
~~~

The definition's IgnorePatterns method lazily reads its configured ignore
files from the same resolved source. Patterns are cached only after a
successful read. Missing optional ignore files produce no patterns; read and
parse errors are returned.

## Independent ignore source

Ignore patterns do not have to live beside the definition:

~~~go
patterns, err := ctxClient.IgnorePatterns(ctx,
    ignore.WithSource(source.Input("ignore-files")),
    ignore.WithFiles(".dexnore", ".dockerignore"),
)
~~~

The first existing file wins. A local source defaults to .dockerignore, and
a direct HTTP file source uses the resolved filename automatically. Git,
gateway input, OCI layout, and image sources should specify their candidate
filenames explicitly. A custom parser can be provided with ignore.WithParser.

A raw HTTP ignore file can be loaded without an extra filename option:

~~~go
patterns, err := ctxClient.IgnorePatterns(ctx,
    ignore.WithSource(source.HTTP("https://example.com/.dexnore")),
)
~~~

## Main context without ignores

MainContext does not call BuildDefinition or IgnorePatterns:

~~~go
main, err := ctxClient.MainContext(ctx,
    maincontext.WithSource(source.Image("alpine:latest")),
    maincontext.WithoutIgnorePatterns(),
)
~~~

To apply patterns, pass them explicitly with maincontext.WithIgnorePatterns.
WithSubdirectory scopes the selected context root. For non-local sources,
filtering and scoping are represented as an LLB copy operation; local root
contexts use BuildKit's local-source exclude patterns directly.

## Replace any of the strategies

All three operations expose loader interfaces and function adapters:

~~~go
ctxClient, err := context.New(
    frontendClient,
    context.WithDefinitionLoader(definition.LoaderFunc(loadMyBuildfile)),
    context.WithIgnoreLoader(ignore.LoaderFunc(loadMyIgnoreFiles)),
    context.WithMainContextLoader(maincontext.LoaderFunc(loadMyMainContext)),
)
~~~

A strategy can also be replaced on its own without changing the others. This
is useful when a frontend already has a parser/definition reader, uses custom
ignore syntax, or needs a non-standard context source.

Named contexts remain available through NamedContext and BaseContext,
using the existing Docker image, OCI, Git, HTTP, input, and local loaders.
Their local ignore-file behavior is configurable through
loader.ContextOpt.IgnoreFiles and loader.ContextOpt.IgnoreParser.
