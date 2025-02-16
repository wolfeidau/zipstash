package server

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"connectrpc.com/connect"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
	"go.opentelemetry.io/otel/attribute"

	v1 "github.com/wolfeidau/zipstash/api/gen/proto/go/cache/v1"
	"github.com/wolfeidau/zipstash/internal/ciauth"
	"github.com/wolfeidau/zipstash/internal/index"
	"github.com/wolfeidau/zipstash/pkg/trace"
)

const (
	cacheRecordInflightTTL = 30 * time.Minute
	cacheRecordTTL         = 7 * 24 * time.Hour
)

type CacheConfig struct {
	GetS3Client S3ClientFunc
	CacheBucket string
}

type S3ClientFunc func() *s3.Client

type CacheServiceHandler struct {
	s3Client  *s3.Client
	presigner *Presigner
	store     *index.Store
	cfg       CacheConfig
}

func NewCacheServiceHandler(ctx context.Context, cfg CacheConfig, store *index.Store) *CacheServiceHandler {
	s3Client := cfg.GetS3Client()
	return &CacheServiceHandler{
		s3Client:  s3Client,
		presigner: NewPresigner(s3Client, cfg.CacheBucket),
		store:     store,
		cfg:       cfg,
	}
}

// CheckEntry checks if a cache entry exists for the given key.
func (zs *CacheServiceHandler) CheckEntry(ctx context.Context, checkReq *connect.Request[v1.CheckEntryRequest]) (*connect.Response[v1.CheckEntryResponse], error) {
	ctx, span := trace.Start(ctx, "Cache.CheckEntry")
	defer span.End()

	owner := checkReq.Msg.Owner

	log.Info().
		Str("Owner", owner).
		Str("ProviderType", fromProviderV1(checkReq.Msg.ProviderType)).
		Msg("check the tenant exists")

	// validate the owner
	_, err := zs.validateOwner(ctx, owner, fromProviderV1(checkReq.Msg.ProviderType))
	if err != nil {
		return nil, err // already a connect error
	}

	cacheID := buildCacheKey(owner, fromProviderV1(checkReq.Msg.ProviderType), checkReq.Msg.Platform.OperatingSystem, checkReq.Msg.Platform.Architecture, checkReq.Msg.Key)

	// does the cache entry already exist?
	exists, cacheRec, err := zs.store.ExistsCache(ctx, cacheID)
	if err != nil {
		log.Error().Err(err).Msg("failed to check if cache entry exists")
		return nil, connect.NewError(connect.CodeInternal, errors.New("cache.v1.CacheService.CheckEntry internal error"))
	}

	span.SetAttributes(attribute.String("cache_id", cacheID), attribute.Bool("exists", exists))

	return connect.NewResponse(&v1.CheckEntryResponse{
		Exists:    exists,
		Sha256Sum: cacheRec.Sha256,
	}), nil
}

