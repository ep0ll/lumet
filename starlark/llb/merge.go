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
func (v *MergeOpValue) String()string{return "<llb.MergeOp>"}
func (v *MergeOpValue) Type()string{return "llb.MergeOp"}
func (v *MergeOpValue) Freeze(){}
func (v *MergeOpValue) Truth()starlark.Bool{return starlark.Bool(v!=nil&&v.op!=nil)}
func (v *MergeOpValue) Hash()(uint32,error){return 0,fmt.Errorf("unhashable: llb.MergeOp")}
func (v *MergeOpValue) Attr(name string)(starlark.Value,error){
	if v==nil||v.op==nil{return nil,fmt.Errorf("llb.MergeOp: nil receiver")}
	switch name{
	case "Output","output":
		return boundBuiltin("MergeOp.Output",func(_ *starlark.Thread,args starlark.Tuple,kwargs []starlark.Tuple)(starlark.Value,error){
			if err:=arity("MergeOp.Output",args,kwargs,0,0);err!=nil{return nil,err}
			output:=v.op.Output();if output==nil{return starlark.None,nil};return WrapOutput(output),nil
		}),nil
	case "Inputs","inputs":
		return boundBuiltin("MergeOp.Inputs",func(_ *starlark.Thread,args starlark.Tuple,kwargs []starlark.Tuple)(starlark.Value,error){
			if err:=arity("MergeOp.Inputs",args,kwargs,0,0);err!=nil{return nil,err}
			return outputListValue(v.op.Inputs()),nil
		}),nil
	case "Validate","validate":
		return boundBuiltin("MergeOp.Validate",func(thread *starlark.Thread,args starlark.Tuple,kwargs []starlark.Tuple)(starlark.Value,error){
			if err:=arity("MergeOp.Validate",args,kwargs,0,-1);err!=nil{return nil,err}
			constraints,err:=constraintsFromArgs(args,0);if err!=nil{return nil,err}
			if err:=v.op.Validate(callContext(thread),constraints);err!=nil{return nil,err}
			return starlark.None,nil
		}),nil
	case "Marshal","marshal":
		return boundBuiltin("MergeOp.Marshal",func(thread *starlark.Thread,args starlark.Tuple,kwargs []starlark.Tuple)(starlark.Value,error){
			if err:=arity("MergeOp.Marshal",args,kwargs,0,-1);err!=nil{return nil,err}
			constraints,err:=constraintsFromArgs(args,0);if err!=nil{return nil,err}
			dgst,data,metadata,sources,err:=v.op.Marshal(callContext(thread),constraints);if err!=nil{return nil,err}
			return marshalResult(dgst,data,metadata,sources),nil
		}),nil
	default:return nil,nil
	}
}
func (v *MergeOpValue) AttrNames()[]string{return []string{"Output","Inputs","Validate","Marshal","output","inputs","validate","marshal"}}

func mergeBuiltins()map[string]starlark.Value{
	return map[string]starlark.Value{
		"Merge":builtin("Merge",mergeBuiltin),
		"NewMerge":builtin("NewMerge",newMergeBuiltin),
	}
}
func mergeBuiltin(_ *starlark.Thread,_ *starlark.Builtin,args starlark.Tuple,kwargs []starlark.Tuple)(starlark.Value,error){
	if err:=arity("Merge",args,kwargs,1,-1);err!=nil{return nil,err}
	list,ok:=args[0].(starlark.Indexable);if !ok{return nil,fmt.Errorf("Merge: first argument must be a list or tuple of llb.State")}
	states:=make([]buildkitllb.State,list.Len())
	for i:=range states{
		state,err:=stateFromValue(list.Index(i));if err!=nil{return nil,fmt.Errorf("Merge: state %d: %w",i,err)}
		states[i]=state
	}
	options,err:=optionsFromArgs[buildkitllb.ConstraintsOpt](args,1,"ConstraintsOpt");if err!=nil{return nil,err}
	return WrapState(buildkitllb.Merge(states,options...)),nil
}
func newMergeBuiltin(_ *starlark.Thread,_ *starlark.Builtin,args starlark.Tuple,kwargs []starlark.Tuple)(starlark.Value,error){
	if err:=arity("NewMerge",args,kwargs,2,2);err!=nil{return nil,err}
	list,ok:=args[0].(starlark.Indexable);if !ok{return nil,fmt.Errorf("NewMerge: first argument must be a list or tuple")}
	states:=make([]buildkitllb.State,list.Len())
	for i:=range states{state,err:=stateFromValue(list.Index(i));if err!=nil{return nil,fmt.Errorf("NewMerge: state %d: %w",i,err)};states[i]=state}
	constraints,ok:=args[1].(*ConstraintsValue);if !ok||constraints==nil{return nil,fmt.Errorf("NewMerge: second argument must be llb.Constraints")}
	return WrapMergeOp(buildkitllb.NewMerge(states,constraints.Unwrap())),nil
}

var(
	_ starlark.Value=(*MergeOpValue)(nil)
	_ starlark.HasAttrs=(*MergeOpValue)(nil)
)
