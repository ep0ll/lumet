package llb

import (
	"fmt"

	buildkitllb "github.com/moby/buildkit/client/llb"
	"go.starlark.net/starlark"
)

// PassthroughInputValue describes an input to a PassthroughOp.
type PassthroughInputValue struct {
	input buildkitllb.PassthroughInput
}

// WrapPassthroughInput adapts a native passthrough input.
func WrapPassthroughInput(input buildkitllb.PassthroughInput)*PassthroughInputValue{
	return &PassthroughInputValue{input:input}
}
func (v *PassthroughInputValue) String()string{return "<llb.passthrough_input>"}
func (v *PassthroughInputValue) Type()string{return "llb.passthrough_input"}
func (v *PassthroughInputValue) Freeze(){}
func (v *PassthroughInputValue) Truth()starlark.Bool{return starlark.Bool(v!=nil)}
func (v *PassthroughInputValue) Hash()(uint32,error){return 0,fmt.Errorf("unhashable: llb.passthrough_input")}
func (v *PassthroughInputValue) Attr(name string)(starlark.Value,error){
		if v==nil{return nil,fmt.Errorf("llb.passthrough_input: nil receiver")}
	switch name{
	case "state":return WrapState(v.input.State),nil
	case "output":return starlark.Bool(v.input.Output),nil
	default:return nil,nil
	}
}
func (v *PassthroughInputValue) AttrNames()[]string{return SnakeCaseNames([]string{"state","output","state","output"})}

// PassthroughOpValue wraps a multi-output passthrough vertex.
type PassthroughOpValue struct { op *buildkitllb.PassthroughOp }

// WrapPassthroughOp adapts a native PassthroughOp.
func WrapPassthroughOp(op *buildkitllb.PassthroughOp)*PassthroughOpValue{return &PassthroughOpValue{op:op}}
func (v *PassthroughOpValue) Unwrap()*buildkitllb.PassthroughOp{if v==nil{return nil};return v.op}
func (v *PassthroughOpValue) String()string{return "<llb.passthrough_op>"}
func (v *PassthroughOpValue) Type()string{return "llb.passthrough_op"}
func (v *PassthroughOpValue) Freeze(){}
func (v *PassthroughOpValue) Truth()starlark.Bool{return starlark.Bool(v!=nil&&v.op!=nil)}
func (v *PassthroughOpValue) Hash()(uint32,error){return 0,fmt.Errorf("unhashable: llb.passthrough_op")}
func (v *PassthroughOpValue) Attr(name string)(starlark.Value,error){
		if v==nil||v.op==nil{return nil,fmt.Errorf("llb.passthrough_op: nil receiver")}
	switch name{
	case "output":
		return boundBuiltin("passthrough_op.output",func(_ *starlark.Thread,args starlark.Tuple,kwargs []starlark.Tuple)(starlark.Value,error){
			if err:=arity("passthrough_op.output",args,kwargs,0,0);err!=nil{return nil,err}
			return WrapOutput(v.op.Output()),nil
		}),nil
	case "output_at":
		return boundBuiltin("passthrough_op.output_at",func(_ *starlark.Thread,args starlark.Tuple,kwargs []starlark.Tuple)(starlark.Value,error){
			if err:=arity("passthrough_op.output_at",args,kwargs,1,1);err!=nil{return nil,err}
			index,err:=intAt("passthrough_op.output_at",args,0);if err!=nil{return nil,err}
			return WrapOutput(v.op.OutputAt(index)),nil
		}),nil
	case "inputs":
		return boundBuiltin("passthrough_op.inputs",func(_ *starlark.Thread,args starlark.Tuple,kwargs []starlark.Tuple)(starlark.Value,error){
			if err:=arity("passthrough_op.inputs",args,kwargs,0,0);err!=nil{return nil,err}
			return outputListValue(v.op.Inputs()),nil
		}),nil
	case "validate":
		return boundBuiltin("passthrough_op.validate",func(thread *starlark.Thread,args starlark.Tuple,kwargs []starlark.Tuple)(starlark.Value,error){
			if err:=arity("passthrough_op.validate",args,kwargs,0,-1);err!=nil{return nil,err}
			constraints,err:=constraintsFromArgs(args,0);if err!=nil{return nil,err}
			if err:=v.op.Validate(callContext(thread),constraints);err!=nil{return nil,err}
			return starlark.None,nil
		}),nil
	case "marshal":
		return boundBuiltin("passthrough_op.marshal",func(thread *starlark.Thread,args starlark.Tuple,kwargs []starlark.Tuple)(starlark.Value,error){
			if err:=arity("passthrough_op.marshal",args,kwargs,0,-1);err!=nil{return nil,err}
			constraints,err:=constraintsFromArgs(args,0);if err!=nil{return nil,err}
			dgst,data,metadata,sources,err:=v.op.Marshal(callContext(thread),constraints);if err!=nil{return nil,err}
			return marshalResult(dgst,data,metadata,sources),nil
		}),nil
	default:return nil,nil
	}
}
func (v *PassthroughOpValue) AttrNames()[]string{return SnakeCaseNames([]string{"output","output_at","inputs","validate","marshal","output","output_at","inputs","validate","marshal"})}

