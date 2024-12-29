package client

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/rs/zerolog/log"

	"github.com/wolfeidau/cache-service/internal/archive"
	"github.com/wolfeidau/cache-service/internal/commands"
	"github.com/wolfeidau/cache-service/internal/trace"
	"github.com/wolfeidau/cache-service/internal/uploader"
	"github.com/wolfeidau/cache-service/pkg/client"
)

type SaveCmd struct {
	Endpoint string `help:"endpoint to call" default:"http://localhost:8080" env:"INPUT_ENDPOINT"`
	Token    string `help:"token to use" required:""`
	Key      string `help:"key to use for the cache entry" required:"" env:"INPUT_KEY"`
	Path     string `help:"Path list for a cache entry." env:"INPUT_PATH"`
}

func (c *SaveCmd) Run(ctx context.Context, globals *commands.Globals) error {
	ctx, span := trace.Start(ctx, "SaveCmd.Run")
	defer span.End()

	return c.save(ctx, globals)
}

func (c *SaveCmd) save(ctx context.Context, globals *commands.Globals) error {
	ctx, span := trace.Start(ctx, "SaveCmd.save")
	defer span.End()

	paths, err := checkPath(c.Path)
	if err != nil {
		return fmt.Errorf("failed to check path: %w", err)
	}

	fileInfo, err := archive.BuildArchive(ctx, paths, c.Key)
	if err != nil {
		return fmt.Errorf("failed to build archive: %w", err)
	}

	log.Info().Any("fileInfo", fileInfo).Msg("archive info")

	cl, err := newClient(c.Endpoint, c.Token)

	if err != nil {
		return fmt.Errorf("failed to create client: %w", err)
	}

	createResp, err := cl.CreateCacheEntryWithResponse(ctx, "GitHubActions", client.CreateCacheEntryJSONRequestBody{
		CacheEntry: client.CacheEntry{
			Key:         c.Key,
			Compression: "zip",
			FileSize:    fileInfo.Size,
			Sha256sum:   fileInfo.Sha256sum,
			Paths:       paths,
		},
	})
	if err != nil {
		return fmt.Errorf("failed to create cache entry: %w", err)
	}

	if createResp.StatusCode() != http.StatusCreated {
		return fmt.Errorf("failed to create cache entry: %s", createResp.JSONDefault.Message)
	}

	log.Info().Str("id", createResp.JSON201.Id).Msg("creating cache entry")

	upl := uploader.NewUploader(ctx, fileInfo.ArchivePath, createResp.JSON201.UploadInstructions, 20)

	etags, err := upl.Upload(ctx)
	if err != nil {
		return fmt.Errorf("failed to upload: %w", err)
	}

	log.Info().Str("id", createResp.JSON201.Id).Int("len", len(etags)).Msg("updating cache entry")

	updateResp, err := cl.UpdateCacheEntryWithResponse(ctx, "GitHubActions", client.CacheEntryUpdateRequest{
		Id:             createResp.JSON201.Id,
		Key:            c.Key,
		MultipartEtags: etags,
	})
	if err != nil {
		return fmt.Errorf("failed to update cache entry: %w", err)
	}

	if updateResp.StatusCode() != http.StatusOK {
		return fmt.Errorf("failed to update cache entry: %s", updateResp.JSONDefault.Message)
	}

	log.Info().Str("id", createResp.JSON201.Id).Msg("updated cache entry")

	return nil
}

func checkPath(path string) ([]string, error) {
	files := strings.Fields(path)
	if len(files) == 0 {
		return nil, fmt.Errorf("no paths provided")
	}

	for i, file := range files {
		if strings.HasPrefix(file, "~/") {
			homedir, err := os.UserHomeDir()
			if err != nil {
				return nil, fmt.Errorf("failed to get home directory: %w", err)
			}

			// update the path to be the full path
			files[i] = filepath.Join(homedir, file[2:])
		}
	}

	return files, nil
}
