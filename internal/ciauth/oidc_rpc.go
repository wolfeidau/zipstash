package ciauth

import (
	"context"
	"errors"

	"connectrpc.com/connect"
	"github.com/coreos/go-oidc/v3/oidc"
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

			// store the token in the context
			ctx = context.WithValue(ctx, idTokenKey{}, idToken)

			return next(ctx, req)
		}
	}
}

type idTokenKey struct{}

func GetIDToken(ctx context.Context) *oidc.IDToken {
	idToken, ok := ctx.Value(idTokenKey{}).(*oidc.IDToken)
	if !ok {
		return nil
	}
	return idToken
}
