// Package imagemetaresolver provides the Starlark wrapper for BuildKit's
// client/llb/imagemetaresolver package.
package imagemetaresolver

import (
	"fmt"

	starlarkllb "github.com/ep0ll/lumet/starlark/llb"
	buildkitresolver "github.com/moby/buildkit/client/llb/imagemetaresolver"
	ocispecs "github.com/opencontainers/image-spec/specs-go/v1"
	"go.starlark.net/starlark"
)

type imageMetaResolverOptionValue struct {
	option buildkitresolver.ImageMetaResolverOpt
}

func (v *imageMetaResolverOptionValue) String() string { return "<imagemetaresolver.image_meta_resolver_opt>" }
func (v *imageMetaResolverOptionValue) Type() string { return "imagemetaresolver.image_meta_resolver_opt" }
func (v *imageMetaResolverOptionValue) Freeze() {}
func (v *imageMetaResolverOptionValue) Truth() starlark.Bool { return starlark.Bool(v != nil) }
func (v *imageMetaResolverOptionValue) Hash() (uint32,error) {
	return 0,fmt.Errorf("unhashable: imagemetaresolver.image_meta_resolver_opt")
}

// Members returns this subpackage's Starlark globals. Centralized
// construction, freezing and registration are owned by starlark/modules.
func Members() starlark.StringDict {
	return starlark.StringDict{
		"with_default":          starlarkllb.WrapOption(buildkitresolver.WithDefault),
		"with_default_platform": starlark.NewBuiltin("with_default_platform", withDefaultPlatformBuiltin),
		"new":                   starlark.NewBuiltin("new", newResolverBuiltin),
		"default":               starlark.NewBuiltin("default", defaultResolverBuiltin),
	}
}

func withDefaultPlatformBuiltin(_ *starlark.Thread,_ *starlark.Builtin,args starlark.Tuple,kwargs []starlark.Tuple)(starlark.Value,error){
	var platformValue starlark.Value=starlark.None
	if err:=starlarkllb.UnpackArgs("with_default_platform",args,kwargs,"platform",&platformValue);err!=nil{
		return nil,err
	}
	var platform *ocispecs.Platform
	if platformValue!=starlark.None{
		value,ok:=platformValue.(*starlarkllb.OCIPlatformValue)
		if !ok || value==nil{
			return nil,fmt.Errorf("with_default_platform: got %s, want llb.oci_platform or none",platformValue.Type())
		}
		p:=value.Unwrap()
		platform=&p
	}
	return &imageMetaResolverOptionValue{option:buildkitresolver.WithDefaultPlatform(platform)},nil
}

func newResolverBuiltin(_ *starlark.Thread,_ *starlark.Builtin,args starlark.Tuple,kwargs []starlark.Tuple)(starlark.Value,error){
	options:=make([]buildkitresolver.ImageMetaResolverOpt,len(args))
	if len(kwargs)!=0{return nil,fmt.Errorf("new: options must be passed positionally")}
	for i,arg:=range args{
		value,ok:=arg.(*imageMetaResolverOptionValue)
		if !ok || value==nil{
			return nil,fmt.Errorf("new: argument %d: got %s, want imagemetaresolver.image_meta_resolver_opt",i+1,arg.Type())
		}
		options[i]=value.option
	}
	return starlarkllb.WrapImageMetaResolver(buildkitresolver.New(options...)),nil
}

func defaultResolverBuiltin(_ *starlark.Thread,_ *starlark.Builtin,args starlark.Tuple,kwargs []starlark.Tuple)(starlark.Value,error){
	if len(args)!=0 || len(kwargs)!=0{return nil,fmt.Errorf("default: does not accept arguments")}
	return starlarkllb.WrapImageMetaResolver(buildkitresolver.Default()),nil
}

var (
	_ starlark.Value=(*imageMetaResolverOptionValue)(nil)
)
