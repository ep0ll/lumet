package client

import (
	"fmt"

	gwclient "github.com/moby/buildkit/frontend/gateway/client"
	gatewaypb "github.com/moby/buildkit/frontend/gateway/pb"
	"github.com/moby/buildkit/solver/result"
	digest "github.com/opencontainers/go-digest"
	"go.starlark.net/starlark"
	"google.golang.org/protobuf/proto"
)

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

type AttestationValue struct { attestation *gwclient.Attestation }

func WrapAttestation(value *gwclient.Attestation) *AttestationValue { return &AttestationValue{attestation: value} }
func (v *AttestationValue) String() string { return "<gateway.client.attestation>" }
func (v *AttestationValue) Type() string { return "gateway.client.attestation" }
func (v *AttestationValue) Freeze() {}
func (v *AttestationValue) Truth() starlark.Bool { return starlark.Bool(v != nil && v.attestation != nil) }
func (v *AttestationValue) Hash() (uint32, error) { return 0, fmt.Errorf("unhashable: gateway.client.attestation") }
func (v *AttestationValue) Attr(name string) (starlark.Value, error) {
	if v == nil || v.attestation == nil { return nil, fmt.Errorf("gateway.client.attestation: nil receiver") }
	switch name {
	case "kind": return starlark.MakeInt(int(v.attestation.Kind)), nil
	case "metadata": return stringBytesMapValue(v.attestation.Metadata), nil
	case "ref":
		if v.attestation.Ref == nil { return starlark.None, nil }
		return WrapReference(v.attestation.Ref), nil
	case "path": return starlark.String(v.attestation.Path), nil
	case "predicate_type": return starlark.String(v.attestation.InToto.PredicateType), nil
	case "subjects":
		values := make([]starlark.Value, len(v.attestation.InToto.Subjects))
		for i, subject := range v.attestation.InToto.Subjects {
			digests := make([]starlark.Value, len(subject.Digest))
			for j, dgst := range subject.Digest { digests[j] = starlark.String(dgst.String()) }
			values[i] = structValue("in_toto_subject", starlark.StringDict{
				"kind": starlark.MakeInt(int(subject.Kind)), "name": starlark.String(subject.Name), "digest": starlark.NewList(digests),
			})
		}
		return starlark.NewList(values), nil
	case "to_pb":
		return bind("attestation.to_pb", func(_ *starlark.Thread, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
			if err := starlark.UnpackArgs("attestation.to_pb", args, kwargs); err != nil { return nil, err }
			message, err := gwclient.AttestationToPB(v.attestation)
			if err != nil { return nil, err }
			data, err := proto.Marshal(message)
			if err != nil { return nil, err }
			return starlark.Bytes(data), nil
		}), nil
	default: return nil, nil
	}
}
func (v *AttestationValue) AttrNames() []string { return []string{"kind", "metadata", "path", "predicate_type", "ref", "subjects", "to_pb"} }

func attestationFromValue(value starlark.Value) (*gwclient.Attestation, error) {
	if v, ok := value.(*AttestationValue); ok && v != nil && v.attestation != nil { return v.attestation, nil }
	return nil, fmt.Errorf("got %s, want gateway.client.attestation", valueType(value))
}

func attestationBuiltin(_ *starlark.Thread, _ *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	var kind int
	var metadataValue, refValue, subjectsValue starlark.Value = starlark.None, starlark.None, starlark.None
	var path, predicateType string
	if err := starlark.UnpackArgs("attestation", args, kwargs,
		"kind?", &kind, "metadata?", &metadataValue, "ref?", &refValue, "path?", &path,
		"predicate_type?", &predicateType, "subjects?", &subjectsValue,
	); err != nil { return nil, err }
	value := &gwclient.Attestation{Kind: gatewaypb.AttestationKind(kind), Path: path}
	if metadataValue != starlark.None {
		dict, ok := metadataValue.(*starlark.Dict)
		if !ok { return nil, fmt.Errorf("attestation.metadata: want dict") }
		value.Metadata = make(map[string][]byte, dict.Len())
		for _, item := range dict.Items() {
			key, err := stringAt("attestation.metadata key", item[0]); if err != nil { return nil, err }
			data, err := bytesAt("attestation.metadata["+key+"]", item[1]); if err != nil { return nil, err }
			value.Metadata[key] = data
		}
	}
	if refValue != starlark.None {
		ref, ok := refValue.(*ReferenceValue)
		if !ok || ref == nil { return nil, fmt.Errorf("attestation.ref: got %s, want gateway.client.reference", valueType(refValue)) }
		value.Ref = ref.ref
	}
	value.InToto.PredicateType = predicateType
	if subjectsValue != starlark.None {
		iterable, ok := subjectsValue.(starlark.Iterable)
		if !ok { return nil, fmt.Errorf("attestation.subjects: want iterable") }
		iter := iterable.Iterate()
		for {
			var item starlark.Value
			if !iter.Next(&item) { break }
			dict, ok := item.(*starlark.Dict)
			if !ok { iter.Done(); return nil, fmt.Errorf("attestation.subjects: each item must be a dict") }
			var subject result.InTotoSubject
			for _, pair := range dict.Items() {
				key, err := stringAt("attestation.subjects key", pair[0]); if err != nil { iter.Done(); return nil, err }
				switch key {
				case "kind":
					n, err := intAt("attestation.subjects.kind", pair[1]); if err != nil { iter.Done(); return nil, err }
					subject.Kind = gatewaypb.InTotoSubjectKind(n)
				case "name":
					subject.Name, err = stringAt("attestation.subjects.name", pair[1]); if err != nil { iter.Done(); return nil, err }
				case "digest":
					values, err := stringListAt("attestation.subjects.digest", pair[1]); if err != nil { iter.Done(); return nil, err }
					for _, d := range values { subject.Digest = append(subject.Digest, digest.Digest(d)) }
				default:
					iter.Done(); return nil, fmt.Errorf("attestation.subjects: unknown field %q", key)
				}
			}
			value.InToto.Subjects = append(value.InToto.Subjects, subject)
		}
		iter.Done()
	}
	return WrapAttestation(value), nil
}

func attestationFromPBBuiltin(_ *starlark.Thread, _ *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	var dataValue starlark.Value
	if err := starlark.UnpackArgs("attestation_from_pb", args, kwargs, "data", &dataValue); err != nil { return nil, err }
	data, err := bytesAt("attestation_from_pb.data", dataValue)
	if err != nil { return nil, err }
	message := &gatewaypb.Attestation{}
	if err := proto.Unmarshal(data, message); err != nil { return nil, err }
	value, err := gwclient.AttestationFromPB[gwclient.Reference](message)
	if err != nil { return nil, err }
	return WrapAttestation(value), nil
}

