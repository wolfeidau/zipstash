package server

import (
	"context"
	"errors"
	"path"
	"sort"

	"connectrpc.com/connect"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/rs/zerolog/log"
	"go.opentelemetry.io/otel/trace"

	v1 "github.com/wolfeidau/zipstash/api/zipstash/v1"
)

type ZipStashServiceHandler struct {
	s3client  *s3.Client
	presigner *Presigner
	cfg       Config
}

func NewZipStashServiceHandler(ctx context.Context, cfg Config) *ZipStashServiceHandler {
	s3client := cfg.GetS3Client()
	return &ZipStashServiceHandler{
		s3client:  s3client,
		presigner: NewPresigner(s3client, cfg),
		cfg:       cfg,
	}
}

func (zs *ZipStashServiceHandler) CreateCacheEntry(ctx context.Context, createReq *connect.Request[v1.CreateCacheEntryRequest]) (*connect.Response[v1.CreateCacheEntryResponse], error) {
	span := trace.SpanFromContext(ctx)
	span.SetName("ZipStash.CreateCacheEntry")
	defer span.End()

	prefix := path.Join(createReq.Msg.CacheEntry.Name, createReq.Msg.CacheEntry.Branch)

	log.Info().
		Str("Key", createReq.Msg.CacheEntry.Key).
		Str("Prefix", prefix).
		Str("Bucket", zs.cfg.CacheBucket).
		Int64("FileSize", createReq.Msg.CacheEntry.FileSize).
		Msg("presign upload request")

	uploadInstructs, err := zs.presigner.GenerateFileUploadInstructions(
		ctx,
		path.Join(prefix, createReq.Msg.CacheEntry.Key),
		createReq.Msg.CacheEntry.Sha256Sum,
		createReq.Msg.CacheEntry.Compression,
		createReq.Msg.CacheEntry.FileSize,
	)
	if err != nil {
		log.Error().Err(err).Msg("failed to presign upload")
		return nil, connect.NewError(connect.CodeInternal, errors.New("zipstash.v1.ZipStashService.CreateCacheEntry internal error"))
	}

	return connect.NewResponse(&v1.CreateCacheEntryResponse{
		Id:                 uploadInstructs.MultipartUploadId,
		Multipart:          uploadInstructs.Multipart,
		UploadInstructions: fromUploadInstructions(uploadInstructs.UploadInstructions),
	}), nil
}

func (zs *ZipStashServiceHandler) UpdateCacheEntry(ctx context.Context, updateReq *connect.Request[v1.UpdateCacheEntryRequest]) (*connect.Response[v1.UpdateCacheEntryResponse], error) {
	span := trace.SpanFromContext(ctx)
	span.SetName("ZipStash.UpdateCacheEntry")
	defer span.End()

	prefix := path.Join(updateReq.Msg.Name, updateReq.Msg.Branch)
	s3key := path.Join(prefix, updateReq.Msg.Key)

	log.Info().
		Str("Id", updateReq.Msg.Id).
		Str("Prefix", prefix).
		Str("Key", updateReq.Msg.Key).
		Str("S3Key", s3key).
		Msg("cache entry update request")

	if updateReq.Msg.Id != "" {
		_, err := zs.s3client.CompleteMultipartUpload(ctx, &s3.CompleteMultipartUploadInput{
			Bucket:   aws.String(zs.cfg.CacheBucket),
			Key:      aws.String(s3key),
			UploadId: aws.String(updateReq.Msg.Id),
			MultipartUpload: &types.CompletedMultipartUpload{
				Parts: fromCachePartETagV1(updateReq.Msg.MultipartEtags),
			},
		})
		if err != nil {
			log.Error().Err(err).Msg("failed to complete multipart upload")
			return nil, connect.NewError(connect.CodeInternal, errors.New("zipstash.v1.ZipStashService.UpdateCacheEntry internal error"))
		}
	}

	return connect.NewResponse(&v1.UpdateCacheEntryResponse{
		Id: updateReq.Msg.Id,
	}), nil
}

func (zs *ZipStashServiceHandler) GetCacheEntry(ctx context.Context, getReq *connect.Request[v1.GetCacheEntryRequest]) (*connect.Response[v1.GetCacheEntryResponse], error) {
	span := trace.SpanFromContext(ctx)
	span.SetName("ZipStash.GetCacheEntryByKey")
	defer span.End()

	prefix := path.Join(getReq.Msg.Name, getReq.Msg.Branch)
	s3key := path.Join(prefix, getReq.Msg.Key)

	log.Ctx(ctx).Info().
		Str("Prefix", prefix).
		Str("Key", getReq.Msg.Key).
		Msg("cache entry get request")

	res, err := zs.s3client.HeadObject(ctx, &s3.HeadObjectInput{
		Bucket: aws.String(zs.cfg.CacheBucket),
		Key:    aws.String(s3key),
	})
	if err != nil {
		var nsk *types.NotFound
		if errors.As(err, &nsk) {
			return nil, connect.NewError(connect.CodeNotFound, errors.New("zipstash.v1.ZipStashService.GetCacheEntry not found"))
		}
	}

	downloadInstructs, err := zs.presigner.GenerateFileDownloadInstructions(
		ctx,
		s3key,
		aws.ToInt64(res.ContentLength),
	)
	if err != nil {
		log.Ctx(ctx).Error().Err(err).Msg("failed to presign download")
		return nil, connect.NewError(connect.CodeInternal, errors.New("zipstash.v1.ZipStashService.GetCacheEntry internal error"))
	}

	return connect.NewResponse(&v1.GetCacheEntryResponse{
		Multipart:            downloadInstructs.Multipart,
		DownloadInstructions: fromInstructToDownloadV1(downloadInstructs.DownloadInstructions),
	}), nil
}

func fromUploadInstructions(uploadInstructs []CacheURLInstruction) []*v1.CacheUploadInstruction {
	uploadInstsV1 := make([]*v1.CacheUploadInstruction, len(uploadInstructs))

	for i, uploadInstruction := range uploadInstructs {
		uploadInstV1 := &v1.CacheUploadInstruction{
			Url:    uploadInstruction.Url,
			Method: uploadInstruction.Method,
		}

		if uploadInstruction.Offset != nil {
			uploadInstV1.Offset = &v1.Offset{
				Start: uploadInstruction.Offset.Start,
				End:   uploadInstruction.Offset.End,
				Part:  uploadInstruction.Offset.Part,
			}
		}

		uploadInstsV1[i] = uploadInstV1
	}

	return uploadInstsV1
}

func fromInstructToDownloadV1(instructs []CacheURLInstruction) []*v1.CacheDownloadInstruction {
	res := make([]*v1.CacheDownloadInstruction, len(instructs))

	for i, instruct := range instructs {
		di := &v1.CacheDownloadInstruction{
			Url:    instruct.Url,
			Method: instruct.Method,
		}

		if instruct.Offset != nil {
			di.Offset = &v1.Offset{
				Start: instruct.Offset.Start,
				End:   instruct.Offset.End,
				Part:  instruct.Offset.Part,
			}
		}

		res[i] = di
	}

	return res
}

func fromCachePartETagV1(multipartEtags []*v1.CachePartETag) []types.CompletedPart {
	// sort the parts by part number
	sort.Slice(multipartEtags, func(i, j int) bool {
		return multipartEtags[i].Part < multipartEtags[j].Part
	})

	parts := make([]types.CompletedPart, 0, len(multipartEtags))
	for _, part := range multipartEtags {
		parts = append(parts, types.CompletedPart{
			ETag:       aws.String(part.Etag),
			PartNumber: aws.Int32(part.Part),
		})
	}

	return parts
}
