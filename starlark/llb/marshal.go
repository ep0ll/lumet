package llb

import (
	"bytes"
	"fmt"

	buildkitllb "github.com/moby/buildkit/client/llb"
	"github.com/moby/buildkit/solver/pb"
	digest "github.com/opencontainers/go-digest"
	"go.starlark.net/starlark"
	"google.golang.org/protobuf/proto"
)

// DefinitionValue is the Starlark representation of a marshalled BuildKit
// definition. The definition remains a native BuildKit value.
type DefinitionValue struct {
	definition *buildkitllb.Definition
}

// WrapDefinition adapts a native BuildKit Definition.
func WrapDefinition(definition *buildkitllb.Definition) *DefinitionValue {
	return &DefinitionValue{definition: definition}
}

// Unwrap returns the native BuildKit Definition.
func (v *DefinitionValue) Unwrap() *buildkitllb.Definition {
	if v == nil {
		return nil
	}
	return v.definition
}

func (v *DefinitionValue) String() string        { return "<llb.definition>" }
func (v *DefinitionValue) Type() string          { return "llb.definition" }
func (v *DefinitionValue) Freeze()               {}
func (v *DefinitionValue) Truth() starlark.Bool  { return starlark.Bool(v != nil && v.definition != nil) }
func (v *DefinitionValue) Hash() (uint32, error) { return 0, fmt.Errorf("unhashable: llb.definition") }

func (v *DefinitionValue) Attr(name string) (starlark.Value, error) {
		if v == nil || v.definition == nil {
		return nil, fmt.Errorf("llb.definition: nil receiver")
	}
	switch name {
	case "def":
		values := make([]starlark.Value, len(v.definition.Def))
		for i, data := range v.definition.Def {
			values[i] = starlark.Bytes(append([]byte(nil), data...))
		}
		return starlark.NewList(values), nil
	case "metadata":
		dict := starlark.NewDict(len(v.definition.Metadata))
		for key, metadata := range v.definition.Metadata {
			if err := dict.SetKey(starlark.String(key.String()), WrapOpMetadata(metadata)); err != nil {
				return nil, err
			}
		}
		return dict, nil
	case "source":
		if v.definition.Source == nil {
			return starlark.None, nil
		}
		return WrapSource(v.definition.Source), nil
	case "constraints":
		if v.definition.Constraints == nil {
			return starlark.None, nil
		}
		return WrapConstraints(*v.definition.Constraints), nil
	case "head":
		return boundBuiltin("definition.head", func(_ *starlark.Thread, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
			if err := arity("definition.head", args, kwargs, 0, 0); err != nil {
				return nil, err
			}
			head, err := v.definition.Head()
			if err != nil {
				return nil, err
			}
			return starlark.String(head.String()), nil
		}), nil
	case "to_pb":
		return boundBuiltin("definition.to_pb", func(_ *starlark.Thread, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
			if err := arity("definition.to_pb", args, kwargs, 0, 0); err != nil {
				return nil, err
			}
			return WrapDefinitionProto(v.definition.ToPB()), nil
		}), nil
	case "marshal", "bytes":
		return boundBuiltin("definition.marshal", func(_ *starlark.Thread, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
			if err := arity("definition.marshal", args, kwargs, 0, 0); err != nil {
				return nil, err
			}
			data, err := proto.Marshal(v.definition.ToPB())
			if err != nil {
				return nil, err
			}
			return starlark.Bytes(data), nil
		}), nil
	default:
		return nil, nil
	}
}

func (v *DefinitionValue) AttrNames() []string {
	return SnakeCaseNames([]string{"def", "metadata", "source", "constraints", "head", "to_pb", "marshal", "bytes",
		"def", "metadata", "source", "constraints", "head", "to_pb", "marshal", "bytes"})
}

// DefinitionProtoValue wraps solver/pb.Definition for APIs that consume the
// protobuf form of an LLB definition.
type DefinitionProtoValue struct {
	definition *pb.Definition
}

// WrapDefinitionProto adapts a native protobuf Definition.
func WrapDefinitionProto(definition *pb.Definition) *DefinitionProtoValue {
	return &DefinitionProtoValue{definition: definition}
}

