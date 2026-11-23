package llb

import (
	"fmt"

	buildkitllb "github.com/moby/buildkit/client/llb"
	"github.com/moby/buildkit/solver/pb"
	"go.starlark.net/starlark"
)

// PositionValue wraps a solver/pb.Position used by source ranges.
type PositionValue struct { position *pb.Position }

// WrapPosition adapts a native source position.
func WrapPosition(position *pb.Position) *PositionValue { return &PositionValue{position: position} }

func (v *PositionValue) String() string { return "<llb.SourcePosition>" }
func (v *PositionValue) Type() string { return "llb.SourcePosition" }
func (v *PositionValue) Freeze() {}
func (v *PositionValue) Truth() starlark.Bool { return starlark.Bool(v != nil && v.position != nil) }
func (v *PositionValue) Hash() (uint32,error) { return 0,fmt.Errorf("unhashable: llb.SourcePosition") }
func (v *PositionValue) Attr(name string)(starlark.Value,error){
	if v==nil||v.position==nil{return nil,fmt.Errorf("llb.SourcePosition: nil receiver")}
	switch name{
	case "line","Line":return starlark.MakeInt(int(v.position.Line)),nil
	case "character","Character":return starlark.MakeInt(int(v.position.Character)),nil
	default:return nil,nil
	}
}
func (v *PositionValue) AttrNames()[]string{return []string{"line","character","Line","Character"}}

// RangeValue wraps a protobuf source range.
type RangeValue struct { rangeValue *pb.Range }

// WrapRange adapts a native source range.
func WrapRange(r *pb.Range)*RangeValue{return &RangeValue{rangeValue:r}}
func (v *RangeValue) String()string{return "<llb.SourceRange>"}
func (v *RangeValue) Type()string{return "llb.SourceRange"}
func (v *RangeValue) Freeze(){}
func (v *RangeValue) Truth()starlark.Bool{return starlark.Bool(v!=nil&&v.rangeValue!=nil)}
func (v *RangeValue) Hash()(uint32,error){return 0,fmt.Errorf("unhashable: llb.SourceRange")}
func (v *RangeValue) Attr(name string)(starlark.Value,error){
	if v==nil||v.rangeValue==nil{return nil,fmt.Errorf("llb.SourceRange: nil receiver")}
	switch name{
	case "start","Start":return WrapPosition(v.rangeValue.Start),nil
	case "end","End":return WrapPosition(v.rangeValue.End),nil
	default:return nil,nil
	}
}
func (v *RangeValue) AttrNames()[]string{return []string{"start","end","Start","End"}}

// SourceMapValue wraps BuildKit's source-file mapping value.
type SourceMapValue struct { sourceMap *buildkitllb.SourceMap }

// WrapSourceMap adapts a native BuildKit SourceMap.
func WrapSourceMap(sourceMap *buildkitllb.SourceMap)*SourceMapValue{return &SourceMapValue{sourceMap:sourceMap}}

// Unwrap returns the native SourceMap.
func (v *SourceMapValue) Unwrap()*buildkitllb.SourceMap{if v==nil{return nil};return v.sourceMap}
func (v *SourceMapValue) String()string{return "<llb.SourceMap>"}
func (v *SourceMapValue) Type()string{return "llb.SourceMap"}
func (v *SourceMapValue) Freeze(){}
func (v *SourceMapValue) Truth()starlark.Bool{return starlark.Bool(v!=nil&&v.sourceMap!=nil)}
func (v *SourceMapValue) Hash()(uint32,error){return 0,fmt.Errorf("unhashable: llb.SourceMap")}
func (v *SourceMapValue) Attr(name string)(starlark.Value,error){
	if v==nil||v.sourceMap==nil{return nil,fmt.Errorf("llb.SourceMap: nil receiver")}
	switch name{
	case "state","State":
		if v.sourceMap.State==nil{return starlark.None,nil}
		return WrapState(*v.sourceMap.State),nil
	case "definition","Definition":
		if v.sourceMap.Definition==nil{return starlark.None,nil}
		return WrapDefinition(v.sourceMap.Definition),nil
	case "filename","Filename":return starlark.String(v.sourceMap.Filename),nil
	case "language","Language":return starlark.String(v.sourceMap.Language),nil
	case "data","Data":return starlark.Bytes(append([]byte(nil),v.sourceMap.Data...)),nil
	case "Location","location":
		return boundBuiltin("SourceMap.Location",func(_ *starlark.Thread,args starlark.Tuple,kwargs []starlark.Tuple)(starlark.Value,error){
			if err:=arity("SourceMap.Location",args,kwargs,1,1);err!=nil{return nil,err}
			ranges,err:=rangesFromValue(args[0]);if err!=nil{return nil,err}
			return wrapOption("ConstraintsOpt",v.sourceMap.Location(ranges)),nil
		}),nil
	default:return nil,nil
	}
}
func (v *SourceMapValue) AttrNames()[]string{
	return []string{"state","definition","filename","language","data","Location","location","State","Definition","Filename","Language","Data"}
}

