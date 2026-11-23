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

// AttestationToPB and AttestationFromPB mirror BuildKit's exported protobuf
// conversion helpers for host applications using this adapter package.
func AttestationToPB[T any](value *result.Attestation[T]) (*gatewaypb.Attestation, error) {
	return gwclient.AttestationToPB(value)
}

func AttestationFromPB[T any](value *gatewaypb.Attestation) (*result.Attestation[T], error) {
	return gwclient.AttestationFromPB[T](value)
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

func attestationToPBBuiltin(_ *starlark.Thread, _ *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	var value starlark.Value
	if err := starlark.UnpackArgs("attestation_to_pb", args, kwargs, "attestation", &value); err != nil { return nil, err }
	attestation, err := attestationFromValue(value)
	if err != nil { return nil, err }
	message, err := AttestationToPB(attestation)
	if err != nil { return nil, err }
	data, err := proto.Marshal(message)
	if err != nil { return nil, err }
	return starlark.Bytes(data), nil
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

