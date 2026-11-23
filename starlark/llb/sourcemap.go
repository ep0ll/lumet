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

func (v *PositionValue) String() string { return "<llb.source_position>" }
func (v *PositionValue) Type() string { return "llb.source_position" }
func (v *PositionValue) Freeze() {}
func (v *PositionValue) Truth() starlark.Bool { return starlark.Bool(v != nil && v.position != nil) }
func (v *PositionValue) Hash() (uint32,error) { return 0,fmt.Errorf("unhashable: llb.source_position") }
func (v *PositionValue) Attr(name string)(starlark.Value,error){
		if v==nil||v.position==nil{return nil,fmt.Errorf("llb.source_position: nil receiver")}
	switch name{
	case "line":return starlark.MakeInt(int(v.position.Line)),nil
	case "character":return starlark.MakeInt(int(v.position.Character)),nil
	default:return nil,nil
	}
}
func (v *PositionValue) AttrNames()[]string{return SnakeCaseNames([]string{"line","character","line","character"})}

// RangeValue wraps a protobuf source range.
type RangeValue struct { rangeValue *pb.Range }

// WrapRange adapts a native source range.
func WrapRange(r *pb.Range)*RangeValue{return &RangeValue{rangeValue:r}}
func (v *RangeValue) String()string{return "<llb.source_range>"}
func (v *RangeValue) Type()string{return "llb.source_range"}
func (v *RangeValue) Freeze(){}
func (v *RangeValue) Truth()starlark.Bool{return starlark.Bool(v!=nil&&v.rangeValue!=nil)}
func (v *RangeValue) Hash()(uint32,error){return 0,fmt.Errorf("unhashable: llb.source_range")}
func (v *RangeValue) Attr(name string)(starlark.Value,error){
		if v==nil||v.rangeValue==nil{return nil,fmt.Errorf("llb.source_range: nil receiver")}
	switch name{
	case "start":return WrapPosition(v.rangeValue.Start),nil
	case "end":return WrapPosition(v.rangeValue.End),nil
	default:return nil,nil
	}
}
func (v *RangeValue) AttrNames()[]string{return SnakeCaseNames([]string{"start","end","start","end"})}

// SourceMapValue wraps BuildKit's source-file mapping value.
type SourceMapValue struct { sourceMap *buildkitllb.SourceMap }

// WrapSourceMap adapts a native BuildKit SourceMap.
func WrapSourceMap(sourceMap *buildkitllb.SourceMap)*SourceMapValue{return &SourceMapValue{sourceMap:sourceMap}}

// Unwrap returns the native SourceMap.
func (v *SourceMapValue) Unwrap()*buildkitllb.SourceMap{if v==nil{return nil};return v.sourceMap}
func (v *SourceMapValue) String()string{return "<llb.source_map>"}
func (v *SourceMapValue) Type()string{return "llb.source_map"}
func (v *SourceMapValue) Freeze(){}
func (v *SourceMapValue) Truth()starlark.Bool{return starlark.Bool(v!=nil&&v.sourceMap!=nil)}
func (v *SourceMapValue) Hash()(uint32,error){return 0,fmt.Errorf("unhashable: llb.source_map")}
func (v *SourceMapValue) Attr(name string)(starlark.Value,error){
		if v==nil||v.sourceMap==nil{return nil,fmt.Errorf("llb.source_map: nil receiver")}
	switch name{
	case "state":
		if v.sourceMap.State==nil{return starlark.None,nil}
		return WrapState(*v.sourceMap.State),nil
	case "definition":
		if v.sourceMap.Definition==nil{return starlark.None,nil}
		return WrapDefinition(v.sourceMap.Definition),nil
	case "filename":return starlark.String(v.sourceMap.Filename),nil
	case "language":return starlark.String(v.sourceMap.Language),nil
	case "data":return starlark.Bytes(append([]byte(nil),v.sourceMap.Data...)),nil
	case "location":
		return boundBuiltin("source_map.location",func(_ *starlark.Thread,args starlark.Tuple,kwargs []starlark.Tuple)(starlark.Value,error){
			if err:=arity("source_map.location",args,kwargs,1,1);err!=nil{return nil,err}
			ranges,err:=rangesFromValue(args[0]);if err!=nil{return nil,err}
			return wrapOption("constraints_opt",v.sourceMap.Location(ranges)),nil
		}),nil
	default:return nil,nil
	}
}
func (v *SourceMapValue) AttrNames()[]string{
	return SnakeCaseNames([]string{"state","definition","filename","language","data","location","location","state","definition","filename","language","data"})
}

