package commands

import (
	"context"
	"fmt"
	"net/http"

	"connectrpc.com/connect"
	"connectrpc.com/otelconnect"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/awslabs/aws-lambda-go-api-proxy/httpadapter"
	"github.com/rs/zerolog/log"
	"github.com/wolfeidau/lambda-go-extras/lambdaextras"
	lmw "github.com/wolfeidau/lambda-go-extras/middleware"
	"github.com/wolfeidau/lambda-go-extras/middleware/raw"
	zlog "github.com/wolfeidau/lambda-go-extras/middleware/zerolog"

	"github.com/wolfeidau/zipstash/api/gen/proto/go/cache/v1/cachev1connect"
	"github.com/wolfeidau/zipstash/internal/ciauth"
	"github.com/wolfeidau/zipstash/internal/server"
	"github.com/wolfeidau/zipstash/pkg/trace"
)

type LambdaServerCmd struct {
	CacheBucket     string `help:"bucket to store cache" env:"CACHE_BUCKET"`
	CacheIndexTable string `help:"table to store cache index" env:"CACHE_INDEX_TABLE"`
	TrustRemote     bool   `help:"trust remote spans"`
}

func (s *LambdaServerCmd) Run(ctx context.Context, globals *Globals) error {
	tp, err := trace.NewProvider(ctx, "github.com/wolfeidau/zipstash", globals.Version)
	if err != nil {
		log.Fatal().Msgf("failed to create trace provider: %v", err)
	}
	defer func() {
		_ = tp.Shutdown(ctx)
	}()

	opts := []connect.HandlerOption{}

	awscfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	s3ClientFunc := func() *s3.Client {
		return s3.NewFromConfig(awscfg)
	}
	ddbClientFunc := func() *dynamodb.Client {
		return dynamodb.NewFromConfig(awscfg)
	}

	// Add OIDC interceptor
	opts = append(opts, connect.WithInterceptors(
		ciauth.NewInterceptorWithConfig(ciauth.Config{
			Providers: ciauth.DefaultProviderEndpoints,
		}),
	))

	var oteloptions []otelconnect.Option
	oteloptions = append(oteloptions, otelconnect.WithTracerProvider(tp))
	if s.TrustRemote {
		oteloptions = append(oteloptions, otelconnect.WithTrustRemote())
	}

	otelInterceptor, err := otelconnect.NewInterceptor(oteloptions...)
	if err != nil {
		return fmt.Errorf("failed to create otel interceptor: %w", err)
	}
	opts = append(opts, connect.WithInterceptors(otelInterceptor))

	csh := server.NewCacheServiceHandler(ctx, server.Config{
		CacheBucket:       s.CacheBucket,
		CacheIndexTable:   s.CacheIndexTable,
		GetS3Client:       s3ClientFunc,
		GetDynamoDBClient: ddbClientFunc,
	})
	mux := http.NewServeMux()
	path, handler := cachev1connect.NewCacheServiceHandler(csh, opts...)
	mux.Handle(path, handler)

	flds := lmw.FieldMap{"version": "dev"}

	ch := lmw.New(
		raw.New(raw.Fields(flds)),
		zlog.New(zlog.Fields(flds)),
	).Then(lambdaextras.GenericHandler(httpadapter.NewV2(mux).ProxyWithContext))

	lambda.Start(ch)

	return nil
}
