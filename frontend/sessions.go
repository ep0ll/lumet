package frontend

import (
	"maps"
	"strings"
)

const localSessionIDPrefix = "local-sessionid:"

// LocalSessions maps a local context name to the session that serves it.
type LocalSessions map[string]string

// ParseLocalSessions source: https://github.com/moby/buildkit/blob/9fcedf9807077097847bda8d4db8b9c71d3110bf/frontend/dockerui/attr.go#L128-L136
func ParseLocalSessions(opts map[string]string) LocalSessions {
	m := LocalSessions{}
	for k, v := range opts {
		if name, ok := strings.CutPrefix(k, localSessionIDPrefix); ok {
			m[name] = v
		}
	}
	return m
}

func (s LocalSessions) clone() LocalSessions { return maps.Clone(s) }
