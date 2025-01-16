package ciauth

import (
	"context"
	"errors"

	"connectrpc.com/connect"
	"github.com/rs/zerolog/log"
	"go.opentelemetry.io/otel/attribute"

	"github.com/wolfeidau/zipstash/pkg/trace"
)

func NewInterceptorWithConfig(cfg Config) connect.UnaryInterceptorFunc {
	providers := NewOIDCProviders(cfg.Providers)

	return func(next connect.UnaryFunc) connect.UnaryFunc {
		return func(ctx context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
			ctx, span := trace.Start(ctx, "OIDC.Interceptor")
			defer span.End()

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

			log.Ctx(ctx).Info().
				Str("subject", idToken.Subject).
				Strs("audience", idToken.Audience).
				Time("expiry", idToken.Expiry).
				Str("provider", providerName).
				Str("issuer", idToken.Issuer).
				Msg("token verified")

			span.SetAttributes(
				attribute.String("subject", idToken.Subject),
				attribute.StringSlice("audience", idToken.Audience),
				attribute.String("expiry", idToken.Expiry.String()),
				attribute.String("provider", providerName),
				attribute.String("issuer", idToken.Issuer),
			)

			// store the token in the context
			ctx = context.WithValue(ctx, idTokenKey{}, &CIAuthIdentity{
				Provider: providerName,
				IDToken:  idToken,
			})

			return next(ctx, req)
		}
	}
}

type idTokenKey struct{}

func GetCIAuthIdentity(ctx context.Context) *CIAuthIdentity {
	idToken, ok := ctx.Value(idTokenKey{}).(*CIAuthIdentity)
	if !ok {
		return nil
	}
	return idToken
}