// Unwrap returns the native protobuf Definition.
func (v *DefinitionProtoValue) Unwrap() *pb.Definition {
	if v == nil {
		return nil
	}
	return v.definition
}

func (v *DefinitionProtoValue) String() string        { return "<llb.definition_pb>" }
func (v *DefinitionProtoValue) Type() string          { return "llb.definition_pb" }
func (v *DefinitionProtoValue) Freeze()               {}
func (v *DefinitionProtoValue) Truth() starlark.Bool  { return starlark.Bool(v != nil && v.definition != nil) }
func (v *DefinitionProtoValue) Hash() (uint32, error) { return 0, fmt.Errorf("unhashable: llb.definition_pb") }

func (v *DefinitionProtoValue) Attr(name string) (starlark.Value, error) {
		if v == nil || v.definition == nil {
		return nil, fmt.Errorf("llb.definition_pb: nil receiver")
	}
	switch name {
	case "marshal", "bytes":
		return boundBuiltin("definition_pb.marshal", func(_ *starlark.Thread, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
			if err := arity("definition_pb.marshal", args, kwargs, 0, 0); err != nil {
				return nil, err
			}
			data, err := proto.Marshal(v.definition)
			if err != nil {
				return nil, err
			}
			return starlark.Bytes(data), nil
		}), nil
	default:
		return nil, nil
	}
}

func (v *DefinitionProtoValue) AttrNames() []string {
	return SnakeCaseNames([]string{"marshal", "bytes", "marshal", "bytes"})
}

// OpProtoValue wraps a BuildKit solver/pb.Op generated from MarshalConstraints.
type OpProtoValue struct {
	op *pb.Op
}

// WrapOpProto adapts a protobuf operation.
func WrapOpProto(op *pb.Op) *OpProtoValue { return &OpProtoValue{op: op} }

func (v *OpProtoValue) Unwrap() *pb.Op {
	if v == nil { return nil }
	return v.op
}
func (v *OpProtoValue) String() string        { return "<llb.op_pb>" }
func (v *OpProtoValue) Type() string          { return "llb.op_pb" }
func (v *OpProtoValue) Freeze()               {}
func (v *OpProtoValue) Truth() starlark.Bool  { return starlark.Bool(v != nil && v.op != nil) }
func (v *OpProtoValue) Hash() (uint32, error) { return 0, fmt.Errorf("unhashable: llb.op_pb") }
func (v *OpProtoValue) Attr(name string) (starlark.Value,error) {
		if v==nil||v.op==nil{return nil,fmt.Errorf("llb.op_pb: nil receiver")}
	switch name {
	case "marshal","bytes":
		return boundBuiltin("op_pb.marshal",func(_ *starlark.Thread,args starlark.Tuple,kwargs []starlark.Tuple)(starlark.Value,error){
			if err:=arity("op_pb.marshal",args,kwargs,0,0);err!=nil{return nil,err}
			data,err:=proto.Marshal(v.op);if err!=nil{return nil,err}
			return starlark.Bytes(data),nil
		}),nil
	default:return nil,nil
	}
}
func (v *OpProtoValue) AttrNames()[]string{return SnakeCaseNames([]string{"marshal","bytes","marshal","bytes"})}

