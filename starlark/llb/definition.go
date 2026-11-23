package llb

import (
	"fmt"

	buildkitllb "github.com/moby/buildkit/client/llb"
	"go.starlark.net/starlark"
)

// DefinitionOpValue wraps BuildKit's Vertex implementation reconstructed from
// a protobuf definition.
type DefinitionOpValue struct {
	op *buildkitllb.DefinitionOp
}

// WrapDefinitionOp adapts a native DefinitionOp.
func WrapDefinitionOp(op *buildkitllb.DefinitionOp) *DefinitionOpValue {
	return &DefinitionOpValue{op: op}
}

// Unwrap returns the native DefinitionOp.
func (v *DefinitionOpValue) Unwrap() *buildkitllb.DefinitionOp {
	if v == nil { return nil }
	return v.op
}

func (v *DefinitionOpValue) String() string        { return "<llb.DefinitionOp>" }
func (v *DefinitionOpValue) Type() string          { return "llb.DefinitionOp" }
func (v *DefinitionOpValue) Freeze()               {}
func (v *DefinitionOpValue) Truth() starlark.Bool  { return starlark.Bool(v != nil && v.op != nil) }
func (v *DefinitionOpValue) Hash() (uint32, error) { return 0, fmt.Errorf("unhashable: llb.DefinitionOp") }

func (v *DefinitionOpValue) Attr(name string) (starlark.Value, error) {
	if v == nil || v.op == nil { return nil, fmt.Errorf("llb.DefinitionOp: nil receiver") }
	switch name {
	case "State", "state":
		return boundBuiltin("DefinitionOp.State",func(_ *starlark.Thread,args starlark.Tuple,kwargs []starlark.Tuple)(starlark.Value,error){
			if err:=arity("DefinitionOp.State",args,kwargs,0,0);err!=nil{return nil,err}
			return WrapState(buildkitllb.NewState(v.op)),nil
		}),nil
	case "Output","output":
		return boundBuiltin("DefinitionOp.Output",func(_ *starlark.Thread,args starlark.Tuple,kwargs []starlark.Tuple)(starlark.Value,error){
			if err:=arity("DefinitionOp.Output",args,kwargs,0,0);err!=nil{return nil,err}
			output:=v.op.Output()
			if output==nil{return starlark.None,nil}
			return WrapOutput(output),nil
		}),nil
	case "Inputs","inputs":
		return boundBuiltin("DefinitionOp.Inputs",func(_ *starlark.Thread,args starlark.Tuple,kwargs []starlark.Tuple)(starlark.Value,error){
			if err:=arity("DefinitionOp.Inputs",args,kwargs,0,0);err!=nil{return nil,err}
			return outputListValue(v.op.Inputs()),nil
		}),nil
	case "Validate","validate":
		return boundBuiltin("DefinitionOp.Validate",func(thread *starlark.Thread,args starlark.Tuple,kwargs []starlark.Tuple)(starlark.Value,error){
			if err:=arity("DefinitionOp.Validate",args,kwargs,0,-1);err!=nil{return nil,err}
			constraints,err:=constraintsFromArgs(args,0);if err!=nil{return nil,err}
			if err:=v.op.Validate(callContext(thread),constraints);err!=nil{return nil,err}
			return starlark.None,nil
		}),nil
	case "Marshal","marshal":
		return boundBuiltin("DefinitionOp.Marshal",func(thread *starlark.Thread,args starlark.Tuple,kwargs []starlark.Tuple)(starlark.Value,error){
			if err:=arity("DefinitionOp.Marshal",args,kwargs,0,-1);err!=nil{return nil,err}
			constraints,err:=constraintsFromArgs(args,0);if err!=nil{return nil,err}
			dgst,data,metadata,sources,err:=v.op.Marshal(callContext(thread),constraints)
			if err!=nil{return nil,err}
			return marshalResult(dgst,data,metadata,sources),nil
		}),nil
	case "ToInput","to_input":
		return boundBuiltin("DefinitionOp.ToInput",func(thread *starlark.Thread,args starlark.Tuple,kwargs []starlark.Tuple)(starlark.Value,error){
			if err:=arity("DefinitionOp.ToInput",args,kwargs,0,-1);err!=nil{return nil,err}
			constraints,err:=constraintsFromArgs(args,0);if err!=nil{return nil,err}
			input,err:=v.op.ToInput(callContext(thread),constraints);if err!=nil{return nil,err}
			if input==nil{return starlark.None,nil}
			return starlarkstructValue("llb.Input",starlark.StringDict{
				"digest":starlark.String(input.Digest),
				"index":starlark.MakeInt64(input.Index),
			}),nil
		}),nil
	default:return nil,nil
	}
}

func (v *DefinitionOpValue) AttrNames()[]string{
	return []string{"State","Output","Inputs","Validate","Marshal","ToInput","state","output","inputs","validate","marshal","to_input"}
}

func outputListValue(outputs []buildkitllb.Output) starlark.Value {
	values:=make([]starlark.Value,len(outputs))
	for i,output:=range outputs{values[i]=WrapOutput(output)}
	return starlark.NewList(values)
}