func passthroughBuiltins()map[string]starlark.Value{
	return map[string]starlark.Value{
		"passthrough_input":builtin("passthrough_input",passthroughInputBuiltin),
		"new_passthrough_op":builtin("new_passthrough_op",newPassthroughOpBuiltin),
		"new_passthrough":builtin("new_passthrough",newPassthroughBuiltin),
		"passthrough":builtin("passthrough",newPassthroughBuiltin),
	}
}
func passthroughInputBuiltin(_ *starlark.Thread,_ *starlark.Builtin,args starlark.Tuple,kwargs []starlark.Tuple)(starlark.Value,error){
	var stateValue starlark.Value
	var output bool=true
	if err:=UnpackArgs("passthrough_input",args,kwargs,"state",&stateValue,"output?",&output);err!=nil{return nil,err}
	state,err:=stateFromValue(stateValue);if err!=nil{return nil,err}
	return WrapPassthroughInput(buildkitllb.PassthroughInput{State:state,Output:output}),nil
}
func passthroughInputsFromValue(value starlark.Value)([]buildkitllb.PassthroughInput,error){
	list,ok:=value.(starlark.Indexable);if !ok{return nil,fmt.Errorf("passthrough inputs: got %s, want list or tuple",valueType(value))}
	inputs:=make([]buildkitllb.PassthroughInput,list.Len())
	for i:=range inputs{
		item,ok:=list.Index(i).(*PassthroughInputValue)
		if !ok||item==nil{return nil,fmt.Errorf("passthrough inputs[%d]: got %s, want llb.passthrough_input",i,valueType(list.Index(i)))}
		inputs[i]=item.input
	}
	return inputs,nil
}
func newPassthroughOpBuiltin(_ *starlark.Thread,_ *starlark.Builtin,args starlark.Tuple,kwargs []starlark.Tuple)(starlark.Value,error){
	if err:=arity("new_passthrough_op",args,kwargs,2,-1);err!=nil{return nil,err}
	id,err:=stringAt("new_passthrough_op",args,0);if err!=nil{return nil,err}
	inputs,err:=passthroughInputsFromValue(args[1]);if err!=nil{return nil,err}
	options,err:=optionsFromArgs[buildkitllb.ConstraintsOpt](args,2,"constraints_opt");if err!=nil{return nil,err}
	return WrapPassthroughOp(buildkitllb.NewPassthroughOp(id,inputs,options...)),nil
}
func newPassthroughBuiltin(_ *starlark.Thread,_ *starlark.Builtin,args starlark.Tuple,kwargs []starlark.Tuple)(starlark.Value,error){
	if err:=arity("new_passthrough",args,kwargs,2,-1);err!=nil{return nil,err}
	id,err:=stringAt("new_passthrough",args,0);if err!=nil{return nil,err}
	inputs,err:=passthroughInputsFromValue(args[1]);if err!=nil{return nil,err}
	options,err:=optionsFromArgs[buildkitllb.ConstraintsOpt](args,2,"constraints_opt");if err!=nil{return nil,err}
	return WrapState(buildkitllb.NewPassthrough(id,inputs,options...)),nil
}

var(
	_ starlark.Value=(*PassthroughInputValue)(nil)
	_ starlark.HasAttrs=(*PassthroughInputValue)(nil)
	_ starlark.Value=(*PassthroughOpValue)(nil)
	_ starlark.HasAttrs=(*PassthroughOpValue)(nil)
)
