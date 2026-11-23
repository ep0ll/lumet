package client

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"sort"
	"reflect"

	gwclient "github.com/moby/buildkit/frontend/gateway/client"
	"github.com/moby/buildkit/solver/pb"
	sourcepolicy "github.com/moby/buildkit/sourcepolicy/pb"
	"github.com/tonistiigi/fsutil/types"
	starlarkllb "github.com/ep0ll/lumet/starlark/llb"
	"go.starlark.net/starlark"
	"go.starlark.net/starlarkstruct"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
)

type recordValue struct {
	name   string
	native any
	fields starlark.StringDict
}

func newRecord(name string, native any, fields starlark.StringDict) *recordValue {
	return &recordValue{name: name, native: native, fields: fields}
}

func (v *recordValue) String() string { return "<gateway.client." + v.name + ">" }
func (v *recordValue) Type() string { return "gateway.client." + v.name }
func (v *recordValue) Freeze() { v.fields.Freeze() }
func (v *recordValue) Truth() starlark.Bool { return starlark.Bool(v != nil) }
func (v *recordValue) Hash() (uint32, error) { return 0, fmt.Errorf("unhashable: %s", v.Type()) }
func (v *recordValue) Attr(name string) (starlark.Value, error) {
	if v == nil { return nil, fmt.Errorf("gateway.client: nil %s", name) }
	if value, ok := v.fields[name]; ok { return value, nil }
	return nil, nil
}
func (v *recordValue) AttrNames() []string {
	names := make([]string, 0, len(v.fields))
	for name := range v.fields { names = append(names, name) }
	sort.Strings(names)
	return names
}

func recordFrom[T any](value starlark.Value, name string) (T, error) {
	var zero T
	record, ok := value.(*recordValue)
	if !ok || record == nil { return zero, fmt.Errorf("got %s, want gateway.client.%s", valueType(value), name) }
	native, ok := record.native.(T)
	if !ok { return zero, fmt.Errorf("gateway.client.%s has invalid native value", name) }
	return native, nil
}

func valueType(value starlark.Value) string {
	if value == nil { return "nil" }
	return value.Type()
}

func stringAt(name string, value starlark.Value) (string, error) {
	text, ok := value.(starlark.String)
	if !ok { return "", fmt.Errorf("%s: got %s, want string", name, valueType(value)) }
	return string(text), nil
}

func boolAt(name string, value starlark.Value) (bool, error) {
	b, ok := value.(starlark.Bool)
	if !ok { return false, fmt.Errorf("%s: got %s, want bool", name, valueType(value)) }
	return bool(b), nil
}

func intAt(name string, value starlark.Value) (int, error) {
	n, ok := value.(starlark.Int)
	if !ok { return 0, fmt.Errorf("%s: got %s, want int", name, valueType(value)) }
	result, err := starlark.AsInt32(n)
	if err != nil { return 0, fmt.Errorf("%s: %w", name, err) }
	return int(result), nil
}

func bytesAt(name string, value starlark.Value) ([]byte, error) {
	switch v := value.(type) {
	case starlark.Bytes:
		return append([]byte(nil), []byte(v)...), nil
	case starlark.String:
		return []byte(string(v)), nil
	default:
		return nil, fmt.Errorf("%s: got %s, want bytes or string", name, valueType(value))
	}
}

func stringListAt(name string, value starlark.Value) ([]string, error) {
	list, ok := value.(starlark.Iterable)
	if !ok { return nil, fmt.Errorf("%s: got %s, want iterable of strings", name, valueType(value)) }
	iter := list.Iterate()
	defer iter.Done()
	var out []string
	var item starlark.Value
	for iter.Next(&item) {
		text, err := stringAt(name, item)
		if err != nil { return nil, err }
		out = append(out, text)
	}
	return out, nil
}

func stringMapAt(name string, value starlark.Value) (map[string]string, error) {
	if value == starlark.None { return nil, nil }
	dict, ok := value.(*starlark.Dict)
	if !ok { return nil, fmt.Errorf("%s: got %s, want dict", name, valueType(value)) }
	out := make(map[string]string, dict.Len())
	for _, item := range dict.Items() {
		key, err := stringAt(name+" key", item[0])
		if err != nil { return nil, err }
		val, err := stringAt(name+"["+key+"]", item[1])
		if err != nil { return nil, err }
		out[key] = val
	}
	return out, nil
}