// CreateEntry creates a new cache entry, this is the first step in the cache entry creation process.
func (zs *CacheServiceHandler) CreateEntry(ctx context.Context, createReq *connect.Request[v1.CreateEntryRequest]) (*connect.Response[v1.CreateEntryResponse], error) {
	ctx, span := trace.Start(ctx, "Cache.CreateEntry")
	defer span.End()

	span.SetAttributes(
		attribute.String("key", createReq.Msg.CacheEntry.Key),
		attribute.String("owner", createReq.Msg.CacheEntry.Owner),
		attribute.String("provider", fromProviderV1(createReq.Msg.ProviderType)),
	)

	owner := createReq.Msg.CacheEntry.Owner

	log.Info().
		Str("Owner", createReq.Msg.CacheEntry.Owner).
		Str("ProviderType", fromProviderV1(createReq.Msg.ProviderType)).
		Msg("check the tenant exists")

	// validate the owner
	_, err := zs.validateOwner(ctx, owner, fromProviderV1(createReq.Msg.ProviderType))
	if err != nil {
		return nil, err // already a connect error
	}

	// TODO: validate the name and branch
	name := createReq.Msg.CacheEntry.Name
	branch := createReq.Msg.CacheEntry.Branch

	cacheID := buildCacheKey(owner, fromProviderV1(createReq.Msg.ProviderType), createReq.Msg.Platform.OperatingSystem, createReq.Msg.Platform.Architecture, createReq.Msg.CacheEntry.Key)

	// does the cache entry already exist?
	exists, _, err := zs.store.ExistsCache(ctx, cacheID)
	if err != nil {
		log.Error().Err(err).Msg("failed to check if cache entry exists")
		return nil, connect.NewError(connect.CodeInternal, errors.New("cache.v1.CacheService.CreateEntry internal error"))
	}

	// NOTE: this is a secondary check for exists to prevent a race conditions
	if exists {
		log.Info().Msg("cache entry already exists")
		return nil, connect.NewError(connect.CodeAlreadyExists, errors.New("cache.v1.CacheService.CreateEntry cache entry already exists"))
	}

	// generate an identifier to track the upload
	uploadID := uuid.New().String()

	log.Info().
		Str("Key", createReq.Msg.CacheEntry.Key).
		Str("uploadID", uploadID).
		Str("Sha256Sum", createReq.Msg.CacheEntry.Sha256Sum).
		Msg("presign upload request")

	uploadInstructs, err := zs.presigner.GenerateFileUploadInstructions(
		ctx,
		cacheID,
		createReq.Msg.CacheEntry.Sha256Sum,
		createReq.Msg.CacheEntry.Compression,
		createReq.Msg.CacheEntry.FileSize,
	)
	if err != nil {
		log.Error().Err(err).Msg("failed to presign upload")
		return nil, connect.NewError(connect.CodeInternal, errors.New("cache.v1.CacheService.CreateEntry internal error"))
	}

	cacheRec := index.CacheRecord{
		Key:               createReq.Msg.CacheEntry.Key,
		Paths:             strings.Join(createReq.Msg.CacheEntry.Paths, "\n"),
		Name:              name,
		Branch:            branch,
		Architecture:      createReq.Msg.Platform.Architecture,
		OperatingSystem:   createReq.Msg.Platform.OperatingSystem,
		CpuCount:          createReq.Msg.Platform.CpuCount,
		Owner:             createReq.Msg.CacheEntry.Owner,
		Provider:          fromProviderV1(createReq.Msg.ProviderType),
		Sha256:            createReq.Msg.CacheEntry.Sha256Sum,
		Compression:       createReq.Msg.CacheEntry.Compression,
		MultipartUploadId: uploadInstructs.MultipartUploadId,
		FileSize:          createReq.Msg.CacheEntry.FileSize,
		UpdatedAt:         time.Now(),
	}

	identity := ciauth.GetOIDCIdentity(ctx)
	cacheRec.Identity = &index.Identity{
		Subject: identity.Subject(),
		Issuer:  identity.Issuer(),
	}

	// TODO: we should enable customization of this field to allow for removal of fields to change behavior
	created := strings.Join([]string{
		createReq.Msg.CacheEntry.Owner,
		fromProviderV1(createReq.Msg.ProviderType),
		createReq.Msg.Platform.OperatingSystem,
		createReq.Msg.Platform.Architecture,
		name,
		hashValue(branch),
		time.Now().UTC().Format(time.RFC3339),
	}, "#")

	// create/update the cache entry in the cache index
	// TODO: should we store this record after generating the upload instructions? So we can put the upload ID in the record and validate the upload ID when the upload is complete?
	err = zs.store.PutCache(ctx, uploadID, created, cacheRec, cacheRecordInflightTTL)
	if err != nil {
		log.Error().Err(err).Msg("failed to create cache entry")
		return nil, connect.NewError(connect.CodeInternal, errors.New("cache.v1.CacheService.CreateEntry internal error"))
	}

	return connect.NewResponse(&v1.CreateEntryResponse{
		Id:                 uploadID,
		Multipart:          uploadInstructs.Multipart,
		UploadInstructions: fromUploadInstructions(uploadInstructs.UploadInstructions),
	}), nil
}

