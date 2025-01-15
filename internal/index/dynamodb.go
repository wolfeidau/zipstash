package index

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/rs/zerolog/log"
	"github.com/wolfeidau/dynastorev2"

	"github.com/wolfeidau/zipstash/pkg/trace"
)

var (
	ErrNotFound = errors.New("not found")
)

type StoreConfig struct {
	TableName string
	Create    bool
}

type Store struct {
	dynamodbClient *dynamodb.Client
	dynamodb       *dynastorev2.Store[string, string, CacheRecord]
}

func MustNewStore(ctx context.Context, dynamodbClient *dynamodb.Client, config StoreConfig) *Store {
	s := &Store{
		dynamodbClient: dynamodbClient,
		dynamodb:       dynastorev2.New[string, string, CacheRecord](dynamodbClient, config.TableName),
	}

	if config.Create {
		err := s.createTable(ctx, config.TableName)
		if err != nil {
			log.Fatal().Err(err).Msg("failed to create table")
		}
	}

	return s
}

func (s *Store) GetCache(ctx context.Context, id string) (CacheRecord, error) {
	ctx, span := trace.Start(ctx, "Store.Get")
	defer span.End()

	_, cacheRec, err := s.dynamodb.Get(ctx, "cache", id)
	if err != nil {
		if errors.Is(err, dynastorev2.ErrKeyNotExists) {
			return CacheRecord{}, ErrNotFound
		}

		return CacheRecord{}, fmt.Errorf("failed to get cache record: %w", err)
	}

	return cacheRec, err
}

func (s *Store) ExistsCache(ctx context.Context, id string) (bool, CacheRecord, error) {
	ctx, span := trace.Start(ctx, "Store.Get")
	defer span.End()

	_, cacheRec, err := s.dynamodb.Get(ctx, "cache", id)
	if err != nil {
		if errors.Is(err, dynastorev2.ErrKeyNotExists) {
			return false, CacheRecord{}, nil
		}

		return false, CacheRecord{}, fmt.Errorf("failed to get cache record: %w", err)
	}

	return true, cacheRec, err
}

func (s *Store) PutCache(ctx context.Context, id string, value CacheRecord, lifetime time.Duration) error {
	ctx, span := trace.Start(ctx, "Store.Put")
	defer span.End()

	_, err := s.dynamodb.Create(ctx, "cache", id, value,
		s.dynamodb.WriteWithCreateConstraintDisabled(true),
		s.dynamodb.WriteWithTTL(lifetime),
	)
	if err != nil {
		return fmt.Errorf("failed to put cache record: %w", err)
	}

	return err
}
func (s *Store) DeleteCache(ctx context.Context, id string) error {
	ctx, span := trace.Start(ctx, "Store.Delete")
	defer span.End()

	err := s.dynamodb.Delete(ctx, "cache", id)
	if err != nil {
		return fmt.Errorf("failed to delete cache record: %w", err)
	}

	return err
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
