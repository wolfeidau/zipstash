package uploader

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/cenkalti/backoff/v5"
	"github.com/rs/zerolog/log"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"

	"github.com/wolfeidau/zipstash/pkg/trace"
)

// CachePartETag Part index and ETag
type CachePartETag struct {
	// Etag ETag
	Etag string `json:"etag"`

	// Part Part index
	Part     int32 `json:"part"`
	PartSize int64 `json:"part_size"`
}

// CacheUploadInstruction defines model for CacheUploadInstruction.
type CacheUploadInstruction struct {
	// Method HTTP method
	Method string  `json:"method"`
	Offset *Offset `json:"offset,omitempty"`

	// Url URL
	Url string `json:"url"`
}

// Offset defines model for Offset.
type Offset struct {
	// End End position of the part
	End int64 `json:"end"`

	// Part Part number
	Part int32 `json:"part"`

	// Start Start position of the part
	Start int64 `json:"start"`
}

// Uploader uses go routines to upload files in parallel with a limit of 20 concurrent uploads.
// It is provided a list of URLs to upload and a channel to receive the results.
// The results are sent to the channel as a map of URL to error.
// If an error occurs, the error is sent to the channel and the upload is stopped.
// The uploader will continue to upload files until the channel is closed.
// The uploader will return when all uploads are complete or when an error occurs.
type Uploader struct {
	client          *http.Client
	errors          chan error
	done            chan struct{}
	filePath        string
	uploadInstructs []CacheUploadInstruction
	limit           int
}

func NewUploader(ctx context.Context, filePath string, uploadInstructs []CacheUploadInstruction, limit int) *Uploader {
	return &Uploader{
		filePath:        filePath,
		uploadInstructs: uploadInstructs,
		client:          &http.Client{Transport: otelhttp.NewTransport(http.DefaultTransport)},
		limit:           limit,
		errors:          make(chan error),
		done:            make(chan struct{}),
	}
}

func (u *Uploader) Upload(ctx context.Context) ([]CachePartETag, error) {
	ctx, span := trace.Start(ctx, "Uploader.Upload")
	defer span.End()

	var mu sync.Mutex
	etags := make([]CachePartETag, 0, len(u.uploadInstructs))

	wg := sync.WaitGroup{}
	wg.Add(len(u.uploadInstructs))
	sem := make(chan struct{}, u.limit)
	start := time.Now()

	for _, uploadInstruct := range u.uploadInstructs {
		sem <- struct{}{}
		go func(uploadInstruct CacheUploadInstruction) {
			defer func() {
				<-sem
				wg.Done()
			}()
			etag, err := u.upload(ctx, uploadInstruct)
			if err != nil {
				u.errors <- err
				return
			}
			// Safely collect the etag
			mu.Lock()
			etags = append(etags, etag)
			mu.Unlock()
		}(uploadInstruct)
	}

	go func() {
		wg.Wait()
		close(u.done)
	}()

	select {
	case <-u.done:
		emitSummary(etags, start)
		return etags, nil
	case err := <-u.errors:
		return nil, err
	}
}

func (u *Uploader) upload(ctx context.Context, uploadInstruct CacheUploadInstruction) (CachePartETag, error) {
	ctx, span := trace.Start(ctx, "Uploader.upload")
	defer span.End()
	size := int64(0)
	multipart := uploadInstruct.Offset != nil

	if multipart {
		size = uploadInstruct.Offset.End - uploadInstruct.Offset.Start + 1
	}
	var cachePartEtag CachePartETag

	chunk, err := u.readChunk(ctx, size, uploadInstruct)
	if err != nil {
		return cachePartEtag, fmt.Errorf("failed to read chunk: %w", err)
	}

	etag, err := u.uploadChunk(ctx, uploadInstruct, chunk)
	if err != nil {
		return cachePartEtag, fmt.Errorf("failed to upload chunk: %w", err)
	}

	cachePartEtag.Etag = etag
	cachePartEtag.Part = 1
	cachePartEtag.PartSize = int64(len(chunk))
	if multipart {
		cachePartEtag.Part = uploadInstruct.Offset.Part
	}

	log.Debug().Str("etag", etag).Int64("size", int64(len(chunk))).Int32("part", cachePartEtag.Part).Msg("uploaded")

	return cachePartEtag, nil
}

func (u *Uploader) readChunk(ctx context.Context, size int64, uploadInstruct CacheUploadInstruction) ([]byte, error) {
	_, span := trace.Start(ctx, "Uploader.readChunk")
	defer span.End()

	file, err := os.Open(u.filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	if uploadInstruct.Offset == nil {
		// read the entire file
		return os.ReadFile(u.filePath)
	}

	// TODO: check if offsets are valid
	log.Debug().Int32("part", uploadInstruct.Offset.Part).Int64("size", size).Int64("start", uploadInstruct.Offset.Start).Int64("end", uploadInstruct.Offset.End).Msg("reading chunk")

	buf := make([]byte, size)
	n, err := file.ReadAt(buf, uploadInstruct.Offset.Start)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	// verify the size of the read
	if int64(n) != size {
		return nil, fmt.Errorf("read size mismatch: got %d, expected %d", n, size)
	}

	return buf, nil
}

func (u *Uploader) uploadChunk(ctx context.Context, uploadInstruct CacheUploadInstruction, chunk []byte) (string, error) {
	ctx, span := trace.Start(ctx, "Uploader.uploadChunk")
	defer span.End()

	operation := func() (string, error) {
		uploadReq, err := http.NewRequestWithContext(ctx, uploadInstruct.Method, uploadInstruct.Url, bytes.NewBuffer(chunk))
		if err != nil {
			return "", fmt.Errorf("failed to create request: %w", err)
		}

		resp, err := u.client.Do(uploadReq)
		if err != nil {
			return "", fmt.Errorf("failed to do upload file: %w", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode == http.StatusBadRequest ||
			resp.StatusCode == http.StatusForbidden ||
			resp.StatusCode == http.StatusLengthRequired {

			// // read the body and log it if there is an error
			// if body, err := io.ReadAll(resp.Body); err == nil {
			// 	log.Error().Str("body", string(body)).Msg("failed to upload file")
			// }

			return "", backoff.Permanent(fmt.Errorf("failed to upload file: %s", resp.Status))
		}

		if resp.StatusCode != http.StatusOK {
			return "", fmt.Errorf("failed to upload file: %s", resp.Status)
		}

		return resp.Header.Get("ETag"), nil
	}

	return backoff.Retry(ctx, operation,
		backoff.WithBackOff(backoff.NewExponentialBackOff()), backoff.WithMaxTries(3))
}

func emitSummary(etags []CachePartETag, start time.Time) {
	since := time.Since(start)

	var totalSize int64
	for _, etag := range etags {
		totalSize += etag.PartSize
	}

	// calculate the average download speed in megabytes per second
	averageSpeed := float64(totalSize) / since.Seconds() / 1024 / 1024

	log.Info().Int64("totalSize", totalSize).Dur("duration_ms", since).Str("transfer_speed", fmt.Sprintf("%.2fMB/s", averageSpeed)).Msg("upload complete")
}
