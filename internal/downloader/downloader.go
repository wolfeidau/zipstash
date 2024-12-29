package downloader

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"sync"

	"github.com/cenkalti/backoff/v5"
	"github.com/rs/zerolog/log"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"

	"github.com/wolfeidau/cache-service/internal/trace"
	"github.com/wolfeidau/cache-service/pkg/client"
)

type DownloadedFile struct {
	URL      string
	FilePath string
	ETag     string
	Part     int
	Size     int64
}

// Downloader uses go routines to download parts of a file in parallel with a limit of 20 concurrent uploads.
// It is provided a list of URLs to upload and a channel to receive the results.
type Downloader struct {
	client            *http.Client
	downloadInstructs []client.CacheDownloadInstruction
	limit             int
	errors            chan error
	done              chan struct{}
}

func NewDownloader(downloadInstructs []client.CacheDownloadInstruction, limit int) *Downloader {
	return &Downloader{
		downloadInstructs: downloadInstructs,
		client:            &http.Client{Transport: otelhttp.NewTransport(http.DefaultTransport)},
		limit:             limit,
		errors:            make(chan error),
		done:              make(chan struct{}),
	}
}
func (d *Downloader) Download(ctx context.Context) ([]DownloadedFile, error) {
	ctx, span := trace.Start(ctx, "Downloader.Download")
	defer span.End()

	var mu sync.Mutex
	downloads := make([]DownloadedFile, 0, len(d.downloadInstructs))

	wg := sync.WaitGroup{}
	wg.Add(len(d.downloadInstructs))
	sem := make(chan struct{}, d.limit)

	for _, downloadInstruct := range d.downloadInstructs {
		sem <- struct{}{}
		go func(downloadInstruct client.CacheDownloadInstruction) {
			defer func() {
				<-sem
				wg.Done()
			}()
			download, err := d.download(ctx, downloadInstruct)
			if err != nil {
				d.errors <- err
			}
			// Safely collect the etag
			mu.Lock()
			downloads = append(downloads, download)
			mu.Unlock()
		}(downloadInstruct)
	}

	go func() {
		wg.Wait()
		close(d.done)
	}()

	select {
	case <-d.done:
		return downloads, nil
	case err := <-d.errors:
		return nil, err
	}
}

func (d *Downloader) download(ctx context.Context, downloadInstruct client.CacheDownloadInstruction) (DownloadedFile, error) {
	ctx, span := trace.Start(ctx, "Downloader.download")
	defer span.End()

	operation := func() (DownloadedFile, error) {

		var download DownloadedFile

		downloadReq, err := http.NewRequestWithContext(ctx, downloadInstruct.Method, downloadInstruct.Url, nil)
		if err != nil {
			return download, fmt.Errorf("failed to create request: %w", err)
		}

		downloadReq.Header.Set("Range", fmt.Sprintf("bytes=%d-%d", downloadInstruct.Offset.Start, downloadInstruct.Offset.End))

		resp, err := d.client.Do(downloadReq)
		if err != nil {
			return download, fmt.Errorf("failed to do upload file: %w", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode == http.StatusBadRequest ||
			resp.StatusCode == http.StatusNotFound ||
			resp.StatusCode == http.StatusInternalServerError {
			return download, fmt.Errorf("failed to download file: %s", resp.Status)
		}

		f, err := os.CreateTemp("", fmt.Sprintf("cache-service-download-%06d-*", downloadInstruct.Offset.Part))
		if err != nil {
			return download, fmt.Errorf("failed to create temp file: %w", err)
		}

		defer f.Close()

		_, err = f.ReadFrom(resp.Body)
		if err != nil {
			return download, fmt.Errorf("failed to read response body: %w", err)
		}

		download.Part = int(downloadInstruct.Offset.Part)
		download.URL = downloadInstruct.Url
		download.ETag = resp.Header.Get("ETag")
		download.Size = resp.ContentLength
		download.FilePath = f.Name()

		log.Info().Int("part", download.Part).Str("path", download.FilePath).Int64("size", download.Size).Msg("downloaded file")

		return download, nil
	}

	return backoff.Retry(ctx, operation,
		backoff.WithBackOff(backoff.NewExponentialBackOff()), backoff.WithMaxTries(3))
}
