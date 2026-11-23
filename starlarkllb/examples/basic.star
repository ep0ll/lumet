# Example Starlark file that builds a tiny Go binary using the llb module.
#
# The host application must register the module with:
#   starlarkllb.Register(predeclared)

st = llb.image("golang:1.22-alpine")

st = st.dir("/src")
st = st.add_env("CGO_ENABLED", "0")
st = st.add_env("GOOS", "linux")

# Simple run (string form → /bin/sh -c ...)
st = st.run("go build -o /out/app ./...").root()

# The final state is left in the global "result" so the host can extract it.
result = st