// SourceLocationValue wraps a source map and its ranges on one LLB vertex.
type SourceLocationValue struct { location *buildkitllb.SourceLocation }

// WrapSourceLocation adapts a native SourceLocation.
func WrapSourceLocation(location *buildkitllb.SourceLocation)*SourceLocationValue{return &SourceLocationValue{location:location}}

// Unwrap returns the native source location.
func (v *SourceLocationValue) Unwrap()*buildkitllb.SourceLocation{if v==nil{return nil};return v.location}
func (v *SourceLocationValue) String()string{return "<llb.SourceLocation>"}
func (v *SourceLocationValue) Type()string{return "llb.SourceLocation"}
func (v *SourceLocationValue) Freeze(){}
func (v *SourceLocationValue) Truth()starlark.Bool{return starlark.Bool(v!=nil&&v.location!=nil)}
func (v *SourceLocationValue) Hash()(uint32,error){return 0,fmt.Errorf("unhashable: llb.SourceLocation")}
func (v *SourceLocationValue) Attr(name string)(starlark.Value,error){
	if v==nil||v.location==nil{return nil,fmt.Errorf("llb.SourceLocation: nil receiver")}
	switch name{
	case "source_map","SourceMap":return WrapSourceMap(v.location.SourceMap),nil
	case "ranges","Ranges":
		values:=make([]starlark.Value,len(v.location.Ranges))
		for i,r:=range v.location.Ranges{values[i]=WrapRange(r)}
		return starlark.NewList(values),nil
	default:return nil,nil
	}
}
func (v *SourceLocationValue) AttrNames()[]string{return []string{"source_map","ranges","SourceMap","Ranges"}}

// SourceValue wraps the source metadata stored in a Definition.
type SourceValue struct { source *pb.Source }

// WrapSource adapts a protobuf source metadata object.
func WrapSource(source *pb.Source)*SourceValue{return &SourceValue{source:source}}
func (v *SourceValue) String()string{return "<llb.Source>"}
func (v *SourceValue) Type()string{return "llb.Source"}
func (v *SourceValue) Freeze(){}
func (v *SourceValue) Truth()starlark.Bool{return starlark.Bool(v!=nil&&v.source!=nil)}
func (v *SourceValue) Hash()(uint32,error){return 0,fmt.Errorf("unhashable: llb.Source")}
func (v *SourceValue) Attr(name string)(starlark.Value,error){
	if v==nil||v.source==nil{return nil,fmt.Errorf("llb.Source: nil receiver")}
	switch name{
	case "infos","Infos":
		values:=make([]starlark.Value,len(v.source.Infos))
		for i,info:=range v.source.Infos{
			if info==nil{values[i]=starlark.None;continue}
			values[i]=starlarkstructValue("llb.SourceInfo",starlark.StringDict{
				"filename":starlark.String(info.Filename),"language":starlark.String(info.Language),
				"data":starlark.Bytes(info.Data),
			})
		}
		return starlark.NewList(values),nil
	case "locations","Locations":
		dict:=starlark.NewDict(len(v.source.Locations))
		for key,locations:=range v.source.Locations{
			if locations==nil{_ = dict.SetKey(starlark.String(key),starlark.None);continue}
			_ = dict.SetKey(starlark.String(key),starlark.MakeInt(len(locations.Locations)))
		}
		return dict,nil
	default:return nil,nil
	}
}
func (v *SourceValue) AttrNames()[]string{return []string{"infos","locations","Infos","Locations"}}

func sourceMapBuiltins()map[string]starlark.Value{
	return map[string]starlark.Value{
		"NewSourceMap":builtin("NewSourceMap",newSourceMapBuiltin),
		"SourceLocation":builtin("SourceLocation",sourceLocationBuiltin),
		"Position":builtin("Position",positionBuiltin),
		"Range":builtin("Range",rangeBuiltin),
	}
}

func newSourceMapBuiltin(_ *starlark.Thread, _ *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	var stateValue, definitionValue starlark.Value = starlark.None, starlark.None
	var filename, language string
	var dataValue starlark.Value = starlark.Bytes(nil)
	if err := starlark.UnpackArgs("NewSourceMap", args, kwargs,
		"filename", &filename,
		"language?", &language,
		"data?", &dataValue,
		"state?", &stateValue,
		"definition?", &definitionValue,
	); err != nil {
		return nil, err
	}
	data, err := bytesAt("NewSourceMap", starlark.Tuple{dataValue}, 0)
	if err != nil {
		return nil, err
	}
	sourceMap := &buildkitllb.SourceMap{Filename: filename, Language: language, Data: data}
	if stateValue != starlark.None {
		state, err := stateFromValue(stateValue)
		if err != nil {
			return nil, err
		}
		sourceMap.State = &state
	}
	if definitionValue != starlark.None {
		definition, ok := definitionValue.(*DefinitionValue)
		if !ok || definition == nil {
			return nil, fmt.Errorf("NewSourceMap.definition: got %s, want llb.Definition", valueType(definitionValue))
		}
		sourceMap.Definition = definition.definition
	}
	if sourceMap.State == nil && sourceMap.Definition == nil {
		return nil, fmt.Errorf("NewSourceMap: state or definition is required")
	}
	return WrapSourceMap(sourceMap), nil
}

