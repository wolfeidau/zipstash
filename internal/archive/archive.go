package archive

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"hash"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/klauspost/compress/zstd"
	"github.com/wolfeidau/quickzip"
	"go.opentelemetry.io/otel/attribute"

	"github.com/wolfeidau/cache-service/internal/trace"
)

const (
	modifiedEpoch = "2024-01-01T00:00:00Z"
	bufferSize    = 1024 * 1024 * 2
	skipOwnership = true
)

type ArchiveInfo struct {
	ArchivePath string
	Size        int64
	Sha256sum   string
	Stats       map[string]int64
}

func BuildArchive(ctx context.Context, paths []string, key string) (*ArchiveInfo, error) {
	_, span := trace.Start(ctx, "BuildArchive")
	defer span.End()

	modified, err := time.Parse(time.RFC3339, modifiedEpoch)
	if err != nil {
		return nil, fmt.Errorf("failed to parse modified epoch: %w", err)
	}

	archiveFile, err := os.CreateTemp("", fmt.Sprintf("%s-*.zip", key))
	if err != nil {
		return nil, fmt.Errorf("failed to create archive file: %w", err)
	}
	defer archiveFile.Close()

	checksummer := NewChecksumSHA256(archiveFile)

	// wrap the file in an io.Writer which records the sha256sum of the file
	arc, err := quickzip.NewArchiver(
		checksummer,
		quickzip.WithArchiverMethod(zstd.ZipMethodWinZip),
		quickzip.WithArchiverBufferSize(bufferSize),
		quickzip.WithModifiedEpoch(modified),
		quickzip.WithSkipOwnership(skipOwnership),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create archiver: %w", err)
	}

	for _, path := range paths {
		_, err := os.Stat(path)
		if err != nil {
			return nil, fmt.Errorf("failed to stat file: %w", err)
		}

		_, err = isUnderHome(path)
		if err != nil {
			return nil, fmt.Errorf("failed directory (%s) outside home directory: %w", path, err)
		}

		files := make(map[string]os.FileInfo)
		err = filepath.Walk(path, func(filename string, fi os.FileInfo, err error) error {
			files[filename] = fi
			return nil
		})
		if err != nil {
			return nil, fmt.Errorf("failed to walk path: %s with error: %w", path, err)
		}

		chroot, err := getChrootPath(path)
		if err != nil {
			return nil, fmt.Errorf("failed to get chroot path: %w", err)
		}

		err = arc.Archive(context.Background(), chroot, files)
		if err != nil {
			return nil, fmt.Errorf("failed to archive path: %s with error: %w", path, err)
		}
	}

	err = arc.Close()
	if err != nil {
		return nil, fmt.Errorf("failed to close archive: %w", err)
	}

	stat, err := archiveFile.Stat()
	if err != nil {
		return nil, fmt.Errorf("failed to stat archive file: %w", err)
	}

	span.SetAttributes(
		attribute.String("Sha256sum", checksummer.Sum()),
		attribute.Int64("Size", stat.Size()),
	)

	return &ArchiveInfo{
		ArchivePath: archiveFile.Name(),
		Size:        stat.Size(),
		Sha256sum:   checksummer.Sum(),
		Stats:       map[string]int64{},
	}, nil
}

func getChrootPath(path string) (string, error) {
	if filepath.IsAbs(path) {
		return os.UserHomeDir()
	}

	// get the current working directory
	return os.Getwd()
}

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

type ChecksumSHA256 struct {
	f      io.Writer
	sha256 hash.Hash
}

func NewChecksumSHA256(f io.Writer) *ChecksumSHA256 {
	return &ChecksumSHA256{
		f:      f,
		sha256: sha256.New(),
	}
}

// implement the io.WriteCloser interface
func (c *ChecksumSHA256) Write(p []byte) (n int, err error) {
	n, err = c.f.Write(p)
	if err != nil {
		return n, err
	}
	c.sha256.Write(p)
	return n, nil
}

func (c *ChecksumSHA256) Sum() string {
	return hex.EncodeToString(c.sha256.Sum(nil))
}