// SourceLocationValue wraps a source map and its ranges on one LLB vertex.
type SourceLocationValue struct { location *buildkitllb.SourceLocation }

// WrapSourceLocation adapts a native SourceLocation.
func WrapSourceLocation(location *buildkitllb.SourceLocation)*SourceLocationValue{return &SourceLocationValue{location:location}}

// Unwrap returns the native source location.
func (v *SourceLocationValue) Unwrap()*buildkitllb.SourceLocation{if v==nil{return nil};return v.location}
func (v *SourceLocationValue) String()string{return "<llb.source_location>"}
func (v *SourceLocationValue) Type()string{return "llb.source_location"}
func (v *SourceLocationValue) Freeze(){}
func (v *SourceLocationValue) Truth()starlark.Bool{return starlark.Bool(v!=nil&&v.location!=nil)}
func (v *SourceLocationValue) Hash()(uint32,error){return 0,fmt.Errorf("unhashable: llb.source_location")}
func (v *SourceLocationValue) Attr(name string)(starlark.Value,error){
		if v==nil||v.location==nil{return nil,fmt.Errorf("llb.source_location: nil receiver")}
	switch name{
	case "source_map":return WrapSourceMap(v.location.SourceMap),nil
	case "ranges":
		values:=make([]starlark.Value,len(v.location.Ranges))
		for i,r:=range v.location.Ranges{values[i]=WrapRange(r)}
		return starlark.NewList(values),nil
	default:return nil,nil
	}
}
func (v *SourceLocationValue) AttrNames()[]string{return SnakeCaseNames([]string{"source_map","ranges","source_map","ranges"})}

// SourceValue wraps the source metadata stored in a Definition.
type SourceValue struct { source *pb.Source }

// WrapSource adapts a protobuf source metadata object.
func WrapSource(source *pb.Source)*SourceValue{return &SourceValue{source:source}}
func (v *SourceValue) String()string{return "<llb.source>"}
func (v *SourceValue) Type()string{return "llb.source"}
func (v *SourceValue) Freeze(){}
func (v *SourceValue) Truth()starlark.Bool{return starlark.Bool(v!=nil&&v.source!=nil)}
func (v *SourceValue) Hash()(uint32,error){return 0,fmt.Errorf("unhashable: llb.source")}
func (v *SourceValue) Attr(name string)(starlark.Value,error){
		if v==nil||v.source==nil{return nil,fmt.Errorf("llb.source: nil receiver")}
	switch name{
	case "infos":
		values:=make([]starlark.Value,len(v.source.Infos))
		for i,info:=range v.source.Infos{
			if info==nil{values[i]=starlark.None;continue}
			values[i]=starlarkstructValue("llb.source_info",starlark.StringDict{
				"filename":starlark.String(info.Filename),"language":starlark.String(info.Language),
				"data":starlark.Bytes(info.Data),
			})
		}
		return starlark.NewList(values),nil
	case "locations":
		dict:=starlark.NewDict(len(v.source.Locations))
		for key,locations:=range v.source.Locations{
			if locations==nil{_ = dict.SetKey(starlark.String(key),starlark.None);continue}
			_ = dict.SetKey(starlark.String(key),starlark.MakeInt(len(locations.Locations)))
		}
		return dict,nil
	default:return nil,nil
	}
}
func (v *SourceValue) AttrNames()[]string{return SnakeCaseNames([]string{"infos","locations","infos","locations"})}

func sourceMapBuiltins()map[string]starlark.Value{
	return map[string]starlark.Value{
		"new_source_map":builtin("new_source_map",newSourceMapBuiltin),
		"source_location":builtin("source_location",sourceLocationBuiltin),
		"position":builtin("position",positionBuiltin),
		"range":builtin("range",rangeBuiltin),
	}
}

