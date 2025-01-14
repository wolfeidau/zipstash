package server

import (
	"context"
	"errors"
	"path"
	"sort"
	"strings"
	"time"

	"connectrpc.com/connect"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/rs/zerolog/log"
	"go.opentelemetry.io/otel/trace"

	v1 "github.com/wolfeidau/zipstash/api/zipstash/v1"
	"github.com/wolfeidau/zipstash/internal/index"
)

type ZipStashServiceHandler struct {
	s3Client  *s3.Client
	ddbClient *dynamodb.Client
	presigner *Presigner
	store     *index.Store
	cfg       Config
}

func NewZipStashServiceHandler(ctx context.Context, cfg Config) *ZipStashServiceHandler {
	s3Client := cfg.GetS3Client()
	ddbClient := cfg.GetDynamoDBClient()
	return &ZipStashServiceHandler{
		s3Client:  s3Client,
		ddbClient: ddbClient,
		presigner: NewPresigner(s3Client, cfg),
		store: index.MustNewStore(ctx, ddbClient, index.StoreConfig{
			TableName: cfg.CacheIndexTable,
			Create:    cfg.CreateCacheIndexTable,
		}),
		cfg: cfg,
	}
}

func (zs *ZipStashServiceHandler) CreateCacheEntry(ctx context.Context, createReq *connect.Request[v1.CreateCacheEntryRequest]) (*connect.Response[v1.CreateCacheEntryResponse], error) {
	span := trace.SpanFromContext(ctx)
	span.SetName("ZipStash.CreateCacheEntry")
	defer span.End()

	prefix := path.Join(createReq.Msg.CacheEntry.Name, createReq.Msg.CacheEntry.Branch)
	s3key := path.Join(prefix, createReq.Msg.CacheEntry.Key)

	log.Info().
		Str("Key", createReq.Msg.CacheEntry.Key).
		Str("Prefix", prefix).
		Str("Bucket", zs.cfg.CacheBucket).
		Str("Sha256Sum", createReq.Msg.CacheEntry.Sha256Sum).
		Int64("FileSize", createReq.Msg.CacheEntry.FileSize).
		Msg("presign upload request")

	// does the cache entry already exist?
	exists, cacheRec, err := zs.store.Exists(ctx, s3key)
	if err != nil {
		log.Error().Err(err).Msg("failed to check if cache entry exists")
		return nil, connect.NewError(connect.CodeInternal, errors.New("zipstash.v1.ZipStashService.CreateCacheEntry internal error"))
	}

	if exists {
		log.Info().Msg("cache entry already exists")
		if cacheRec.Sha256 == createReq.Msg.CacheEntry.Sha256Sum {
			log.Info().Msg("cache entry already exists with matching sha256sum")
			return nil, connect.NewError(connect.CodeAlreadyExists, errors.New("zipstash.v1.ZipStashService.CreateCacheEntry cache entry already exists"))
		}
	}

	// create/update the cache entry in the cache index
	// TODO: should we store this record after generating the upload instructions? So we can put the upload ID in the record and validate the upload ID when the upload is complete?
	err = zs.store.Put(ctx, s3key, index.CacheRecord{
		ID:     createReq.Msg.CacheEntry.Key,
		Paths:  strings.Join(createReq.Msg.CacheEntry.Paths, "\n"),
		Name:   createReq.Msg.CacheEntry.Name,
		Branch: createReq.Msg.CacheEntry.Branch,
		// TokenSource: createReq.Msg.CacheEntry.TokenSource,
		Sha256:    createReq.Msg.CacheEntry.Sha256Sum,
		FileSize:  createReq.Msg.CacheEntry.FileSize,
		UpdatedAt: time.Now(),
	})
	if err != nil {
		log.Error().Err(err).Msg("failed to create cache entry")
		return nil, connect.NewError(connect.CodeInternal, errors.New("zipstash.v1.ZipStashService.CreateCacheEntry internal error"))
	}

	uploadInstructs, err := zs.presigner.GenerateFileUploadInstructions(
		ctx,
		s3key,
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

	// does the cache entry exist?
	exists, _, err := zs.store.Exists(ctx, s3key)
	if err != nil {
		log.Error().Err(err).Msg("failed to check if cache entry exists")
		return nil, connect.NewError(connect.CodeInternal, errors.New("zipstash.v1.ZipStashService.CreateCacheEntry internal error"))
	}

	if !exists {
		log.Info().Msg("cache entry does not exist")
		return nil, connect.NewError(connect.CodeNotFound, errors.New("zipstash.v1.ZipStashService.UpdateCacheEntry cache entry does not exist"))
	}

	log.Info().
		Str("Id", updateReq.Msg.Id).
		Str("Prefix", prefix).
		Str("Key", updateReq.Msg.Key).
		Str("S3Key", s3key).
		Msg("cache entry update request")

	if updateReq.Msg.Id != "" {
		_, err := zs.s3Client.CompleteMultipartUpload(ctx, &s3.CompleteMultipartUploadInput{
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

	log.Info().
		Str("Prefix", prefix).
		Str("Key", getReq.Msg.Key).
		Msg("cache entry get request")

	// does the cache entry exist?
	exists, _, err := zs.store.Exists(ctx, s3key)
	if err != nil {
		log.Error().Err(err).Msg("failed to check if cache entry exists")
		return nil, connect.NewError(connect.CodeInternal, errors.New("zipstash.v1.ZipStashService.CreateCacheEntry internal error"))
	}

	if !exists {
		log.Info().Msg("cache entry does not exist")
		return nil, connect.NewError(connect.CodeNotFound, errors.New("zipstash.v1.ZipStashService.UpdateCacheEntry cache entry does not exist"))
	}

	exists, res, err := zs.exists(ctx, getReq.Msg.Name, getReq.Msg.Branch, getReq.Msg.Key)
	if err != nil {
		log.Error().Err(err).Msg("failed to get cache entry")
		return nil, connect.NewError(connect.CodeInternal, errors.New("zipstash.v1.ZipStashService.GetCacheEntry internal error"))
	}

	if !exists {
		return nil, connect.NewError(connect.CodeNotFound, errors.New("zipstash.v1.ZipStashService.GetCacheEntry not found"))
	}

	log.Info().
		Str("Prefix", prefix).
		Str("Key", getReq.Msg.Key).
		Str("sha256sum", aws.ToString(res.ChecksumSHA256)).Msg("cache entry found")

	downloadInstructs, err := zs.presigner.GenerateFileDownloadInstructions(
		ctx,
		s3key,
		aws.ToInt64(res.ContentLength),
	)
	if err != nil {
		log.Error().Err(err).Msg("failed to presign download")
		return nil, connect.NewError(connect.CodeInternal, errors.New("zipstash.v1.ZipStashService.GetCacheEntry internal error"))
	}

	return connect.NewResponse(&v1.GetCacheEntryResponse{
		CacheEntry: &v1.CacheEntry{
			Key:      getReq.Msg.Key,
			Name:     getReq.Msg.Name,
			Branch:   getReq.Msg.Branch,
			FileSize: aws.ToInt64(res.ContentLength),
		},
		Multipart:            downloadInstructs.Multipart,
		DownloadInstructions: fromInstructToDownloadV1(downloadInstructs.DownloadInstructions),
	}), nil
}

func (zs *ZipStashServiceHandler) exists(ctx context.Context, name, branch, key string) (bool, *s3.HeadObjectOutput, error) {
	span := trace.SpanFromContext(ctx)
	span.SetName("ZipStash.exists")
	defer span.End()

	prefix := path.Join(name, branch)
	s3key := path.Join(prefix, key)
	res, err := zs.s3Client.HeadObject(ctx, &s3.HeadObjectInput{
		Bucket: aws.String(zs.cfg.CacheBucket),
		Key:    aws.String(s3key),
	})
	if err != nil {
		var nsk *types.NotFound
		if errors.As(err, &nsk) {
			return false, nil, nil
		}
		log.Ctx(ctx).Error().Err(err).Msg("failed to get cache entry")
		return false, nil, err
	}

	return true, res, nil
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
