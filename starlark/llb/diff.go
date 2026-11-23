package llb

import (
	"fmt"

	buildkitllb "github.com/moby/buildkit/client/llb"
	"go.starlark.net/starlark"
)

// DiffOpValue wraps BuildKit's two-input diff vertex.
type DiffOpValue struct { op *buildkitllb.DiffOp }

// WrapDiffOp adapts a native DiffOp.
func WrapDiffOp(op *buildkitllb.DiffOp)*DiffOpValue{return &DiffOpValue{op:op}}
func (v *DiffOpValue) Unwrap()*buildkitllb.DiffOp{if v==nil{return nil};return v.op}
func (v *DiffOpValue) String()string{return "<llb.DiffOp>"}
func (v *DiffOpValue) Type()string{return "llb.DiffOp"}
func (v *DiffOpValue) Freeze(){}
func (v *DiffOpValue) Truth()starlark.Bool{return starlark.Bool(v!=nil&&v.op!=nil)}
func (v *DiffOpValue) Hash()(uint32,error){return 0,fmt.Errorf("unhashable: llb.DiffOp")}
func (v *DiffOpValue) Attr(name string)(starlark.Value,error){
	if v==nil||v.op==nil{return nil,fmt.Errorf("llb.DiffOp: nil receiver")}
	switch name{
	case "Output","output":
		return boundBuiltin("DiffOp.Output",func(_ *starlark.Thread,args starlark.Tuple,kwargs []starlark.Tuple)(starlark.Value,error){
			if err:=arity("DiffOp.Output",args,kwargs,0,0);err!=nil{return nil,err}
			output:=v.op.Output();if output==nil{return starlark.None,nil};return WrapOutput(output),nil
		}),nil
	case "Inputs","inputs":
		return boundBuiltin("DiffOp.Inputs",func(_ *starlark.Thread,args starlark.Tuple,kwargs []starlark.Tuple)(starlark.Value,error){
			if err:=arity("DiffOp.Inputs",args,kwargs,0,0);err!=nil{return nil,err}
			return outputListValue(v.op.Inputs()),nil
		}),nil
	case "Validate","validate":
		return boundBuiltin("DiffOp.Validate",func(thread *starlark.Thread,args starlark.Tuple,kwargs []starlark.Tuple)(starlark.Value,error){
			if err:=arity("DiffOp.Validate",args,kwargs,0,-1);err!=nil{return nil,err}
			constraints,err:=constraintsFromArgs(args,0);if err!=nil{return nil,err}
			if err:=v.op.Validate(callContext(thread),constraints);err!=nil{return nil,err}
			return starlark.None,nil
		}),nil
	case "Marshal","marshal":
		return boundBuiltin("DiffOp.Marshal",func(thread *starlark.Thread,args starlark.Tuple,kwargs []starlark.Tuple)(starlark.Value,error){
			if err:=arity("DiffOp.Marshal",args,kwargs,0,-1);err!=nil{return nil,err}
			constraints,err:=constraintsFromArgs(args,0);if err!=nil{return nil,err}
			dgst,data,metadata,sources,err:=v.op.Marshal(callContext(thread),constraints);if err!=nil{return nil,err}
			return marshalResult(dgst,data,metadata,sources),nil
		}),nil
	default:return nil,nil
	}
}
func (v *DiffOpValue) AttrNames()[]string{return []string{"Output","Inputs","Validate","Marshal","output","inputs","validate","marshal"}}

func diffBuiltins()map[string]starlark.Value{
	return map[string]starlark.Value{
		"Diff":builtin("Diff",diffBuiltin),
		"NewDiff":builtin("NewDiff",newDiffBuiltin),
	}
}
func diffBuiltin(_ *starlark.Thread,_ *starlark.Builtin,args starlark.Tuple,kwargs []starlark.Tuple)(starlark.Value,error){
	if err:=arity("Diff",args,kwargs,2,-1);err!=nil{return nil,err}
	lower,err:=stateFromValue(args[0]);if err!=nil{return nil,err}
	upper,err:=stateFromValue(args[1]);if err!=nil{return nil,err}
	options,err:=optionsFromArgs[buildkitllb.ConstraintsOpt](args,2,"ConstraintsOpt");if err!=nil{return nil,err}
	return WrapState(buildkitllb.Diff(lower,upper,options...)),nil
}
func newDiffBuiltin(_ *starlark.Thread,_ *starlark.Builtin,args starlark.Tuple,kwargs []starlark.Tuple)(starlark.Value,error){
	if err:=arity("NewDiff",args,kwargs,3,3);err!=nil{return nil,err}
	lower,err:=stateFromValue(args[0]);if err!=nil{return nil,err}
	upper,err:=stateFromValue(args[1]);if err!=nil{return nil,err}
	constraints,ok:=args[2].(*ConstraintsValue);if !ok||constraints==nil{return nil,fmt.Errorf("NewDiff: argument 3: want llb.Constraints")}
	return WrapDiffOp(buildkitllb.NewDiff(lower,upper,constraints.Unwrap())),nil
}

var(
	_ starlark.Value=(*DiffOpValue)(nil)
	_ starlark.HasAttrs=(*DiffOpValue)(nil)
)