func newSourceMapBuiltin(_ *starlark.Thread, _ *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	var stateValue, definitionValue starlark.Value = starlark.None, starlark.None
	var filename, language string
	var dataValue starlark.Value = starlark.Bytes(nil)
	if err := UnpackArgs("new_source_map", args, kwargs,
		"filename", &filename,
		"language?", &language,
		"data?", &dataValue,
		"state?", &stateValue,
		"definition?", &definitionValue,
	); err != nil {
		return nil, err
	}
	data, err := bytesAt("new_source_map", starlark.Tuple{dataValue}, 0)
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
			return nil, fmt.Errorf("new_source_map.definition: got %s, want llb.definition", valueType(definitionValue))
		}
		sourceMap.Definition = definition.definition
	}
	if sourceMap.State == nil && sourceMap.Definition == nil {
		return nil, fmt.Errorf("new_source_map: state or definition is required")
	}
	return WrapSourceMap(sourceMap), nil
}

func sourceLocationBuiltin(_ *starlark.Thread,_ *starlark.Builtin,args starlark.Tuple,kwargs []starlark.Tuple)(starlark.Value,error){
	var sourceMapValue starlark.Value
	var rangeValues starlark.Value=starlark.NewList(nil)
	if err:=UnpackArgs("source_location",args,kwargs,"source_map",&sourceMapValue,"ranges?",&rangeValues);err!=nil{return nil,err}
	sourceMap,ok:=sourceMapValue.(*SourceMapValue)
	if !ok||sourceMap==nil||sourceMap.sourceMap==nil{return nil,fmt.Errorf("source_location.source_map: got %s, want llb.source_map",valueType(sourceMapValue))}
	ranges,err:=rangesFromValue(rangeValues);if err!=nil{return nil,err}
	return WrapSourceLocation(&buildkitllb.SourceLocation{SourceMap:sourceMap.sourceMap,Ranges:ranges}),nil
}
func positionBuiltin(_ *starlark.Thread,_ *starlark.Builtin,args starlark.Tuple,kwargs []starlark.Tuple)(starlark.Value,error){
	var line,character int
	if err:=UnpackArgs("position",args,kwargs,"line",&line,"character?",&character);err!=nil{return nil,err}
	return WrapPosition(&pb.Position{Line:int32(line),Character:int32(character)}),nil
}
func rangeBuiltin(_ *starlark.Thread,_ *starlark.Builtin,args starlark.Tuple,kwargs []starlark.Tuple)(starlark.Value,error){
	var startValue,endValue starlark.Value
	if err:=UnpackArgs("range",args,kwargs,"start",&startValue,"end",&endValue);err!=nil{return nil,err}
	start,ok:=startValue.(*PositionValue);if !ok||start==nil{return nil,fmt.Errorf("range.start: got %s, want llb.source_position",valueType(startValue))}
	end,ok:=endValue.(*PositionValue);if !ok||end==nil{return nil,fmt.Errorf("range.end: got %s, want llb.source_position",valueType(endValue))}
	return WrapRange(&pb.Range{Start:start.position,End:end.position}),nil
}
func rangesFromValue(value starlark.Value)([]*pb.Range,error){
	list,ok:=value.(starlark.Indexable);if !ok{return nil,fmt.Errorf("ranges: got %s, want list or tuple",valueType(value))}
	result:=make([]*pb.Range,list.Len())
	for i:=range result{
		item,ok:=list.Index(i).(*RangeValue)
		if !ok||item==nil||item.rangeValue==nil{return nil,fmt.Errorf("ranges[%d]: got %s, want llb.source_range",i,valueType(list.Index(i)))}
		result[i]=item.rangeValue
	}
	return result,nil
}
func sourceLocationsFromValue(value starlark.Value)([]*buildkitllb.SourceLocation,error){
	list,ok:=value.(starlark.Indexable);if !ok{return nil,fmt.Errorf("source locations: got %s, want list or tuple",valueType(value))}
	result:=make([]*buildkitllb.SourceLocation,list.Len())
	for i:=range result{
		item,ok:=list.Index(i).(*SourceLocationValue)
		if !ok||item==nil||item.location==nil{return nil,fmt.Errorf("source locations[%d]: got %s, want llb.source_location",i,valueType(list.Index(i)))}
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
