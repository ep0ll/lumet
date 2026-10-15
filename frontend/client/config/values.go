package config

import (
	"strings"

	"github.com/tonistiigi/go-csvvalue"
)

// filterPrefix returns opts entries whose key starts with prefix, with the prefix removed.
func filterPrefix(opts map[string]string, prefix string) map[string]string {
	m := map[string]string{}
	for k, v := range opts {
		if after, ok := strings.CutPrefix(k, prefix); ok {
			m[after] = v
		}
	}
	return m
}

// eachCSVField calls fn for every field of the CSV string v.
func eachCSVField(v string, fn func(field string) error) error {
	if v == "" {
		return nil
	}
	fields, err := csvvalue.Fields(v, nil)
	if err != nil {
		return err
	}
	for i := 0; i < len(fields); i++ {
		if err := fn(fields[i]); err != nil {
			return err
		}
	}
	return nil
}
