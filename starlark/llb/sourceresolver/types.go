// Package sourceresolver exposes BuildKit's client/llb/sourceresolver
// request and response types to Starlark.
package sourceresolver

import (
	"fmt"
	starlarkllb "github.com/ep0ll/lumet/starlark/llb"
	buildkitsourceresolver "github.com/moby/buildkit/client/llb/sourceresolver"
	"github.com/moby/buildkit/solver/pb"
	sourcepolicy "github.com/moby/buildkit/sourcepolicy/pb"
	digest "github.com/opencontainers/go-digest"
	ocispecs "github.com/opencontainers/image-spec/specs-go/v1"
	"go.starlark.net/starlark"
	"go.starlark.net/starlarkstruct"
	"google.golang.org/protobuf/proto"
)

// recordValue is the Starlark representation of one of the small public
// request/response structs in BuildKit's sourceresolver package.
type recordValue struct {
	name   string
	native any
	fields starlark.StringDict
}

func newRecord(name string, native any, fields starlark.StringDict) *recordValue {
	return &recordValue{name: name, native: native, fields: fields}
}
func (v *recordValue) String() string { return "<sourceresolver." + v.name + ">" }
func (v *recordValue) Type() string { return "sourceresolver." + v.name }
func (v *recordValue) Freeze() { v.fields.Freeze() }
func (v *recordValue) Truth() starlark.Bool { return starlark.Bool(v != nil) }
func (v *recordValue) Hash() (uint32,error) { return 0, fmt.Errorf("unhashable: sourceresolver.%s", v.name) }
func (v *recordValue) Attr(name string) (starlark.Value,error) {
	if v == nil { return nil, fmt.Errorf("sourceresolver: nil %s", name) }
	if value, ok := v.fields[name]; ok { return value,nil }
	return nil,nil
}
func (v *recordValue) AttrNames() []string {
	names := make([]string,0,len(v.fields))
	for name := range v.fields { names = append(names,name) }
	return names
}

func stringField(name string, value string) starlark.Value { return starlark.String(value) }
func bytesField(value []byte) starlark.Value { return starlark.Bytes(append([]byte(nil),value...)) }
func digestField(value digest.Digest) starlark.Value { return starlark.String(value.String()) }
func boolField(value bool) starlark.Value { return starlark.Bool(value) }
func intField(value int) starlark.Value { return starlark.MakeInt(value) }
func stringList(values []string) starlark.Value {
	items:=make([]starlark.Value,len(values))
	for i,v:=range values { items[i]=starlark.String(v) }
	return starlark.NewList(items)
}
func recordFrom[T any](value starlark.Value, name string) (T,error) {
	var zero T
	record, ok:=value.(*recordValue)
	if !ok || record==nil { return zero,fmt.Errorf("got %s, want sourceresolver.%s",starlarkType(value),name) }
	result,ok:=record.native.(T)
	if !ok { return zero,fmt.Errorf("sourceresolver.%s has invalid native value",name) }
	return result,nil
}
func starlarkType(value starlark.Value) string {
	if value==nil { return "nil" }
	return value.Type()
}
func stringArg(name string, value starlark.Value) (string,error) {
	v,ok:=value.(starlark.String)
	if !ok { return "",fmt.Errorf("%s: got %s, want string",name,starlarkType(value)) }
	return string(v),nil
}
func boolArg(name string, value starlark.Value) (bool,error) {
	v,ok:=value.(starlark.Bool)
	if !ok { return false,fmt.Errorf("%s: got %s, want bool",name,starlarkType(value)) }
	return bool(v),nil
}
func bytesArg(name string, value starlark.Value) ([]byte,error) {
	switch v:=value.(type) {
	case starlark.Bytes: return append([]byte(nil),[]byte(v)...),nil
	case starlark.String: return []byte(string(v)),nil
	default: return nil,fmt.Errorf("%s: got %s, want bytes or string",name,starlarkType(value))
	}
}
func platformArg(name string, value starlark.Value) (*ocispecs.Platform,error) {
	if value==starlark.None { return nil,nil }
	v,ok:=value.(*starlarkllb.OCIPlatformValue)
	if !ok || v==nil { return nil,fmt.Errorf("%s: got %s, want llb.OCIPlatform or None",name,starlarkType(value)) }
	p:=v.Unwrap()
	return &p,nil
}