func valueToJSON(value starlark.Value) (any, error) {
	switch v := value.(type) {
	case starlark.NoneType:
		return nil, nil
	case starlark.String:
		return string(v), nil
	case starlark.Bytes:
		return base64.StdEncoding.EncodeToString([]byte(v)), nil
	case starlark.Bool:
		return bool(v), nil
	case starlark.Int:
		if n, err := v.Int64(); err == nil { return n, nil }
		if n, err := v.Uint64(); err == nil { return n, nil }
		return nil, fmt.Errorf("integer is outside JSON range")
	case starlark.Float:
		return float64(v), nil
	case *starlark.Dict:
		out := make(map[string]any, v.Len())
		for _, item := range v.Items() {
			key, err := stringAt("protobuf field name", item[0])
			if err != nil { return nil, err }
			val, err := valueToJSON(item[1])
			if err != nil { return nil, err }
			out[key] = val
		}
		return out, nil
	case *starlark.List, starlark.Tuple:
		iter := starlark.Iterate(value)
		if iter == nil { return nil, fmt.Errorf("%s is not iterable", value.Type()) }
		defer iter.Done()
		var out []any
		var item starlark.Value
		for iter.Next(&item) {
			val, err := valueToJSON(item)
			if err != nil { return nil, err }
			out = append(out, val)
		}
		return out, nil
	case *recordValue:
		return valueToJSON(starlarkstruct.FromStringDict(starlark.String(v.name), v.fields))
	default:
		return nil, fmt.Errorf("cannot convert %s to a protobuf field value", valueType(value))
	}
}

func protoFromValue(value starlark.Value, message proto.Message, name string) error {
	if value == nil || value == starlark.None { return nil }
	if dict, ok := value.(*starlark.Dict); ok {
		object, err := valueToJSON(dict)
		if err != nil { return err }
		data, err := json.Marshal(object)
		if err != nil { return err }
		if err := (protojson.UnmarshalOptions{}).Unmarshal(data, message); err != nil {
			return fmt.Errorf("%s: %w", name, err)
		}
		return nil
	}
	if data, ok := value.(starlark.Bytes); ok {
		if err := proto.Unmarshal([]byte(data), message); err != nil { return fmt.Errorf("%s: %w", name, err) }
		return nil
	}
	return fmt.Errorf("%s: got %s, want dict or protobuf bytes", name, valueType(value))
}

func structValue(name string, fields starlark.StringDict) starlark.Value {
	return starlarkstruct.FromStringDict(starlark.String(name), fields)
}

func valueList(values []starlark.Value) *starlark.List { return starlark.NewList(values) }

