package main

import (
	"context"
	"os"

	"github.com/alecthomas/kong"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	oteltrace "go.opentelemetry.io/otel/trace"

	"github.com/wolfeidau/zipstash/internal/commands"
	"github.com/wolfeidau/zipstash/internal/commands/client"
	"github.com/wolfeidau/zipstash/pkg/trace"
)

var (
	version = "dev"

	cli struct {
		Debug   bool `help:"Enable debug mode."`
		Version kong.VersionFlag
		Save    client.SaveCmd    `cmd:"" help:"save a cache entry."`
		Restore client.RestoreCmd `cmd:"" help:"restore a cache entry."`
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

	var span oteltrace.Span
	ctx, span = trace.Start(ctx, "zipstash")
	defer span.End()

	cmd := kong.Parse(&cli,
		kong.Vars{
			"version": version,
		},
		kong.BindTo(ctx, (*context.Context)(nil)))
	enableDebug(cli.Debug) // enable debug logging
	err = cmd.Run(&commands.Globals{Debug: cli.Debug, Version: version})
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
