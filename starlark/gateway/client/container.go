package client

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"sync"
	"syscall"

	buildkitllb "github.com/moby/buildkit/client/llb"
	gwclient "github.com/moby/buildkit/frontend/gateway/client"
	"github.com/moby/buildkit/solver/pb"
	fstypes "github.com/tonistiigi/fsutil/types"
	"go.starlark.net/starlark"
)

type ReferenceValue struct { ref gwclient.Reference }

func WrapReference(ref gwclient.Reference) *ReferenceValue { return &ReferenceValue{ref: ref} }
func (v *ReferenceValue) Unwrap() gwclient.Reference { if v == nil { return nil }; return v.ref }
func (v *ReferenceValue) String() string { return "<gateway.client.reference>" }
func (v *ReferenceValue) Type() string { return "gateway.client.reference" }
func (v *ReferenceValue) Freeze() {}
func (v *ReferenceValue) Truth() starlark.Bool { return starlark.Bool(v != nil && v.ref != nil) }
func (v *ReferenceValue) Hash() (uint32, error) { return 0, fmt.Errorf("unhashable: gateway.client.reference") }

func (v *ReferenceValue) Attr(name string) (starlark.Value, error) {
	if v == nil || v.ref == nil { return nil, fmt.Errorf("gateway.client.reference: nil receiver") }
	switch name {
	case "to_state":
		return bind("reference.to_state", func(thread *starlark.Thread, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
			if err := starlark.UnpackArgs("reference.to_state", args, kwargs); err != nil { return nil, err }
			state, err := v.ref.ToState()
			if err != nil { return nil, err }
			return starlarkllb.WrapState(state), nil
		}), nil
	case "evaluate":
		return bind("reference.evaluate", func(thread *starlark.Thread, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
			if err := starlark.UnpackArgs("reference.evaluate", args, kwargs); err != nil { return nil, err }
			if err := v.ref.Evaluate(threadContext(thread)); err != nil { return nil, err }
			return starlark.None, nil
		}), nil
	case "read_file":
		return bind("reference.read_file", func(thread *starlark.Thread, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
			var request starlark.Value
			if err := starlark.UnpackArgs("reference.read_file", args, kwargs, "request", &request); err != nil { return nil, err }
			req, err := recordFrom[gwclient.ReadRequest](request, "read_request")
			if err != nil { return nil, err }
			data, err := v.ref.ReadFile(threadContext(thread), req)
			if err != nil { return nil, err }
			return starlark.Bytes(data), nil
		}), nil
	case "stat_file":
		return bind("reference.stat_file", func(thread *starlark.Thread, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
			var request starlark.Value
			if err := starlark.UnpackArgs("reference.stat_file", args, kwargs, "request", &request); err != nil { return nil, err }
			req, err := recordFrom[gwclient.StatRequest](request, "stat_request")
			if err != nil { return nil, err }
			stat, err := v.ref.StatFile(threadContext(thread), req)
			if err != nil { return nil, err }
			return statValue(stat), nil
		}), nil
	case "read_dir":
		return bind("reference.read_dir", func(thread *starlark.Thread, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
			var request starlark.Value
			if err := starlark.UnpackArgs("reference.read_dir", args, kwargs, "request", &request); err != nil { return nil, err }
			req, err := recordFrom[gwclient.ReadDirRequest](request, "read_dir_request")
			if err != nil { return nil, err }
			entries, err := v.ref.ReadDir(threadContext(thread), req)
			if err != nil { return nil, err }
			return statsValue(entries), nil
		}), nil
	default: return nil, nil
	}
}

func (v *ReferenceValue) AttrNames() []string { return []string{"evaluate", "read_dir", "read_file", "stat_file", "to_state"} }

type ContainerValue struct { container gwclient.Container }

func WrapContainer(container gwclient.Container) *ContainerValue { return &ContainerValue{container: container} }
func (v *ContainerValue) String() string { return "<gateway.client.container>" }
func (v *ContainerValue) Type() string { return "gateway.client.container" }
func (v *ContainerValue) Freeze() {}
func (v *ContainerValue) Truth() starlark.Bool { return starlark.Bool(v != nil && v.container != nil) }
func (v *ContainerValue) Hash() (uint32, error) { return 0, fmt.Errorf("unhashable: gateway.client.container") }

