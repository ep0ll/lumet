// Package sourceresolver exposes BuildKit's client/llb/sourceresolver
// request and response types to Starlark.
package sourceresolver

import (
	"fmt"
	"sort"
	starlarkllb "github.com/ep0ll/lumet/starlark/llb"
	buildkitsourceresolver "github.com/moby/buildkit/client/llb/sourceresolver"
	"github.com/moby/buildkit/solver/pb"
	sourcepolicy "github.com/moby/buildkit/sourcepolicy/pb"
	digest "github.com/opencontainers/go-digest"
	ocispecs "github.com/opencontainers/image-spec/specs-go/v1"
	"go.starlark.net/starlark"
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
	normalized, err := starlarkllb.NormalizeMembers(fields)
	if err != nil {
		panic(err)
	}
	return &recordValue{name: name, native: native, fields: normalized}
}
func (v *recordValue) String() string { return "<sourceresolver." + starlarkllb.SnakeCaseIdentifier(v.name) + ">" }
func (v *recordValue) Type() string { return "sourceresolver." + starlarkllb.SnakeCaseIdentifier(v.name) }
func (v *recordValue) Freeze() { v.fields.Freeze() }
func (v *recordValue) Truth() starlark.Bool { return starlark.Bool(v != nil) }
func (v *recordValue) Hash() (uint32,error) { return 0, fmt.Errorf("unhashable: sourceresolver.%s", v.name) }
func (v *recordValue) Attr(name string) (starlark.Value,error) {
	if v == nil { return nil, fmt.Errorf("sourceresolver: nil %s", name) }
	if value, ok := v.fields[name]; ok { return value,nil }
	return nil,nil
}
func (v *recordValue) AttrNames() []string {
	names := make([]string, 0, len(v.fields))
	for name := range v.fields {
		names = append(names, name)
	}
	sort.Strings(names)
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
	if !ok || v==nil { return nil,fmt.Errorf("%s: got %s, want llb.oci_platform or none",name,starlarkType(value)) }
	p:=v.Unwrap()
	return &p,nil
}

func sourceOpBuiltin(_ *starlark.Thread,_ *starlark.Builtin,args starlark.Tuple,kwargs []starlark.Tuple)(starlark.Value,error) {
	var identifier string
	var attrsValue starlark.Value = starlark.None
	if err:=starlarkllb.UnpackArgs("source_op",args,kwargs,"identifier",&identifier,"attrs?",&attrsValue); err!=nil { return nil,err }
	attrs:=map[string]string{}
	if attrsValue!=starlark.None {
		dict,ok:=attrsValue.(*starlark.Dict)
		if !ok { return nil,fmt.Errorf("source_op.attrs: got %s, want dict",starlarkType(attrsValue)) }
		for _,item:=range dict.Items() {
			key,err:=stringArg("source_op.attrs key",item[0]); if err!=nil{return nil,err}
			value,err:=stringArg("source_op.attrs value",item[1]); if err!=nil{return nil,err}
			attrs[key]=value
		}
	}
	return &SourceOpValue{op:&pb.SourceOp{Identifier:identifier,Attrs:attrs}},nil
}

// SourceOpValue wraps solver/pb.SourceOp, not client/llb.SourceOp.
type SourceOpValue struct { op *pb.SourceOp }
func (v *SourceOpValue) String() string { return "<sourceresolver.source_op>" }
func (v *SourceOpValue) Type() string { return "sourceresolver.source_op" }
func (v *SourceOpValue) Freeze() {}
func (v *SourceOpValue) Truth() starlark.Bool { return starlark.Bool(v!=nil && v.op!=nil) }
func (v *SourceOpValue) Hash()(uint32,error){return 0,fmt.Errorf("unhashable: sourceresolver.source_op")}
func (v *SourceOpValue) Attr(name string)(starlark.Value,error){
	if v==nil||v.op==nil{return nil,fmt.Errorf("sourceresolver.source_op: nil receiver")}
	switch name {
	case "identifier":return starlark.String(v.op.Identifier),nil
	case "attrs":
		d:=starlark.NewDict(len(v.op.Attrs))
		for k,val:=range v.op.Attrs { _=d.SetKey(starlark.String(k),starlark.String(val)) }
		return d,nil
	default:return nil,nil
	}
}
func (v *SourceOpValue) AttrNames()[]string{return []string{"identifier", "attrs"}}
func (v *SourceOpValue) Unwrap()*pb.SourceOp { if v==nil{return nil};return v.op }

