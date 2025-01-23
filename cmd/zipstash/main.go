package main

import (
	"context"
	"net/http"
	"os"

	"connectrpc.com/connect"
	"connectrpc.com/otelconnect"
	"github.com/alecthomas/kong"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"go.opentelemetry.io/otel"
	oteltrace "go.opentelemetry.io/otel/trace"

	"github.com/wolfeidau/zipstash/api/gen/proto/go/cache/v1/cachev1connect"
	"github.com/wolfeidau/zipstash/internal/commands/client"
	"github.com/wolfeidau/zipstash/pkg/trace"
)

var (
	version = "dev"

	cli struct {
		Save     client.SaveCmd    `cmd:"" help:"save a cache entry."`
		Restore  client.RestoreCmd `cmd:"" help:"restore a cache entry."`
		Endpoint string            `help:"endpoint to call" default:"http://localhost:8080" env:"INPUT_ENDPOINT"`
		Debug    bool              `help:"Enable debug mode."`
		Version  kong.VersionFlag
	}
)

func main() {
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr}).
		With().Caller().Logger()

	ctx := context.Background()

	tp, err := trace.NewProvider(ctx, "github.com/wolfeidau/zipstash", version)
	if err != nil {
		log.Fatal().Msgf("failed to create trace provider: %v", err)
	}
	defer func() {
		_ = tp.Shutdown(ctx)
	}()

	otelInterceptor, err := otelconnect.NewInterceptor(
		otelconnect.WithTracerProvider(tp),
		otelconnect.WithPropagator(otel.GetTextMapPropagator()),
	)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to create otel interceptor")
	}

	var span oteltrace.Span
	ctx, span = trace.Start(ctx, "zipstash")
	defer span.End()

	cmd := kong.Parse(&cli,
		kong.Vars{
			"version": version,
		},
		kong.BindTo(ctx, (*context.Context)(nil)))
	enableDebug(cli.Debug) // enable debug logging
	err = cmd.Run(&client.Globals{Debug: cli.Debug, Version: version, Client: buildClient(cli.Endpoint, otelInterceptor)})
	span.RecordError(err)
	cmd.FatalIfErrorf(err)
}

func enableDebug(debug bool) {
	if debug {
		zerolog.SetGlobalLevel(zerolog.DebugLevel)
	} else {
		zerolog.SetGlobalLevel(zerolog.InfoLevel)
	}
}

func buildClient(endpoint string, otelInterceptor *otelconnect.Interceptor) cachev1connect.CacheServiceClient {
	return cachev1connect.NewCacheServiceClient(http.DefaultClient, endpoint, connect.WithInterceptors(otelInterceptor))
}