func solveRequestBuiltin(_ *starlark.Thread, _ *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	var evaluate bool
	var definitionValue starlark.Value = starlark.None
	var frontend string
	var frontendOptValue starlark.Value = starlark.None
	var frontendInputsValue starlark.Value = starlark.None
	var cacheImportsValue starlark.Value = starlark.None
	var sourcePoliciesValue starlark.Value = starlark.None
	if err := starlark.UnpackArgs("solve_request", args, kwargs,
		"evaluate?", &evaluate, "definition?", &definitionValue, "frontend?", &frontend,
		"frontend_opt?", &frontendOptValue, "frontend_inputs?", &frontendInputsValue,
		"cache_imports?", &cacheImportsValue, "source_policies?", &sourcePoliciesValue,
	); err != nil { return nil, err }

	req := gwclient.SolveRequest{Evaluate: evaluate, Frontend: frontend}
	if definitionValue != starlark.None {
		switch value := definitionValue.(type) {
		case *starlarkllb.DefinitionValue:
			definition := value.Unwrap()
			if definition == nil { return nil, fmt.Errorf("solve_request.definition: nil llb.definition") }
			req.Definition = definition.ToPB()
		case *starlarkllb.DefinitionProtoValue:
			req.Definition = value.Unwrap()
		default:
			return nil, fmt.Errorf("solve_request.definition: got %s, want llb.definition or llb.definition_pb", valueType(definitionValue))
		}
	}
	var err error
	if frontendOptValue != starlark.None {
		req.FrontendOpt, err = stringMapAt("solve_request.frontend_opt", frontendOptValue)
		if err != nil { return nil, err }
	}
	if frontendInputsValue != starlark.None {
		dict, ok := frontendInputsValue.(*starlark.Dict)
		if !ok { return nil, fmt.Errorf("solve_request.frontend_inputs: got %s, want dict", valueType(frontendInputsValue)) }
		req.FrontendInputs = make(map[string]*pb.Definition, dict.Len())
		for _, item := range dict.Items() {
			key, err := stringAt("solve_request.frontend_inputs key", item[0])
			if err != nil { return nil, err }
			switch value := item[1].(type) {
			case *starlarkllb.DefinitionValue:
				if value.Unwrap() == nil { return nil, fmt.Errorf("solve_request.frontend_inputs[%q]: nil definition", key) }
				req.FrontendInputs[key] = value.Unwrap().ToPB()
			case *starlarkllb.DefinitionProtoValue:
				req.FrontendInputs[key] = value.Unwrap()
			default:
				return nil, fmt.Errorf("solve_request.frontend_inputs[%q]: got %s, want llb.definition or llb.definition_pb", key, valueType(item[1]))
			}
		}
	}
	if cacheImportsValue != starlark.None {
		list, ok := cacheImportsValue.(starlark.Iterable)
		if !ok { return nil, fmt.Errorf("solve_request.cache_imports: got %s, want iterable", valueType(cacheImportsValue)) }
		iter := list.Iterate()
		for {
			var item starlark.Value
			if !iter.Next(&item) { break }
			entry, err := cacheOptionsFromValue(item)
			if err != nil { iter.Done(); return nil, err }
			req.CacheImports = append(req.CacheImports, entry)
		}
		iter.Done()
	}
	if sourcePoliciesValue != starlark.None {
		list, ok := sourcePoliciesValue.(starlark.Iterable)
		if !ok { return nil, fmt.Errorf("solve_request.source_policies: got %s, want iterable", valueType(sourcePoliciesValue)) }
		iter := list.Iterate()
		for {
			var item starlark.Value
			if !iter.Next(&item) { break }
			policy := &sourcepolicy.Policy{}
			if err := protoFromValue(item, policy, "solve_request.source_policies"); err != nil { iter.Done(); return nil, err }
			req.SourcePolicies = append(req.SourcePolicies, policy)
		}
		iter.Done()
	}
	return solveRequestValue(req), nil
}

func solveRequestValue(req gwclient.SolveRequest) *recordValue {
	fields := starlark.StringDict{
		"evaluate": starlark.Bool(req.Evaluate),
		"definition": starlark.None,
		"frontend": starlark.String(req.Frontend),
		"frontend_opt": stringMapValue(req.FrontendOpt),
	}
	if req.Definition != nil { fields["definition"] = starlarkllb.WrapDefinitionProto(req.Definition) }
	return newRecord("solve_request", req, fields)
}

func cacheOptionsFromValue(value starlark.Value) (gwclient.CacheOptionsEntry, error) {
	switch item := value.(type) {
	case *recordValue:
		return recordFrom[gwclient.CacheOptionsEntry](item, "cache_options_entry")
	case *starlark.Dict:
		var entry gwclient.CacheOptionsEntry
		for _, pair := range item.Items() {
			key, err := stringAt("cache_options_entry key", pair[0]); if err != nil { return entry, err }
			switch key {
			case "type":
				entry.Type, err = stringAt("cache_options_entry.type", pair[1])
			case "attrs":
				entry.Attrs, err = stringMapAt("cache_options_entry.attrs", pair[1])
			default:
				return entry, fmt.Errorf("cache_options_entry: unknown field %q", key)
			}
			if err != nil { return entry, err }
		}
		return entry, nil
	default:
		return gwclient.CacheOptionsEntry{}, fmt.Errorf("cache_imports: got %s, want cache_options_entry or dict", valueType(value))
	}
}

func cacheOptionsEntryBuiltin(_ *starlark.Thread, _ *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	var typ string
	var attrsValue starlark.Value = starlark.None
	if err := starlark.UnpackArgs("cache_options_entry", args, kwargs, "type?", &typ, "attrs?", &attrsValue); err != nil { return nil, err }
	attrs, err := stringMapAt("cache_options_entry.attrs", attrsValue)
	if err != nil { return nil, err }
	entry := gwclient.CacheOptionsEntry{Type: typ, Attrs: attrs}
	return newRecord("cache_options_entry", entry, starlark.StringDict{"type": starlark.String(typ), "attrs": stringMapValue(attrs)}), nil
}