func optBuiltin(_ *starlark.Thread,_ *starlark.Builtin,args starlark.Tuple,kwargs []starlark.Tuple)(starlark.Value,error){
	if len(args)!=0{return nil,fmt.Errorf("opt: use named arguments")}
	var opt buildkitsourceresolver.Opt
	for _,kw:=range kwargs{
		key,ok:=kw[0].(starlark.String);if !ok{return nil,fmt.Errorf("opt: field names must be strings")}
		switch string(key){
		case "log_name":
			value,err:=stringArg("opt.log_name",kw[1]);if err!=nil{return nil,err};opt.LogName=value
		case "image":
			if kw[1]==starlark.None {opt.ImageOpt=nil;continue}
			value,err:=recordFrom[buildkitsourceresolver.ResolveImageOpt](kw[1],"resolve_image_opt");if err!=nil{return nil,err};opt.ImageOpt=&value
		case "git":
			if kw[1]==starlark.None {opt.GitOpt=nil;continue}
			value,err:=recordFrom[buildkitsourceresolver.ResolveGitOpt](kw[1],"resolve_git_opt");if err!=nil{return nil,err};opt.GitOpt=&value
		case "http":
			if kw[1]==starlark.None {opt.HTTPOpt=nil;continue}
			value,err:=recordFrom[buildkitsourceresolver.ResolveHTTPOpt](kw[1],"resolve_http_opt");if err!=nil{return nil,err};opt.HTTPOpt=&value
		case "oci_layout":
			if kw[1]==starlark.None {opt.OCILayoutOpt=nil;continue}
			value,err:=recordFrom[buildkitsourceresolver.ResolveOCILayoutOpt](kw[1],"resolve_oci_layout_opt");if err!=nil{return nil,err};opt.OCILayoutOpt=&value
		case "source_policies":
			list,ok:=kw[1].(starlark.Indexable);if !ok{return nil,fmt.Errorf("opt.source_policies: got %s, want list",starlarkType(kw[1]))}
			opt.SourcePolicies=make([]*sourcepolicy.Policy,list.Len())
			for i:=0;i<list.Len();i++{
				data,err:=bytesArg("opt.source_policies",list.Index(i));if err!=nil{return nil,err}
				policy:=&sourcepolicy.Policy{}
				if err:=proto.Unmarshal(data,policy);err!=nil{return nil,fmt.Errorf("opt.source_policies[%d]: %w",i,err)}
				opt.SourcePolicies[i]=policy
			}
		default:return nil,fmt.Errorf("opt: unknown field %q",key)
		}
	}
	return starlarkllb.WrapSourceResolverOpt(opt),nil
}

