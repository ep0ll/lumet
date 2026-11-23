// Package llbbuild exposes BuildKit's llbbuild extension as a Starlark module.
package llbbuild

import (
	"fmt"

	starlarkllb "github.com/ep0ll/lumet/starlark/llb"
	buildkitllb "github.com/moby/buildkit/client/llb"
	buildkitllbbuild "github.com/moby/buildkit/client/llb/llbbuild"
	"go.starlark.net/starlark"
	"go.starlark.net/starlarkstruct"
)

type buildOptionValue struct { option buildkitllbbuild.BuildOption }
func (v *buildOptionValue) String()string{return "<llbbuild.BuildOption>"}
func (v *buildOptionValue) Type()string{return "llbbuild.BuildOption"}
func (v *buildOptionValue) Freeze(){}
func (v *buildOptionValue) Truth()starlark.Bool{return starlark.Bool(v!=nil&&v.option!=nil)}
func (v *buildOptionValue) Hash()(uint32,error){return 0,fmt.Errorf("unhashable: llbbuild.BuildOption")}

type buildInfoValue struct { info *buildkitllbbuild.BuildInfo }
func (v *buildInfoValue) String()string{return "<llbbuild.BuildInfo>"}
func (v *buildInfoValue) Type()string{return "llbbuild.BuildInfo"}
func (v *buildInfoValue) Freeze(){}
func (v *buildInfoValue) Truth()starlark.Bool{return starlark.Bool(v!=nil&&v.info!=nil)}
func (v *buildInfoValue) Hash()(uint32,error){return 0,fmt.Errorf("unhashable: llbbuild.BuildInfo")}
func (v *buildInfoValue) Attr(name string)(starlark.Value,error){
	if v==nil||v.info==nil{return nil,fmt.Errorf("llbbuild.BuildInfo: nil receiver")}
	switch name{
	case "definition_filename","DefinitionFilename":
		return starlark.String(v.info.DefinitionFilename),nil
	case "constraints","Constraints":
		return starlarkllb.WrapConstraints(v.info.Constraints),nil
	default:return nil,nil
	}
}
func (v *buildInfoValue) AttrNames()[]string{return []string{"definition_filename","constraints","DefinitionFilename","Constraints"}}

func newModule()*starlarkstruct.Module{
	members:=starlark.StringDict{
		"Build":starlark.NewBuiltin("Build",buildBuiltin),
		"NewBuildOp":starlark.NewBuiltin("NewBuildOp",newBuildOpBuiltin),
		"BuildInfo":starlark.NewBuiltin("BuildInfo",buildInfoBuiltin),
		"WithFilename":starlark.NewBuiltin("WithFilename",withFilenameBuiltin),
		"WithConstraints":starlark.NewBuiltin("WithConstraints",withConstraintsBuiltin),
	}
	module:=&starlarkstruct.Module{Name:"llbbuild",Members:members}
	module.Freeze()
	return module
}

// Module is the frozen Starlark module for BuildKit's llbbuild subpackage.
var Module=newModule()

func Load(_ *starlark.Thread,name string)(starlark.StringDict,error){
	switch name{
	case "llb/llbbuild","llb/llbbuild.star","llbbuild","llbbuild.star":
		return Module.Members,nil
	default:return nil,fmt.Errorf("llbbuild: unknown module %q",name)
	}
}
func init(){
	if err:=starlarkllb.RegisterModule("llb/llbbuild",Module.Members);err!=nil{panic(err)}
}

func buildInfoBuiltin(_ *starlark.Thread,_ *starlark.Builtin,args starlark.Tuple,kwargs []starlark.Tuple)(starlark.Value,error){
	var filename string
	if err:=starlark.UnpackArgs("BuildInfo",args,kwargs,"definition_filename?",&filename);err!=nil{return nil,err}
	return &buildInfoValue{info:&buildkitllbbuild.BuildInfo{DefinitionFilename:filename}},nil
}
func withFilenameBuiltin(_ *starlark.Thread,_ *starlark.Builtin,args starlark.Tuple,kwargs []starlark.Tuple)(starlark.Value,error){
	var filename string
	if err:=starlark.UnpackArgs("WithFilename",args,kwargs,"filename",&filename);err!=nil{return nil,err}
	return &buildOptionValue{option:buildkitllbbuild.WithFilename(filename)},nil
}
func withConstraintsBuiltin(_ *starlark.Thread,_ *starlark.Builtin,args starlark.Tuple,kwargs []starlark.Tuple)(starlark.Value,error){
	if len(kwargs)!=0||len(args)!=1{return nil,fmt.Errorf("WithConstraints: expected one llb.ConstraintsOpt")}
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
		if !ok||value==nil||value.option==nil{return nil,fmt.Errorf("llbbuild: argument %d: got %s, want llbbuild.BuildOption",i+1,arg.Type())}
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
	return nil,fmt.Errorf("NewBuildOp: got %s, want llb.State or llb.Output",value.Type())
}
