package instruction

import (
	"strconv"
	"strings"
)

// dockerWord quotes an instruction operand when whitespace or quote
// characters would otherwise make the ONBUILD representation ambiguous.
func dockerWord(value string) string {
	if strings.ContainsAny(value, " \t\r\n\"'") {
		return strconv.Quote(value)
	}
	return value
}

// setEnv updates one environment entry while preserving the order of
// unrelated inherited entries and ensuring only one value per key remains.
func setEnv(values []string, key, value string) []string {
	out := make([]string, 0, len(values)+1)
	replaced := false
	for _, entry := range values {
		current, _, _ := strings.Cut(entry, "=")
		if current == key {
			if !replaced {
				out = append(out, key+"="+value)
				replaced = true
			}
			continue
		}
		out = append(out, entry)
	}
	if !replaced {
		out = append(out, key+"="+value)
	}
	return out
}

func uniqueStageNames(names []StageName) []StageName {
	seen := make(map[string]struct{}, len(names))
	result := make([]StageName, 0, len(names))
	for _, name := range names {
		key := strings.ToLower(strings.TrimSpace(string(name)))
		if key == "" {
			continue
		}
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		result = append(result, name)
	}
	return result
}
