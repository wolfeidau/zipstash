package client

import (
	"context"

	"github.com/wolfeidau/cache-service/internal/commands"
	"github.com/wolfeidau/cache-service/internal/trace"
)

type RestoreCmd struct {
	Endpoint  string `help:"endpoint to call" default:"http://localhost:8080" env:"INPUT_ENDPOINT"`
	Token     string `help:"token to use" required:""`
	Key       string `help:"key to use for the cache entry" required:"" env:"INPUT_KEY"`
	Path      string `help:"Path list for a cache entry." env:"INPUT_PATH"`
	CacheFile string `help:"file to save"`
	Skip      bool   `help:"skip confirmation"`
}

func (c *RestoreCmd) Run(ctx context.Context, globals *commands.Globals) error {
	_, span := trace.Start(ctx, "RestoreCmd.Run")
	defer span.End()
	return c.restore(ctx, globals)
}

func (c *RestoreCmd) restore(ctx context.Context, globals *commands.Globals) error {
	_, span := trace.Start(ctx, "RestoreCmd.restore")
	defer span.End()

	return nil
}
