package main

import (
	"context"

	"github.com/alecthomas/kong"

	"github.com/wolfeidau/zipstash/internal/commands"
)

var (
	version = "dev"

	cli struct {
		Debug   bool `help:"Enable debug mode."`
		Version kong.VersionFlag
		RPC     commands.RPCServerCmd    `cmd:"" help:"start a rpc server."`
		Lambda  commands.LambdaServerCmd `cmd:"" help:"start a server in aws lambda."`
	}
)

func main() {
	ctx := context.Background()

	cmd := kong.Parse(&cli,
		kong.Vars{
			"version": version,
		},
		kong.BindTo(ctx, (*context.Context)(nil)))
	err := cmd.Run(&commands.Globals{Debug: cli.Debug, Version: version})
	cmd.FatalIfErrorf(err)
}
