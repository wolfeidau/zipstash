package server

import (
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

type Config struct {
	CacheBucket string
	GetS3Client S3ClientFunc
}

type S3ClientFunc func() *s3.Client
