package commands

import (
	"context"
	"fmt"
	"net/http"

	"connectrpc.com/connect"
	"connectrpc.com/otelconnect"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/rs/zerolog/log"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"

	"github.com/wolfeidau/zipstash/api/zipstash/v1/zipstashv1connect"
	"github.com/wolfeidau/zipstash/internal/ciauth"
	"github.com/wolfeidau/zipstash/internal/server"
	"github.com/wolfeidau/zipstash/pkg/trace"
)

type RPCServerCmd struct {
	Listen      string `help:"listen address" default:"localhost:8080"`
	CacheBucket string `help:"bucket to store cache" env:"CACHE_BUCKET"`
	Local       bool   `help:"run in local mode"`
	Endpoint    string `help:"s3 endpoint" env:"S3_ENDPOINT" default:"http://minio.zipstash.orb.local:9000"`
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
	opts := []connect.HandlerOption{}
	if s.Local {
		s3ClientFunc, err = newLocalS3Client(s.Endpoint)
		if err != nil {
			return fmt.Errorf("failed to create local s3 client: %w", err)
		}

	} else {
		awscfg, err := config.LoadDefaultConfig(ctx)
		if err != nil {
			return fmt.Errorf("failed to load config: %w", err)
		}
		s3ClientFunc = func() *s3.Client {
			return s3.NewFromConfig(awscfg)
		}

		opts = append(opts, connect.WithInterceptors(
			ciauth.NewInterceptorWithConfig(ciauth.Config{
				Providers: ciauth.DefaultProviderEndpoints,
			}),
		))
	}

	otelInterceptor, err := otelconnect.NewInterceptor(otelconnect.WithTracerProvider(tp))
	if err != nil {
		return fmt.Errorf("failed to create otel interceptor: %w", err)
	}
	opts = append(opts, connect.WithInterceptors(otelInterceptor))

	zs := server.NewZipStashServiceHandler(ctx, server.Config{
		CacheBucket: s.CacheBucket,
		GetS3Client: s3ClientFunc,
	})
	mux := http.NewServeMux()
	path, handler := zipstashv1connect.NewZipStashServiceHandler(zs, opts...)
	mux.Handle(path, handler)

	return http.ListenAndServe(
		s.Listen,
		// Use h2c so we can serve HTTP/2 without TLS.
		h2c.NewHandler(mux, &http2.Server{}),
	)
}
