package server

import (
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

type Config struct {
	CacheBucket           string
	CacheIndexTable       string
	CreateCacheIndexTable bool
	GetS3Client           S3ClientFunc
	GetDynamoDBClient     DynamoDBClientFunc
}

type S3ClientFunc func() *s3.Client
type DynamoDBClientFunc func() *dynamodb.Client
