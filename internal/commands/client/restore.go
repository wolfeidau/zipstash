package client

import (
	"cmp"
	"context"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"

	"connectrpc.com/connect"
	"github.com/rs/zerolog/log"
	"go.opentelemetry.io/otel/attribute"
	"golang.org/x/exp/slices"

	v1 "github.com/wolfeidau/zipstash/api/gen/proto/go/cache/v1"
	"github.com/wolfeidau/zipstash/pkg/archive"
	"github.com/wolfeidau/zipstash/pkg/downloader"
	"github.com/wolfeidau/zipstash/pkg/tokens"
	"github.com/wolfeidau/zipstash/pkg/trace"
)

type RestoreCmd struct {
	Key         string `help:"key to use for the cache entry" required:"" env:"INPUT_KEY"`
	Path        string `help:"Path list for a cache entry." env:"INPUT_PATH"`
	TokenSource string `help:"token source" default:"github_actions" env:"INPUT_TOKEN_SOURCE"`
	Clean       bool   `help:"clean the path before restore" env:"INPUT_CLEAN"`
	GitHub      GitHub `embed:"" prefix:"github_"`
	Local       Local  `embed:"" prefix:"local_"`
}

func (c *RestoreCmd) Run(ctx context.Context, globals *Globals) error {
	ctx, span := trace.Start(ctx, "RestoreCmd.Run")
	defer span.End()

	span.SetAttributes(
		attribute.String("key", c.Key),
		attribute.String("path", c.Path),
		attribute.Bool("clean", c.Clean),
		attribute.String("token_source", c.TokenSource),
	)

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

	req := newAuthenticatedProviderRequest(&v1.GetEntryRequest{
		Key:      c.Key,
		Name:     repo,
		Branch:   branch,
		Provider: convertProviderV1(c.TokenSource),
	}, token, c.TokenSource, globals.Version)

	getEntryResp, err := cl.GetEntry(ctx, req)
	if err != nil {
		if connect.CodeOf(err) == connect.CodeNotFound {
			log.Info().Msg("cache entry not found")
			return nil
		}
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

	if c.Clean {
		for _, path := range paths {
			path, err := archive.ResolveHomeDir(path)
			if err != nil {
				return fmt.Errorf("failed to resolve home dir: %w", err)
			}

			log.Info().Str("path", path).Msg("cleaning path")
			err = removeAll(path)
			if err != nil {
				return fmt.Errorf("failed to clean path: %w", err)
			}
		}
	}

	log.Info().Strs("paths", paths).Msg("extracting files")

	err = archive.ExtractFiles(ctx, zipFile, zipFileLen, paths)
	if err != nil {
		return fmt.Errorf("failed to restore files: %w", err)
	}

	// cleanup zip file
	defer os.Remove(zipFile.Name())

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
		n, err := appendToFileAndRemove(zipFile, d.FilePath)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to write file: %w", err)
		}
		zipFileLen += n
	}

	span.SetAttributes(attribute.Int64("zipFileLen", zipFileLen))

	return zipFile, zipFileLen, nil
}

func appendToFileAndRemove(f *os.File, path string) (int64, error) {
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

// RemoveAll removes a directory written by Download or Unzip, first applying
// any permission changes needed to do so.
func removeAll(dir string) error {
	log.Info().Str("path", dir).Msg("changing permissions")

	// Module cache has 0555 directories; make them writable in order to remove content.
	err := filepath.WalkDir(dir, func(path string, info fs.DirEntry, err error) error {
		if err != nil {
			return nil // ignore errors walking in file system
		}
		if info.IsDir() {
			return os.Chmod(path, 0777)
		}
		return nil
	})
	if err != nil {
		return err
	}
	return os.RemoveAll(dir)
}
