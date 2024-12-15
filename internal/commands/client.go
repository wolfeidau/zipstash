package commands

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"

	"github.com/wolfeidau/cache-service/internal/trace"
	"github.com/wolfeidau/cache-service/pkg/api"
)

type ClientCmd struct {
	Endpoint  string `help:"endpoint to call" default:"http://localhost:8080"`
	Token     string `help:"token to use" required:""`
	Action    string `help:"action to perform" enum:"save,restore" required:""`
	CacheFile string `help:"file to save"`
}

func (c *ClientCmd) Run(ctx context.Context, globals *Globals) error {
	_, span := trace.Start(ctx, "ClientCmdRun")
	defer span.End()

	switch c.Action {
	case "save":
		return c.save(ctx, globals)
	case "restore":
		return c.restore(ctx, globals)
	}

	return nil
}

func (c *ClientCmd) save(ctx context.Context, globals *Globals) error {
	file, err := os.Open(c.CacheFile)
	if err != nil {
		return fmt.Errorf("failed to open file: %w", err)
	}

	defer file.Close()

	fileInfo, err := file.Stat()
	if err != nil {
		return fmt.Errorf("failed to stat file: %w", err)
	}

	fileName := filepath.Base(c.CacheFile)

	cl, err := api.NewClientWithResponses(c.Endpoint, api.WithRequestEditorFn(func(ctx context.Context, req *http.Request) error {
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.Token))
		return nil
	}))
	if err != nil {
		return fmt.Errorf("failed to create client: %w", err)
	}

	createResp, err := cl.CreateCacheEntryWithResponse(ctx, "GitHubActions", api.CreateCacheEntryJSONRequestBody{
		CacheEntry: api.CacheEntry{
			Key:         fileName,
			Compression: "zip",
			FileSize:    fileInfo.Size(),
			Sha256sum:   "8b61aa44a35df9defab3c94f6180e1b921788f252640b9856784438d71e140a8",
		},
	})
	if err != nil {
		return fmt.Errorf("failed to create cache entry: %w", err)
	}

	if createResp.StatusCode() != http.StatusCreated {
		return fmt.Errorf("failed to create cache entry: %s", createResp.JSONDefault.Message)
	}

	fmt.Println(string(createResp.Body))

	id := createResp.JSON201.Id

	fmt.Println(id)

	uploadInst := createResp.JSON201.UploadInstructions[0]

	uploadReq, err := http.NewRequest(
		uploadInst.Method,
		uploadInst.Url, file)

	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	uploadReq.ContentLength = fileInfo.Size()

	cacheResp, err := http.DefaultClient.Do(uploadReq)
	if err != nil {
		return fmt.Errorf("failed to upload file: %w", err)
	}

	defer cacheResp.Body.Close()

	data, err := io.ReadAll(cacheResp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response: %w", err)
	}

	if cacheResp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to upload file: %s %s", cacheResp.Status, string(data))
	}

	fmt.Println(cacheResp.Status)

	updateResp, err := cl.UpdateCacheEntryWithResponse(ctx, "GitHubActions", api.CacheEntryUpdateRequest{
		Id:  id,
		Key: fileName,
		MultipartEtags: []api.CachePartETag{
			{
				Etag: cacheResp.Header.Get("ETag"),
				Part: 1,
			},
		},
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

func (c *ClientCmd) restore(ctx context.Context, globals *Globals) error {
	return nil
}
