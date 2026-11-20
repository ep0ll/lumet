# Frontend-neutral build context

The context package loads a frontend's build definition, main build context,
and named contexts without assuming that the definition is a Dockerfile. A
frontend supplies a small frontend.Spec describing its conventions; the
loader owns the BuildKit session, input, source, ignore-file, and source-map
plumbing.

## Naming and ignore-file precedence

A frontend.Spec declares:

- DefaultEntrypoint and optional EntrypointFallback paths.
- The default main-context and build-definition local names.
- DefinitionIgnoreFiles, checked first and treated as an entrypoint-specific
  override.
- IgnoreFiles, checked next against the main context.
- An optional IgnoreParser; without one the Moby ignorefile parser is used.

Only the first existing ignore file is parsed. A missing candidate advances
to the next candidate. Permission, transport, size-limit, and parse failures
are returned instead of being silently treated as an absent file.

## Dockerfile frontend

Dockerfile.dockerignore takes precedence over .dockerignore, matching the
usual Dockerfile frontend convention.

~~~go
dockerfileSpec := frontend.Spec{
    Name: "dockerfile",
    DefaultEntrypoint: func(string) string {
        return "Dockerfile"
    },
    EntrypointFallback: frontend.LowerBaseFallback("Dockerfile"),
    DefinitionIgnoreFiles: func(filename string) []string {
        return []string{filename + ".dockerignore"}
    },
    IgnoreFiles: func(string) []string {
        return []string{".dockerignore"}
    },
    DefaultContextLocalName: "context",
    EntrypointLocalNames:    []string{"dockerfile"},
}
~~~

## Dexfile frontend

Dexfile reads the ignore file adjacent to its build definition. SuffixIgnore
describes that convention without adding Dexfile-specific behavior to the
context package.

~~~go
dexfileSpec := frontend.Spec{
    Name: "dexfile",
    DefaultEntrypoint: func(string) string {
        return "Dexfile"
    },
    EntrypointFallback:      frontend.LowerBaseFallback("Dexfile"),
    DefinitionIgnoreFiles: frontend.SuffixIgnore(".dexnore"),
    DefaultContextLocalName: "context",
    EntrypointLocalNames:    []string{"dockerfile"},
    OptKeys: frontend.ContextOptKeys{
        EntrypointLocalNames: []string{"dexfilekey", "dockerfilekey"},
        EntrypointSubDir:     "dexfilesubdir",
    },
}
~~~

## DALEC and other frontends

DALEC currently reads its spec through BuildKit's Dockerfile UI entrypoint
reader using the dalec language identifier. A compatible adapter can use the
same Dockerfile entrypoint and ignore-file conventions while setting
Name: "dalec" so source-map diagnostics retain the correct language.

Frontends with a different format only need to change the spec. For example,
a YAML-based frontend can set DefaultEntrypoint to its YAML filename,
provide a YAML-specific fallback, and supply a parser if its ignore syntax
differs from Docker's.

## Loading

~~~go
client, err := frontend.NewClient(gatewayClient)
if err != nil {
    return nil, err
}

contexts := context.New(dockerfileSpec, client)

definition, err := contexts.BuildDefinition(ctx)
if err != nil {
    return nil, err
}
ignorePatterns, err := definition.IgnorePatterns(ctx)
if err != nil {
    return nil, err
}

main, err := contexts.MainContext(ctx)
if err != nil {
    return nil, err
}
~~~

BuildDefinition returns context/definition.Source, which exposes source-map
locations, warning forwarding, and lazily loaded ignore patterns. MainContext
returns external Git/HTTP/input states as-is; for local sources it attaches
the parsed ignore patterns to the LLB local source.

## Custom source schemes

The default resolver understands BuildKit Git references, HTTP(S) sources,
frontend inputs, and local sessions. Add another source scheme without
changing the orchestration layer:

~~~go
type customResolver struct{}

func (customResolver) Resolve(
    ctx context.Context,
    input string,
    opt context.ResolveOptions,
    client frontend.Client,
) (*context.ResolvedSource, error) {
    // Resolve a custom URI to an LLB state. Return nil, nil when the input
    // belongs to the normal local-session or frontend-input path.
    return nil, nil
}

contexts := context.NewWithOptions(
    customSpec,
    client,
    context.WithSourceResolver(customResolver{}),
)
~~~

Named contexts retain the existing local, Git, HTTP, image, OCI-layout, and
frontend-input loaders. ContextOpt.IgnoreFiles and ContextOpt.IgnoreParser
allow those local named contexts to use the frontend's own ignore convention.
