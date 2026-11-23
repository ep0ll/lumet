package client

import (
	"fmt"

	gwclient "github.com/moby/buildkit/frontend/gateway/client"
	"go.starlark.net/starlark"
)

// NewResult returns BuildKit's native empty result.
func NewResult() *gwclient.Result { return gwclient.NewResult() }

type ResultValue struct { result *gwclient.Result }

func WrapResult(value *gwclient.Result) *ResultValue { return &ResultValue{result: value} }
func (v *ResultValue) Unwrap() *gwclient.Result { if v == nil { return nil }; return v.result }
func (v *ResultValue) String() string { return "<gateway.client.result>" }
func (v *ResultValue) Type() string { return "gateway.client.result" }
func (v *ResultValue) Freeze() {}
func (v *ResultValue) Truth() starlark.Bool { return starlark.Bool(v != nil && v.result != nil) }
func (v *ResultValue) Hash() (uint32, error) { return 0, fmt.Errorf("unhashable: gateway.client.result") }

func (v *ResultValue) Attr(name string) (starlark.Value, error) {
	if v == nil || v.result == nil { return nil, fmt.Errorf("gateway.client.result: nil receiver") }
	switch name {
	case "ref":
		if v.result.Ref == nil { return starlark.None, nil }
		return WrapReference(v.result.Ref), nil
	case "refs":
		dict := starlark.NewDict(len(v.result.Refs))
		for key, ref := range v.result.Refs { if err := dict.SetKey(starlark.String(key), WrapReference(ref)); err != nil { return nil, err } }
		return dict, nil
	case "metadata":
		return stringBytesMapValue(v.result.Metadata), nil
	case "attestations":
		dict := starlark.NewDict(len(v.result.Attestations))
		for key, attestations := range v.result.Attestations {
			values := make([]starlark.Value, len(attestations))
			for i := range attestations { values[i] = WrapAttestation(&attestations[i]) }
			if err := dict.SetKey(starlark.String(key), starlark.NewList(values)); err != nil { return nil, err }
		}
		return dict, nil
	case "clone":
		return bind("result.clone", func(_ *starlark.Thread, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
			if err := starlark.UnpackArgs("result.clone", args, kwargs); err != nil { return nil, err }
			return WrapResult(v.result.Clone()), nil
		}), nil
	case "add_meta":
		return bind("result.add_meta", func(_ *starlark.Thread, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
			var key string
			var data starlark.Value
			if err := starlark.UnpackArgs("result.add_meta", args, kwargs, "key", &key, "value", &data); err != nil { return nil, err }
			value, err := bytesAt("result.add_meta.value", data)
			if err != nil { return nil, err }
			v.result.AddMeta(key, value)
			return starlark.None, nil
		}), nil
	case "add_ref":
		return bind("result.add_ref", func(_ *starlark.Thread, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
			var key string
			var value starlark.Value
			if err := starlark.UnpackArgs("result.add_ref", args, kwargs, "key", &key, "ref", &value); err != nil { return nil, err }
			ref, ok := value.(*ReferenceValue)
			if !ok || ref == nil || ref.ref == nil { return nil, fmt.Errorf("result.add_ref.ref: got %s, want gateway.client.reference", valueType(value)) }
			v.result.AddRef(key, ref.ref)
			return starlark.None, nil
		}), nil
	case "set_ref":
		return bind("result.set_ref", func(_ *starlark.Thread, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
			var value starlark.Value
			if err := starlark.UnpackArgs("result.set_ref", args, kwargs, "ref", &value); err != nil { return nil, err }
			ref, ok := value.(*ReferenceValue)
			if !ok || ref == nil { return nil, fmt.Errorf("result.set_ref.ref: got %s, want gateway.client.reference", valueType(value)) }
			v.result.SetRef(ref.ref)
			return starlark.None, nil
		}), nil
	case "find_ref":
		return bind("result.find_ref", func(_ *starlark.Thread, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
			var key string
			if err := starlark.UnpackArgs("result.find_ref", args, kwargs, "key", &key); err != nil { return nil, err }
			ref, found := v.result.FindRef(key)
			if !found || ref == nil { return starlark.Tuple{starlark.None, starlark.False}, nil }
			return starlark.Tuple{WrapReference(ref), starlark.True}, nil
		}), nil
	case "single_ref":
		return bind("result.single_ref", func(_ *starlark.Thread, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
			if err := starlark.UnpackArgs("result.single_ref", args, kwargs); err != nil { return nil, err }
			ref, err := v.result.SingleRef()
			if err != nil { return nil, err }
			if ref == nil { return starlark.None, nil }
			return WrapReference(ref), nil
		}), nil
	case "is_empty":
		return bind("result.is_empty", func(_ *starlark.Thread, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
			if err := starlark.UnpackArgs("result.is_empty", args, kwargs); err != nil { return nil, err }
			return starlark.Bool(v.result.IsEmpty()), nil
		}), nil
	case "each_ref":
		return bind("result.each_ref", func(thread *starlark.Thread, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
			var callable starlark.Callable
			if err := starlark.UnpackArgs("result.each_ref", args, kwargs, "callback", &callable); err != nil { return nil, err }
			err := v.result.EachRef(func(ref gwclient.Reference) error {
				_, callErr := starlark.Call(thread, callable, starlark.Tuple{WrapReference(ref)}, nil)
				return callErr
			})
			if err != nil { return nil, err }
			return starlark.None, nil
		}), nil
	case "add_attestation":
		return bind("result.add_attestation", func(_ *starlark.Thread, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
			var key string
			var value starlark.Value
			if err := starlark.UnpackArgs("result.add_attestation", args, kwargs, "key", &key, "attestation", &value); err != nil { return nil, err }
			attestation, err := attestationFromValue(value)
			if err != nil { return nil, err }
			v.result.AddAttestation(key, *attestation)
			return starlark.None, nil
		}), nil
	default:
		return nil, nil
	}
}

func (v *ResultValue) AttrNames() []string {
	return []string{"add_attestation", "add_meta", "add_ref", "attestations", "clone", "each_ref", "find_ref", "is_empty", "metadata", "ref", "refs", "set_ref", "single_ref"}
}

func newResultBuiltin(_ *starlark.Thread, _ *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	if err := starlark.UnpackArgs("new_result", args, kwargs); err != nil { return nil, err }
	return WrapResult(gwclient.NewResult()), nil
}

