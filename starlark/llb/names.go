package llb

import (
	"fmt"
	"sort"
	"regexp"
	"strings"

	"go.starlark.net/starlark"
)

var (
	snakeAcronyms = strings.NewReplacer("UIDGID", "UID_GID")
	snakeUpperBoundary = regexp.MustCompile(`([A-Z]+)([A-Z][a-z])`)
	snakeLowerBoundary = regexp.MustCompile(`([a-z0-9])([A-Z])`)
)

func SnakeCaseIdentifier(name string) string {
	name = strings.TrimSpace(name)
	name = strings.ReplaceAll(name, "-", "_")
	name = snakeAcronyms.Replace(name)
	name = snakeUpperBoundary.ReplaceAllString(name, `${1}_${2}`)
	name = snakeLowerBoundary.ReplaceAllString(name, `${1}_${2}`)
	name = strings.ToLower(name)
	for strings.Contains(name, "__") {
		name = strings.ReplaceAll(name, "__", "_")
	}
	return strings.Trim(name, "_")
}

func SnakeCaseNames(names []string) []string {
	result := make([]string, 0, len(names))
	seen := make(map[string]struct{}, len(names))
	for _, name := range names {
		name = SnakeCaseIdentifier(name)
		if name == "" {
			continue
		}
		if _, ok := seen[name]; ok {
			continue
		}
		seen[name] = struct{}{}
		result = append(result, name)
	}
	sort.Strings(result)
	return result
}

func NormalizeMembers(members starlark.StringDict) (starlark.StringDict, error) {
	if members == nil {
		return nil, fmt.Errorf("starlark/llb: nil member dictionary")
	}
	result := make(starlark.StringDict, len(members))
	original := make(map[string]string, len(members))
	for name, value := range members {
		key := SnakeCaseIdentifier(name)
		if key == "" || value == nil {
			return nil, fmt.Errorf("starlark/llb: invalid member %q", name)
		}
		if previous, ok := original[key]; ok {
			return nil, fmt.Errorf("starlark/llb: members %q and %q both normalize to %q", previous, name, key)
		}
		original[key] = name
		result[key] = value
	}
	return result, nil
}

func UnpackArgs(fnname string, args starlark.Tuple, kwargs []starlark.Tuple, pairs ...any) error {
	specs := make([]any, len(pairs))
	for i, pair := range pairs {
		name, ok := pair.(string)
		if !ok {
			specs[i] = pair
			continue
		}
		prefix := ""
		for strings.HasPrefix(name, "*") {
			prefix += "*"
			name = strings.TrimPrefix(name, "*")
		}
		suffix := ""
		if strings.HasSuffix(name, "?") {
			suffix = "?"
			name = strings.TrimSuffix(name, "?")
		}
		specs[i] = prefix + SnakeCaseIdentifier(name) + suffix
	}
	normalizedKwargs := make([]starlark.Tuple, len(kwargs))
	for i, keyword := range kwargs {
		normalizedKwargs[i] = append(starlark.Tuple(nil), keyword...)
		if len(keyword) > 0 {
			if name, ok := keyword[0].(starlark.String); ok {
				normalizedKwargs[i][0] = starlark.String(SnakeCaseIdentifier(string(name)))
			}
		}
	}
	return starlark.UnpackArgs(SnakeCaseIdentifier(fnname), args, normalizedKwargs, specs...)
}
