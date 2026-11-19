package archive

import (
	"archive/tar"
	"bytes"
)

// IsArchive reports whether the given header looks like a compressed or tar archive.
func IsArchive(header []byte) bool {
	for _, magic := range [][]byte{
		{0x42, 0x5A, 0x68},                   // bzip2
		{0x1F, 0x8B, 0x08},                   // gzip
		{0xFD, 0x37, 0x7A, 0x58, 0x5A, 0x00}, // xz
	} {
		if len(header) >= len(magic) && bytes.Equal(magic, header[:len(magic)]) {
			return true
		}
	}

	r := tar.NewReader(bytes.NewBuffer(header))
	_, err := r.Next()
	return err == nil
}
