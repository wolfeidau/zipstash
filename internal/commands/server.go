package commands

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	transport "github.com/aws/smithy-go/endpoints"
	"github.com/labstack/echo/v4"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	echo_middleware "github.com/wolfeidau/echo-middleware"
	"go.opentelemetry.io/contrib/instrumentation/github.com/labstack/echo/otelecho"

	"github.com/wolfeidau/zipstash/internal/ciauth"
	"github.com/wolfeidau/zipstash/internal/server"
	"github.com/wolfeidau/zipstash/pkg/trace"
)

type ServerCmd struct {
	Listen      string `help:"listen address" default:"localhost:8080"`
	JWKSURL     string `help:"jwks url" default:"https://token.actions.githubusercontent.com/.well-known/jwks"`
	CacheBucket string `help:"bucket to store cache" env:"CACHE_BUCKET"`
	Local       bool   `help:"run in local mode"`
	Endpoint    string `help:"s3 endpoint" env:"S3_ENDPOINT" default:"http://minio.zipstash.orb.local:9000"`
}

func (s *ServerCmd) Run(ctx context.Context, globals *Globals) error {
	e := echo.New()

	e.HideBanner = true
	e.Logger.SetOutput(io.Discard)

	tp, err := trace.NewProvider(ctx, "github.com/wolfeidau/zipstash", globals.Version)
	if err != nil {
		log.Fatal().Msgf("failed to create trace provider: %v", err)
	}
	defer func() {
		_ = tp.Shutdown(ctx)
	}()

	e.Use(otelecho.Middleware("listener", otelecho.WithTracerProvider(tp)))

	e.Use(echo_middleware.ZeroLogWithConfig(echo_middleware.ZeroLogConfig{
		Level:  zerolog.DebugLevel,
		Caller: true,
		Fields: map[string]any{"version": globals.Version},
	}))

	var s3ClientFunc server.S3ClientFunc

	if s.Local {
		endpointURL, err := url.Parse(s.Endpoint)
		if err != nil {
			return fmt.Errorf("failed to parse endpoint url: %w", err)
		}

		s3ClientFunc = func() *s3.Client {
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
		}
	} else {
		oidc, err := ciauth.NewWithConfig(ctx, ciauth.Config{
			Providers: ciauth.DefaultProviderEndpoints,
		})
		if err != nil {
			return fmt.Errorf("failed to create oidc middleware: %w", err)
		}

		e.Use(oidc)

		awscfg, err := config.LoadDefaultConfig(ctx)
		if err != nil {
			return fmt.Errorf("failed to load config: %w", err)
		}

		s3ClientFunc = func() *s3.Client {
			return s3.NewFromConfig(awscfg)
		}
	}

	err = server.Setup(ctx, e, server.Config{
		JWKSURL:     s.JWKSURL,
		CacheBucket: s.CacheBucket,
		GetS3Client: s3ClientFunc,
	})
	if err != nil {
		return fmt.Errorf("failed to setup server: %w", err)
	}

	svr := &http.Server{
		Handler:           e.Server.Handler,
		Addr:              s.Listen,
		ReadHeaderTimeout: 10 * time.Second,
	}

	log.Info().Str("addr", s.Listen).Msg("starting server")

	return svr.ListenAndServe()
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
