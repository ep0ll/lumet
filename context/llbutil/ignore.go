package llbutil

import (
	"bytes"

	"github.com/moby/patternmatcher/ignorefile"
)

// IgnoreParser parses the contents of one ignore file.
type IgnoreParser func(filename string, data []byte) ([]string, error)

// ParseIgnoreFile parses an ignore file with the supplied parser or Moby's
// standard parser when no override is provided.
func ParseIgnoreFile(parser IgnoreParser, filename string, data []byte) ([]string, error) {
	if parser != nil {
		return parser(filename, data)
	}
	return ignorefile.ReadAll(bytes.NewReader(data))
}
