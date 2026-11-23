package client_test

import (
	"context"
	"strings"
	"testing"

	buildkitllb "github.com/moby/buildkit/client/llb"
	buildkitsourceresolver "github.com/moby/buildkit/client/llb/sourceresolver"
	gwclient "github.com/moby/buildkit/frontend/gateway/client"
	"github.com/moby/buildkit/solver/pb"
	digest "github.com/opencontainers/go-digest"
	gatewayclient "github.com/ep0ll/lumet/starlark/gateway/client"
	"github.com/ep0ll/lumet/starlark/modules"
	"go.starlark.net/starlark"
)

type fakeGateway struct {
	lastSolve gwclient.SolveRequest
	resolveOptions buildkitsourceresolver.Opt
	solveCalls int
}

func (f *fakeGateway) ResolveSourceMetadata(_ context.Context, _ *pb.SourceOp, _ buildkitsourceresolver.Opt) (*buildkitsourceresolver.MetaResponse, error) {
	return nil, nil
}
func (f *fakeGateway) Solve(_ context.Context, req gwclient.SolveRequest) (*gwclient.Result, error) {
	f.lastSolve = req
	f.solveCalls++
	return gwclient.NewResult(), nil
}
func (f *fakeGateway) ResolveImageConfig(_ context.Context, _ string, opt buildkitsourceresolver.Opt) (string, digest.Digest, []byte, error) {
	f.resolveOptions = opt
	return "docker.io/library/alpine:latest", digest.Digest("sha256:abcdef"), []byte("config"), nil
}
func (f *fakeGateway) BuildOpts() gwclient.BuildOpts { return gwclient.BuildOpts{} }
func (f *fakeGateway) Inputs(context.Context) (map[string]buildkitllb.State, error) {
	return nil, nil
}
func (f *fakeGateway) NewContainer(context.Context, gwclient.NewContainerRequest) (gwclient.Container, error) {
	return nil, nil
}
func (f *fakeGateway) Warn(context.Context, digest.Digest, string, gwclient.WarnOpts) error {
	return nil
}

func TestGatewayModuleRequestBuiltins(t *testing.T) {
	ctx := context.Background()
	gateway := &fakeGateway{}
source := strings.Join([]string{
"load(",
"    \"gateway/client\",",
"    \"attestation\",",
"    \"attestation_from_pb\",",
"    \"attestation_to_pb\",",
"    \"cache_options_entry\",",
"    \"client\",",
"    \"file_range\",",
"    \"mount\",",
"    \"new_container_request\",",
"    \"new_result\",",
"    \"read_request\",",
"    \"solve_request\",",
"    \"start_request\",",
"    \"win_size\",",
")",
"",
"request = solve_request(",
"    evaluate = True,",
"    frontend = \"dockerfile.v0\",",
"    frontend_opt = {\"MODE\": \"release\"},",
"    cache_imports = [",
"        cache_options_entry(type = \"registry\", attrs = {\"ref\": \"example/cache:latest\"}),",
"    ],",
")",
"mount_value = mount(dest = \"/workspace\", readonly = True)",
"container_request = new_container_request(",
"    mounts = [mount_value],",
"    hostname = \"worker\",",
")",
"process_request = start_request(",
"    args = [\"sh\", \"-c\", \"echo hello\"],",
"    stdin = b\"input\",",
"    security_mode = 0,",
")",
"read = read_request(filename = \"/etc/os-release\", range = file_range(offset = 0, length = 64))",
"size = win_size(rows = 40, cols = 120)",
"empty_result = new_result()",
"att = attestation(path = \"attestation.json\", predicate_type = \"https://example.test/predicate\")",
"att_roundtrip = attestation_from_pb(attestation_to_pb(att))",
"",
"def build(gateway):",
"    return gateway.solve(solve_request(evaluate = True))"
}, "\n")

	globals, err := modules.ExecWithGateway(ctx, gateway, "gateway_test.star", source, nil, modules.ExecOptions{})
	if err != nil {
		t.Fatalf("ExecWithGateway() error: %v", err)
	}
	request, ok := globals["request"].(starlark.HasAttrs)
	if !ok {
		t.Fatalf("request has type %T, want starlark.HasAttrs", globals["request"])
	}
	evaluate, err := request.Attr("evaluate")
	if err != nil || evaluate != starlark.True {
		t.Fatalf("request.evaluate = %v, err = %v; want True", evaluate, err)
	}
	frontendOpt, err := request.Attr("frontend_opt")
	if err != nil {
		t.Fatalf("request.frontend_opt: %v", err)
	}
	dict, ok := frontendOpt.(*starlark.Dict)
	if !ok {
		t.Fatalf("frontend_opt has type %T, want *starlark.Dict", frontendOpt)
	}
	mode, found, err := dict.Get(starlark.String("MODE"))
	if err != nil || !found || mode != starlark.String("release") {
		t.Fatalf("frontend_opt[MODE] = %v, found=%v, err=%v", mode, found, err)
	}

	attestation, ok := globals["att_roundtrip"].(starlark.HasAttrs)
	if !ok {
		t.Fatalf("att_roundtrip has type %T, want starlark.HasAttrs", globals["att_roundtrip"])
	}
	path, err := attestation.Attr("path")
	if err != nil || path != starlark.String("attestation.json") {
		t.Fatalf("att_roundtrip.path = %v, err = %v", path, err)
	}

	emptyResult, ok := globals["empty_result"].(*gatewayclient.ResultValue)
	if !ok {
		t.Fatalf("empty_result has type %T, want *ResultValue", globals["empty_result"])
	}
	if emptyResult.Unwrap() == nil || !emptyResult.Unwrap().IsEmpty() {
		t.Fatal("new_result() should produce an empty native BuildKit result")
	}

	build, ok := globals["build"].(starlark.Callable)
	if !ok {
		t.Fatalf("build has type %T, want starlark.Callable", globals["build"])
	}
	result, err := gatewayclient.BuildCallback(build)(ctx, gateway)
	if err != nil {
		t.Fatalf("BuildCallback() error: %v", err)
	}
	if result == nil || !result.IsEmpty() {
		t.Fatal("build callback should return the fake gateway's empty result")
	}
	if gateway.solveCalls != 1 || !gateway.lastSolve.Evaluate {
		t.Fatalf("gateway Solve calls=%d, request=%+v", gateway.solveCalls, gateway.lastSolve)
	}
	if gateway.resolveOptions.ImageOpt == nil || gateway.resolveOptions.ImageOpt.ResolveMode != "pull" {
		t.Fatalf("ResolveImageConfig received options %+v; want resolve mode pull", gateway.resolveOptions)
	}

	gateway2 := &fakeGateway{}
	buildFunc := modules.BuildFunc("gateway_build.star", source, modules.ExecOptions{})
	built, err := buildFunc(ctx, gateway2)
	if err != nil {
		t.Fatalf("modules.BuildFunc() error: %v", err)
	}
	if built == nil || !built.IsEmpty() || gateway2.solveCalls != 1 {
		t.Fatalf("modules.BuildFunc() result=%v solve_calls=%d", built, gateway2.solveCalls)
	}
}