func sourceOpBuiltin(_ *starlark.Thread,_ *starlark.Builtin,args starlark.Tuple,kwargs []starlark.Tuple)(starlark.Value,error) {
	var identifier string
	var attrsValue starlark.Value = starlark.None
	if err:=starlark.UnpackArgs("SourceOp",args,kwargs,"identifier",&identifier,"attrs?",&attrsValue); err!=nil { return nil,err }
	attrs:=map[string]string{}
	if attrsValue!=starlark.None {
		dict,ok:=attrsValue.(*starlark.Dict)
		if !ok { return nil,fmt.Errorf("SourceOp.attrs: got %s, want dict",starlarkType(attrsValue)) }
		for _,item:=range dict.Items() {
			key,err:=stringArg("SourceOp.attrs key",item[0]); if err!=nil{return nil,err}
			value,err:=stringArg("SourceOp.attrs value",item[1]); if err!=nil{return nil,err}
			attrs[key]=value
		}
	}
	return &SourceOpValue{op:&pb.SourceOp{Identifier:identifier,Attrs:attrs}},nil
}

// SourceOpValue wraps solver/pb.SourceOp, not client/llb.SourceOp.
type SourceOpValue struct { op *pb.SourceOp }
func (v *SourceOpValue) String() string { return "<sourceresolver.SourceOp>" }
func (v *SourceOpValue) Type() string { return "sourceresolver.SourceOp" }
func (v *SourceOpValue) Freeze() {}
func (v *SourceOpValue) Truth() starlark.Bool { return starlark.Bool(v!=nil && v.op!=nil) }
func (v *SourceOpValue) Hash()(uint32,error){return 0,fmt.Errorf("unhashable: sourceresolver.SourceOp")}
func (v *SourceOpValue) Attr(name string)(starlark.Value,error){
	if v==nil||v.op==nil{return nil,fmt.Errorf("sourceresolver.SourceOp: nil receiver")}
	switch name {
	case "identifier","Identifier":return starlark.String(v.op.Identifier),nil
	case "attrs","Attrs":
		d:=starlark.NewDict(len(v.op.Attrs))
		for k,val:=range v.op.Attrs { _=d.SetKey(starlark.String(k),starlark.String(val)) }
		return d,nil
	default:return nil,nil
	}
}
func (v *SourceOpValue) AttrNames()[]string{return []string{"identifier","attrs","Identifier","Attrs"}}
func (v *SourceOpValue) Unwrap()*pb.SourceOp { if v==nil{return nil};return v.op }

func optBuiltin(_ *starlark.Thread,_ *starlark.Builtin,args starlark.Tuple,kwargs []starlark.Tuple)(starlark.Value,error){
	if len(args)!=0{return nil,fmt.Errorf("Opt: use named arguments")}
	var opt buildkitsourceresolver.Opt
	for _,kw:=range kwargs{
		key,ok:=kw[0].(starlark.String);if !ok{return nil,fmt.Errorf("Opt: field names must be strings")}
		switch string(key){
		case "log_name":
			value,err:=stringArg("Opt.log_name",kw[1]);if err!=nil{return nil,err};opt.LogName=value
		case "image":
			if kw[1]==starlark.None {opt.ImageOpt=nil;continue}
			value,err:=recordFrom[buildkitsourceresolver.ResolveImageOpt](kw[1],"ResolveImageOpt");if err!=nil{return nil,err};opt.ImageOpt=&value
		case "git":
			if kw[1]==starlark.None {opt.GitOpt=nil;continue}
			value,err:=recordFrom[buildkitsourceresolver.ResolveGitOpt](kw[1],"ResolveGitOpt");if err!=nil{return nil,err};opt.GitOpt=&value
		case "http":
			if kw[1]==starlark.None {opt.HTTPOpt=nil;continue}
			value,err:=recordFrom[buildkitsourceresolver.ResolveHTTPOpt](kw[1],"ResolveHTTPOpt");if err!=nil{return nil,err};opt.HTTPOpt=&value
		case "oci_layout":
			if kw[1]==starlark.None {opt.OCILayoutOpt=nil;continue}
			value,err:=recordFrom[buildkitsourceresolver.ResolveOCILayoutOpt](kw[1],"ResolveOCILayoutOpt");if err!=nil{return nil,err};opt.OCILayoutOpt=&value
		case "source_policies":
			list,ok:=kw[1].(starlark.Indexable);if !ok{return nil,fmt.Errorf("Opt.source_policies: got %s, want list",starlarkType(kw[1]))}
			opt.SourcePolicies=make([]*sourcepolicy.Policy,list.Len())
			for i:=0;i<list.Len();i++{
				data,err:=bytesArg("Opt.source_policies",list.Index(i));if err!=nil{return nil,err}
				policy:=&sourcepolicy.Policy{}
				if err:=proto.Unmarshal(data,policy);err!=nil{return nil,fmt.Errorf("Opt.source_policies[%d]: %w",i,err)}
				opt.SourcePolicies[i]=policy
			}
		default:return nil,fmt.Errorf("Opt: unknown field %q",key)
		}
	}
	return starlarkllb.WrapSourceResolverOpt(opt),nil
}