func (v *ContainerValue) Attr(name string) (starlark.Value, error) {
	if v == nil || v.container == nil { return nil, fmt.Errorf("gateway.client.container: nil receiver") }
	switch name {
	case "start":
		return bind("container.start", func(thread *starlark.Thread, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
			var value starlark.Value
			if err := starlark.UnpackArgs("container.start", args, kwargs, "request", &value); err != nil { return nil, err }
			request, ok := value.(*StartRequestValue)
			if !ok || request == nil { return nil, fmt.Errorf("container.start.request: got %s, want gateway.client.start_request", valueType(value)) }
			process, err := v.container.Start(threadContext(thread), request.request)
			if err != nil { return nil, err }
			return WrapContainerProcess(process, request), nil
		}), nil
	case "release":
		return bind("container.release", func(thread *starlark.Thread, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
			if err := starlark.UnpackArgs("container.release", args, kwargs); err != nil { return nil, err }
			if err := v.container.Release(threadContext(thread)); err != nil { return nil, err }
			return starlark.None, nil
		}), nil
	case "read_file":
		return bind("container.read_file", func(thread *starlark.Thread, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
			var request starlark.Value
			if err := starlark.UnpackArgs("container.read_file", args, kwargs, "request", &request); err != nil { return nil, err }
			req, err := recordFrom[gwclient.ReadContainerRequest](request, "read_container_request")
			if err != nil { return nil, err }
			data, err := v.container.ReadFile(threadContext(thread), req)
			if err != nil { return nil, err }
			return starlark.Bytes(data), nil
		}), nil
	case "stat_file":
		return bind("container.stat_file", func(thread *starlark.Thread, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
			var request starlark.Value
			if err := starlark.UnpackArgs("container.stat_file", args, kwargs, "request", &request); err != nil { return nil, err }
			req, err := recordFrom[gwclient.StatContainerRequest](request, "stat_container_request")
			if err != nil { return nil, err }
			stat, err := v.container.StatFile(threadContext(thread), req)
			if err != nil { return nil, err }
			return statValue(stat), nil
		}), nil
	case "read_dir":
		return bind("container.read_dir", func(thread *starlark.Thread, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
			var request starlark.Value
			if err := starlark.UnpackArgs("container.read_dir", args, kwargs, "request", &request); err != nil { return nil, err }
			req, err := recordFrom[gwclient.ReadDirContainerRequest](request, "read_dir_container_request")
			if err != nil { return nil, err }
			entries, err := v.container.ReadDir(threadContext(thread), req)
			if err != nil { return nil, err }
			return statsValue(entries), nil
		}), nil
	default: return nil, nil
	}
}

func (v *ContainerValue) AttrNames() []string { return []string{"read_dir", "read_file", "release", "start", "stat_file"} }

type lockedBuffer struct {
	mu sync.Mutex
	buffer bytes.Buffer
	closed bool
}
func (b *lockedBuffer) Write(p []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.closed { return 0, io.ErrClosedPipe }
	return b.buffer.Write(p)
}
func (b *lockedBuffer) Close() error {
	b.mu.Lock()
	b.closed = true
	b.mu.Unlock()
	return nil
}
func (b *lockedBuffer) Bytes() []byte {
	b.mu.Lock()
	defer b.mu.Unlock()
	return append([]byte(nil), b.buffer.Bytes()...)
}

type StartRequestValue struct {
	request gwclient.StartRequest
	stdout *lockedBuffer
	stderr *lockedBuffer
}

func (v *StartRequestValue) String() string { return "<gateway.client.start_request>" }
func (v *StartRequestValue) Type() string { return "gateway.client.start_request" }
func (v *StartRequestValue) Freeze() {}
func (v *StartRequestValue) Truth() starlark.Bool { return starlark.Bool(v != nil) }
func (v *StartRequestValue) Hash() (uint32, error) { return 0, fmt.Errorf("unhashable: gateway.client.start_request") }
func (v *StartRequestValue) Attr(name string) (starlark.Value, error) {
	if v == nil { return nil, fmt.Errorf("gateway.client.start_request: nil receiver") }
	switch name {
	case "args": return stringListValue(v.request.Args), nil
	case "env": return stringListValue(v.request.Env), nil
	case "user": return starlark.String(v.request.User), nil
	case "cwd": return starlark.String(v.request.Cwd), nil
	case "tty": return starlark.Bool(v.request.Tty), nil
	case "stdout": return starlark.Bytes(v.stdout.Bytes()), nil
	case "stderr": return starlark.Bytes(v.stderr.Bytes()), nil
	case "security_mode": return starlark.MakeInt(int(v.request.SecurityMode)), nil
	case "remove_mount_stubs_recursive": return starlark.Bool(v.request.RemoveMountStubsRecursive), nil
	default: return nil, nil
	}
}
func (v *StartRequestValue) AttrNames() []string {
	return []string{"args", "cwd", "env", "remove_mount_stubs_recursive", "security_mode", "stderr", "stdout", "tty", "user"}
}

