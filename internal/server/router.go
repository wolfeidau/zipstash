package server

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/labstack/echo/v4"
	oapi_middleware "github.com/oapi-codegen/echo-middleware"
	echo_middleware "github.com/wolfeidau/echo-middleware"

	"github.com/wolfeidau/cache-service/internal/api"
)

type Config struct {
	JWKSURL     string
	CacheBucket string
	GetS3Client S3ClientFunc
}

func Setup(ctx context.Context, e *echo.Echo, cfg Config, mws ...echo.MiddlewareFunc) error {
	swagger, err := api.GetSwagger()
	if err != nil {
		return fmt.Errorf("failed to load swagger spec: %w", err)
	}

	swagger.Servers = nil

	cacheAPI := NewCache(ctx, cfg)

	e.Use(echo_middleware.ZeroLogRequestLog())

	for _, m := range mws {
		e.Use(m)
	}

	e.Use(oapi_middleware.OapiRequestValidator(swagger))

	api.RegisterHandlers(e, cacheAPI)

	return nil
}

type S3ClientFunc func() *s3.Client