// UpdateEntry updates an existing cache entry, this is the second step in the cache entry creation process and is called after the upload is complete.
func (zs *CacheServiceHandler) UpdateEntry(ctx context.Context, updateReq *connect.Request[v1.UpdateEntryRequest]) (*connect.Response[v1.UpdateEntryResponse], error) {
	ctx, span := trace.Start(ctx, "Cache.UpdateEntry")
	defer span.End()

	// does the in flight cache entry exist?
	exists, cacheRec, err := zs.store.ExistsCache(ctx, updateReq.Msg.Id)
	if err != nil {
		log.Error().Err(err).Msg("failed to check if cache entry exists")
		return nil, connect.NewError(connect.CodeInternal, errors.New("cache.v1.CacheService.UpdateEntry internal error"))
	}

	if !exists {
		log.Info().Msg("cache entry does not exist")
		return nil, connect.NewError(connect.CodeNotFound, errors.New("cache.v1.CacheService.UpdateEntry cache entry does not exist"))
	}

	cacheID := buildCacheKey(cacheRec.Owner, cacheRec.Provider, cacheRec.OperatingSystem, cacheRec.Architecture, cacheRec.Key)

	log.Info().
		Str("Id", updateReq.Msg.Id).
		Str("Key", cacheRec.Key).
		Str("cacheID", cacheID).
		Msg("cache entry update request")

	// complete the multipart upload if it exists and the upload ID matches
	if cacheRec.MultipartUploadId != nil {
		_, err := zs.s3Client.CompleteMultipartUpload(ctx, &s3.CompleteMultipartUploadInput{
			Bucket:   aws.String(zs.cfg.CacheBucket),
			Key:      aws.String(cacheID),
			UploadId: cacheRec.MultipartUploadId,
			MultipartUpload: &types.CompletedMultipartUpload{
				Parts: fromCachePartETagV1(updateReq.Msg.MultipartEtags),
			},
		})
		if err != nil {
			log.Error().Err(err).Msg("failed to complete multipart upload")
			return nil, connect.NewError(connect.CodeInternal, errors.New("cache.v1.CacheService.UpdateEntry internal error"))
		}
	}

	// update the cache entry in the cache index
	cacheRec.UpdatedAt = time.Now()

	// TODO: we should enable customization of this field to allow for removal of fields to change behavior
	created := strings.Join([]string{
		cacheRec.Owner,
		cacheRec.Provider,
		cacheRec.OperatingSystem,
		cacheRec.Architecture,
		cacheRec.Name,
		hashValue(cacheRec.Branch),
		time.Now().UTC().Format(time.RFC3339),
	}, "#")

	// move the inflight cache entry to the cache index
	err = zs.store.PutCache(ctx, cacheID, created, cacheRec, cacheRecordTTL)
	if err != nil {
		log.Error().Err(err).Msg("failed to update cache entry")
		return nil, connect.NewError(connect.CodeInternal, errors.New("cache.v1.CacheService.UpdateEntry internal error"))
	}

	return connect.NewResponse(&v1.UpdateEntryResponse{
		Id: updateReq.Msg.Id,
	}), nil
}

// GetEntry returns a cache entry by ID and is used to download a cache entry.
func (zs *CacheServiceHandler) GetEntry(ctx context.Context, getReq *connect.Request[v1.GetEntryRequest]) (*connect.Response[v1.GetEntryResponse], error) {
	ctx, span := trace.Start(ctx, "Cache.GetEntry")
	defer span.End()

	span.SetAttributes(
		attribute.String("key", getReq.Msg.Key),
		attribute.String("owner", getReq.Msg.Owner),
		attribute.String("provider", fromProviderV1(getReq.Msg.ProviderType)),
	)

	// validate the owner
	_, err := zs.validateOwner(ctx, getReq.Msg.Owner, fromProviderV1(getReq.Msg.ProviderType))
	if err != nil {
		return nil, err // already a connect error
	}

	// does the cache entry exist?
	existsWithFallbackRes, err := zs.existsWithFallback(ctx, getReq)
	if err != nil {
		log.Error().Err(err).Msg("failed to check if cache entry exists")
		return nil, connect.NewError(connect.CodeInternal, errors.New("cache.v1.CacheService.GetEntry internal error"))
	}

	if !existsWithFallbackRes.exists {
		log.Info().Msg("cache entry does not exist")
		return nil, connect.NewError(connect.CodeNotFound, errors.New("cache.v1.CacheService.GetEntry cache entry does not exist"))
	}

	exists, res, err := zs.existsInS3(ctx, existsWithFallbackRes.cacheID)
	if err != nil {
		log.Error().Err(err).Msg("failed to get cache entry")
		return nil, connect.NewError(connect.CodeInternal, errors.New("cache.v1.CacheService.GetEntry internal error"))
	}

	if !exists {
		return nil, connect.NewError(connect.CodeNotFound, errors.New("cache.v1.CacheService.GetEntry not found"))
	}

	log.Info().
		Str("cacheID", existsWithFallbackRes.cacheID).
		Bool("exists", existsWithFallbackRes.exists).
		Bool("fallback", existsWithFallbackRes.fallback).
		Str("sha256sum", aws.ToString(res.ChecksumSHA256)).Msg("cache entry found")

	downloadInstructs, err := zs.presigner.GenerateFileDownloadInstructions(
		ctx,
		existsWithFallbackRes.cacheID,
		aws.ToInt64(res.ContentLength),
	)
	if err != nil {
		log.Error().Err(err).Msg("failed to presign download")
		return nil, connect.NewError(connect.CodeInternal, errors.New("cache.v1.CacheService.GetEntry internal error"))
	}

	// TODO: touch the s3 object to update the last modified time

	record := existsWithFallbackRes.record

	return connect.NewResponse(&v1.GetEntryResponse{
		CacheEntry: &v1.CacheEntry{
			Key:         existsWithFallbackRes.cacheID,
			Name:        record.Name,
			Branch:      record.Branch,
			Compression: record.Compression,
			Sha256Sum:   record.Sha256,
			FileSize:    aws.ToInt64(res.ContentLength),
		},
		Multipart:            downloadInstructs.Multipart,
		Fallback:             existsWithFallbackRes.fallback,
		DownloadInstructions: fromInstructToDownloadV1(downloadInstructs.DownloadInstructions),
	}), nil
}

