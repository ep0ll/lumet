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
	name = SnakeCaseIdentifier(name)
	if v == nil || v.definition == nil {
		return nil, fmt.Errorf("llb.definition: nil receiver")
	}
	switch name {
	case "Def", "def":
		values := make([]starlark.Value, len(v.definition.Def))
		for i, data := range v.definition.Def {
			values[i] = starlark.Bytes(append([]byte(nil), data...))
		}
		return starlark.NewList(values), nil
	case "Metadata", "metadata":
		dict := starlark.NewDict(len(v.definition.Metadata))
		for key, metadata := range v.definition.Metadata {
			if err := dict.SetKey(starlark.String(key.String()), WrapOpMetadata(metadata)); err != nil {
				return nil, err
			}
		}
		return dict, nil
	case "Source", "source":
		if v.definition.Source == nil {
			return starlark.None, nil
		}
		return WrapSource(v.definition.Source), nil
	case "Constraints", "constraints":
		if v.definition.Constraints == nil {
			return starlark.None, nil
		}
		return WrapConstraints(*v.definition.Constraints), nil
	case "Head", "head":
		return boundBuiltin("Definition.Head", func(_ *starlark.Thread, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
			if err := arity("Definition.Head", args, kwargs, 0, 0); err != nil {
				return nil, err
			}
			head, err := v.definition.Head()
			if err != nil {
				return nil, err
			}
			return starlark.String(head.String()), nil
		}), nil
	case "ToPB", "to_pb":
		return boundBuiltin("Definition.ToPB", func(_ *starlark.Thread, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
			if err := arity("Definition.ToPB", args, kwargs, 0, 0); err != nil {
				return nil, err
			}
			return WrapDefinitionProto(v.definition.ToPB()), nil
		}), nil
	case "Marshal", "marshal", "Bytes", "bytes":
		return boundBuiltin("Definition.Marshal", func(_ *starlark.Thread, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
			if err := arity("Definition.Marshal", args, kwargs, 0, 0); err != nil {
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
	return SnakeCaseNames([]string{"Def", "Metadata", "Source", "Constraints", "Head", "ToPB", "Marshal", "Bytes",
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
	name = SnakeCaseIdentifier(name)
	if v == nil || v.definition == nil {
		return nil, fmt.Errorf("llb.definition_pb: nil receiver")
	}
	switch name {
	case "Marshal", "marshal", "Bytes", "bytes":
		return boundBuiltin("DefinitionPB.Marshal", func(_ *starlark.Thread, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
			if err := arity("DefinitionPB.Marshal", args, kwargs, 0, 0); err != nil {
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
	return SnakeCaseNames([]string{"Marshal", "Bytes", "marshal", "bytes"})
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
	name = SnakeCaseIdentifier(name)
	if v==nil||v.op==nil{return nil,fmt.Errorf("llb.op_pb: nil receiver")}
	switch name {
	case "Marshal","marshal","Bytes","bytes":
		return boundBuiltin("OpPB.Marshal",func(_ *starlark.Thread,args starlark.Tuple,kwargs []starlark.Tuple)(starlark.Value,error){
			if err:=arity("OpPB.Marshal",args,kwargs,0,0);err!=nil{return nil,err}
			data,err:=proto.Marshal(v.op);if err!=nil{return nil,err}
			return starlark.Bytes(data),nil
		}),nil
	default:return nil,nil
	}
}
func (v *OpProtoValue) AttrNames()[]string{return SnakeCaseNames([]string{"Marshal","Bytes","marshal","bytes"})}

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
	name = SnakeCaseIdentifier(name)
	if v==nil||v.metadata==nil{return nil,fmt.Errorf("llb.op_metadata_pb: nil receiver")}
	switch name{
	case "Marshal","marshal","Bytes","bytes":
		return boundBuiltin("OpMetadataPB.Marshal",func(_ *starlark.Thread,args starlark.Tuple,kwargs []starlark.Tuple)(starlark.Value,error){
			if err:=arity("OpMetadataPB.Marshal",args,kwargs,0,0);err!=nil{return nil,err}
			data,err:=proto.Marshal(v.metadata);if err!=nil{return nil,err}
			return starlark.Bytes(data),nil
		}),nil
	default:return nil,nil
	}
}
func (v *OpMetadataProtoValue) AttrNames()[]string{return SnakeCaseNames([]string{"Marshal","Bytes","marshal","bytes"})}

// MarshalCacheValue exposes BuildKit's mutex-protected MarshalCache.
type MarshalCacheValue struct { cache *buildkitllb.MarshalCache }
func WrapMarshalCache(cache *buildkitllb.MarshalCache)*MarshalCacheValue{return &MarshalCacheValue{cache:cache}}
func (v *MarshalCacheValue) String()string{return "<llb.marshal_cache>"}
func (v *MarshalCacheValue) Type()string{return "llb.marshal_cache"}
func (v *MarshalCacheValue) Freeze(){}
func (v *MarshalCacheValue) Truth()starlark.Bool{return starlark.Bool(v!=nil&&v.cache!=nil)}
func (v *MarshalCacheValue) Hash()(uint32,error){return 0,fmt.Errorf("unhashable: llb.marshal_cache")}
func (v *MarshalCacheValue) Attr(name string)(starlark.Value,error){
	name = SnakeCaseIdentifier(name)
	if v==nil||v.cache==nil{return nil,fmt.Errorf("llb.marshal_cache: nil receiver")}
	switch name{
	case "Acquire","acquire":
		return boundBuiltin("MarshalCache.Acquire",func(_ *starlark.Thread,args starlark.Tuple,kwargs []starlark.Tuple)(starlark.Value,error){
			if err:=arity("MarshalCache.Acquire",args,kwargs,0,0);err!=nil{return nil,err}
			return WrapMarshalCacheInstance(v.cache.Acquire()),nil
		}),nil
	default:return nil,nil
	}
}
func (v *MarshalCacheValue) AttrNames()[]string{return SnakeCaseNames([]string{"Acquire","acquire"})}

// MarshalCacheInstanceValue is a lock-owning handle and must be released once.
type MarshalCacheInstanceValue struct { instance *buildkitllb.MarshalCacheInstance }
func WrapMarshalCacheInstance(instance *buildkitllb.MarshalCacheInstance)*MarshalCacheInstanceValue{return &MarshalCacheInstanceValue{instance:instance}}
func (v *MarshalCacheInstanceValue) String()string{return "<llb.marshal_cache_instance>"}
func (v *MarshalCacheInstanceValue) Type()string{return "llb.marshal_cache_instance"}
func (v *MarshalCacheInstanceValue) Freeze(){}
func (v *MarshalCacheInstanceValue) Truth()starlark.Bool{return starlark.Bool(v!=nil&&v.instance!=nil)}
func (v *MarshalCacheInstanceValue) Hash()(uint32,error){return 0,fmt.Errorf("unhashable: llb.marshal_cache_instance")}
func (v *MarshalCacheInstanceValue) Attr(name string)(starlark.Value,error){
	name = SnakeCaseIdentifier(name)
	if v==nil||v.instance==nil{return nil,fmt.Errorf("llb.marshal_cache_instance: nil receiver")}
	switch name{
	case "Load","load":
		return boundBuiltin("MarshalCacheInstance.Load",func(_ *starlark.Thread,args starlark.Tuple,kwargs []starlark.Tuple)(starlark.Value,error){
			if err:=arity("MarshalCacheInstance.Load",args,kwargs,1,1);err!=nil{return nil,err}
			constraints,ok:=args[0].(*ConstraintsValue);if !ok||constraints==nil{return nil,fmt.Errorf("Load: got %s, want llb.constraints",valueType(args[0]))}
			dgst,data,metadata,sources,err:=v.instance.Load(&constraints.constraints)
			if err!=nil{return nil,err}
			return marshalResult(dgst,data,metadata,sources),nil
		}),nil
	case "Store","store":
		return boundBuiltin("MarshalCacheInstance.Store",func(_ *starlark.Thread,args starlark.Tuple,kwargs []starlark.Tuple)(starlark.Value,error){
			if err:=arity("MarshalCacheInstance.Store",args,kwargs,4,4);err!=nil{return nil,err}
			data,err:=bytesAt("MarshalCacheInstance.Store",args,0);if err!=nil{return nil,err}
			var metadata *pb.OpMetadata
			if args[1]!=starlark.None{
				value,ok:=args[1].(*OpMetadataProtoValue);if !ok||value==nil{return nil,fmt.Errorf("Store: argument 2: got %s, want llb.op_metadata_pb or None",valueType(args[1]))}
				metadata=value.metadata
			}
			sources,err:=sourceLocationsFromValue(args[2]);if err!=nil{return nil,err}
			constraints,ok:=args[3].(*ConstraintsValue);if !ok||constraints==nil{return nil,fmt.Errorf("Store: argument 4: got %s, want llb.constraints",valueType(args[3]))}
			dgst,dt,md,locs,err:=v.instance.Store(data,metadata,sources,&constraints.constraints)
			if err!=nil{return nil,err}
			return marshalResult(dgst,dt,md,locs),nil
		}),nil
	case "Release","release":
		return boundBuiltin("MarshalCacheInstance.Release",func(_ *starlark.Thread,args starlark.Tuple,kwargs []starlark.Tuple)(starlark.Value,error){
			if err:=arity("MarshalCacheInstance.Release",args,kwargs,0,0);err!=nil{return nil,err}
			v.instance.Release()
			v.instance=nil
			return starlark.None,nil
		}),nil
	default:return nil,nil
	}
}
func (v *MarshalCacheInstanceValue) AttrNames()[]string{return SnakeCaseNames([]string{"Load","Store","Release","load","store","release"})}

func marshalResult(dgst digest.Digest,data []byte,metadata *pb.OpMetadata,sources []*buildkitllb.SourceLocation)starlark.Value{
	var md starlark.Value=starlark.None
	if metadata!=nil{md=WrapOpMetadataProto(metadata)}
	locs:=make([]starlark.Value,len(sources))
	for i,source:=range sources{locs[i]=WrapSourceLocation(source)}
	return starlark.Tuple{starlark.String(dgst.String()),starlark.Bytes(data),md,starlark.NewList(locs)}
}

func (v *StateValue) stateMarshalAttr(name string)(starlark.Value,bool,error){
	switch name{
	case "SetMarshalDefaults","set_marshal_defaults":
		return boundBuiltin("State.SetMarshalDefaults",func(_ *starlark.Thread,args starlark.Tuple,kwargs []starlark.Tuple)(starlark.Value,error){
			if err:=arity("State.SetMarshalDefaults",args,kwargs,0,-1);err!=nil{return nil,err}
			options,err:=optionsFromArgs[buildkitllb.ConstraintsOpt](args,0,"ConstraintsOpt");if err!=nil{return nil,err}
			return WrapState(v.state.SetMarshalDefaults(options...)),nil
		}),true,nil
	case "Marshal","marshal":
		return boundBuiltin("State.Marshal",func(thread *starlark.Thread,args starlark.Tuple,kwargs []starlark.Tuple)(starlark.Value,error){
			if err:=arity("State.Marshal",args,kwargs,0,-1);err!=nil{return nil,err}
			options,err:=optionsFromArgs[buildkitllb.ConstraintsOpt](args,0,"ConstraintsOpt");if err!=nil{return nil,err}
			definition,err:=v.state.Marshal(callContext(thread),options...)
			if err!=nil{return nil,err}
			return WrapDefinition(definition),nil
		}),true,nil
	case "WithImageConfig","with_image_config":
		return boundBuiltin("State.WithImageConfig",func(_ *starlark.Thread,args starlark.Tuple,kwargs []starlark.Tuple)(starlark.Value,error){
			if err:=arity("State.WithImageConfig",args,kwargs,1,1);err!=nil{return nil,err}
			data,err:=bytesAt("State.WithImageConfig",args,0);if err!=nil{return nil,err}
			state,err:=v.state.WithImageConfig(data);if err!=nil{return nil,err}
			return WrapState(state),nil
		}),true,nil
	default:return nil,false,nil
	}
}
var stateMarshalAttrNames=[]string{"Marshal","SetMarshalDefaults","WithImageConfig","marshal","set_marshal_defaults","with_image_config"}

func marshalBuiltins()map[string]starlark.Value{
	return map[string]starlark.Value{
		"NewMarshalCache":builtin("NewMarshalCache",newMarshalCacheBuiltin),
		"MarshalConstraints":builtin("MarshalConstraints",marshalConstraintsBuiltin),
		"WriteTo":builtin("WriteTo",writeToBuiltin),
		"ReadFrom":builtin("ReadFrom",readFromBuiltin),
	}
}
func newMarshalCacheBuiltin(_ *starlark.Thread,_ *starlark.Builtin,args starlark.Tuple,kwargs []starlark.Tuple)(starlark.Value,error){
	if err:=arity("NewMarshalCache",args,kwargs,0,0);err!=nil{return nil,err}
	return WrapMarshalCache(&buildkitllb.MarshalCache{}),nil
}
func marshalConstraintsBuiltin(_ *starlark.Thread,_ *starlark.Builtin,args starlark.Tuple,kwargs []starlark.Tuple)(starlark.Value,error){
	if err:=arity("MarshalConstraints",args,kwargs,2,2);err!=nil{return nil,err}
	base,ok:=args[0].(*ConstraintsValue);if !ok||base==nil{return nil,fmt.Errorf("MarshalConstraints: argument 1: want llb.constraints")}
	override,ok:=args[1].(*ConstraintsValue);if !ok||override==nil{return nil,fmt.Errorf("MarshalConstraints: argument 2: want llb.constraints")}
	op,metadata:=buildkitllb.MarshalConstraints(&base.constraints,&override.constraints)
	return starlark.Tuple{WrapOpProto(op),WrapOpMetadataProto(metadata)},nil
}
func writeToBuiltin(_ *starlark.Thread,_ *starlark.Builtin,args starlark.Tuple,kwargs []starlark.Tuple)(starlark.Value,error){
	if err:=arity("WriteTo",args,kwargs,1,1);err!=nil{return nil,err}
	definition,ok:=args[0].(*DefinitionValue);if !ok||definition==nil||definition.definition==nil{return nil,fmt.Errorf("WriteTo: expected llb.definition")}
	var buffer bytes.Buffer
	if err:=buildkitllb.WriteTo(definition.definition,&buffer);err!=nil{return nil,err}
	return starlark.Bytes(buffer.Bytes()),nil
}
func readFromBuiltin(_ *starlark.Thread,_ *starlark.Builtin,args starlark.Tuple,kwargs []starlark.Tuple)(starlark.Value,error){
	if err:=arity("ReadFrom",args,kwargs,1,1);err!=nil{return nil,err}
	data,err:=bytesAt("ReadFrom",args,0);if err!=nil{return nil,err}
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
