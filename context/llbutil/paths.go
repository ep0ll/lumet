package llbutil

import (
	"fmt"
	"path"
	"strings"
)

// CleanRelativePath normalizes a source-relative path and rejects paths that
// are empty, absolute, or escape their source root.
func CleanRelativePath(value string) (string, error) {
	value = strings.TrimSpace(strings.ReplaceAll(value, "\\", "/"))
	if value == "" || path.IsAbs(value) {
		return "", fmt.Errorf("path %q must be a non-empty relative path", value)
	}
	clean := path.Clean(value)
	if clean == "." || clean == ".." || strings.HasPrefix(clean, "../") {
		return "", fmt.Errorf("path %q escapes the source root", value)
	}
	return clean, nil
}

// CleanRelativePaths normalizes and de-duplicates source-relative paths while
// preserving their first occurrence and ordering.
func CleanRelativePaths(paths []string) ([]string, error) {
	cleaned := make([]string, 0, len(paths))
	seen := make(map[string]struct{}, len(paths))
	for _, value := range paths {
		clean, err := CleanRelativePath(value)
		if err != nil {
			return nil, err
		}
		if _, ok := seen[clean]; ok {
			continue
		}
		seen[clean] = struct{}{}
		cleaned = append(cleaned, clean)
	}
	return cleaned, nil
}