func mountBuiltin(_ *starlark.Thread, _ *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	var selector, dest, resultID string
	var refValue starlark.Value = starlark.None
	var readonly bool
	var mountType int
	var cacheOptValue, secretOptValue, sshOptValue starlark.Value = starlark.None, starlark.None, starlark.None
	if err := starlark.UnpackArgs("mount", args, kwargs,
		"selector?", &selector, "dest?", &dest, "result_id?", &resultID,
		"ref?", &refValue, "readonly?", &readonly, "mount_type?", &mountType,
		"cache_opt?", &cacheOptValue, "secret_opt?", &secretOptValue, "ssh_opt?", &sshOptValue,
	); err != nil { return nil, err }
	m := gwclient.Mount{Selector: selector, Dest: dest, ResultID: resultID, Readonly: readonly, MountType: pb.MountType(mountType)}
	if refValue != starlark.None {
		ref, ok := refValue.(*ReferenceValue)
		if !ok || ref == nil { return nil, fmt.Errorf("mount.ref: got %s, want gateway.client.reference", valueType(refValue)) }
		m.Ref = ref.ref
	}
	if cacheOptValue != starlark.None {
		value := &pb.CacheOpt{}
		if err := protoFromValue(cacheOptValue, value, "mount.cache_opt"); err != nil { return nil, err }
		m.CacheOpt = value
	}
	if secretOptValue != starlark.None {
		value := &pb.SecretOpt{}
		if err := protoFromValue(secretOptValue, value, "mount.secret_opt"); err != nil { return nil, err }
		m.SecretOpt = value
	}
	if sshOptValue != starlark.None {
		value := &pb.SSHOpt{}
		if err := protoFromValue(sshOptValue, value, "mount.ssh_opt"); err != nil { return nil, err }
		m.SSHOpt = value
	}
	return newRecord("mount", m, starlark.StringDict{
		"selector": starlark.String(selector), "dest": starlark.String(dest), "result_id": starlark.String(resultID),
		"ref": refValue, "readonly": starlark.Bool(readonly), "mount_type": starlark.MakeInt(mountType),
		"cache_opt": cacheOptValue, "secret_opt": secretOptValue, "ssh_opt": sshOptValue,
	}), nil
}

func newContainerRequestBuiltin(_ *starlark.Thread, _ *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	var mountsValue starlark.Value = starlark.None
	var hostname string
	var netMode int
	var extraHostsValue, platformValue, constraintsValue starlark.Value = starlark.None, starlark.None, starlark.None
	if err := starlark.UnpackArgs("new_container_request", args, kwargs,
		"mounts?", &mountsValue, "hostname?", &hostname, "net_mode?", &netMode,
		"extra_hosts?", &extraHostsValue, "platform?", &platformValue, "constraints?", &constraintsValue,
	); err != nil { return nil, err }
	req := gwclient.NewContainerRequest{Hostname: hostname, NetMode: pb.NetMode(netMode)}
	if mountsValue != starlark.None {
		iterable, ok := mountsValue.(starlark.Iterable)
		if !ok { return nil, fmt.Errorf("new_container_request.mounts: got %s, want iterable", valueType(mountsValue)) }
		iter := iterable.Iterate()
		for {
			var value starlark.Value
			if !iter.Next(&value) { break }
			mount, err := recordFrom[gwclient.Mount](value, "mount")
			if err != nil { iter.Done(); return nil, err }
			req.Mounts = append(req.Mounts, mount)
		}
		iter.Done()
	}
	if extraHostsValue != starlark.None {
		iterable, ok := extraHostsValue.(starlark.Iterable)
		if !ok { return nil, fmt.Errorf("new_container_request.extra_hosts: got %s, want iterable", valueType(extraHostsValue)) }
		iter := iterable.Iterate()
		for {
			var value starlark.Value
			if !iter.Next(&value) { break }
			host := &pb.HostIP{}
			if err := protoFromValue(value, host, "new_container_request.extra_hosts"); err != nil { iter.Done(); return nil, err }
			req.ExtraHosts = append(req.ExtraHosts, host)
		}
		iter.Done()
	}
	if platformValue != starlark.None {
		req.Platform = &pb.Platform{}
		if err := protoFromValue(platformValue, req.Platform, "new_container_request.platform"); err != nil { return nil, err }
	}
	if constraintsValue != starlark.None {
		req.Constraints = &pb.WorkerConstraints{}
		if err := protoFromValue(constraintsValue, req.Constraints, "new_container_request.constraints"); err != nil { return nil, err }
	}
	return newRecord("new_container_request", req, starlark.StringDict{
		"hostname": starlark.String(hostname), "net_mode": starlark.MakeInt(netMode),
		"mounts": mountsValue, "extra_hosts": extraHostsValue, "platform": platformValue, "constraints": constraintsValue,
	}), nil
}

