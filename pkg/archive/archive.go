package archive

import (
	"encoding/hex"
	"fmt"
	"hash"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/minio/crc64nvme"
)

const (
	modifiedEpoch = "2024-01-01T00:00:00Z"
	bufferSize    = 1024 * 1024 * 20
	skipOwnership = true
)

// isUnderHome checks if the given path is under the user's home directory.
// It first gets the absolute path of the given path, then gets the user's
// home directory, and finally checks if the absolute path starts with the
// home directory path.
func isUnderHome(path string) (bool, error) {
	if path == "" {
		return false, fmt.Errorf("path is empty")
	}

	// Get absolute path
	absPath, err := filepath.Abs(path)
	if err != nil {
		return false, fmt.Errorf("failed to get absolute path: %w", err)
	}

	// Get user's home directory
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return false, fmt.Errorf("failed to get home directory: %w", err)
	}

	// Clean both paths to normalize them
	cleanPath := filepath.Clean(absPath)
	cleanHome := filepath.Clean(homeDir)

	// Check if the path starts with home directory
	return strings.HasPrefix(cleanPath, cleanHome), nil
}

type ChecksumWriter struct {
	f io.Writer
	h hash.Hash
}

func NewChecksumWriter(f io.Writer) *ChecksumWriter {
	return &ChecksumWriter{
		f: f,
		h: crc64nvme.New(),
	}
}

// implement the io.WriteCloser interface
func (c *ChecksumWriter) Write(p []byte) (n int, err error) {
	n, err = c.f.Write(p)
	if err != nil {
		return n, err
	}
	c.h.Write(p)
	return n, nil
}

func (c *ChecksumWriter) Sum() string {
	return hex.EncodeToString(c.h.Sum(nil))
}
