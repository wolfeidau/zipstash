package commands

import (
	"context"
	"fmt"
	"net/http"
	"os"

	"connectrpc.com/connect"
	"connectrpc.com/otelconnect"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"

	"github.com/wolfeidau/zipstash/api/gen/proto/go/cache/v1/cachev1connect"
	"github.com/wolfeidau/zipstash/api/gen/proto/go/provision/v1/provisionv1connect"
	"github.com/wolfeidau/zipstash/internal/ciauth"
	"github.com/wolfeidau/zipstash/internal/index"
	"github.com/wolfeidau/zipstash/internal/provider"
	"github.com/wolfeidau/zipstash/internal/server"
	"github.com/wolfeidau/zipstash/pkg/trace"
)

type RPCServerCmd struct {
	Listen                string `help:"listen address" default:"localhost:8080"`
	CacheBucket           string `help:"bucket to store cache" env:"CACHE_BUCKET"`
	CacheIndexTable       string `help:"table to store cache index" env:"CACHE_INDEX_TABLE"`
	CreateCacheIndexTable bool   `help:"create cache index table if it does not exist" env:"CREATE_CACHE_INDEX_TABLE" default:"false"`
	Local                 bool   `help:"run in local mode"`
	TrustRemote           bool   `help:"trust remote spans"`
	S3Endpoint            string `help:"s3 endpoint, used in local mode" env:"S3_ENDPOINT" default:"http://minio.zipstash.orb.local:9000"`
	DynamoEndpoint        string `help:"s3 endpoint, used in local mode" env:"DYNAMO_ENDPOINT" default:"http://dynamodb-local.zipstash.orb.local:8000"`
}

func (s *RPCServerCmd) Run(ctx context.Context, globals *Globals) error {
	tp, err := trace.NewProvider(ctx, "github.com/wolfeidau/zipstash", globals.Version)
	if err != nil {
		log.Fatal().Msgf("failed to create trace provider: %v", err)
	}
	defer func() {
		_ = tp.Shutdown(ctx)
	}()

	var s3ClientFunc server.S3ClientFunc
	var ddbClientFunc index.DynamoDBClientFunc
	opts := []connect.HandlerOption{}
	if s.Local {
		log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr}).
			With().Caller().Logger()

		s3ClientFunc = newLocalS3Client(s.S3Endpoint)
		ddbClientFunc = newLocalDDBClient(s.DynamoEndpoint)
	} else {
		awscfg, err := config.LoadDefaultConfig(ctx)
		if err != nil {
			return fmt.Errorf("failed to load config: %w", err)
		}
		s3ClientFunc = func() *s3.Client {
			return s3.NewFromConfig(awscfg)
		}
		ddbClientFunc = func() *dynamodb.Client {
			return dynamodb.NewFromConfig(awscfg)
		}

		opts = append(opts, connect.WithInterceptors(
			ciauth.NewInterceptorWithConfig(ciauth.Config{
				Providers: provider.DefaultEndpoints,
			}),
		))

	}

	store := index.MustNewStore(ctx, index.StoreConfig{
		CacheIndexTable:   s.CacheIndexTable,
		Create:            s.CreateCacheIndexTable,
		GetDynamoDBClient: ddbClientFunc,
	})

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

	csh := server.NewCacheServiceHandler(ctx, server.CacheConfig{
		CacheBucket: s.CacheBucket,
		GetS3Client: s3ClientFunc,
	}, store)

	psh := server.NewProvisionServiceHandler(store)

	mux := http.NewServeMux()
	path, handler := cachev1connect.NewCacheServiceHandler(csh, opts...)

	log.Info().Str("path", path).Str("add", s.Listen).Msg("serving")
	mux.Handle(path, handler)

	path, handler = provisionv1connect.NewProvisionServiceHandler(psh, opts...)

	log.Info().Str("path", path).Str("add", s.Listen).Msg("serving")
	mux.Handle(path, handler)

	return http.ListenAndServe(
		s.Listen,
		// Use h2c so we can serve HTTP/2 without TLS.
		h2c.NewHandler(mux, &http2.Server{}),
	)
}