// definitionBuiltins exposes BuildKit's definition-backed Vertex constructor.
func definitionBuiltins() map[string]starlark.Value {
	return map[string]starlark.Value{
		"NewDefinitionOp":builtin("NewDefinitionOp",newDefinitionOpBuiltin),
	}
}

func newDefinitionOpBuiltin(_ *starlark.Thread,_ *starlark.Builtin,args starlark.Tuple,kwargs []starlark.Tuple)(starlark.Value,error){
	if err:=arity("NewDefinitionOp",args,kwargs,1,1);err!=nil{return nil,err}
	var definition *pb.Definition
	switch value:=args[0].(type){
	case *DefinitionProtoValue:
		if value!=nil{definition=value.definition}
	case *DefinitionValue:
		if value!=nil&&value.definition!=nil{definition=value.definition.ToPB()}
	default:
		return nil,fmt.Errorf("NewDefinitionOp: got %s, want llb.Definition or llb.DefinitionPB",valueType(args[0]))
	}
	if definition==nil{return nil,fmt.Errorf("NewDefinitionOp: definition must not be nil")}
	op,err:=buildkitllb.NewDefinitionOp(definition)
	if err!=nil{return nil,err}
	return WrapDefinitionOp(op),nil
}

func (v *VertexValue) Attr(name string)(starlark.Value,error){
	if v==nil||v.vertex==nil{return nil,fmt.Errorf("llb.Vertex: nil receiver")}
	switch name{
	case "Output","output":
		return boundBuiltin("Vertex.Output",func(_ *starlark.Thread,args starlark.Tuple,kwargs []starlark.Tuple)(starlark.Value,error){
			if err:=arity("Vertex.Output",args,kwargs,0,0);err!=nil{return nil,err}
			output:=v.vertex.Output()
			if output==nil{return starlark.None,nil}
			return WrapOutput(output),nil
		}),nil
	case "Inputs","inputs":
		return boundBuiltin("Vertex.Inputs",func(_ *starlark.Thread,args starlark.Tuple,kwargs []starlark.Tuple)(starlark.Value,error){
			if err:=arity("Vertex.Inputs",args,kwargs,0,0);err!=nil{return nil,err}
			return outputListValue(v.vertex.Inputs()),nil
		}),nil
	case "Validate","validate":
		return boundBuiltin("Vertex.Validate",func(thread *starlark.Thread,args starlark.Tuple,kwargs []starlark.Tuple)(starlark.Value,error){
			if err:=arity("Vertex.Validate",args,kwargs,0,-1);err!=nil{return nil,err}
			constraints,err:=constraintsFromArgs(args,0);if err!=nil{return nil,err}
			if err:=v.vertex.Validate(callContext(thread),constraints);err!=nil{return nil,err}
			return starlark.None,nil
		}),nil
	case "Marshal","marshal":
		return boundBuiltin("Vertex.Marshal",func(thread *starlark.Thread,args starlark.Tuple,kwargs []starlark.Tuple)(starlark.Value,error){
			if err:=arity("Vertex.Marshal",args,kwargs,0,-1);err!=nil{return nil,err}
			constraints,err:=constraintsFromArgs(args,0);if err!=nil{return nil,err}
			dgst,data,metadata,sources,err:=v.vertex.Marshal(callContext(thread),constraints)
			if err!=nil{return nil,err}
			return marshalResult(dgst,data,metadata,sources),nil
		}),nil
	default:return nil,nil
	}
}
func (v *VertexValue) AttrNames()[]string{return []string{"Output","Inputs","Validate","Marshal","output","inputs","validate","marshal"}}

func (v *OutputValue) Attr(name string)(starlark.Value,error){
	if v==nil||v.output==nil{return nil,fmt.Errorf("llb.Output: nil receiver")}
	switch name{
	case "Vertex","vertex":
		return boundBuiltin("Output.Vertex",func(thread *starlark.Thread,args starlark.Tuple,kwargs []starlark.Tuple)(starlark.Value,error){
			if err:=arity("Output.Vertex",args,kwargs,0,-1);err!=nil{return nil,err}
			constraints,err:=constraintsFromArgs(args,0);if err!=nil{return nil,err}
			return WrapVertex(v.output.Vertex(callContext(thread),constraints)),nil
		}),nil
	case "ToInput","to_input":
		return boundBuiltin("Output.ToInput",func(thread *starlark.Thread,args starlark.Tuple,kwargs []starlark.Tuple)(starlark.Value,error){
			if err:=arity("Output.ToInput",args,kwargs,0,-1);err!=nil{return nil,err}
			constraints,err:=constraintsFromArgs(args,0);if err!=nil{return nil,err}
			input,err:=v.output.ToInput(callContext(thread),constraints);if err!=nil{return nil,err}
			if input==nil{return starlark.None,nil}
			return starlarkstructValue("llb.Input",starlark.StringDict{"digest":starlark.String(input.Digest),"index":starlark.MakeInt64(input.Index)}),nil
		}),nil
	default:return nil,nil
	}
}
func (v *OutputValue) AttrNames()[]string{return []string{"Vertex","ToInput","vertex","to_input"}}

var (
	_ starlark.Value=(*DefinitionOpValue)(nil)
	_ starlark.HasAttrs=(*DefinitionOpValue)(nil)
)
