package server

import (
	"context"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"net/http"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/rs/zerolog/log"

	"github.com/wolfeidau/zipstash/pkg/trace"
)

const (
	MinPartSize       int64         = 10 * 1024 * 1024 // 5MB minimum
	DefaultExpiration time.Duration = 60 * time.Minute
)

type Presigner struct {
	s3client        *s3.Client
	presignS3Client *s3.PresignClient
	cacheBucket     string
}

func NewPresigner(s3client *s3.Client, cacheBucket string) *Presigner {
	presignS3Client := s3.NewPresignClient(s3client)

	return &Presigner{
		s3client:        s3client,
		presignS3Client: presignS3Client,
		cacheBucket:     cacheBucket,
	}
}

// GenerateFileUploadInstructions generates the necessary instructions for uploading a file to S3, including presigned URLs and multipart upload details.
// If the file size is less than the minimum multipart upload part size, a single presigned PUT URL is returned.
// Otherwise, the function calculates the necessary offsets for a multipart upload and returns the presigned URLs for each part.
func (p *Presigner) GenerateFileUploadInstructions(ctx context.Context, s3key, sha256sum, compression string, totalSize int64) (*UploadInstructionsResp, error) {
	ctx, span := trace.Start(ctx, "Presigner.GenerateFileUploadInstructions")
	defer span.End()

	// minimum multipart upload part size is 5 MB
	// https://docs.aws.amazon.com/AmazonS3/latest/userguide/qfacts.html
	if totalSize < MinPartSize {
		req, err := p.presignS3Client.PresignPutObject(ctx, &s3.PutObjectInput{
			Bucket:         aws.String(p.cacheBucket),
			Key:            aws.String(s3key),
			ChecksumSHA256: aws.String(convertSha256ToBase64(sha256sum)),
			ContentType:    aws.String(compressionToContentType(compression)),
		}, func(opts *s3.PresignOptions) {
			opts.Expires = DefaultExpiration
		})

		if err != nil {
			return nil, fmt.Errorf("failed to presign upload: %w", err)
		}

		return &UploadInstructionsResp{
			Multipart: false,
			UploadInstructions: []CacheURLInstruction{
				{
					Url:    req.URL,
					Method: http.MethodPut,
				},
			},
		}, nil
	}

	createMultiResp, err := p.s3client.CreateMultipartUpload(ctx, &s3.CreateMultipartUploadInput{
		Bucket:      aws.String(p.cacheBucket),
		Key:         aws.String(s3key),
		ContentType: aws.String(compressionToContentType(compression)),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create multipart upload: %w", err)
	}

	log.Info().Int64("totalSize", totalSize).Int64("MinPartSize", MinPartSize).Msg("multipart upload")

	// Maximum multipart upload part size is 5 GB
	// Maximum number of parts per upload is 10,000
	offsets := calculateOffsets(totalSize, MinPartSize)
	reqs := make([]CacheURLInstruction, 0, len(offsets))

	for _, offset := range offsets {
		req, err := p.presignS3Client.PresignUploadPart(ctx, &s3.UploadPartInput{
			Bucket:         aws.String(p.cacheBucket),
			Key:            aws.String(s3key),
			PartNumber:     aws.Int32(offset.Part),
			UploadId:       createMultiResp.UploadId,
			ChecksumSHA256: aws.String(convertSha256ToBase64(sha256sum)),
		}, func(opts *s3.PresignOptions) {
			opts.Expires = DefaultExpiration
		})
		if err != nil {
			return nil, fmt.Errorf("failed to presign upload: %w", err)
		}
		reqs = append(reqs, CacheURLInstruction{
			Url:    req.URL,
			Method: http.MethodPut,
			Offset: offset,
		})
	}

	return &UploadInstructionsResp{
		UploadInstructions: reqs,
		Multipart:          true,
		MultipartUploadId:  createMultiResp.UploadId,
	}, nil
}

// GenerateFileDownloadInstructions generates the necessary instructions for downloading a file from the S3 cache.
// If the file size is less than the minimum multipart upload part size, it generates a single download instruction.
// Otherwise, it generates multiple download instructions for downloading the file in parts.
// The returned instructions include the presigned URLs and HTTP methods to use for the downloads.
func (p *Presigner) GenerateFileDownloadInstructions(ctx context.Context, s3key string, totalSize int64) (*DownloadInstructionsResp, error) {
	ctx, span := trace.Start(ctx, "Presigner.GenerateFileDownloadInstructions")
	defer span.End()

	// minimum multipart upload part size is 5 MB
	// https://docs.aws.amazon.com/AmazonS3/latest/userguide/qfacts.html
	if totalSize < MinPartSize {
		req, err := p.presignS3Client.PresignGetObject(ctx, &s3.GetObjectInput{
			Bucket: aws.String(p.cacheBucket),
			Key:    aws.String(s3key),
		}, func(opts *s3.PresignOptions) {
			opts.Expires = DefaultExpiration
		})

		if err != nil {
			return nil, fmt.Errorf("failed to presign upload: %w", err)
		}
		return &DownloadInstructionsResp{
			DownloadInstructions: []CacheURLInstruction{
				{
					Url:    req.URL,
					Method: http.MethodGet,
				},
			},
		}, nil
	}

	// Maximum multipart upload part size is 5 GB
	// Maximum number of parts per upload is 10,000
	offsets := calculateOffsets(totalSize, MinPartSize)
	reqs := make([]CacheURLInstruction, 0, len(offsets))
	for _, offset := range offsets {
		req, err := p.presignS3Client.PresignGetObject(ctx, &s3.GetObjectInput{
			Bucket: aws.String(p.cacheBucket),
			Key:    aws.String(s3key),
			Range:  aws.String(fmt.Sprintf("bytes=%d-%d", offset.Start, offset.End)),
		}, func(opts *s3.PresignOptions) {
			opts.Expires = DefaultExpiration
		})
		if err != nil {
			return nil, fmt.Errorf("failed to presign upload: %w", err)
		}
		reqs = append(reqs, CacheURLInstruction{
			Url:    req.URL,
			Method: http.MethodGet,
			Offset: offset,
		})
	}

	return &DownloadInstructionsResp{
		DownloadInstructions: reqs,
		Multipart:            true,
	}, nil
}

type Offset struct {
	Part  int32
	Start int64
	End   int64
}

type CacheURLInstruction struct {
	Offset *Offset
	Url    string
	Method string
}

type UploadInstructionsResp struct {
	MultipartUploadId  *string
	UploadInstructions []CacheURLInstruction
	Multipart          bool
}

type DownloadInstructionsResp struct {
	DownloadInstructions []CacheURLInstruction
	Multipart            bool
}

// calculateOffsets calculates the offsets for range queries for a given total size and part size.
// It returns a slice of offset structs, where each offset represents a part of the total size
// that can be downloaded or uploaded separately.
// The part number starts at 1 in S3 multipart uploads.
// The last part may have a different size than the other parts.
func calculateOffsets(totalSize int64, partSize int64) []*Offset {
	offsets := make([]*Offset, 0, (totalSize/partSize)+1)
	i := int32(1) // part number starts at 1 in s3 multipart uploads
	start := int64(0)
	end := partSize
	for end < totalSize {
		offsets = append(offsets, &Offset{
			Part:  i,
			Start: start,
			End:   end,
		})

		start = end + 1
		end += partSize
		i++
	}

	end = totalSize - 1

	// add the last part
	offsets = append(offsets, &Offset{
		Part:  i,
		Start: start,
		End:   end,
	})

	return offsets
}

func compressionToContentType(compression string) string {
	switch compression {
	case "zip":
		return "application/zip"
	default:
		return "application/octet-stream"
	}
}

func convertSha256ToBase64(sha256 string) string {
	decoded, err := hex.DecodeString(sha256)
	if err != nil {
		return ""
	}
	return base64.StdEncoding.EncodeToString(decoded)
}
