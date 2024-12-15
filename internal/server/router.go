package server

import (
	"context"
	"fmt"

	"github.com/labstack/echo/v4"
	oapi_middleware "github.com/oapi-codegen/echo-middleware"
	echo_middleware "github.com/wolfeidau/echo-middleware"

	"github.com/wolfeidau/cache-service/pkg/api"
)

type Config struct {
	JWKSURL     string
	CacheBucket string
}

func Setup(ctx context.Context, e *echo.Echo, cfg Config, mws ...echo.MiddlewareFunc) error {
	swagger, err := api.GetSwagger()
	if err != nil {
		return fmt.Errorf("failed to load swagger spec: %w", err)
	}

	swagger.Servers = nil

	cacheAPI, err := NewCache(ctx, cfg)
	if err != nil {
		return fmt.Errorf("failed to create cache: %w", err)
	}

	e.Use(echo_middleware.ZeroLogRequestLog())

	for _, m := range mws {
		e.Use(m)
	}

	e.Use(oapi_middleware.OapiRequestValidator(swagger))

	api.RegisterHandlers(e, cacheAPI)

	return nil
}
