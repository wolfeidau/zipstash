package ciauth

import (
	"context"
	"net/http"

	"github.com/rs/zerolog/log"
	"github.com/wolfeidau/zipstash/pkg/trace"
	"go.opentelemetry.io/otel/codes"
)

func NewOIDCAuthMiddleware(audience string, validator *OIDCCachingValidator) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx, span := trace.Start(r.Context(), "OIDC.AuthMiddleware")
			defer span.End()

			rawIDToken, err := extractBearerToken(r.Header)
			if err != nil {
				log.Error().Err(err).Msg("failed to extract bearer token")
				span.SetStatus(codes.Error, err.Error())
				http.Error(w, "invalid token", http.StatusUnauthorized)
				return
			}

			oidcIdentity, err := validator.ValidateToken(ctx, rawIDToken, audience)
			if err != nil {
				log.Error().Err(err).Msg("failed to validate token")
				span.SetStatus(codes.Error, err.Error())
				http.Error(w, "invalid token", http.StatusUnauthorized)
				return
			}

			log.Info().
				Str("provider", oidcIdentity.Provider()).
				Str("subject", oidcIdentity.Subject()).
				Str("issuer", oidcIdentity.Issuer()).
				Str("owner", oidcIdentity.Owner()).
				Msg("OIDC identity")

			ctx = context.WithValue(ctx, oidcIdentityKey{}, oidcIdentity)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
