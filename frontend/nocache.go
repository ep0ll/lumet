package frontend

import "strings"

const keyNoCache = "no-cache"

// NoCache records which stages must bypass the cache.
type NoCache struct {
	all    bool
	stages []string
}

// ParseNoCache reads the "no-cache" option: absent = nothing, empty = all stages,
// otherwise a comma separated stage list.
func ParseNoCache(opts map[string]string) NoCache {
	v, ok := opts[keyNoCache]
	if !ok {
		return NoCache{}
	}
	if v == "" {
		return NoCache{all: true}
	}
	return NoCache{stages: strings.Split(v, ",")}
}

// Matches reports whether the named stage must not use the cache.
func (n NoCache) Matches(stage string) bool {
	if n.all {
		return true
	}
	for i := 0; i < len(n.stages); i++ {
		if strings.EqualFold(n.stages[i], stage) {
			return true
		}
	}
	return false
}

func (n NoCache) clone() NoCache {
	n.stages = append([]string(nil), n.stages...)
	return n
}