// validateOwner validates the owner of the cache entry using the oidc identity. The owner needs to exist in the tenant index.
func (zs *CacheServiceHandler) validateOwner(ctx context.Context, owner, provider string) (index.TenantRecord, error) {
	ctx, span := trace.Start(ctx, "Cache.validateOwner")
	defer span.End()

	identity := ciauth.GetOIDCIdentity(ctx)

	log.Info().
		Str("Owner", owner).
		Str("identity.Owner", identity.Owner()).
		Any("claims", identity.Claims()).
		Str("ProviderType", provider).
		Msg("check the tenant exists")

	exists, rec, err := zs.store.ExistsTenantByKey(ctx, index.TenantKey(provider, owner))
	if err != nil {
		log.Error().Err(err).Msg("failed to validate owner")
		return index.TenantRecord{}, connect.NewError(connect.CodeInternal, errors.New("cache.v1.CacheService.UpdateEntry internal error"))
	}

	if !exists {
		return index.TenantRecord{}, connect.NewError(connect.CodePermissionDenied, errors.New("cache.v1.CacheService permission denied"))
	}

	return rec, nil
}

// existsInS3 checks if the cache entry data exists in S3.
func (zs *CacheServiceHandler) existsInS3(ctx context.Context, s3key string) (bool, *s3.HeadObjectOutput, error) {
	ctx, span := trace.Start(ctx, "Cache.existsInS3")
	defer span.End()

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

type existsWithFallbackResult struct {
	cacheID  string
	record   index.CacheRecord
	exists   bool
	fallback bool
}

// existsWithFallback checks if the cache entry exists in the cache index. If it does not exist, it uses the fallback branch to check if the cache entry using a prefix search of the cache created
func (zs *CacheServiceHandler) existsWithFallback(ctx context.Context, getReq *connect.Request[v1.GetEntryRequest]) (existsWithFallbackResult, error) {
	ctx, span := trace.Start(ctx, "Cache.existsWithFallback")
	defer span.End()

	cacheID := buildCacheKey(getReq.Msg.Owner, fromProviderV1(getReq.Msg.ProviderType), getReq.Msg.Platform.OperatingSystem, getReq.Msg.Platform.Architecture, getReq.Msg.Key)

	log.Info().
		Str("key", getReq.Msg.Key).
		Str("cacheID", cacheID).
		Msg("cache entry get request")

	keyExists, record, err := zs.store.ExistsCache(ctx, cacheID)
	if err != nil {
		return existsWithFallbackResult{}, fmt.Errorf("failed to check if cache entry exists: %w", err)
	}

	span.SetAttributes(attribute.Bool("keyExists", keyExists), attribute.String("cacheKey", cacheID))

	if keyExists {
		return existsWithFallbackResult{exists: true, record: record, cacheID: cacheID}, nil
	}

	// TODO: we should enable customization of this field to allow for removal of fields to change behavior
	createdPrefix := strings.Join([]string{
		getReq.Msg.Owner,
		fromProviderV1(getReq.Msg.ProviderType),
		getReq.Msg.Platform.OperatingSystem,
		getReq.Msg.Platform.Architecture,
		getReq.Msg.Name,
		hashValue(getReq.Msg.FallbackBranch),
	}, "#")

	fallbackExists, record, err := zs.store.ExistsCacheByFallbackBranch(ctx, createdPrefix)
	if err != nil {
		return existsWithFallbackResult{}, fmt.Errorf("failed to check fallback cache entry exists: %w", err)
	}

	cacheID = buildCacheKey(record.Owner, record.Provider, record.OperatingSystem, record.Architecture, record.Key)

	log.Info().
		Str("key", getReq.Msg.Key).
		Str("cacheID", cacheID).
		Msg("cache entry get request with fallback")

	span.SetAttributes(attribute.Bool("fallbackExists", fallbackExists), attribute.String("cacheKey", cacheID))

	if fallbackExists {
		return existsWithFallbackResult{exists: true, record: record, cacheID: cacheID, fallback: true}, nil
	}

	return existsWithFallbackResult{}, nil
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
