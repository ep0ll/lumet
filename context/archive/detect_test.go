package archive

import (
	"archive/tar"
	"bytes"
	"testing"
)

func TestIsArchiveCompressedMagic(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		header []byte
	}{
		{name: "bzip2", header: []byte{0x42, 0x5A, 0x68}},
		{name: "gzip", header: []byte{0x1F, 0x8B, 0x08}},
		{name: "xz", header: []byte{0xFD, 0x37, 0x7A, 0x58, 0x5A, 0x00}},
		{name: "zstd", header: []byte{0x28, 0xB5, 0x2F, 0xFD}},
		{name: "lz4", header: []byte{0x04, 0x22, 0x4D, 0x18}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if !IsArchive(tt.header) {
				t.Fatalf("IsArchive(%x) = false, want true", tt.header)
			}
		})
	}
}

func TestIsArchiveTar(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	writer := tar.NewWriter(&buf)
	if err := writer.WriteHeader(&tar.Header{Name: "file", Mode: 0o600, Size: 0}); err != nil {
		t.Fatal(err)
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}

	if !IsArchive(buf.Bytes()) {
		t.Fatal("IsArchive(tar header) = false, want true")
	}
}

func TestIsArchiveRejectsShortOrPlainInput(t *testing.T) {
	t.Parallel()

	for _, header := range [][]byte{nil, {}, []byte("plain text"), make([]byte, 511)} {
		if IsArchive(header) {
			t.Fatalf("IsArchive(%q) = true, want false", header)
		}
	}
}
