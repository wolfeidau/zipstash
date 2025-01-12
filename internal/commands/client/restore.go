package client

import (
	"cmp"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/klauspost/compress/zip"
	"github.com/rs/zerolog/log"
	"github.com/wolfeidau/quickzip"
	"go.opentelemetry.io/otel/attribute"
	"golang.org/x/exp/slices"

	v1 "github.com/wolfeidau/zipstash/api/zipstash/v1"
	"github.com/wolfeidau/zipstash/pkg/archive"
	"github.com/wolfeidau/zipstash/pkg/downloader"
	"github.com/wolfeidau/zipstash/pkg/tokens"
	"github.com/wolfeidau/zipstash/pkg/trace"
)

type RestoreCmd struct {
	Key         string `help:"key to use for the cache entry" required:"" env:"INPUT_KEY"`
	Path        string `help:"Path list for a cache entry." env:"INPUT_PATH"`
	TokenSource string `help:"token source" default:"github_actions" env:"INPUT_TOKEN_SOURCE"`
	GitHub      GitHub `embed:"" prefix:"github_"`
	Local       Local  `embed:"" prefix:"local_"`
}

func (c *RestoreCmd) Run(ctx context.Context, globals *Globals) error {
	ctx, span := trace.Start(ctx, "RestoreCmd.Run")
	defer span.End()

	return c.restore(ctx, globals)
}

func (c *RestoreCmd) restore(ctx context.Context, globals *Globals) error {
	ctx, span := trace.Start(ctx, "RestoreCmd.restore")
	defer span.End()

	cl := globals.Client

	repo, branch, err := getRepoAndBranch(c.GitHub, c.Local)
	if err != nil {
		return fmt.Errorf("failed to get repo and branch: %w", err)
	}

	token, err := tokens.GetToken(ctx, c.TokenSource, audience, nil)
	if err != nil {
		return fmt.Errorf("failed to get token: %w", err)
	}

	req := newAuthenticatedProviderRequest(&v1.GetCacheEntryRequest{
		Key:      c.Key,
		Name:     repo,
		Branch:   branch,
		Provider: convertProviderV1(c.TokenSource),
	}, token, c.TokenSource, globals.Version)

	getEntryResp, err := cl.GetCacheEntry(ctx, req)
	if err != nil {
		return fmt.Errorf("failed to get cache entry: %w", err)
	}

	log.Info().
		Str("key", getEntryResp.Msg.CacheEntry.Key).
		Str("compression", getEntryResp.Msg.CacheEntry.Compression).
		Int64("size", getEntryResp.Msg.CacheEntry.FileSize).
		Msg("cache entry")

	downloads, err := downloader.NewDownloader(
		convertToDownloadInstructions(getEntryResp.Msg.DownloadInstructions),
		20,
	).Download(ctx)
	if err != nil {
		return fmt.Errorf("failed to download cache entry: %w", err)
	}

	slices.SortFunc(downloads, func(a, b downloader.DownloadedFile) int {
		return cmp.Compare(a.Part, b.Part)
	})

	for _, d := range downloads {
		log.Debug().
			Int("part", d.Part).
			Str("etag", d.ETag).
			Msg("download")
	}

	zipFile, zipFileLen, err := combineParts(ctx, downloads)
	if err != nil {
		return fmt.Errorf("failed to create temp file: %w", err)
	}
	defer zipFile.Close()

	log.Info().Int64("zipFileLen", zipFileLen).Str("name", zipFile.Name()).Msg("zip file len")

	paths, err := checkPath(c.Path)
	if err != nil {
		return fmt.Errorf("failed to check path: %w", err)
	}

	err = restoreFiles(ctx, zipFile, zipFileLen, paths)
	if err != nil {
		return fmt.Errorf("failed to restore files: %w", err)
	}

	// cleanup zip file
	defer os.Remove(zipFile.Name())

	return nil
}

func restoreFiles(ctx context.Context, zipFile *os.File, zipFileLen int64, paths []string) error {
	_, span := trace.Start(ctx, "restoreFiles")
	defer span.End()
	extract, err := quickzip.NewExtractorFromReader(zipFile, zipFileLen)
	if err != nil {
		return fmt.Errorf("failed to create extractor: %w", err)
	}

	mappings, err := archive.PathsToMappings(paths)
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

// pass in a list of paths and turn them into a zip file stream to enable extraction
func combineParts(ctx context.Context, downloads []downloader.DownloadedFile) (*os.File, int64, error) {
	_, span := trace.Start(ctx, "combineParts")
	defer span.End()

	zipFile, err := os.CreateTemp("", "zipstash-download-*.zip")
	if err != nil {
		return nil, 0, fmt.Errorf("failed to create temp file: %w", err)
	}

	zipFileLen := int64(0)

	for _, d := range downloads {
		n, err := appendToFile(zipFile, d.FilePath)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to write file: %w", err)
		}
		zipFileLen += n
	}

	span.SetAttributes(attribute.Int64("zipFileLen", zipFileLen))

	return zipFile, zipFileLen, nil
}

func appendToFile(f *os.File, path string) (int64, error) {
	pf, err := os.Open(filepath.Clean(path))
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

func convertToDownloadInstructions(instructs []*v1.CacheDownloadInstruction) []downloader.CacheDownloadInstruction {
	res := make([]downloader.CacheDownloadInstruction, len(instructs))

	for i, downloadInstruct := range instructs {

		cdi := downloader.CacheDownloadInstruction{
			Method: downloadInstruct.Method,
			Url:    downloadInstruct.Url,
		}

		if downloadInstruct.Offset != nil {
			cdi.Offset = &downloader.Offset{
				Start: downloadInstruct.Offset.Start,
				End:   downloadInstruct.Offset.End,
				Part:  downloadInstruct.Offset.Part,
			}
		}

		res[i] = cdi
	}

	return res
}

func convertProviderV1(tokenSource string) v1.Provider {
	switch tokenSource {
	case "github_actions":
		return v1.Provider_PROVIDER_GITHUB_ACTIONS
	case "buildkite":
		return v1.Provider_PROVIDER_BUILDKITE
	case "gitlab":
		return v1.Provider_PROVIDER_GITLAB
	default:
		return v1.Provider_PROVIDER_UNSPECIFIED
	}
}
