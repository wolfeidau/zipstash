package index

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/rs/zerolog/log"
	"github.com/wolfeidau/dynastorev2"
	"go.opentelemetry.io/otel/attribute"

	"github.com/wolfeidau/zipstash/pkg/trace"
)

var (
	ErrNotFound      = errors.New("not found")
	ErrAlreadyExists = errors.New("already exists")
)

type DynamoDBClientFunc func() *dynamodb.Client

type StoreConfig struct {
	GetDynamoDBClient DynamoDBClientFunc
	CacheIndexTable   string
	Create            bool
}

type Store struct {
	dynamodbClient *dynamodb.Client
	cacheStore     *dynastorev2.Store[string, string, CacheRecord]
	tenantStore    *dynastorev2.Store[string, string, TenantRecord]
}

func MustNewStore(ctx context.Context, config StoreConfig) *Store {
	ddbClient := config.GetDynamoDBClient()

	s := &Store{
		dynamodbClient: ddbClient,
		cacheStore:     dynastorev2.New[string, string, CacheRecord](ddbClient, config.CacheIndexTable),
		tenantStore:    dynastorev2.New[string, string, TenantRecord](ddbClient, config.CacheIndexTable),
	}

	if config.Create {
		err := s.createTable(ctx, config.CacheIndexTable)
		if err != nil {
			log.Fatal().Err(err).Msg("failed to create table")
		}
	}

	return s
}

func (s *Store) GetCache(ctx context.Context, id string) (CacheRecord, error) {
	ctx, span := trace.Start(ctx, "Store.GetCache")
	defer span.End()

	_, cacheRec, err := s.cacheStore.Get(ctx, "cache", id)
	if err != nil {
		if errors.Is(err, dynastorev2.ErrKeyNotExists) {
			return CacheRecord{}, ErrNotFound
		}

		return CacheRecord{}, fmt.Errorf("failed to get cache record: %w", err)
	}

	return cacheRec, err
}

func (s *Store) ExistsCache(ctx context.Context, id string) (bool, CacheRecord, error) {
	ctx, span := trace.Start(ctx, "Store.ExistsCache")
	defer span.End()

	_, cacheRec, err := s.cacheStore.Get(ctx, "cache", id)
	if err != nil {
		if errors.Is(err, dynastorev2.ErrKeyNotExists) {
			return false, CacheRecord{}, nil
		}

		return false, CacheRecord{}, fmt.Errorf("failed to get cache record: %w", err)
	}

	return true, cacheRec, err
}

func (s *Store) ExistsCacheByFallbackBranch(ctx context.Context, owner, provider, os, arch, branch string) (bool, CacheRecord, error) {
	ctx, span := trace.Start(ctx, "Store.ExistsCacheByFallbackBranch")
	defer span.End()

	created := strings.Join([]string{
		owner,
		provider,
		os,
		arch,
		branch,
	}, "#")

	_, res, err := s.cacheStore.ListBySortKeyPrefix(ctx, "cache", created,
		s.tenantStore.ReadWithLimit(1),
		s.tenantStore.ReadWithIndex("idx_created", "id", "created"))
	if err != nil {
		span.RecordError(err)

		return false, CacheRecord{}, fmt.Errorf("failed to list tenants: %w", err)
	}

	if len(res) == 0 {
		return false, CacheRecord{}, nil
	}

	return true, res[0], nil
}

func (s *Store) PutCache(ctx context.Context, id string, value CacheRecord, lifetime time.Duration) error {
	ctx, span := trace.Start(ctx, "Store.PutCache")
	defer span.End()

	created := strings.Join([]string{
		value.Owner,
		value.Provider,
		value.OperatingSystem,
		value.Architecture,
		value.Branch,
		time.Now().UTC().Format(time.RFC3339),
	}, "#")

	span.SetAttributes(attribute.String("created", created))

	_, err := s.cacheStore.Create(ctx, "cache", id, value,
		s.cacheStore.WriteWithCreateConstraintDisabled(true),
		s.cacheStore.WriteWithTTL(lifetime),
		s.cacheStore.WriteWithExtraFields(map[string]any{
			// bit of a hack as created is just updated without create constraint
			"created": created,
			"pk1":     "cache#owner",
			"sk1":     TenantKey(value.Provider, value.Owner),
		}),
	)
	if err != nil {
		span.RecordError(err)

		return fmt.Errorf("failed to put cache record: %w", err)
	}

	return err
}

func (s *Store) DeleteCache(ctx context.Context, id string) error {
	ctx, span := trace.Start(ctx, "Store.DeleteCache")
	defer span.End()

	err := s.cacheStore.Delete(ctx, "cache", id)
	if err != nil {
		span.RecordError(err)

		return fmt.Errorf("failed to delete cache record: %w", err)
	}

	return err
}

func (s *Store) GetTenant(ctx context.Context, id string) (TenantRecord, error) {
	ctx, span := trace.Start(ctx, "Store.GetTenant")
	defer span.End()

	_, tenantRec, err := s.tenantStore.Get(ctx, "tenant", id)
	if err != nil {
		span.RecordError(err)

		if errors.Is(err, dynastorev2.ErrKeyNotExists) {
			return TenantRecord{}, ErrNotFound
		}
		return TenantRecord{}, fmt.Errorf("failed to get tenant record: %w", err)
	}

	return tenantRec, nil
}

