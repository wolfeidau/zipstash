package ciauth

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/labstack/echo/v4"
	"github.com/rs/zerolog/log"
	"github.com/wolfeidau/cache-service/pkg/api"
)

type Config struct {
	Providers map[string]string
}

func NewWithConfig(ctx context.Context, cfg Config) (echo.MiddlewareFunc, error) {

	providers := NewOIDCProviders(cfg.Providers)

	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			ctx := c.Request().Context()

			rawIDToken, err := extractBearerToken(c.Request())
			if err != nil {
				log.Ctx(ctx).Error().Err(err).Msg("failed to parse token")
				return c.JSON(http.StatusUnauthorized, api.Error{
					Message: "invalid token",
				})
			}

			// provider is required in the path
			providerName := c.Param("provider")
			if providerName == "" {
				return c.JSON(http.StatusBadRequest, api.Error{
					Message: "missing provider",
				})
			}

			idToken, err := providers.VerifyToken(ctx, providerName, rawIDToken)
			if err != nil {
				log.Ctx(ctx).Error().Err(err).Msg("failed to parse token")
				return c.JSON(http.StatusUnauthorized, api.Error{
					Message: "invalid token",
				})
			}

			log.Ctx(ctx).Info().Any("tok", idToken).Msg("token")

			c.Set("idToken", idToken)

			return next(c)
		}
	}, nil
}

func extractBearerToken(req *http.Request) (string, error) {
	reqToken := req.Header.Get("Authorization")
	splitToken := strings.Split(reqToken, "Bearer")
	if len(splitToken) != 2 {
		return "", fmt.Errorf("malformed token")
	}

	return strings.TrimSpace(splitToken[1]), nil
}
