package llb

import (
	"fmt"

	buildkitllb "github.com/moby/buildkit/client/llb"
	"go.starlark.net/starlark"
)

// MergeOpValue wraps BuildKit's overlay merge vertex.
type MergeOpValue struct { op *buildkitllb.MergeOp }

// WrapMergeOp adapts a native MergeOp.
func WrapMergeOp(op *buildkitllb.MergeOp)*MergeOpValue{return &MergeOpValue{op:op}}
func (v *MergeOpValue) Unwrap()*buildkitllb.MergeOp{if v==nil{return nil};return v.op}
func (v *MergeOpValue) String()string{return "<llb.merge_op>"}
func (v *MergeOpValue) Type()string{return "llb.merge_op"}
func (v *MergeOpValue) Freeze(){}
func (v *MergeOpValue) Truth()starlark.Bool{return starlark.Bool(v!=nil&&v.op!=nil)}
func (v *MergeOpValue) Hash()(uint32,error){return 0,fmt.Errorf("unhashable: llb.merge_op")}
func (v *MergeOpValue) Attr(name string)(starlark.Value,error){
		if v==nil||v.op==nil{return nil,fmt.Errorf("llb.merge_op: nil receiver")}
	switch name{
	case "output":
		return boundBuiltin("merge_op.output",func(_ *starlark.Thread,args starlark.Tuple,kwargs []starlark.Tuple)(starlark.Value,error){
			if err:=arity("merge_op.output",args,kwargs,0,0);err!=nil{return nil,err}
			output:=v.op.Output();if output==nil{return starlark.None,nil};return WrapOutput(output),nil
		}),nil
	case "inputs":
		return boundBuiltin("merge_op.inputs",func(_ *starlark.Thread,args starlark.Tuple,kwargs []starlark.Tuple)(starlark.Value,error){
			if err:=arity("merge_op.inputs",args,kwargs,0,0);err!=nil{return nil,err}
			return outputListValue(v.op.Inputs()),nil
		}),nil
	case "validate":
		return boundBuiltin("merge_op.validate",func(thread *starlark.Thread,args starlark.Tuple,kwargs []starlark.Tuple)(starlark.Value,error){
			if err:=arity("merge_op.validate",args,kwargs,0,-1);err!=nil{return nil,err}
			constraints,err:=constraintsFromArgs(args,0);if err!=nil{return nil,err}
			if err:=v.op.Validate(callContext(thread),constraints);err!=nil{return nil,err}
			return starlark.None,nil
		}),nil
	case "marshal":
		return boundBuiltin("merge_op.marshal",func(thread *starlark.Thread,args starlark.Tuple,kwargs []starlark.Tuple)(starlark.Value,error){
			if err:=arity("merge_op.marshal",args,kwargs,0,-1);err!=nil{return nil,err}
			constraints,err:=constraintsFromArgs(args,0);if err!=nil{return nil,err}
			dgst,data,metadata,sources,err:=v.op.Marshal(callContext(thread),constraints);if err!=nil{return nil,err}
			return marshalResult(dgst,data,metadata,sources),nil
		}),nil
	default:return nil,nil
	}
}
func (v *MergeOpValue) AttrNames()[]string{return SnakeCaseNames([]string{"output","inputs","validate","marshal","output","inputs","validate","marshal"})}

func mergeBuiltins()map[string]starlark.Value{
	return map[string]starlark.Value{
		"merge":builtin("merge",mergeBuiltin),
		"new_merge":builtin("new_merge",newMergeBuiltin),
	}
}
func mergeBuiltin(_ *starlark.Thread,_ *starlark.Builtin,args starlark.Tuple,kwargs []starlark.Tuple)(starlark.Value,error){
	if err:=arity("merge",args,kwargs,1,-1);err!=nil{return nil,err}
	list,ok:=args[0].(starlark.Indexable);if !ok{return nil,fmt.Errorf("merge: first argument must be a list or tuple of llb.state")}
	states:=make([]buildkitllb.State,list.Len())
	for i:=range states{
		state,err:=stateFromValue(list.Index(i));if err!=nil{return nil,fmt.Errorf("merge: state %d: %w",i,err)}
		states[i]=state
	}
	options,err:=optionsFromArgs[buildkitllb.ConstraintsOpt](args,1,"constraints_opt");if err!=nil{return nil,err}
	return WrapState(buildkitllb.Merge(states,options...)),nil
}
func newMergeBuiltin(_ *starlark.Thread,_ *starlark.Builtin,args starlark.Tuple,kwargs []starlark.Tuple)(starlark.Value,error){
	if err:=arity("new_merge",args,kwargs,2,2);err!=nil{return nil,err}
	list,ok:=args[0].(starlark.Indexable);if !ok{return nil,fmt.Errorf("new_merge: first argument must be a list or tuple")}
	states:=make([]buildkitllb.State,list.Len())
	for i:=range states{state,err:=stateFromValue(list.Index(i));if err!=nil{return nil,fmt.Errorf("new_merge: state %d: %w",i,err)};states[i]=state}
	constraints,ok:=args[1].(*ConstraintsValue);if !ok||constraints==nil{return nil,fmt.Errorf("new_merge: second argument must be llb.constraints")}
	return WrapMergeOp(buildkitllb.NewMerge(states,constraints.Unwrap())),nil
}

var(
	_ starlark.Value=(*MergeOpValue)(nil)
	_ starlark.HasAttrs=(*MergeOpValue)(nil)
)
