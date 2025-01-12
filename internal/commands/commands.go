package commands

import (
	"context"
	"fmt"
	"net/url"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	transport "github.com/aws/smithy-go/endpoints"
	"github.com/wolfeidau/zipstash/internal/server"
)

type Globals struct {
	Debug   bool
	Version string
}

type Resolver struct {
	URL *url.URL
}

func (r *Resolver) ResolveEndpoint(_ context.Context, params s3.EndpointParameters) (transport.Endpoint, error) {
	u := *r.URL
	if params.Bucket != nil {
		u.Path += "/" + *params.Bucket
	}
	return transport.Endpoint{URI: u}, nil
}

func newLocalS3Client(endpoint string) (server.S3ClientFunc, error) {
	endpointURL, err := url.Parse(endpoint)
	if err != nil {
		return nil, fmt.Errorf("failed to parse endpoint: %w", err)
	}

	return func() *s3.Client {
		return s3.New(s3.Options{
			UsePathStyle:       true,
			EndpointResolverV2: &Resolver{URL: endpointURL},
			Credentials: aws.CredentialsProviderFunc(func(ctx context.Context) (aws.Credentials, error) {
				return aws.Credentials{
					AccessKeyID:     "minioadmin",
					SecretAccessKey: "minioadmin",
				}, nil
			}),
		})
	}, nil
}
