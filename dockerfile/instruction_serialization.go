package dockerfile

import (
	"encoding/json"
	"strconv"
	"strings"
)

func jsonMarshal(value any) (string, error) {
	encoded, err := json.Marshal(value)
	if err != nil {
		return "", err
	}
	return string(encoded), nil
}

// dockerWord quotes a value for the typed ONBUILD serializer. It is not used
// to parse Dockerfile text or to perform shell expansion.
func dockerWord(value string) string {
	if strings.ContainsAny(value, " \t\r\n\"'") {
		return strconv.Quote(value)
	}
	return value
}
