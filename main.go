package main

import (
	"context"

	"github.com/alecthomas/kong"
	"github.com/rs/zerolog/log"
	oteltrace "go.opentelemetry.io/otel/trace"

	"github.com/wolfeidau/cache-service/internal/commands"
	"github.com/wolfeidau/cache-service/internal/commands/client"
	"github.com/wolfeidau/cache-service/internal/trace"
)

var (
	version = "dev"

	cli struct {
		Debug        bool `help:"Enable debug mode."`
		Version      kong.VersionFlag
		Server       commands.ServerCmd       `cmd:"" help:"start a server."`
		LambdaServer commands.LambdaServerCmd `cmd:"" help:"start a server in aws lambda."`
		Client       client.ClientCmd         `cmd:"" help:"run a client."`
	}
)

func main() {
	ctx := context.Background()

	tp, err := trace.NewProvider(ctx, "github.com/wolfeidau/cache-service", version)
	if err != nil {
		log.Fatal().Msgf("failed to create trace provider: %v", err)
	}
	defer func() {
		_ = tp.Shutdown(ctx)
	}()

	var span oteltrace.Span
	ctx, span = trace.Start(ctx, "object-cache-service")
	defer span.End()

	cmd := kong.Parse(&cli,
		kong.Vars{
			"version": version,
		},
		kong.BindTo(ctx, (*context.Context)(nil)))
	err = cmd.Run(&commands.Globals{Debug: cli.Debug, Version: version})
	span.RecordError(err)
	cmd.FatalIfErrorf(err)
}
