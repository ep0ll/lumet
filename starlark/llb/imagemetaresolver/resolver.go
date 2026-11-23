// Package imagemetaresolver provides the Starlark wrapper for BuildKit's
// client/llb/imagemetaresolver package.
package imagemetaresolver

import (
	"fmt"

	starlarkllb "github.com/ep0ll/lumet/starlark/llb"
	buildkitresolver "github.com/moby/buildkit/client/llb/imagemetaresolver"
	ocispecs "github.com/opencontainers/image-spec/specs-go/v1"
	"go.starlark.net/starlark"
	"go.starlark.net/starlarkstruct"
)

type imageMetaResolverOptionValue struct {
	option buildkitresolver.ImageMetaResolverOpt
}

func (v *imageMetaResolverOptionValue) String() string { return "<imagemetaresolver.ImageMetaResolverOpt>" }
func (v *imageMetaResolverOptionValue) Type() string { return "imagemetaresolver.ImageMetaResolverOpt" }
func (v *imageMetaResolverOptionValue) Freeze() {}
func (v *imageMetaResolverOptionValue) Truth() starlark.Bool { return starlark.Bool(v != nil) }
func (v *imageMetaResolverOptionValue) Hash() (uint32,error) {
	return 0,fmt.Errorf("unhashable: imagemetaresolver.ImageMetaResolverOpt")
}

func newModule() *starlarkstruct.Module {
	members:=starlark.StringDict{
		"WithDefault":starlarkllb.WrapOption(buildkitresolver.WithDefault),
		"WithDefaultPlatform":starlark.NewBuiltin("WithDefaultPlatform",withDefaultPlatformBuiltin),
		"New":starlark.NewBuiltin("New",newResolverBuiltin),
		"Default":starlark.NewBuiltin("Default",defaultResolverBuiltin),
	}
	module:=&starlarkstruct.Module{Name:"imagemetaresolver",Members:members}
	module.Freeze()
	return module
}

// Module is the frozen Starlark module for BuildKit's imagemetaresolver package.
var Module=newModule()

func Load(_ *starlark.Thread,name string)(starlark.StringDict,error){
	switch name{
	case "llb/imagemetaresolver","llb/imagemetaresolver.star","imagemetaresolver","imagemetaresolver.star":
		return Module.Members,nil
	default:
		return nil,fmt.Errorf("imagemetaresolver: unknown module %q",name)
	}
}

func init(){
	if err:=starlarkllb.RegisterModule("llb/imagemetaresolver",Module.Members);err!=nil{
		panic(err)
	}
}

func withDefaultPlatformBuiltin(_ *starlark.Thread,_ *starlark.Builtin,args starlark.Tuple,kwargs []starlark.Tuple)(starlark.Value,error){
	var platformValue starlark.Value=starlark.None
	if err:=starlark.UnpackArgs("WithDefaultPlatform",args,kwargs,"platform",&platformValue);err!=nil{
		return nil,err
	}
	var platform *ocispecs.Platform
	if platformValue!=starlark.None{
		value,ok:=platformValue.(*starlarkllb.OCIPlatformValue)
		if !ok || value==nil{
			return nil,fmt.Errorf("WithDefaultPlatform: got %s, want llb.OCIPlatform or None",platformValue.Type())
		}
		p:=value.Unwrap()
		platform=&p
	}
	return &imageMetaResolverOptionValue{option:buildkitresolver.WithDefaultPlatform(platform)},nil
}

func newResolverBuiltin(_ *starlark.Thread,_ *starlark.Builtin,args starlark.Tuple,kwargs []starlark.Tuple)(starlark.Value,error){
	options:=make([]buildkitresolver.ImageMetaResolverOpt,len(args))
	if len(kwargs)!=0{return nil,fmt.Errorf("New: options must be passed positionally")}
	for i,arg:=range args{
		value,ok:=arg.(*imageMetaResolverOptionValue)
		if !ok || value==nil{
			return nil,fmt.Errorf("New: argument %d: got %s, want imagemetaresolver.ImageMetaResolverOpt",i+1,arg.Type())
		}
		options[i]=value.option
	}
	return starlarkllb.WrapImageMetaResolver(buildkitresolver.New(options...)),nil
}

func defaultResolverBuiltin(_ *starlark.Thread,_ *starlark.Builtin,args starlark.Tuple,kwargs []starlark.Tuple)(starlark.Value,error){
	if len(args)!=0 || len(kwargs)!=0{return nil,fmt.Errorf("Default: does not accept arguments")}
	return starlarkllb.WrapImageMetaResolver(buildkitresolver.Default()),nil
}

var (
	_ starlark.Value=(*imageMetaResolverOptionValue)(nil)
)
