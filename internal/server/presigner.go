package server

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"

	"github.com/wolfeidau/cache-service/pkg/api"
)

const (
	MinUploadPartSize int64         = 1024 * 1024 * 5 // 5 MB
	DefaultExpiration time.Duration = 60 * time.Minute
)

type Presigner struct {
	presignS3Client *s3.PresignClient
	cfg             Config
}

func NewPresigner(s3client *s3.Client, cfg Config) *Presigner {
	presignS3Client := s3.NewPresignClient(s3client)

	return &Presigner{
		presignS3Client: presignS3Client,
		cfg:             cfg,
	}
}

func (p *Presigner) GenerateFileUploadInstructions(ctx context.Context, key string, totalSize int64) ([]api.CacheUploadInstruction, error) {
	// minimum multipart upload part size is 5 MB
	// https://docs.aws.amazon.com/AmazonS3/latest/userguide/qfacts.html
	if totalSize < MinUploadPartSize {
		req, err := p.presignS3Client.PresignPutObject(ctx, &s3.PutObjectInput{
			Bucket: aws.String(p.cfg.CacheBucket),
			Key:    aws.String(key),
		}, func(opts *s3.PresignOptions) {
			opts.Expires = DefaultExpiration
		})

		if err != nil {
			return nil, fmt.Errorf("failed to presign upload: %w", err)
		}

		return []api.CacheUploadInstruction{
			{
				Url:    req.URL,
				Method: http.MethodPut,
			},
		}, nil
	}

	// Maximum multipart upload part size is 5 GB
	// Maximum number of parts per upload is 10,000
	offsets := calculateOffsets(totalSize, MinUploadPartSize)
	reqs := make([]api.CacheUploadInstruction, 0, len(offsets))

	for _, offset := range offsets {
		req, err := p.presignS3Client.PresignUploadPart(ctx, &s3.UploadPartInput{
			Bucket: aws.String(p.cfg.CacheBucket),
			Key:    aws.String(key),
		}, func(opts *s3.PresignOptions) {
			opts.Expires = DefaultExpiration
		})
		if err != nil {
			return nil, fmt.Errorf("failed to presign upload: %w", err)
		}
		reqs = append(reqs, api.CacheUploadInstruction{
			Url:    req.URL,
			Method: http.MethodPut,
			Offset: &api.Offset{
				Part:  offset.Part,
				Start: offset.Start,
				End:   offset.End,
			},
		})
	}

	return reqs, nil
}

func (p *Presigner) GenerateFileDownloadInstructions(ctx context.Context, key string, totalSize int64) ([]api.CacheDownloadInstruction, error) {
	// minimum multipart upload part size is 5 MB
	// https://docs.aws.amazon.com/AmazonS3/latest/userguide/qfacts.html
	if totalSize < MinUploadPartSize {
		req, err := p.presignS3Client.PresignGetObject(ctx, &s3.GetObjectInput{
			Bucket: aws.String(p.cfg.CacheBucket),
			Key:    aws.String(key),
		}, func(opts *s3.PresignOptions) {
			opts.Expires = DefaultExpiration
		})

		if err != nil {
			return nil, fmt.Errorf("failed to presign upload: %w", err)
		}
		return []api.CacheDownloadInstruction{
			{
				Url:    req.URL,
				Method: http.MethodGet,
			},
		}, nil
	}

	// Maximum multipart upload part size is 5 GB
	// Maximum number of parts per upload is 10,000
	offsets := calculateOffsets(totalSize, MinUploadPartSize)
	reqs := make([]api.CacheDownloadInstruction, 0, len(offsets))
	for _, offset := range offsets {
		req, err := p.presignS3Client.PresignGetObject(ctx, &s3.GetObjectInput{
			Bucket: aws.String(p.cfg.CacheBucket),
			Key:    aws.String(key),
			Range:  aws.String(fmt.Sprintf("bytes=%d-%d", offset.Start, offset.End)),
		}, func(opts *s3.PresignOptions) {
			opts.Expires = DefaultExpiration
		})
		if err != nil {
			return nil, fmt.Errorf("failed to presign upload: %w", err)
		}
		reqs = append(reqs, api.CacheDownloadInstruction{
			Url:    req.URL,
			Method: http.MethodGet,
			Offset: &api.Offset{
				Part:  offset.Part,
				Start: offset.Start,
				End:   offset.End,
			},
		})
	}

	return reqs, nil
}

type offset struct {
	Part  int32
	Start int64
	End   int64
}

// calculate the offsets for range queries for a given total size and part size
func calculateOffsets(totalSize int64, partSize int64) []offset {
	offsets := []offset{}
	i := int32(1)
	start := int64(0)
	end := partSize
	for end < totalSize {
		offsets = append(offsets, offset{
			Part:  i,
			Start: start,
			End:   end,
		})
		start = end + 1
		end += partSize
		i++
	}

	// add the last part
	offsets = append(offsets, offset{
		Start: start,
		End:   totalSize,
	})
	return offsets
}
