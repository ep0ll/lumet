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
	name = SnakeCaseIdentifier(name)
	if v==nil{return nil,fmt.Errorf("llb.passthrough_input: nil receiver")}
	switch name{
	case "state","State":return WrapState(v.input.State),nil
	case "output","Output":return starlark.Bool(v.input.Output),nil
	default:return nil,nil
	}
}
func (v *PassthroughInputValue) AttrNames()[]string{return SnakeCaseNames([]string{"state","output","State","Output"})}

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
	name = SnakeCaseIdentifier(name)
	if v==nil||v.op==nil{return nil,fmt.Errorf("llb.passthrough_op: nil receiver")}
	switch name{
	case "Output","output":
		return boundBuiltin("PassthroughOp.Output",func(_ *starlark.Thread,args starlark.Tuple,kwargs []starlark.Tuple)(starlark.Value,error){
			if err:=arity("PassthroughOp.Output",args,kwargs,0,0);err!=nil{return nil,err}
			return WrapOutput(v.op.Output()),nil
		}),nil
	case "OutputAt","output_at":
		return boundBuiltin("PassthroughOp.OutputAt",func(_ *starlark.Thread,args starlark.Tuple,kwargs []starlark.Tuple)(starlark.Value,error){
			if err:=arity("PassthroughOp.OutputAt",args,kwargs,1,1);err!=nil{return nil,err}
			index,err:=intAt("PassthroughOp.OutputAt",args,0);if err!=nil{return nil,err}
			return WrapOutput(v.op.OutputAt(index)),nil
		}),nil
	case "Inputs","inputs":
		return boundBuiltin("PassthroughOp.Inputs",func(_ *starlark.Thread,args starlark.Tuple,kwargs []starlark.Tuple)(starlark.Value,error){
			if err:=arity("PassthroughOp.Inputs",args,kwargs,0,0);err!=nil{return nil,err}
			return outputListValue(v.op.Inputs()),nil
		}),nil
	case "Validate","validate":
		return boundBuiltin("PassthroughOp.Validate",func(thread *starlark.Thread,args starlark.Tuple,kwargs []starlark.Tuple)(starlark.Value,error){
			if err:=arity("PassthroughOp.Validate",args,kwargs,0,-1);err!=nil{return nil,err}
			constraints,err:=constraintsFromArgs(args,0);if err!=nil{return nil,err}
			if err:=v.op.Validate(callContext(thread),constraints);err!=nil{return nil,err}
			return starlark.None,nil
		}),nil
	case "Marshal","marshal":
		return boundBuiltin("PassthroughOp.Marshal",func(thread *starlark.Thread,args starlark.Tuple,kwargs []starlark.Tuple)(starlark.Value,error){
			if err:=arity("PassthroughOp.Marshal",args,kwargs,0,-1);err!=nil{return nil,err}
			constraints,err:=constraintsFromArgs(args,0);if err!=nil{return nil,err}
			dgst,data,metadata,sources,err:=v.op.Marshal(callContext(thread),constraints);if err!=nil{return nil,err}
			return marshalResult(dgst,data,metadata,sources),nil
		}),nil
	default:return nil,nil
	}
}
func (v *PassthroughOpValue) AttrNames()[]string{return SnakeCaseNames([]string{"Output","OutputAt","Inputs","Validate","Marshal","output","output_at","inputs","validate","marshal"})}

func passthroughBuiltins()map[string]starlark.Value{
	return map[string]starlark.Value{
		"PassthroughInput":builtin("PassthroughInput",passthroughInputBuiltin),
		"NewPassthroughOp":builtin("NewPassthroughOp",newPassthroughOpBuiltin),
		"NewPassthrough":builtin("NewPassthrough",newPassthroughBuiltin),
		"Passthrough":builtin("Passthrough",newPassthroughBuiltin),
	}
}
func passthroughInputBuiltin(_ *starlark.Thread,_ *starlark.Builtin,args starlark.Tuple,kwargs []starlark.Tuple)(starlark.Value,error){
	var stateValue starlark.Value
	var output bool=true
	if err:=UnpackArgs("PassthroughInput",args,kwargs,"state",&stateValue,"output?",&output);err!=nil{return nil,err}
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
	if err:=arity("NewPassthroughOp",args,kwargs,2,-1);err!=nil{return nil,err}
	id,err:=stringAt("NewPassthroughOp",args,0);if err!=nil{return nil,err}
	inputs,err:=passthroughInputsFromValue(args[1]);if err!=nil{return nil,err}
	options,err:=optionsFromArgs[buildkitllb.ConstraintsOpt](args,2,"ConstraintsOpt");if err!=nil{return nil,err}
	return WrapPassthroughOp(buildkitllb.NewPassthroughOp(id,inputs,options...)),nil
}
func newPassthroughBuiltin(_ *starlark.Thread,_ *starlark.Builtin,args starlark.Tuple,kwargs []starlark.Tuple)(starlark.Value,error){
	if err:=arity("NewPassthrough",args,kwargs,2,-1);err!=nil{return nil,err}
	id,err:=stringAt("NewPassthrough",args,0);if err!=nil{return nil,err}
	inputs,err:=passthroughInputsFromValue(args[1]);if err!=nil{return nil,err}
	options,err:=optionsFromArgs[buildkitllb.ConstraintsOpt](args,2,"ConstraintsOpt");if err!=nil{return nil,err}
	return WrapState(buildkitllb.NewPassthrough(id,inputs,options...)),nil
}

var(
	_ starlark.Value=(*PassthroughInputValue)(nil)
	_ starlark.HasAttrs=(*PassthroughInputValue)(nil)
	_ starlark.Value=(*PassthroughOpValue)(nil)
	_ starlark.HasAttrs=(*PassthroughOpValue)(nil)
)