func resolveImageOptBuiltin(_ *starlark.Thread,_ *starlark.Builtin,args starlark.Tuple,kwargs []starlark.Tuple)(starlark.Value,error){
	var platformValue starlark.Value=starlark.None
	var resolveMode string
	var noConfig,attestationChain bool
	var attestations []string
	if err:=starlarkllb.UnpackArgs("resolve_image_opt",args,kwargs,
		"platform?",&platformValue,"resolve_mode?",&resolveMode,"no_config?",&noConfig,
		"attestation_chain?",&attestationChain,"resolve_attestations?",&attestations);err!=nil{return nil,err}
	platform,err:=platformArg("resolve_image_opt.platform",platformValue);if err!=nil{return nil,err}
	opt:=buildkitsourceresolver.ResolveImageOpt{Platform:platform,ResolveMode:resolveMode,NoConfig:noConfig,AttestationChain:attestationChain,ResolveAttestations:attestations}
	return newRecord("resolve_image_opt",opt,starlark.StringDict{
		"platform":platformValue,"resolve_mode":starlark.String(resolveMode),"no_config":boolField(noConfig),
		"attestation_chain":boolField(attestationChain),"resolve_attestations":stringList(attestations),
	}),nil
}
func resolveGitOptBuiltin(_ *starlark.Thread,_ *starlark.Builtin,args starlark.Tuple,kwargs []starlark.Tuple)(starlark.Value,error){
	var returnObject bool
	if err:=starlarkllb.UnpackArgs("resolve_git_opt",args,kwargs,"return_object?",&returnObject);err!=nil{return nil,err}
	opt:=buildkitsourceresolver.ResolveGitOpt{ReturnObject:returnObject}
	return newRecord("resolve_git_opt",opt,starlark.StringDict{"return_object":boolField(returnObject)}),nil
}
func resolveHTTPOptBuiltin(_ *starlark.Thread,_ *starlark.Builtin,args starlark.Tuple,kwargs []starlark.Tuple)(starlark.Value,error){
	var requestValue starlark.Value=starlark.None
	if err:=starlarkllb.UnpackArgs("resolve_http_opt",args,kwargs,"checksum_req?",&requestValue);err!=nil{return nil,err}
	var request *buildkitsourceresolver.ResolveHTTPChecksumRequest
	if requestValue!=starlark.None{
		v,err:=recordFrom[buildkitsourceresolver.ResolveHTTPChecksumRequest](requestValue,"resolve_http_checksum_request");if err!=nil{return nil,err}
		request=&v
	}
	opt:=buildkitsourceresolver.ResolveHTTPOpt{ChecksumReq:request}
	fields:=starlark.StringDict{"checksum_req":requestValue}
	return newRecord("resolve_http_opt",opt,fields),nil
}
func resolveOCILayoutOptBuiltin(_ *starlark.Thread,_ *starlark.Builtin,args starlark.Tuple,kwargs []starlark.Tuple)(starlark.Value,error){
	var platformValue starlark.Value=starlark.None
	var storeValue starlark.Value=starlark.None
	if err:=starlarkllb.UnpackArgs("resolve_oci_layout_opt",args,kwargs,"platform?",&platformValue,"store?",&storeValue);err!=nil{return nil,err}
	platform,err:=platformArg("resolve_oci_layout_opt.platform",platformValue);if err!=nil{return nil,err}
	var store buildkitsourceresolver.ResolveImageConfigOptStore
	if storeValue!=starlark.None{
		value,err:=recordFrom[buildkitsourceresolver.ResolveImageConfigOptStore](storeValue,"resolve_image_config_opt_store");if err!=nil{return nil,err}
		store=value
	}
	opt:=buildkitsourceresolver.ResolveOCILayoutOpt{Platform:platform,Store:store}
	return newRecord("resolve_oci_layout_opt",opt,starlark.StringDict{"platform":platformValue,"store":storeValue}),nil
}
func resolveImageConfigStoreBuiltin(_ *starlark.Thread,_ *starlark.Builtin,args starlark.Tuple,kwargs []starlark.Tuple)(starlark.Value,error){
	var sessionID,storeID string
	if err:=starlarkllb.UnpackArgs("resolve_image_config_opt_store",args,kwargs,"session_id?",&sessionID,"store_id?",&storeID);err!=nil{return nil,err}
	store:=buildkitsourceresolver.ResolveImageConfigOptStore{SessionID:sessionID,StoreID:storeID}
	return newRecord("resolve_image_config_opt_store",store,starlark.StringDict{"session_id":stringField("session_id",sessionID),"store_id":stringField("store_id",storeID)}),nil
}
func resolveHTTPChecksumRequestBuiltin(_ *starlark.Thread, _ *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	var algo int
	var suffixValue starlark.Value = starlark.None
	if err := starlarkllb.UnpackArgs("resolve_http_checksum_request", args, kwargs,
		"algo?", &algo, "suffix?", &suffixValue,
	); err != nil {
		return nil, err
	}
	var suffix []byte
	if suffixValue != starlark.None {
		var err error
		suffix, err = bytesArg("resolve_http_checksum_request.suffix", suffixValue)
		if err != nil { return nil, err }
	}
	request := buildkitsourceresolver.ResolveHTTPChecksumRequest{
		Algo: buildkitsourceresolver.ResolveHTTPChecksumAlgo(algo),
		Suffix: suffix,
	}
	return newRecord("resolve_http_checksum_request", request, starlark.StringDict{
		"algo": intField(algo), "suffix": bytesField(suffix),
	}), nil
}