func fileRangeBuiltin(_ *starlark.Thread, _ *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	var offset, length int
	if err := starlark.UnpackArgs("file_range", args, kwargs, "offset?", &offset, "length?", &length); err != nil { return nil, err }
	r := gwclient.FileRange{Offset: offset, Length: length}
	return newRecord("file_range", r, starlark.StringDict{"offset": starlark.MakeInt(offset), "length": starlark.MakeInt(length)}), nil
}

func readRequestBuiltin(_ *starlark.Thread, _ *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	var filename string
	var rangeValue starlark.Value = starlark.None
	if err := starlark.UnpackArgs("read_request", args, kwargs, "filename", &filename, "range?", &rangeValue); err != nil { return nil, err }
	var fileRange *gwclient.FileRange
	if rangeValue != starlark.None {
		value, err := recordFrom[gwclient.FileRange](rangeValue, "file_range"); if err != nil { return nil, err }
		fileRange = &value
	}
	req := gwclient.ReadRequest{Filename: filename, Range: fileRange}
	return newRecord("read_request", req, starlark.StringDict{"filename": starlark.String(filename), "range": rangeValue}), nil
}

func readContainerRequestBuiltin(_ *starlark.Thread, _ *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	var filename string
	var rangeValue starlark.Value = starlark.None
	var mountIndex int
	if err := starlark.UnpackArgs("read_container_request", args, kwargs, "filename", &filename, "range?", &rangeValue, "mount_index?", &mountIndex); err != nil { return nil, err }
	var fileRange *gwclient.FileRange
	if rangeValue != starlark.None {
		value, err := recordFrom[gwclient.FileRange](rangeValue, "file_range"); if err != nil { return nil, err }
		fileRange = &value
	}
	req := gwclient.ReadContainerRequest{ReadRequest: gwclient.ReadRequest{Filename: filename, Range: fileRange}, MountIndex: mountIndex}
	return newRecord("read_container_request", req, starlark.StringDict{"filename": starlark.String(filename), "range": rangeValue, "mount_index": starlark.MakeInt(mountIndex)}), nil
}

func readDirRequestBuiltin(_ *starlark.Thread, _ *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	var path, pattern string
	if err := starlark.UnpackArgs("read_dir_request", args, kwargs, "path", &path, "include_pattern?", &pattern); err != nil { return nil, err }
	req := gwclient.ReadDirRequest{Path: path, IncludePattern: pattern}
	return newRecord("read_dir_request", req, starlark.StringDict{"path": starlark.String(path), "include_pattern": starlark.String(pattern)}), nil
}

func readDirContainerRequestBuiltin(_ *starlark.Thread, _ *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	var path, pattern string
	var mountIndex int
	if err := starlark.UnpackArgs("read_dir_container_request", args, kwargs, "path", &path, "include_pattern?", &pattern, "mount_index?", &mountIndex); err != nil { return nil, err }
	req := gwclient.ReadDirContainerRequest{ReadDirRequest: gwclient.ReadDirRequest{Path: path, IncludePattern: pattern}, MountIndex: mountIndex}
	return newRecord("read_dir_container_request", req, starlark.StringDict{"path": starlark.String(path), "include_pattern": starlark.String(pattern), "mount_index": starlark.MakeInt(mountIndex)}), nil
}

func statRequestBuiltin(_ *starlark.Thread, _ *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	var path string
	if err := starlark.UnpackArgs("stat_request", args, kwargs, "path", &path); err != nil { return nil, err }
	req := gwclient.StatRequest{Path: path}
	return newRecord("stat_request", req, starlark.StringDict{"path": starlark.String(path)}), nil
}