// OpMetadataProtoValue wraps solver/pb.OpMetadata.
type OpMetadataProtoValue struct { metadata *pb.OpMetadata }
func WrapOpMetadataProto(metadata *pb.OpMetadata)*OpMetadataProtoValue{return &OpMetadataProtoValue{metadata:metadata}}
func (v *OpMetadataProtoValue) Unwrap()*pb.OpMetadata{if v==nil{return nil};return v.metadata}
func (v *OpMetadataProtoValue) String()string{return "<llb.op_metadata_pb>"}
func (v *OpMetadataProtoValue) Type()string{return "llb.op_metadata_pb"}
func (v *OpMetadataProtoValue) Freeze(){}
func (v *OpMetadataProtoValue) Truth()starlark.Bool{return starlark.Bool(v!=nil&&v.metadata!=nil)}
func (v *OpMetadataProtoValue) Hash()(uint32,error){return 0,fmt.Errorf("unhashable: llb.op_metadata_pb")}
func (v *OpMetadataProtoValue) Attr(name string)(starlark.Value,error){
		if v==nil||v.metadata==nil{return nil,fmt.Errorf("llb.op_metadata_pb: nil receiver")}
	switch name{
	case "marshal","bytes":
		return boundBuiltin("op_metadata_pb.marshal",func(_ *starlark.Thread,args starlark.Tuple,kwargs []starlark.Tuple)(starlark.Value,error){
			if err:=arity("op_metadata_pb.marshal",args,kwargs,0,0);err!=nil{return nil,err}
			data,err:=proto.Marshal(v.metadata);if err!=nil{return nil,err}
			return starlark.Bytes(data),nil
		}),nil
	default:return nil,nil
	}
}
func (v *OpMetadataProtoValue) AttrNames()[]string{return SnakeCaseNames([]string{"marshal","bytes","marshal","bytes"})}

// MarshalCacheValue exposes BuildKit's mutex-protected MarshalCache.
type MarshalCacheValue struct { cache *buildkitllb.MarshalCache }
func WrapMarshalCache(cache *buildkitllb.MarshalCache)*MarshalCacheValue{return &MarshalCacheValue{cache:cache}}
func (v *MarshalCacheValue) String()string{return "<llb.marshal_cache>"}
func (v *MarshalCacheValue) Type()string{return "llb.marshal_cache"}
func (v *MarshalCacheValue) Freeze(){}
func (v *MarshalCacheValue) Truth()starlark.Bool{return starlark.Bool(v!=nil&&v.cache!=nil)}
func (v *MarshalCacheValue) Hash()(uint32,error){return 0,fmt.Errorf("unhashable: llb.marshal_cache")}
func (v *MarshalCacheValue) Attr(name string)(starlark.Value,error){
		if v==nil||v.cache==nil{return nil,fmt.Errorf("llb.marshal_cache: nil receiver")}
	switch name{
	case "acquire":
		return boundBuiltin("marshal_cache.acquire",func(_ *starlark.Thread,args starlark.Tuple,kwargs []starlark.Tuple)(starlark.Value,error){
			if err:=arity("marshal_cache.acquire",args,kwargs,0,0);err!=nil{return nil,err}
			return WrapMarshalCacheInstance(v.cache.Acquire()),nil
		}),nil
	default:return nil,nil
	}
}
func (v *MarshalCacheValue) AttrNames()[]string{return SnakeCaseNames([]string{"acquire","acquire"})}

