package client

import (
	"context"
	"fmt"
	"strings"
	"time"

	"connectrpc.com/connect"
	"github.com/rs/zerolog/log"

	v1 "github.com/wolfeidau/zipstash/api/zipstash/v1"
	"github.com/wolfeidau/zipstash/pkg/archive"
	"github.com/wolfeidau/zipstash/pkg/tokens"
	"github.com/wolfeidau/zipstash/pkg/trace"
	"github.com/wolfeidau/zipstash/pkg/uploader"
)

type SaveCmd struct {
	Key         string `help:"key to use for the cache entry" required:"" env:"INPUT_KEY"`
	Path        string `help:"Path list for a cache entry." env:"INPUT_PATH"`
	TokenSource string `help:"token source" default:"github_actions" env:"INPUT_TOKEN_SOURCE"`
	GitHub      GitHub `embed:"" prefix:"github_"`
	Local       Local  `embed:"" prefix:"local_"`
}

func (c *SaveCmd) Run(ctx context.Context, globals *Globals) error {
	ctx, span := trace.Start(ctx, "SaveCmd.Run")
	defer span.End()

	return c.save(ctx, globals)
}

func (c *SaveCmd) save(ctx context.Context, globals *Globals) error {
	ctx, span := trace.Start(ctx, "SaveCmd.save")
	defer span.End()

	cl := globals.Client

	repo, branch, err := getRepoAndBranch(c.GitHub, c.Local)
	if err != nil {
		return fmt.Errorf("failed to get repo and branch: %w", err)
	}

	paths, err := checkPath(c.Path)
	if err != nil {
		return fmt.Errorf("failed to check path: %w", err)
	}

	start := time.Now()

	fileInfo, err := archive.BuildArchive(ctx, paths, c.Key)
	if err != nil {
		return fmt.Errorf("failed to build archive: %w", err)
	}

	log.Info().
		Str("path", fileInfo.ArchivePath).
		Int64("size", fileInfo.Size).
		Str("sha256sum", fileInfo.Sha256sum).
		Dur("duration_ms", time.Since(start)).
		Msg("archive built")

	token, err := tokens.GetToken(ctx, c.TokenSource, audience, nil)
	if err != nil {
		return fmt.Errorf("failed to get token: %w", err)
	}

	req := newAuthenticatedProviderRequest(&v1.CreateCacheEntryRequest{
		CacheEntry: &v1.CacheEntry{
			Key:         c.Key,
			Compression: "zip",
			FileSize:    fileInfo.Size,
			Sha256Sum:   fileInfo.Sha256sum,
			Paths:       paths,
			Name:        repo,
			Branch:      branch,
		},
	}, token, c.TokenSource, globals.Version)

	createResp, err := cl.CreateCacheEntry(ctx, req)
	if err != nil {
		if connect.CodeOf(err) == connect.CodeAlreadyExists {
			log.Info().Msg("cache entry found with matching sha256sum")
			return nil
		}
		return fmt.Errorf("failed to create cache entry: %w", err)
	}

	log.Info().Str("id", createResp.Msg.Id).Msg("creating cache entry")

	upl := uploader.NewUploader(ctx, fileInfo.ArchivePath, toUploadInstructions(createResp.Msg.UploadInstructions), 20)

	etags, err := upl.Upload(ctx)
	if err != nil {
		return fmt.Errorf("failed to upload: %w", err)
	}

	updateReq := newAuthenticatedProviderRequest(&v1.UpdateCacheEntryRequest{
		Id:             createResp.Msg.Id,
		Name:           repo,
		Branch:         branch,
		Key:            c.Key,
		MultipartEtags: toEtagsV1(etags),
	}, token, c.TokenSource, globals.Version)

	updateResp, err := cl.UpdateCacheEntry(ctx, updateReq)
	if err != nil {
		return fmt.Errorf("failed to update cache entry: %w", err)
	}

	log.Info().Str("id", updateResp.Msg.Id).Int("parts", len(etags)).Msg("updated cache entry")

	return nil
}
func checkPath(path string) ([]string, error) {
	paths := strings.Fields(path)
	if len(paths) == 0 {
		return nil, fmt.Errorf("no paths provided")
	}

	return paths, nil
}

func newAuthenticatedProviderRequest[T any](msg *T, token, provider, version string) *connect.Request[T] {
	req := &connect.Request[T]{
		Msg: msg,
	}

	req.Header().Set("Authorization", fmt.Sprintf("Bearer %s", token))
	req.Header().Set("X-Provider", provider)
	req.Header().Set("User-Agent", fmt.Sprintf("zipstash/%s", version))

	return req
}

func toUploadInstructions(instructions []*v1.CacheUploadInstruction) []uploader.CacheUploadInstruction {
	uploadInstructions := make([]uploader.CacheUploadInstruction, len(instructions))
	for i, instruction := range instructions {
		ui := uploader.CacheUploadInstruction{
			Method: instruction.Method,
			Url:    instruction.Url,
		}

		if instruction.Offset != nil {
			ui.Offset = &uploader.Offset{
				Start: instruction.Offset.Start,
				End:   instruction.Offset.End,
				Part:  instruction.Offset.Part,
			}
		}

		uploadInstructions[i] = ui
	}

	return uploadInstructions
}

func toEtagsV1(etags []uploader.CachePartETag) []*v1.CachePartETag {
	etagV1 := make([]*v1.CachePartETag, len(etags))
	for i, etag := range etags {
		etagV1[i] = &v1.CachePartETag{
			Etag:     etag.Etag,
			Part:     etag.Part,
			PartSize: etag.PartSize,
		}
	}
	return etagV1
}
