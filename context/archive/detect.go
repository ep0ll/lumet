package archive

import (
	"archive/tar"
	"bytes"
)

// IsArchive reports whether a header looks like a supported compressed or tar archive.
func IsArchive(header []byte) bool {
	magics := [][]byte{
		{0x42, 0x5A, 0x68},                   // bzip2
		{0x1F, 0x8B, 0x08},                   // gzip
		{0xFD, 0x37, 0x7A, 0x58, 0x5A, 0x00}, // xz
		{0x28, 0xB5, 0x2F, 0xFD},             // zstd
		{0x04, 0x22, 0x4D, 0x18},             // lz4 frame
	}
	for _, magic := range magics {
		if len(header) >= len(magic) && bytes.Equal(magic, header[:len(magic)]) {
			return true
		}
	}

	if len(header) < 512 {
		return false
	}
	reader := tar.NewReader(bytes.NewReader(header))
	_, err := reader.Next()
	return err == nil
}