// MarshalCacheInstanceValue is a lock-owning handle and must be released once.
type MarshalCacheInstanceValue struct { instance *buildkitllb.MarshalCacheInstance }
func WrapMarshalCacheInstance(instance *buildkitllb.MarshalCacheInstance)*MarshalCacheInstanceValue{return &MarshalCacheInstanceValue{instance:instance}}
func (v *MarshalCacheInstanceValue) String()string{return "<llb.marshal_cache_instance>"}
func (v *MarshalCacheInstanceValue) Type()string{return "llb.marshal_cache_instance"}
func (v *MarshalCacheInstanceValue) Freeze(){}
func (v *MarshalCacheInstanceValue) Truth()starlark.Bool{return starlark.Bool(v!=nil&&v.instance!=nil)}
func (v *MarshalCacheInstanceValue) Hash()(uint32,error){return 0,fmt.Errorf("unhashable: llb.marshal_cache_instance")}
func (v *MarshalCacheInstanceValue) Attr(name string)(starlark.Value,error){
		if v==nil||v.instance==nil{return nil,fmt.Errorf("llb.marshal_cache_instance: nil receiver")}
	switch name{
	case "load":
		return boundBuiltin("marshal_cache_instance.load",func(_ *starlark.Thread,args starlark.Tuple,kwargs []starlark.Tuple)(starlark.Value,error){
			if err:=arity("marshal_cache_instance.load",args,kwargs,1,1);err!=nil{return nil,err}
			constraints,ok:=args[0].(*ConstraintsValue);if !ok||constraints==nil{return nil,fmt.Errorf("load: got %s, want llb.constraints",valueType(args[0]))}
			dgst,data,metadata,sources,err:=v.instance.Load(&constraints.constraints)
			if err!=nil{return nil,err}
			return marshalResult(dgst,data,metadata,sources),nil
		}),nil
	case "store":
		return boundBuiltin("marshal_cache_instance.store",func(_ *starlark.Thread,args starlark.Tuple,kwargs []starlark.Tuple)(starlark.Value,error){
			if err:=arity("marshal_cache_instance.store",args,kwargs,4,4);err!=nil{return nil,err}
			data,err:=bytesAt("marshal_cache_instance.store",args,0);if err!=nil{return nil,err}
			var metadata *pb.OpMetadata
			if args[1]!=starlark.None{
				value,ok:=args[1].(*OpMetadataProtoValue);if !ok||value==nil{return nil,fmt.Errorf("store: argument 2: got %s, want llb.op_metadata_pb or none",valueType(args[1]))}
				metadata=value.metadata
			}
			sources,err:=sourceLocationsFromValue(args[2]);if err!=nil{return nil,err}
			constraints,ok:=args[3].(*ConstraintsValue);if !ok||constraints==nil{return nil,fmt.Errorf("store: argument 4: got %s, want llb.constraints",valueType(args[3]))}
			dgst,dt,md,locs,err:=v.instance.Store(data,metadata,sources,&constraints.constraints)
			if err!=nil{return nil,err}
			return marshalResult(dgst,dt,md,locs),nil
		}),nil
	case "release":
		return boundBuiltin("marshal_cache_instance.release",func(_ *starlark.Thread,args starlark.Tuple,kwargs []starlark.Tuple)(starlark.Value,error){
			if err:=arity("marshal_cache_instance.release",args,kwargs,0,0);err!=nil{return nil,err}
			v.instance.Release()
			v.instance=nil
			return starlark.None,nil
		}),nil
	default:return nil,nil
	}
}
func (v *MarshalCacheInstanceValue) AttrNames()[]string{return SnakeCaseNames([]string{"load","store","release","load","store","release"})}

func marshalResult(dgst digest.Digest,data []byte,metadata *pb.OpMetadata,sources []*buildkitllb.SourceLocation)starlark.Value{
	var md starlark.Value=starlark.None
	if metadata!=nil{md=WrapOpMetadataProto(metadata)}
	locs:=make([]starlark.Value,len(sources))
	for i,source:=range sources{locs[i]=WrapSourceLocation(source)}
	return starlark.Tuple{starlark.String(dgst.String()),starlark.Bytes(data),md,starlark.NewList(locs)}
}

func (v *StateValue) stateMarshalAttr(name string)(starlark.Value,bool,error){
	switch name{
	case "set_marshal_defaults":
		return boundBuiltin("state.set_marshal_defaults",func(_ *starlark.Thread,args starlark.Tuple,kwargs []starlark.Tuple)(starlark.Value,error){
			if err:=arity("state.set_marshal_defaults",args,kwargs,0,-1);err!=nil{return nil,err}
			options,err:=optionsFromArgs[buildkitllb.ConstraintsOpt](args,0,"constraints_opt");if err!=nil{return nil,err}
			return WrapState(v.state.SetMarshalDefaults(options...)),nil
		}),true,nil
	case "marshal":
		return boundBuiltin("state.marshal",func(thread *starlark.Thread,args starlark.Tuple,kwargs []starlark.Tuple)(starlark.Value,error){
			if err:=arity("state.marshal",args,kwargs,0,-1);err!=nil{return nil,err}
			options,err:=optionsFromArgs[buildkitllb.ConstraintsOpt](args,0,"constraints_opt");if err!=nil{return nil,err}
			definition,err:=v.state.Marshal(callContext(thread),options...)
			if err!=nil{return nil,err}
			return WrapDefinition(definition),nil
		}),true,nil
	case "with_image_config":
		return boundBuiltin("state.with_image_config",func(_ *starlark.Thread,args starlark.Tuple,kwargs []starlark.Tuple)(starlark.Value,error){
			if err:=arity("state.with_image_config",args,kwargs,1,1);err!=nil{return nil,err}
			data,err:=bytesAt("state.with_image_config",args,0);if err!=nil{return nil,err}
			state,err:=v.state.WithImageConfig(data);if err!=nil{return nil,err}
			return WrapState(state),nil
		}),true,nil
	default:return nil,false,nil
	}
}
var stateMarshalAttrNames=[]string{"marshal","set_marshal_defaults","with_image_config","marshal","set_marshal_defaults","with_image_config"}