func statContainerRequestBuiltin(_ *starlark.Thread, _ *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	var path string
	var mountIndex int
	if err := starlark.UnpackArgs("stat_container_request", args, kwargs, "path", &path, "mount_index?", &mountIndex); err != nil { return nil, err }
	req := gwclient.StatContainerRequest{StatRequest: gwclient.StatRequest{Path: path}, MountIndex: mountIndex}
	return newRecord("stat_container_request", req, starlark.StringDict{"path": starlark.String(path), "mount_index": starlark.MakeInt(mountIndex)}), nil
}

func winSizeBuiltin(_ *starlark.Thread, _ *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	var rows, cols int
	if err := starlark.UnpackArgs("win_size", args, kwargs, "rows?", &rows, "cols?", &cols); err != nil { return nil, err }
	size := gwclient.WinSize{Rows: uint32(rows), Cols: uint32(cols)}
	return newRecord("win_size", size, starlark.StringDict{"rows": starlark.MakeInt(rows), "cols": starlark.MakeInt(cols)}), nil
}

func warnOptsBuiltin(_ *starlark.Thread, _ *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	var level int
	var sourceInfoValue, rangesValue, detailValue starlark.Value = starlark.None, starlark.None, starlark.None
	var url string
	if err := starlark.UnpackArgs("warn_opts", args, kwargs,
		"level?", &level, "source_info?", &sourceInfoValue, "range?", &rangesValue, "detail?", &detailValue, "url?", &url,
	); err != nil { return nil, err }
	opts := gwclient.WarnOpts{Level: level, URL: url}
	if sourceInfoValue != starlark.None {
		opts.SourceInfo = &pb.SourceInfo{}
		if err := protoFromValue(sourceInfoValue, opts.SourceInfo, "warn_opts.source_info"); err != nil { return nil, err }
	}
	if rangesValue != starlark.None {
		iterable, ok := rangesValue.(starlark.Iterable)
		if !ok { return nil, fmt.Errorf("warn_opts.range: got %s, want iterable", valueType(rangesValue)) }
		iter := iterable.Iterate()
		for {
			var value starlark.Value
			if !iter.Next(&value) { break }
			r := &pb.Range{}
			if err := protoFromValue(value, r, "warn_opts.range"); err != nil { iter.Done(); return nil, err }
			opts.Range = append(opts.Range, r)
		}
		iter.Done()
	}
	if detailValue != starlark.None {
		iterable, ok := detailValue.(starlark.Iterable)
		if !ok { return nil, fmt.Errorf("warn_opts.detail: got %s, want iterable", valueType(detailValue)) }
		iter := iterable.Iterate()
		for {
			var value starlark.Value
			if !iter.Next(&value) { break }
			data, err := bytesAt("warn_opts.detail", value)
			if err != nil { iter.Done(); return nil, err }
			opts.Detail = append(opts.Detail, data)
		}
		iter.Done()
	}
	return newRecord("warn_opts", opts, starlark.StringDict{
		"level": starlark.MakeInt(level), "source_info": sourceInfoValue, "range": rangesValue, "detail": detailValue, "url": starlark.String(url),
	}), nil
}

func stringMapValue(values map[string]string) starlark.Value {
	dict := starlark.NewDict(len(values))
	for key, value := range values { _ = dict.SetKey(starlark.String(key), starlark.String(value)) }
	return dict
}

func stringBytesMapValue(values map[string][]byte) starlark.Value {
	dict := starlark.NewDict(len(values))
	for key, value := range values { _ = dict.SetKey(starlark.String(key), starlark.Bytes(append([]byte(nil), value...))) }
	return dict
}

func statValue(stat *types.Stat) starlark.Value {
	if stat == nil {
		return starlark.None
	}
	value := reflect.ValueOf(stat)
	if value.Kind() == reflect.Pointer {
		value = value.Elem()
	}
	fields := starlark.StringDict{}
	typ := value.Type()
	for i := 0; i < value.NumField(); i++ {
		field := value.Field(i)
		definition := typ.Field(i)
		if definition.PkgPath != "" || !field.CanInterface() {
			continue
		}
		name := starlarkllb.SnakeCaseIdentifier(definition.Name)
		switch field.Kind() {
		case reflect.String:
			fields[name] = starlark.String(field.String())
		case reflect.Bool:
			fields[name] = starlark.Bool(field.Bool())
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			fields[name] = starlark.MakeInt64(field.Int())
		case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
			fields[name] = starlark.MakeUint64(field.Uint())
		case reflect.Float32, reflect.Float64:
			fields[name] = starlark.Float(field.Float())
		}
	}
	return structValue("stat", fields)
}
