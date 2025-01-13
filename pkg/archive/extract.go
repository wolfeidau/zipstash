package archive

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/klauspost/compress/zip"
	"github.com/wolfeidau/quickzip"

	"github.com/wolfeidau/zipstash/pkg/trace"
)

func ExtractFiles(ctx context.Context, zipFile *os.File, zipFileLen int64, paths []string) error {
	_, span := trace.Start(ctx, "ExtractFiles")
	defer span.End()
	extract, err := quickzip.NewExtractorFromReader(zipFile, zipFileLen)
	if err != nil {
		return fmt.Errorf("failed to create extractor: %w", err)
	}

	mappings, err := PathsToMappings(paths)
	if err != nil {
		return fmt.Errorf("failed to create mappings: %w", err)
	}

	err = extract.ExtractWithPathMapper(ctx, func(file *zip.File) (string, error) {
		for _, mapping := range mappings {
			if strings.HasPrefix(file.Name, mapping.RelativePath) {
				return filepath.Join(mapping.Chroot, file.Name), nil
			}
		}

		return "", fmt.Errorf("failed to find path mapping for: %s", file.Name)
	})
	if err != nil {
		return fmt.Errorf("failed to extract zip file: %w", err)
	}
	return nil
}