func (s *Store) PutTenant(ctx context.Context, id string, value TenantRecord) error {
	ctx, span := trace.Start(ctx, "Store.PutTenant")
	defer span.End()

	_, err := s.tenantStore.Create(ctx, "tenant", id, value,
		s.tenantStore.WriteWithExtraFields(map[string]any{
			"created": time.Now().UTC().Format(time.RFC3339),
			"pk1":     "tenant#key",
			"sk1":     TenantKey(value.ProviderType, value.Owner),
		}),
	)
	if err != nil {
		span.RecordError(err)

		var oc *types.ConditionalCheckFailedException
		if errors.As(err, &oc) {
			return fmt.Errorf("tenant already exists: %w", ErrAlreadyExists)
		}
		return fmt.Errorf("failed to put cache record: %w", err)
	}

	return err
}

func (s *Store) ExistsTenantByKey(ctx context.Context, key string) (bool, TenantRecord, error) {
	ctx, span := trace.Start(ctx, "Store.ExistsTenant")
	defer span.End()

	_, res, err := s.tenantStore.ListBySortKeyPrefix(ctx, "tenant#key", key,
		s.tenantStore.ReadWithLimit(1),
		s.tenantStore.ReadWithIndex("idx_global_1", "pk1", "sk1"))
	if err != nil {
		span.RecordError(err)

		return false, TenantRecord{}, fmt.Errorf("failed to list tenants: %w", err)
	}

	if len(res) == 0 {
		return false, TenantRecord{}, nil
	}

	return true, res[0], nil
}

func (s *Store) createTable(ctx context.Context, tableName string) error {

	params := &dynamodb.CreateTableInput{
		TableName: aws.String(tableName),
		KeySchema: []types.KeySchemaElement{
			{AttributeName: aws.String("id"), KeyType: types.KeyTypeHash},
			{AttributeName: aws.String("name"), KeyType: types.KeyTypeRange},
		},
		LocalSecondaryIndexes: []types.LocalSecondaryIndex{
			{
				IndexName: aws.String("idx_created"),
				KeySchema: []types.KeySchemaElement{
					{AttributeName: aws.String("id"), KeyType: types.KeyTypeHash},
					{AttributeName: aws.String("created"), KeyType: types.KeyTypeRange},
				},
				Projection: &types.Projection{ProjectionType: types.ProjectionTypeAll},
			},
		},
		GlobalSecondaryIndexes: []types.GlobalSecondaryIndex{
			{
				IndexName: aws.String("idx_global_1"),
				KeySchema: []types.KeySchemaElement{
					{AttributeName: aws.String("pk1"), KeyType: types.KeyTypeHash},
					{AttributeName: aws.String("sk1"), KeyType: types.KeyTypeRange},
				},
				Projection: &types.Projection{ProjectionType: types.ProjectionTypeAll},
				ProvisionedThroughput: &types.ProvisionedThroughput{
					ReadCapacityUnits:  aws.Int64(1),
					WriteCapacityUnits: aws.Int64(1),
				},
			},
		},
		AttributeDefinitions: []types.AttributeDefinition{
			{AttributeName: aws.String("id"), AttributeType: types.ScalarAttributeTypeS},
			{AttributeName: aws.String("name"), AttributeType: types.ScalarAttributeTypeS},
			{AttributeName: aws.String("created"), AttributeType: types.ScalarAttributeTypeS},
			{AttributeName: aws.String("pk1"), AttributeType: types.ScalarAttributeTypeS},
			{AttributeName: aws.String("sk1"), AttributeType: types.ScalarAttributeTypeS},
		},
		ProvisionedThroughput: &types.ProvisionedThroughput{
			ReadCapacityUnits:  aws.Int64(1),
			WriteCapacityUnits: aws.Int64(1),
		},
		SSESpecification: &types.SSESpecification{
			Enabled: aws.Bool(true),
			SSEType: types.SSETypeAes256,
		},
	}

	_, err := s.dynamodbClient.CreateTable(ctx, params)
	if err != nil {
		var oe *types.ResourceInUseException
		if errors.As(err, &oe) {
			return nil
		}

		return fmt.Errorf("failed to create table: %w", err)
	}

	err = dynamodb.NewTableExistsWaiter(s.dynamodbClient).Wait(ctx, &dynamodb.DescribeTableInput{
		TableName: aws.String(tableName),
	}, 10*time.Second)
	if err != nil {
		return err
	}

	_, err = s.dynamodbClient.UpdateTimeToLive(ctx, &dynamodb.UpdateTimeToLiveInput{
		TableName: aws.String(tableName),
		TimeToLiveSpecification: &types.TimeToLiveSpecification{
			AttributeName: aws.String("expires"),
			Enabled:       aws.Bool(true),
		},
	})
	if err != nil {
		return err
	}

	return nil
}