func resolveImageOptBuiltin(_ *starlark.Thread,_ *starlark.Builtin,args starlark.Tuple,kwargs []starlark.Tuple)(starlark.Value,error){
	var platformValue starlark.Value=starlark.None
	var resolveMode string
	var noConfig,attestationChain bool
	var attestations []string
	if err:=starlark.UnpackArgs("ResolveImageOpt",args,kwargs,
		"platform?",&platformValue,"resolve_mode?",&resolveMode,"no_config?",&noConfig,
		"attestation_chain?",&attestationChain,"resolve_attestations?",&attestations);err!=nil{return nil,err}
	platform,err:=platformArg("ResolveImageOpt.platform",platformValue);if err!=nil{return nil,err}
	opt:=buildkitsourceresolver.ResolveImageOpt{Platform:platform,ResolveMode:resolveMode,NoConfig:noConfig,AttestationChain:attestationChain,ResolveAttestations:attestations}
	return newRecord("ResolveImageOpt",opt,starlark.StringDict{
		"platform":platformValue,"resolve_mode":starlark.String(resolveMode),"no_config":boolField(noConfig),
		"attestation_chain":boolField(attestationChain),"resolve_attestations":stringList(attestations),
	}),nil
}
func resolveGitOptBuiltin(_ *starlark.Thread,_ *starlark.Builtin,args starlark.Tuple,kwargs []starlark.Tuple)(starlark.Value,error){
	var returnObject bool
	if err:=starlark.UnpackArgs("ResolveGitOpt",args,kwargs,"return_object?",&returnObject);err!=nil{return nil,err}
	opt:=buildkitsourceresolver.ResolveGitOpt{ReturnObject:returnObject}
	return newRecord("ResolveGitOpt",opt,starlark.StringDict{"return_object":boolField(returnObject)}),nil
}
func resolveHTTPOptBuiltin(_ *starlark.Thread,_ *starlark.Builtin,args starlark.Tuple,kwargs []starlark.Tuple)(starlark.Value,error){
	var requestValue starlark.Value=starlark.None
	if err:=starlark.UnpackArgs("ResolveHTTPOpt",args,kwargs,"checksum_req?",&requestValue);err!=nil{return nil,err}
	var request *buildkitsourceresolver.ResolveHTTPChecksumRequest
	if requestValue!=starlark.None{
		v,err:=recordFrom[buildkitsourceresolver.ResolveHTTPChecksumRequest](requestValue,"ResolveHTTPChecksumRequest");if err!=nil{return nil,err}
		request=&v
	}
	opt:=buildkitsourceresolver.ResolveHTTPOpt{ChecksumReq:request}
	fields:=starlark.StringDict{"checksum_req":requestValue}
	return newRecord("ResolveHTTPOpt",opt,fields),nil
}
func resolveOCILayoutOptBuiltin(_ *starlark.Thread,_ *starlark.Builtin,args starlark.Tuple,kwargs []starlark.Tuple)(starlark.Value,error){
	var platformValue starlark.Value=starlark.None
	var storeValue starlark.Value=starlark.None
	if err:=starlark.UnpackArgs("ResolveOCILayoutOpt",args,kwargs,"platform?",&platformValue,"store?",&storeValue);err!=nil{return nil,err}
	platform,err:=platformArg("ResolveOCILayoutOpt.platform",platformValue);if err!=nil{return nil,err}
	var store buildkitsourceresolver.ResolveImageConfigOptStore
	if storeValue!=starlark.None{
		value,err:=recordFrom[buildkitsourceresolver.ResolveImageConfigOptStore](storeValue,"ResolveImageConfigOptStore");if err!=nil{return nil,err}
		store=value
	}
	opt:=buildkitsourceresolver.ResolveOCILayoutOpt{Platform:platform,Store:store}
	return newRecord("ResolveOCILayoutOpt",opt,starlark.StringDict{"platform":platformValue,"store":storeValue}),nil
}
func resolveImageConfigStoreBuiltin(_ *starlark.Thread,_ *starlark.Builtin,args starlark.Tuple,kwargs []starlark.Tuple)(starlark.Value,error){
	var sessionID,storeID string
	if err:=starlark.UnpackArgs("ResolveImageConfigOptStore",args,kwargs,"session_id?",&sessionID,"store_id?",&storeID);err!=nil{return nil,err}
	store:=buildkitsourceresolver.ResolveImageConfigOptStore{SessionID:sessionID,StoreID:storeID}
	return newRecord("ResolveImageConfigOptStore",store,starlark.StringDict{"session_id":stringField("session_id",sessionID),"store_id":stringField("store_id",storeID)}),nil
}
func resolveHTTPChecksumRequestBuiltin(_ *starlark.Thread,_ *starlark.Builtin,args starlark.Tuple,kwargs []starlark.Tuple)(starlark.Value,error){
	var algo int
	var suffix []byte
	if err:=starlark.UnpackArgs("ResolveHTTPChecksumRequest",args,kwargs,"algo?",&algo,"suffix?",&suffix);err!=nil{return nil,err}
	request:=buildkitsourceresolver.ResolveHTTPChecksumRequest{Algo:buildkitsourceresolver.ResolveHTTPChecksumAlgo(algo),Suffix:suffix}
	return newRecord("ResolveHTTPChecksumRequest",request,starlark.StringDict{"algo":intField(algo),"suffix":bytesField(suffix)}),nil
}
func resolverTypeValues()map[string]starlark.Value{
	return map[string]starlark.Value{
		"ResolverTypeRegistry":starlark.MakeInt(int(buildkitsourceresolver.ResolverTypeRegistry)),
		"ResolverTypeOCILayout":starlark.MakeInt(int(buildkitsourceresolver.ResolverTypeOCILayout)),
	}
}
func checksumAlgoValues()map[string]starlark.Value{
	return map[string]starlark.Value{
		"ResolveHTTPChecksumAlgoSHA256":starlark.MakeInt(int(buildkitsourceresolver.ResolveHTTPChecksumAlgoSHA256)),
		"ResolveHTTPChecksumAlgoSHA384":starlark.MakeInt(int(buildkitsourceresolver.ResolveHTTPChecksumAlgoSHA384)),
		"ResolveHTTPChecksumAlgoSHA512":starlark.MakeInt(int(buildkitsourceresolver.ResolveHTTPChecksumAlgoSHA512)),
	}
}
func sourceResolverTypesBuiltins()map[string]starlark.Value{
	return map[string]starlark.Value{
		"SourceOp":builtin("SourceOp",sourceOpBuiltin),
		"Opt":builtin("Opt",optBuiltin),
		"ResolveImageOpt":builtin("ResolveImageOpt",resolveImageOptBuiltin),
		"ResolveGitOpt":builtin("ResolveGitOpt",resolveGitOptBuiltin),
		"ResolveHTTPOpt":builtin("ResolveHTTPOpt",resolveHTTPOptBuiltin),
		"ResolveOCILayoutOpt":builtin("ResolveOCILayoutOpt",resolveOCILayoutOptBuiltin),
		"ResolveImageConfigOptStore":builtin("ResolveImageConfigOptStore",resolveImageConfigStoreBuiltin),
		"ResolveHTTPChecksumRequest":builtin("ResolveHTTPChecksumRequest",resolveHTTPChecksumRequestBuiltin),
	}
}

func newModule() *starlarkstruct.Module {
	members:=builtins()
	for name,value:=range resolverTypeValues(){members[name]=value}
	for name,value:=range checksumAlgoValues(){members[name]=value}
	module:=&starlarkstruct.Module{Name:"sourceresolver",Members:members}
	module.Freeze()
	return module
}

// Module is the frozen Starlark module for BuildKit's sourceresolver API.
var Module=newModule()

// Load resolves the sourceresolver submodule.
func Load(_ *starlark.Thread,name string)(starlark.StringDict,error){
	switch name {
	case "llb/sourceresolver","llb/sourceresolver.star","sourceresolver","sourceresolver.star":
		return Module.Members,nil
	default:
		return nil,fmt.Errorf("sourceresolver: unknown module %q",name)
	}
}

func init(){
	if err:=starlarkllb.RegisterModule("llb/sourceresolver",Module.Members);err!=nil{
		panic(err)
	}
}
