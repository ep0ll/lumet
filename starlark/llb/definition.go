package llb

import (
	"fmt"

	buildkitllb "github.com/moby/buildkit/client/llb"
	"github.com/moby/buildkit/solver/pb"
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

func (v *DefinitionOpValue) String() string        { return "<llb.definition_op>" }
func (v *DefinitionOpValue) Type() string          { return "llb.definition_op" }
func (v *DefinitionOpValue) Freeze()               {}
func (v *DefinitionOpValue) Truth() starlark.Bool  { return starlark.Bool(v != nil && v.op != nil) }
func (v *DefinitionOpValue) Hash() (uint32, error) { return 0, fmt.Errorf("unhashable: llb.definition_op") }

func (v *DefinitionOpValue) Attr(name string) (starlark.Value, error) {
		if v == nil || v.op == nil { return nil, fmt.Errorf("llb.definition_op: nil receiver") }
	switch name {
	case "state":
		return boundBuiltin("definition_op.state",func(_ *starlark.Thread,args starlark.Tuple,kwargs []starlark.Tuple)(starlark.Value,error){
			if err:=arity("definition_op.state",args,kwargs,0,0);err!=nil{return nil,err}
			return WrapState(buildkitllb.NewState(v.op)),nil
		}),nil
	case "output":
		return boundBuiltin("definition_op.output",func(_ *starlark.Thread,args starlark.Tuple,kwargs []starlark.Tuple)(starlark.Value,error){
			if err:=arity("definition_op.output",args,kwargs,0,0);err!=nil{return nil,err}
			output:=v.op.Output()
			if output==nil{return starlark.None,nil}
			return WrapOutput(output),nil
		}),nil
	case "inputs":
		return boundBuiltin("definition_op.inputs",func(_ *starlark.Thread,args starlark.Tuple,kwargs []starlark.Tuple)(starlark.Value,error){
			if err:=arity("definition_op.inputs",args,kwargs,0,0);err!=nil{return nil,err}
			return outputListValue(v.op.Inputs()),nil
		}),nil
	case "validate":
		return boundBuiltin("definition_op.validate",func(thread *starlark.Thread,args starlark.Tuple,kwargs []starlark.Tuple)(starlark.Value,error){
			if err:=arity("definition_op.validate",args,kwargs,0,-1);err!=nil{return nil,err}
			constraints,err:=constraintsFromArgs(args,0);if err!=nil{return nil,err}
			if err:=v.op.Validate(callContext(thread),constraints);err!=nil{return nil,err}
			return starlark.None,nil
		}),nil
	case "marshal":
		return boundBuiltin("definition_op.marshal",func(thread *starlark.Thread,args starlark.Tuple,kwargs []starlark.Tuple)(starlark.Value,error){
			if err:=arity("definition_op.marshal",args,kwargs,0,-1);err!=nil{return nil,err}
			constraints,err:=constraintsFromArgs(args,0);if err!=nil{return nil,err}
			dgst,data,metadata,sources,err:=v.op.Marshal(callContext(thread),constraints)
			if err!=nil{return nil,err}
			return marshalResult(dgst,data,metadata,sources),nil
		}),nil
	case "to_input":
		return boundBuiltin("definition_op.to_input",func(thread *starlark.Thread,args starlark.Tuple,kwargs []starlark.Tuple)(starlark.Value,error){
			if err:=arity("definition_op.to_input",args,kwargs,0,-1);err!=nil{return nil,err}
			constraints,err:=constraintsFromArgs(args,0);if err!=nil{return nil,err}
			input,err:=v.op.ToInput(callContext(thread),constraints);if err!=nil{return nil,err}
			if input==nil{return starlark.None,nil}
			return starlarkstructValue("llb.input",starlark.StringDict{
				"digest":starlark.String(input.Digest),
				"index":starlark.MakeInt64(input.Index),
			}),nil
		}),nil
	default:return nil,nil
	}
}

func (v *DefinitionOpValue) AttrNames()[]string{
	return SnakeCaseNames([]string{"state","output","inputs","validate","marshal","to_input","state","output","inputs","validate","marshal","to_input"})
}

func outputListValue(outputs []buildkitllb.Output) starlark.Value {
	values:=make([]starlark.Value,len(outputs))
	for i,output:=range outputs{values[i]=WrapOutput(output)}
	return starlark.NewList(values)
}

// definitionBuiltins exposes BuildKit's definition-backed Vertex constructor.
func definitionBuiltins() map[string]starlark.Value {
	return map[string]starlark.Value{
		"new_definition_op":builtin("new_definition_op",newDefinitionOpBuiltin),
	}
}

func newDefinitionOpBuiltin(_ *starlark.Thread,_ *starlark.Builtin,args starlark.Tuple,kwargs []starlark.Tuple)(starlark.Value,error){
	if err:=arity("new_definition_op",args,kwargs,1,1);err!=nil{return nil,err}
	var definition *pb.Definition
	switch value:=args[0].(type){
	case *DefinitionProtoValue:
		if value!=nil{definition=value.definition}
	case *DefinitionValue:
		if value!=nil&&value.definition!=nil{definition=value.definition.ToPB()}
	default:
		return nil,fmt.Errorf("new_definition_op: got %s, want llb.definition or llb.definition_pb",valueType(args[0]))
	}
	if definition==nil{return nil,fmt.Errorf("new_definition_op: definition must not be nil")}
	op,err:=buildkitllb.NewDefinitionOp(definition)
	if err!=nil{return nil,err}
	return WrapDefinitionOp(op),nil
}


var (
	_ starlark.Value=(*DefinitionOpValue)(nil)
	_ starlark.HasAttrs=(*DefinitionOpValue)(nil)
)
