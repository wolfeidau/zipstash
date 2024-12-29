package client

import (
	"cmp"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/rs/zerolog/log"
	"golang.org/x/exp/slices"

	"github.com/wolfeidau/cache-service/internal/commands"
	"github.com/wolfeidau/cache-service/internal/downloader"
	"github.com/wolfeidau/cache-service/internal/trace"
)

type RestoreCmd struct {
	Endpoint string `help:"endpoint to call" default:"http://localhost:8080" env:"INPUT_ENDPOINT"`
	Token    string `help:"token to use" required:""`
	Key      string `help:"key to use for the cache entry" required:"" env:"INPUT_KEY"`
	Path     string `help:"Path list for a cache entry." env:"INPUT_PATH"`
}

func (c *RestoreCmd) Run(ctx context.Context, globals *commands.Globals) error {
	_, span := trace.Start(ctx, "RestoreCmd.Run")
	defer span.End()
	return c.restore(ctx, globals)
}

func (c *RestoreCmd) restore(ctx context.Context, globals *commands.Globals) error {
	ctx, span := trace.Start(ctx, "RestoreCmd.restore")
	defer span.End()

	cl, err := newClient(c.Endpoint, c.Token)
	if err != nil {
		return fmt.Errorf("failed to create client: %w", err)
	}

	getEntryResp, err := cl.GetCacheEntryByKeyWithResponse(ctx, "GitHubActions", c.Key)
	if err != nil {
		return fmt.Errorf("failed to get cache entry: %w", err)
	}

	if getEntryResp.JSON200 == nil {
		return fmt.Errorf("failed to get cache entry: %s", getEntryResp.Status())
	}

	log.Info().Any("cache entry", getEntryResp.JSON200).Msg("cache entry")

	downloads, err := downloader.NewDownloader(getEntryResp.JSON200.DownloadInstructions, 20).Download(ctx)
	if err != nil {
		return fmt.Errorf("failed to download cache entry: %w", err)
	}

	slices.SortFunc(downloads, func(a, b downloader.DownloadedFile) int {
		return cmp.Compare(a.Part, b.Part)
	})

	for _, d := range downloads {
		log.Info().Any("download", d).Msg("download")
	}

	zipFile, err := os.CreateTemp("", "cache-service-download-*.zip")
	if err != nil {
		return fmt.Errorf("failed to create temp file: %w", err)
	}
	defer zipFile.Close()

	zipFileLen := int64(0)

	for _, d := range downloads {
		n, err := appendToFile(zipFile, d.FilePath)
		if err != nil {
			return fmt.Errorf("failed to write file: %w", err)
		}
		zipFileLen += n
	}

	log.Info().Int64("zipFileLen", zipFileLen).Str("name", zipFile.Name()).Msg("zip file len")

	// extract, err := quickzip.NewExtractorFromReader(zipFile, zipFileLen)
	// if err != nil {
	// 	return fmt.Errorf("failed to create extractor: %w", err)
	// }

	// pathToMappings, err := pathToMappings(c.Path)
	// if err != nil {
	// 	return fmt.Errorf("failed to create path mappings: %w", err)
	// }

	// err = extract.ExtractWithPathMapper(ctx, func(file *zip.File) (string, error) {
	// 	return filepath.Join()
	// })

	return nil
}

// pass in a list of paths and turn them into a zip file stream to enable extraction

func appendToFile(f *os.File, path string) (int64, error) {
	pf, err := os.Open(path)
	if err != nil {
		return 0, fmt.Errorf("failed to open file: %w", err)
	}
	defer pf.Close()

	n, err := io.Copy(f, pf)
	if err != nil {
		return 0, fmt.Errorf("failed to copy file: %w", err)
	}

	defer os.Remove(path)

	return n, nil
}

func pathToMappings(path string) (map[string]string, error) {
	files := strings.Fields(path)
	if len(files) == 0 {
		return nil, fmt.Errorf("no paths provided")
	}

	mappings := make(map[string]string)

	for _, file := range files {
		if strings.HasPrefix(file, "~/") {
			homedir, err := os.UserHomeDir()
			if err != nil {
				return nil, fmt.Errorf("failed to get home directory: %w", err)
			}

			mappings[file] = filepath.Join(homedir, file[2:])
		}
	}

	return mappings, nil
}