func resolverTypeValues()map[string]starlark.Value{
	return map[string]starlark.Value{
		"resolver_type_registry":starlark.MakeInt(int(buildkitsourceresolver.ResolverTypeRegistry)),
		"resolver_type_oci_layout":starlark.MakeInt(int(buildkitsourceresolver.ResolverTypeOCILayout)),
	}
}
func checksumAlgoValues()map[string]starlark.Value{
	return map[string]starlark.Value{
		"resolve_http_checksum_algo_sha256":starlark.MakeInt(int(buildkitsourceresolver.ResolveHTTPChecksumAlgoSHA256)),
		"resolve_http_checksum_algo_sha384":starlark.MakeInt(int(buildkitsourceresolver.ResolveHTTPChecksumAlgoSHA384)),
		"resolve_http_checksum_algo_sha512":starlark.MakeInt(int(buildkitsourceresolver.ResolveHTTPChecksumAlgoSHA512)),
	}
}
func sourceResolverTypesBuiltins()map[string]starlark.Value{
	return map[string]starlark.Value{
		"source_op":builtin("source_op",sourceOpBuiltin),
		"opt":builtin("opt",optBuiltin),
		"resolve_image_opt":builtin("resolve_image_opt",resolveImageOptBuiltin),
		"resolve_git_opt":builtin("resolve_git_opt",resolveGitOptBuiltin),
		"resolve_http_opt":builtin("resolve_http_opt",resolveHTTPOptBuiltin),
		"resolve_oci_layout_opt":builtin("resolve_oci_layout_opt",resolveOCILayoutOptBuiltin),
		"resolve_image_config_opt_store":builtin("resolve_image_config_opt_store",resolveImageConfigStoreBuiltin),
		"resolve_http_checksum_request":builtin("resolve_http_checksum_request",resolveHTTPChecksumRequestBuiltin),
	}
}

// Members returns the Starlark globals exported by BuildKit's
// sourceresolver package. Module construction and registration live in
// github.com/ep0ll/lumet/starlark/modules.
func Members() starlark.StringDict {
	members := builtins()
	for name, value := range resolverTypeValues() {
		members[name] = value
	}
	for name, value := range checksumAlgoValues() {
		members[name] = value
	}
	return starlark.StringDict(members)
}

// builtin is the package-local Starlark builtin constructor shared by the
// sourceresolver module files.
func builtin(name string, fn func(*starlark.Thread, *starlark.Builtin, starlark.Tuple, []starlark.Tuple) (starlark.Value, error)) *starlark.Builtin {
	return starlark.NewBuiltin(starlarkllb.SnakeCaseIdentifier(name), fn)
}


// WrapSourceOp adapts the solver protobuf source operation returned by
// gateway/client.MetaResponse. This is distinct from llb.SourceOpValue.
func WrapSourceOp(op *pb.SourceOp) *SourceOpValue {
	return &SourceOpValue{op: op}
}

// UnwrapOpt extracts the native BuildKit sourceresolver.Opt created by opt().
func UnwrapOpt(value starlark.Value) (buildkitsourceresolver.Opt, error) {
	option, ok := value.(*starlarkllb.SourceResolverOptValue)
	if !ok || option == nil {
		return buildkitsourceresolver.Opt{}, fmt.Errorf("got %s, want llb.source_resolver_opt", starlarkType(value))
	}
	return option.Unwrap(), nil
}