func startRequestBuiltin(_ *starlark.Thread, _ *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	var argsValue, envValue, secretEnvValue, stdinValue starlark.Value = starlark.None, starlark.None, starlark.None, starlark.None
	var user, cwd string
	var tty bool
	var securityMode int
	var removeMountStubsRecursive bool
	if err := starlark.UnpackArgs("start_request", args, kwargs,
		"args?", &argsValue, "env?", &envValue, "secret_env?", &secretEnvValue,
		"user?", &user, "cwd?", &cwd, "tty?", &tty, "stdin?", &stdinValue,
		"security_mode?", &securityMode, "remove_mount_stubs_recursive?", &removeMountStubsRecursive,
	); err != nil { return nil, err }
	req := gwclient.StartRequest{
		User: user, Cwd: cwd, Tty: tty, SecurityMode: pb.SecurityMode(securityMode),
		RemoveMountStubsRecursive: removeMountStubsRecursive,
	}
	var err error
	if argsValue != starlark.None { req.Args, err = stringListAt("start_request.args", argsValue); if err != nil { return nil, err } }
	if envValue != starlark.None { req.Env, err = stringListAt("start_request.env", envValue); if err != nil { return nil, err } }
	if secretEnvValue != starlark.None {
		iterable, ok := secretEnvValue.(starlark.Iterable)
		if !ok { return nil, fmt.Errorf("start_request.secret_env: got %s, want iterable", valueType(secretEnvValue)) }
		iter := iterable.Iterate()
		for {
			var item starlark.Value
			if !iter.Next(&item) { break }
			env := &pb.SecretEnv{}
			if err := protoFromValue(item, env, "start_request.secret_env"); err != nil { iter.Done(); return nil, err }
			req.SecretEnv = append(req.SecretEnv, env)
		}
		iter.Done()
	}
	if stdinValue != starlark.None {
		data, err := bytesAt("start_request.stdin", stdinValue)
		if err != nil { return nil, err }
		req.Stdin = io.NopCloser(bytes.NewReader(data))
	}
	stdout, stderr := &lockedBuffer{}, &lockedBuffer{}
	req.Stdout, req.Stderr = stdout, stderr
	return &StartRequestValue{request: req, stdout: stdout, stderr: stderr}, nil
}

type ContainerProcessValue struct {
	process gwclient.ContainerProcess
	request *StartRequestValue
}

func WrapContainerProcess(process gwclient.ContainerProcess, request *StartRequestValue) *ContainerProcessValue {
	return &ContainerProcessValue{process: process, request: request}
}
func (v *ContainerProcessValue) String() string { return "<gateway.client.container_process>" }
func (v *ContainerProcessValue) Type() string { return "gateway.client.container_process" }
func (v *ContainerProcessValue) Freeze() {}
func (v *ContainerProcessValue) Truth() starlark.Bool { return starlark.Bool(v != nil && v.process != nil) }
func (v *ContainerProcessValue) Hash() (uint32, error) { return 0, fmt.Errorf("unhashable: gateway.client.container_process") }
func (v *ContainerProcessValue) Attr(name string) (starlark.Value, error) {
	if v == nil || v.process == nil { return nil, fmt.Errorf("gateway.client.container_process: nil receiver") }
	switch name {
	case "wait":
		return bind("container_process.wait", func(_ *starlark.Thread, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
			if err := starlark.UnpackArgs("container_process.wait", args, kwargs); err != nil { return nil, err }
			if err := v.process.Wait(); err != nil { return nil, err }
			fields := starlark.StringDict{}
			if v.request != nil {
				fields["stdout"] = starlark.Bytes(v.request.stdout.Bytes())
				fields["stderr"] = starlark.Bytes(v.request.stderr.Bytes())
			}
			return structValue("process_output", fields), nil
		}), nil
	case "resize":
		return bind("container_process.resize", func(thread *starlark.Thread, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
			var sizeValue starlark.Value
			if err := starlark.UnpackArgs("container_process.resize", args, kwargs, "size", &sizeValue); err != nil { return nil, err }
			size, err := recordFrom[gwclient.WinSize](sizeValue, "win_size")
			if err != nil { return nil, err }
			if err := v.process.Resize(threadContext(thread), size); err != nil { return nil, err }
			return starlark.None, nil
		}), nil
	case "signal":
		return bind("container_process.signal", func(thread *starlark.Thread, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
			var signal int
			if err := starlark.UnpackArgs("container_process.signal", args, kwargs, "signal", &signal); err != nil { return nil, err }
			if err := v.process.Signal(threadContext(thread), syscall.Signal(signal)); err != nil { return nil, err }
			return starlark.None, nil
		}), nil
	default: return nil, nil
	}
}
func (v *ContainerProcessValue) AttrNames() []string { return []string{"resize", "signal", "wait"} }

func statsValue(values []*fstypes.Stat) starlark.Value {
	items := make([]starlark.Value, len(values))
	for i, value := range values { items[i] = statValue(value) }
	return starlark.NewList(items)
}

func stringListValue(values []string) starlark.Value {
	items := make([]starlark.Value, len(values))
	for i, value := range values { items[i] = starlark.String(value) }
	return starlark.NewList(items)
}

var _ buildkitllb.State
var _ context.Context
