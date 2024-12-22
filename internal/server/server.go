package server

import (
	"context"
	"errors"
	"net/http"
	"sort"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/rs/zerolog/log"

	"github.com/wolfeidau/cache-service/pkg/api"
)

var _ api.ServerInterface = (*Cache)(nil)

type Cache struct {
	s3client  *s3.Client
	presigner *Presigner
	cfg       Config
}

func NewCache(ctx context.Context, cfg Config) *Cache {
	s3client := cfg.GetS3Client()

	return &Cache{
		s3client:  s3client,
		presigner: NewPresigner(s3client, cfg),
		cfg:       cfg,
	}
}

func (ca *Cache) CreateCacheEntry(c echo.Context, provider string) error {
	ctx := c.Request().Context()

	cacheEntryReq := new(api.CacheEntryCreateRequest)
	err := c.Bind(&cacheEntryReq)
	if err != nil {
		return c.JSON(http.StatusBadRequest, api.Error{
			Message: "invalid request",
		})
	}

	log.Ctx(ctx).Info().
		Str("Key", cacheEntryReq.CacheEntry.Key).
		Str("Bucket", ca.cfg.CacheBucket).
		Msg("presign upload request")

	uploadInstructs, err := ca.presigner.GenerateFileUploadInstructions(ctx, cacheEntryReq.CacheEntry, cacheEntryReq.CacheEntry.FileSize)
	if err != nil {
		log.Ctx(ctx).Error().Err(err).Msg("failed to presign upload")
		return c.JSON(http.StatusInternalServerError, api.Error{
			Message: "failed to create cache entry",
		})
	}

	log.Ctx(ctx).Info().Msg("presigned upload request")

	return c.JSON(http.StatusCreated, api.CacheEntryCreateResponse{
		Id:                 uploadInstructs.MultipartUploadId,
		UploadInstructions: uploadInstructs.UploadInstructions,
		Multipart:          uploadInstructs.Multipart,
	})
}

func (ca *Cache) UpdateCacheEntry(c echo.Context, provider string) error {
	ctx := c.Request().Context()

	cacheEntryReq := new(api.CacheEntryUpdateRequest)
	err := c.Bind(&cacheEntryReq)
	if err != nil {
		return c.JSON(http.StatusBadRequest, api.Error{
			Message: "invalid request",
		})
	}

	log.Ctx(ctx).Info().Any("cacheEntryReq", cacheEntryReq).Msg("cache entry update request")

	// TODO: we need a way to record uploads which can't use multipart uploads
	if cacheEntryReq.Id != "" {

		// sort the parts by part number
		sort.Slice(cacheEntryReq.MultipartEtags, func(i, j int) bool {
			return cacheEntryReq.MultipartEtags[i].Part < cacheEntryReq.MultipartEtags[j].Part
		})

		parts := make([]types.CompletedPart, 0, len(cacheEntryReq.MultipartEtags))
		for _, part := range cacheEntryReq.MultipartEtags {
			log.Info().Int32("part", part.Part).Msg("part")
			parts = append(parts, types.CompletedPart{
				ETag:       aws.String(part.Etag),
				PartNumber: aws.Int32(part.Part),
			})
		}

		res, err := ca.s3client.CompleteMultipartUpload(ctx, &s3.CompleteMultipartUploadInput{
			Bucket:   aws.String(ca.cfg.CacheBucket),
			Key:      aws.String(cacheEntryReq.Key),
			UploadId: aws.String(cacheEntryReq.Id),
			MultipartUpload: &types.CompletedMultipartUpload{
				Parts: parts,
			},
		})
		if err != nil {
			log.Ctx(ctx).Error().Err(err).Msg("failed to update cache entry")
			return c.JSON(http.StatusInternalServerError, api.Error{
				Message: "failed to update cache entry",
			})
		}

		log.Ctx(ctx).Info().Any("res", res).Msg("cache entry updated")
	}

	id := uuid.New().String()

	return c.JSON(http.StatusOK, api.CacheEntryUpdateResponse{
		Id: id,
	})
}

func (ca *Cache) GetCacheEntryByKey(c echo.Context, provider, key string) error {
	ctx := c.Request().Context()

	res, err := ca.s3client.HeadObject(ctx, &s3.HeadObjectInput{
		Bucket: aws.String(ca.cfg.CacheBucket),
		Key:    aws.String(key),
	})
	if err != nil {
		var nsk *types.NoSuchKey
		if errors.As(err, &nsk) {
			return c.JSON(http.StatusNotFound, api.Error{
				Message: "cache entry not found",
			})
		}
		log.Ctx(ctx).Error().Err(err).Msg("failed to get cache entry")
		return c.JSON(http.StatusInternalServerError, api.Error{
			Message: "failed to get cache entry",
		})
	}

	log.Ctx(ctx).Info().Any("res", res).Msg("cache entry retrieved")

	downloadInstructs, err := ca.presigner.GenerateFileDownloadInstructions(ctx, key, aws.ToInt64(res.ContentLength))
	if err != nil {
		log.Ctx(ctx).Error().Err(err).Msg("failed to presign download")
		return c.JSON(http.StatusInternalServerError, api.Error{
			Message: "failed to get cache entry",
		})
	}

	return c.JSON(http.StatusOK, api.CacheEntryGetResponse{
		CacheEntry: api.CacheEntry{
			Key:      key,
			FileSize: aws.ToInt64(res.ContentLength),
		},
		DownloadInstructions: downloadInstructs.DownloadInstructions,
		Multipart:            downloadInstructs.Multipart,
	})
}
