package commands

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	echo_middleware "github.com/wolfeidau/echo-middleware"

	"github.com/wolfeidau/cache-service/internal/ciauth"
	"github.com/wolfeidau/cache-service/internal/server"
	"github.com/wolfeidau/cache-service/internal/trace"
)

type ServerCmd struct {
	Listen      string `help:"listen address" default:":8080"`
	JWKSURL     string `help:"jwks url" default:"https://token.actions.githubusercontent.com/.well-known/jwks"`
	CacheBucket string `help:"bucket to store cache" env:"CACHE_BUCKET"`
	Local       bool   `help:"run in local mode"`
}

func (s *ServerCmd) Run(ctx context.Context, globals *Globals) error {
	_, span := trace.Start(ctx, "ServerCmdRun")
	defer span.End()

	e := echo.New()

	e.HideBanner = true
	e.Logger.SetOutput(io.Discard)

	e.Use(echo_middleware.ZeroLogWithConfig(echo_middleware.ZeroLogConfig{
		Level:  zerolog.DebugLevel,
		Fields: map[string]interface{}{"version": "dev"},
	}))

	if !s.Local {
		oidc, err := ciauth.NewWithConfig(ctx, ciauth.Config{
			Providers: ciauth.DefaultProviderEndpoints,
		})
		if err != nil {
			return fmt.Errorf("failed to create oidc middleware: %w", err)
		}

		e.Use(oidc)
	}

	err := server.Setup(ctx, e, server.Config{
		JWKSURL:     s.JWKSURL,
		CacheBucket: s.CacheBucket,
	})
	if err != nil {
		return fmt.Errorf("failed to setup server: %w", err)
	}

	svr := &http.Server{
		Handler:           e.Server.Handler,
		Addr:              s.Listen,
		ReadHeaderTimeout: 10 * time.Second,
	}

	log.Info().Msgf("listening on %s", s.Listen)

	return svr.ListenAndServe()
}