func sourceLocationBuiltin(_ *starlark.Thread,_ *starlark.Builtin,args starlark.Tuple,kwargs []starlark.Tuple)(starlark.Value,error){
	var sourceMapValue starlark.Value
	var rangeValues starlark.Value=starlark.NewList(nil)
	if err:=starlark.UnpackArgs("SourceLocation",args,kwargs,"source_map",&sourceMapValue,"ranges?",&rangeValues);err!=nil{return nil,err}
	sourceMap,ok:=sourceMapValue.(*SourceMapValue)
	if !ok||sourceMap==nil||sourceMap.sourceMap==nil{return nil,fmt.Errorf("SourceLocation.source_map: got %s, want llb.SourceMap",valueType(sourceMapValue))}
	ranges,err:=rangesFromValue(rangeValues);if err!=nil{return nil,err}
	return WrapSourceLocation(&buildkitllb.SourceLocation{SourceMap:sourceMap.sourceMap,Ranges:ranges}),nil
}
func positionBuiltin(_ *starlark.Thread,_ *starlark.Builtin,args starlark.Tuple,kwargs []starlark.Tuple)(starlark.Value,error){
	var line,character int
	if err:=starlark.UnpackArgs("Position",args,kwargs,"line",&line,"character?",&character);err!=nil{return nil,err}
	return WrapPosition(&pb.Position{Line:int32(line),Character:int32(character)}),nil
}
func rangeBuiltin(_ *starlark.Thread,_ *starlark.Builtin,args starlark.Tuple,kwargs []starlark.Tuple)(starlark.Value,error){
	var startValue,endValue starlark.Value
	if err:=starlark.UnpackArgs("Range",args,kwargs,"start",&startValue,"end",&endValue);err!=nil{return nil,err}
	start,ok:=startValue.(*PositionValue);if !ok||start==nil{return nil,fmt.Errorf("Range.start: got %s, want llb.SourcePosition",valueType(startValue))}
	end,ok:=endValue.(*PositionValue);if !ok||end==nil{return nil,fmt.Errorf("Range.end: got %s, want llb.SourcePosition",valueType(endValue))}
	return WrapRange(&pb.Range{Start:start.position,End:end.position}),nil
}
func rangesFromValue(value starlark.Value)([]*pb.Range,error){
	list,ok:=value.(starlark.Indexable);if !ok{return nil,fmt.Errorf("ranges: got %s, want list or tuple",valueType(value))}
	result:=make([]*pb.Range,list.Len())
	for i:=range result{
		item,ok:=list.Index(i).(*RangeValue)
		if !ok||item==nil||item.rangeValue==nil{return nil,fmt.Errorf("ranges[%d]: got %s, want llb.SourceRange",i,valueType(list.Index(i)))}
		result[i]=item.rangeValue
	}
	return result,nil
}
func sourceLocationsFromValue(value starlark.Value)([]*buildkitllb.SourceLocation,error){
	list,ok:=value.(starlark.Indexable);if !ok{return nil,fmt.Errorf("source locations: got %s, want list or tuple",valueType(value))}
	result:=make([]*buildkitllb.SourceLocation,list.Len())
	for i:=range result{
		item,ok:=list.Index(i).(*SourceLocationValue)
		if !ok||item==nil||item.location==nil{return nil,fmt.Errorf("source locations[%d]: got %s, want llb.SourceLocation",i,valueType(list.Index(i)))}
		result[i]=item.location
	}
	return result,nil
}

var(
	_ starlark.Value=(*PositionValue)(nil)
	_ starlark.HasAttrs=(*PositionValue)(nil)
	_ starlark.Value=(*RangeValue)(nil)
	_ starlark.HasAttrs=(*RangeValue)(nil)
	_ starlark.Value=(*SourceMapValue)(nil)
	_ starlark.HasAttrs=(*SourceMapValue)(nil)
	_ starlark.Value=(*SourceLocationValue)(nil)
	_ starlark.HasAttrs=(*SourceLocationValue)(nil)
	_ starlark.Value=(*SourceValue)(nil)
	_ starlark.HasAttrs=(*SourceValue)(nil)
)
