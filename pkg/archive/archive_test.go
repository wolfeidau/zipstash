package archive

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/minio/crc64nvme"
	"github.com/stretchr/testify/require"
)

func TestIsUnderHome(t *testing.T) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		t.Fatalf("failed to get home directory: %v", err)
	}

	tests := []struct {
		name     string
		path     string
		expected bool
		wantErr  bool
	}{
		{
			name:     "path under home directory",
			path:     filepath.Join(homeDir, "documents"),
			expected: true,
			wantErr:  false,
		},
		{
			name:     "path not under home directory",
			path:     "/tmp/test",
			expected: false,
			wantErr:  false,
		},
		// TODO: investigate how we can override the home directory for testing
		// {
		// 	name:     "relative path under home",
		// 	path:     "~/documents",
		// 	expected: true,
		// 	wantErr:  false,
		// },
		{
			name:     "empty path",
			path:     "",
			expected: false,
			wantErr:  true,
		},
		{
			name:     "path with symlinks",
			path:     filepath.Join(homeDir, "..", filepath.Base(homeDir)),
			expected: true,
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := isUnderHome(tt.path)
			if (err != nil) != tt.wantErr {
				t.Errorf("isUnderHome() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if result != tt.expected {
				t.Errorf("isUnderHome() = %v, want %v", result, tt.expected)
			}
		})
	}
}
func TestChecksumWriter_Sum(t *testing.T) {
	tests := []struct {
		name     string
		expected string
		input    []byte
	}{
		{
			name:     "empty input",
			input:    []byte{},
			expected: "0000000000000000",
		},
		{
			name:     "simple string",
			input:    []byte("hello"),
			expected: "3377857006524257",
		},
		{
			name:     "binary data",
			input:    []byte{0xFF, 0x00, 0xAB, 0xCD},
			expected: "15d0acf18b3d9b05",
		},
		{
			name:     "unicode string",
			input:    []byte("Hello 世界"),
			expected: "48a95aa42be13dd9",
		},
		{
			name:     "long input",
			input:    bytes.Repeat([]byte("a"), 1000),
			expected: "eab3232b2997f5e5",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert := require.New(t)
			c := &ChecksumWriter{
				h: crc64nvme.New(),
				f: &bytes.Buffer{},
			}
			_, err := c.Write(tt.input)
			assert.NoError(err)
			result := c.Sum()
			assert.Equal(tt.expected, result)
		})
	}
}
