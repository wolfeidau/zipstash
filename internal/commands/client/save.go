package client

import (
	"context"
	"crypto/sha256"
	"fmt"
	"io"
	"net/http"
	"os"

	"github.com/rs/zerolog/log"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"

	"github.com/wolfeidau/cache-service/internal/commands"
	"github.com/wolfeidau/cache-service/internal/trace"
	"github.com/wolfeidau/cache-service/internal/uploader"
	"github.com/wolfeidau/cache-service/pkg/client"
)

type SaveCmd struct {
	Endpoint  string `help:"endpoint to call" default:"http://localhost:8080" env:"INPUT_ENDPOINT"`
	Token     string `help:"token to use" required:""`
	Key       string `help:"key to use for the cache entry" required:"" env:"INPUT_KEY"`
	Path      string `help:"Path list for a cache entry." env:"INPUT_PATH"`
	CacheFile string `help:"file to save"`
	Skip      bool   `help:"skip confirmation"`
}

func (c *SaveCmd) Run(ctx context.Context, globals *commands.Globals) error {
	_, span := trace.Start(ctx, "SaveCmd.Run")
	defer span.End()

	return c.save(ctx, globals)
}

func (c *SaveCmd) save(ctx context.Context, globals *commands.Globals) error {
	_, span := trace.Start(ctx, "SaveCmd.save")
	defer span.End()

	fileInfo, err := checkCacheFile(ctx, c.CacheFile)
	if err != nil {
		return fmt.Errorf("failed to check cache file: %w", err)
	}

	httpClient := &http.Client{Transport: otelhttp.NewTransport(http.DefaultTransport)}

	cl, err := client.NewClientWithResponses(c.Endpoint, client.WithHTTPClient(httpClient), client.WithRequestEditorFn(func(ctx context.Context, req *http.Request) error {
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.Token))
		return nil
	}))
	if err != nil {
		return fmt.Errorf("failed to create client: %w", err)
	}

	createResp, err := cl.CreateCacheEntryWithResponse(ctx, "GitHubActions", client.CreateCacheEntryJSONRequestBody{
		CacheEntry: client.CacheEntry{
			Key:         c.Key,
			Compression: "zip",
			FileSize:    fileInfo.Size,
			Sha256sum:   fileInfo.Sha256sum,
		},
	})
	if err != nil {
		return fmt.Errorf("failed to create cache entry: %w", err)
	}

	if createResp.StatusCode() != http.StatusCreated {
		return fmt.Errorf("failed to create cache entry: %s", createResp.JSONDefault.Message)
	}

	fmt.Println(string(createResp.Body))

	log.Info().Str("id", createResp.JSON201.Id).Msg("creating cache entry")

	if c.Skip {
		return nil
	}

	upl := uploader.NewUploader(ctx, c.CacheFile, createResp.JSON201.UploadInstructions, 20)

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

	fmt.Println(string(updateResp.Body))

	return nil
}

type fileInfo struct {
	Size      int64
	Sha256sum string
}

func checkCacheFile(ctx context.Context, cacheFile string) (*fileInfo, error) {
	_, span := trace.Start(ctx, "checkCacheFile")
	defer span.End()

	file, err := os.Open(cacheFile)
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %w", err)
	}

	defer file.Close()

	fstat, err := file.Stat()
	if err != nil {
		return nil, fmt.Errorf("failed to stat file: %w", err)
	}

	sha256 := sha256.New()
	if _, err := io.Copy(sha256, file); err != nil {
		return nil, fmt.Errorf("failed to calculate sha256sum: %w", err)
	}

	return &fileInfo{
		Size:      fstat.Size(),
		Sha256sum: fmt.Sprintf("%x", sha256.Sum(nil)),
	}, nil
}