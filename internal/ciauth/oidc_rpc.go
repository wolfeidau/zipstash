package ciauth

import (
	"context"
	"errors"

	"connectrpc.com/connect"
	"github.com/rs/zerolog/log"
)

func NewInterceptorWithConfig(cfg Config) connect.UnaryInterceptorFunc {
	providers := NewOIDCProviders(cfg.Providers)

	return func(next connect.UnaryFunc) connect.UnaryFunc {
		return func(ctx context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
			rawIDToken, err := extractBearerToken(req.Header())
			if err != nil {
				return nil, connect.NewError(connect.CodeUnauthenticated, err)
			}

			providerName := req.Header().Get("X-Provider")
			if providerName == "" {
				return nil, connect.NewError(connect.CodeUnauthenticated, errors.New("missing provider"))
			}

			idToken, err := providers.VerifyToken(ctx, providerName, rawIDToken)
			if err != nil {
				return nil, connect.NewError(connect.CodeUnauthenticated, err)
			}

			log.Ctx(ctx).Info().Any("tok", idToken).Msg("token")
			return next(ctx, req)
		}
	}
}
