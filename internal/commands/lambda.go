package commands

import (
	"context"
	"fmt"
	"io"

	"github.com/aws/aws-lambda-go/lambda"
	"github.com/awslabs/aws-lambda-go-api-proxy/httpadapter"
	"github.com/labstack/echo/v4"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	echo_middleware "github.com/wolfeidau/echo-middleware"
	"github.com/wolfeidau/lambda-go-extras/lambdaextras"
	lmw "github.com/wolfeidau/lambda-go-extras/middleware"
	"github.com/wolfeidau/lambda-go-extras/middleware/raw"
	zlog "github.com/wolfeidau/lambda-go-extras/middleware/zerolog"
	"go.opentelemetry.io/contrib/instrumentation/github.com/labstack/echo/otelecho"

	"github.com/wolfeidau/zipstash/internal/server"
	"github.com/wolfeidau/zipstash/pkg/trace"
)

type LambdaServerCmd struct {
	JWKSURL     string `help:"jwks url" default:"https://token.actions.githubusercontent.com/.well-known/jwks" env:"JWKS_URL"`
	CacheBucket string `help:"bucket to store cache" env:"CACHE_BUCKET"`
}

func (s *LambdaServerCmd) Run(ctx context.Context, globals *Globals) error {
	_, span := trace.Start(ctx, "ServerCmdRun")
	defer span.End()

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
		Fields: map[string]any{"version": globals.Version},
	}))

	err = server.Setup(ctx, e, server.Config{
		JWKSURL:     s.JWKSURL,
		CacheBucket: s.CacheBucket,
	})
	if err != nil {
		return fmt.Errorf("failed to setup server: %w", err)
	}

	flds := lmw.FieldMap{"version": "dev"}

	ch := lmw.New(
		raw.New(raw.Fields(flds)),
		zlog.New(zlog.Fields(flds)),
	).Then(lambdaextras.GenericHandler(httpadapter.NewV2(e.Server.Handler).ProxyWithContext))

	lambda.Start(ch)

	return nil
}
