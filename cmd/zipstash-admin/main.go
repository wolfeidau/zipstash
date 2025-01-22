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

	"github.com/wolfeidau/zipstash/api/gen/proto/go/provision/v1/provisionv1connect"
	"github.com/wolfeidau/zipstash/internal/commands/admin"
	"github.com/wolfeidau/zipstash/pkg/trace"
)

var (
	version = "dev"

	cli struct {
		Debug        bool `help:"Enable debug mode."`
		Version      kong.VersionFlag
		Endpoint     string                `help:"admin endpoint to call" default:"http://localhost:8080" env:"INPUT_ENDPOINT"`
		CreateTenant admin.CreateTenantCmd `cmd:"" help:"create a tenant."`
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
	err = cmd.Run(&admin.Globals{Debug: cli.Debug, Version: version, Client: buildClient(cli.Endpoint, otelInterceptor)})
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

func buildClient(endpoint string, otelInterceptor *otelconnect.Interceptor) provisionv1connect.ProvisionServiceClient {
	return provisionv1connect.NewProvisionServiceClient(http.DefaultClient, endpoint, connect.WithInterceptors(otelInterceptor))
}