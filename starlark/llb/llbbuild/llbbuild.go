// Package llbbuild exposes BuildKit's llbbuild extension as a Starlark module.
package llbbuild

import (
	"fmt"

	starlarkllb "github.com/ep0ll/lumet/starlark/llb"
	buildkitllb "github.com/moby/buildkit/client/llb"
	buildkitllbbuild "github.com/moby/buildkit/client/llb/llbbuild"
	"go.starlark.net/starlark"
)

type buildOptionValue struct { option buildkitllbbuild.BuildOption }
func (v *buildOptionValue) String()string{return "<llbbuild.build_option>"}
func (v *buildOptionValue) Type()string{return "llbbuild.build_option"}
func (v *buildOptionValue) Freeze(){}
func (v *buildOptionValue) Truth()starlark.Bool{return starlark.Bool(v!=nil&&v.option!=nil)}
func (v *buildOptionValue) Hash()(uint32,error){return 0,fmt.Errorf("unhashable: llbbuild.build_option")}

type buildInfoValue struct { info *buildkitllbbuild.BuildInfo }
func (v *buildInfoValue) String()string{return "<llbbuild.build_info>"}
func (v *buildInfoValue) Type()string{return "llbbuild.build_info"}
func (v *buildInfoValue) Freeze(){}
func (v *buildInfoValue) Truth()starlark.Bool{return starlark.Bool(v!=nil&&v.info!=nil)}
func (v *buildInfoValue) Hash()(uint32,error){return 0,fmt.Errorf("unhashable: llbbuild.build_info")}
func (v *buildInfoValue) Attr(name string)(starlark.Value,error){
	name = starlarkllb.SnakeCaseIdentifier(name)
	if v==nil||v.info==nil{return nil,fmt.Errorf("llbbuild.build_info: nil receiver")}
	switch name{
	case "definition_filename","DefinitionFilename":
		return starlark.String(v.info.DefinitionFilename),nil
	case "constraints","Constraints":
		return starlarkllb.WrapConstraints(v.info.Constraints),nil
	default:return nil,nil
	}
}
func (v *buildInfoValue) AttrNames()[]string{return starlarkllb.SnakeCaseNames([]string{"definition_filename","constraints","DefinitionFilename","Constraints"})}

// Members returns this subpackage's Starlark globals. Centralized
// construction, freezing and registration are owned by starlark/modules.
func Members() starlark.StringDict {
	return starlark.StringDict{
		"build":             starlark.NewBuiltin("build", buildBuiltin),
		"new_build_op":      starlark.NewBuiltin("new_build_op", newBuildOpBuiltin),
		"build_info":        starlark.NewBuiltin("build_info", buildInfoBuiltin),
		"with_filename":     starlark.NewBuiltin("with_filename", withFilenameBuiltin),
		"with_constraints":  starlark.NewBuiltin("with_constraints", withConstraintsBuiltin),
	}
}

func buildInfoBuiltin(_ *starlark.Thread,_ *starlark.Builtin,args starlark.Tuple,kwargs []starlark.Tuple)(starlark.Value,error){
	var filename string
	if err:=starlarkllb.UnpackArgs("BuildInfo",args,kwargs,"definition_filename?",&filename);err!=nil{return nil,err}
	return &buildInfoValue{info:&buildkitllbbuild.BuildInfo{DefinitionFilename:filename}},nil
}
func withFilenameBuiltin(_ *starlark.Thread,_ *starlark.Builtin,args starlark.Tuple,kwargs []starlark.Tuple)(starlark.Value,error){
	var filename string
	if err:=starlarkllb.UnpackArgs("WithFilename",args,kwargs,"filename",&filename);err!=nil{return nil,err}
	return &buildOptionValue{option:buildkitllbbuild.WithFilename(filename)},nil
}
func withConstraintsBuiltin(_ *starlark.Thread,_ *starlark.Builtin,args starlark.Tuple,kwargs []starlark.Tuple)(starlark.Value,error){
	if len(kwargs)!=0||len(args)!=1{return nil,fmt.Errorf("WithConstraints: expected one llb.constraints_opt")}
	option,err:=starlarkllbOption(args[0],"ConstraintsOpt")
	if err!=nil{return nil,err}
	return &buildOptionValue{option:buildkitllbbuild.WithConstraints(option)},nil
}
func starlarkllbOption(value starlark.Value,want string)(buildkitllb.ConstraintsOpt,error){
	return starlarkllb.UnwrapOption[buildkitllb.ConstraintsOpt](value,want)
}
func buildBuiltin(_ *starlark.Thread,_ *starlark.Builtin,args starlark.Tuple,kwargs []starlark.Tuple)(starlark.Value,error){
	options,err:=buildOptions(args,kwargs);if err!=nil{return nil,err}
	return starlarkllb.WrapOption(buildkitllbbuild.Build(options...)),nil
}
func newBuildOpBuiltin(_ *starlark.Thread,_ *starlark.Builtin,args starlark.Tuple,kwargs []starlark.Tuple)(starlark.Value,error){
	if len(args)<1{return nil,fmt.Errorf("NewBuildOp: missing source output")}
	source,err:=outputFromStarlark(args[0]);if err!=nil{return nil,err}
	options,err:=buildOptions(args[1:],kwargs);if err!=nil{return nil,err}
	return starlarkllb.WrapVertex(buildkitllbbuild.NewBuildOp(source,options...)),nil
}
func buildOptions(args starlark.Tuple,kwargs []starlark.Tuple)([]buildkitllbbuild.BuildOption,error){
	if len(kwargs)!=0{return nil,fmt.Errorf("llbbuild: use BuildOption values positionally")}
	options:=make([]buildkitllbbuild.BuildOption,len(args))
	for i,arg:=range args{
		value,ok:=arg.(*buildOptionValue)
		if !ok||value==nil||value.option==nil{return nil,fmt.Errorf("llbbuild: argument %d: got %s, want llbbuild.build_option",i+1,arg.Type())}
		options[i]=value.option
	}
	return options,nil
}
func outputFromStarlark(value starlark.Value)(buildkitllb.Output,error){
	switch v:=value.(type){
	case *starlarkllb.StateValue:
		if v!=nil{return v.Unwrap().Output(),nil}
	case *starlarkllb.OutputValue:
		if v!=nil{return v.Unwrap(),nil}
	}
	return nil,fmt.Errorf("NewBuildOp: got %s, want llb.state or llb.output",value.Type())
}
