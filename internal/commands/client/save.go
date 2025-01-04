package client

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/rs/zerolog/log"

	"github.com/wolfeidau/zipstash/internal/commands"
	"github.com/wolfeidau/zipstash/pkg/archive"
	"github.com/wolfeidau/zipstash/pkg/client"
	"github.com/wolfeidau/zipstash/pkg/tokens"
	"github.com/wolfeidau/zipstash/pkg/trace"
	"github.com/wolfeidau/zipstash/pkg/uploader"
)

type SaveCmd struct {
	Endpoint    string `help:"endpoint to call" default:"http://localhost:8080" env:"INPUT_ENDPOINT"`
	Key         string `help:"key to use for the cache entry" required:"" env:"INPUT_KEY"`
	Path        string `help:"Path list for a cache entry." env:"INPUT_PATH"`
	TokenSource string `help:"token source" default:"github_actions" env:"INPUT_TOKEN_SOURCE"`
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

	log.Info().
		Str("path", fileInfo.ArchivePath).
		Int64("size", fileInfo.Size).
		Str("sha256sum", fileInfo.Sha256sum).
		Msg("archive built")

	token, err := tokens.GetToken(ctx, c.TokenSource, audience, nil)
	if err != nil {
		return fmt.Errorf("failed to get token: %w", err)
	}

	cl, err := newClient(c.Endpoint, token, globals.Version)
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

	log.Info().Str("id", createResp.JSON201.Id).Int("parts", len(etags)).Msg("updated cache entry")

	return nil
}

func checkPath(path string) ([]string, error) {
	paths := strings.Fields(path)
	if len(paths) == 0 {
		return nil, fmt.Errorf("no paths provided")
	}

	return paths, nil
}