func marshalBuiltins()map[string]starlark.Value{
	return map[string]starlark.Value{
		"new_marshal_cache":builtin("new_marshal_cache",newMarshalCacheBuiltin),
		"marshal_constraints":builtin("marshal_constraints",marshalConstraintsBuiltin),
		"write_to":builtin("write_to",writeToBuiltin),
		"read_from":builtin("read_from",readFromBuiltin),
	}
}
func newMarshalCacheBuiltin(_ *starlark.Thread,_ *starlark.Builtin,args starlark.Tuple,kwargs []starlark.Tuple)(starlark.Value,error){
	if err:=arity("new_marshal_cache",args,kwargs,0,0);err!=nil{return nil,err}
	return WrapMarshalCache(&buildkitllb.MarshalCache{}),nil
}
func marshalConstraintsBuiltin(_ *starlark.Thread,_ *starlark.Builtin,args starlark.Tuple,kwargs []starlark.Tuple)(starlark.Value,error){
	if err:=arity("marshal_constraints",args,kwargs,2,2);err!=nil{return nil,err}
	base,ok:=args[0].(*ConstraintsValue);if !ok||base==nil{return nil,fmt.Errorf("marshal_constraints: argument 1: want llb.constraints")}
	override,ok:=args[1].(*ConstraintsValue);if !ok||override==nil{return nil,fmt.Errorf("marshal_constraints: argument 2: want llb.constraints")}
	op,metadata:=buildkitllb.MarshalConstraints(&base.constraints,&override.constraints)
	return starlark.Tuple{WrapOpProto(op),WrapOpMetadataProto(metadata)},nil
}
func writeToBuiltin(_ *starlark.Thread,_ *starlark.Builtin,args starlark.Tuple,kwargs []starlark.Tuple)(starlark.Value,error){
	if err:=arity("write_to",args,kwargs,1,1);err!=nil{return nil,err}
	definition,ok:=args[0].(*DefinitionValue);if !ok||definition==nil||definition.definition==nil{return nil,fmt.Errorf("write_to: expected llb.definition")}
	var buffer bytes.Buffer
	if err:=buildkitllb.WriteTo(definition.definition,&buffer);err!=nil{return nil,err}
	return starlark.Bytes(buffer.Bytes()),nil
}
func readFromBuiltin(_ *starlark.Thread,_ *starlark.Builtin,args starlark.Tuple,kwargs []starlark.Tuple)(starlark.Value,error){
	if err:=arity("read_from",args,kwargs,1,1);err!=nil{return nil,err}
	data,err:=bytesAt("read_from",args,0);if err!=nil{return nil,err}
	definition,err:=buildkitllb.ReadFrom(bytes.NewReader(data));if err!=nil{return nil,err}
	return WrapDefinition(definition),nil
}

var (
	_ starlark.Value=(*DefinitionValue)(nil)
	_ starlark.HasAttrs=(*DefinitionValue)(nil)
	_ starlark.Value=(*DefinitionProtoValue)(nil)
	_ starlark.HasAttrs=(*DefinitionProtoValue)(nil)
	_ starlark.Value=(*OpProtoValue)(nil)
	_ starlark.HasAttrs=(*OpProtoValue)(nil)
	_ starlark.Value=(*OpMetadataProtoValue)(nil)
	_ starlark.HasAttrs=(*OpMetadataProtoValue)(nil)
	_ starlark.Value=(*MarshalCacheValue)(nil)
	_ starlark.HasAttrs=(*MarshalCacheValue)(nil)
	_ starlark.Value=(*MarshalCacheInstanceValue)(nil)
	_ starlark.HasAttrs=(*MarshalCacheInstanceValue)(nil)
)
