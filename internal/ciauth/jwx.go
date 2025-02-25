package ciauth

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/lestrrat-go/httprc/v3"
	"github.com/lestrrat-go/jwx/v3/jwk"
	"github.com/lestrrat-go/jwx/v3/jwt"
	"github.com/rs/zerolog/log"
)

// OIDCValidator manages OIDC token validation
type OIDCCachingValidator struct {
	c             *jwk.Cache
	oidcProviders map[string]OIDCProvider
}

func NewOIDCValidator(ctx context.Context, oidcProviders map[string]OIDCProvider) (*OIDCCachingValidator, error) {
	c, err := jwk.NewCache(ctx, httprc.NewClient(
		httprc.WithErrorSink(ZeroLogErrorSink{}),
	))
	if err != nil {
		return nil, fmt.Errorf("failed to create JWK cache: %v", err)
	}

	return &OIDCCachingValidator{
		c:             c,
		oidcProviders: oidcProviders,
	}, nil
}

func (v *OIDCCachingValidator) registerJWKSEndpoints(ctx context.Context, oidcProvider OIDCProvider) error {
	if v.c.IsRegistered(ctx, oidcProvider.JWKSURL) {
		return nil
	}

	if err := v.c.Register(ctx, oidcProvider.JWKSURL,
		jwk.WithMaxInterval(24*time.Hour*7),
		jwk.WithMinInterval(15*time.Minute),
	); err != nil {
		return fmt.Errorf("failed to register providers jwks URL: %v", err)
	}

	return nil
}

func (v *OIDCCachingValidator) ValidateToken(ctx context.Context, tokenStr, expectedAudience string) (OIDCIdentity, error) {
	// parse the JWT with the expected audience
	token, err := jwt.Parse(
		[]byte(tokenStr),
		jwt.WithVerify(false),
		jwt.WithAudience(expectedAudience),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to parse token: %v", err)
	}

	// get the issuer from the token
	tokenIssuer, ok := token.Issuer()
	if !ok {
		return nil, fmt.Errorf("token has no issuer")
	}

	oidcProvider, ok := v.oidcProviders[tokenIssuer]
	if !ok {
		return nil, fmt.Errorf("unknown issuer: %v", tokenIssuer)
	}

	// register the JWK endpoints for the provider
	if err := v.registerJWKSEndpoints(ctx, oidcProvider); err != nil {
		return nil, fmt.Errorf("failed to register JWK endpoints: %v", err)
	}

	// get the JWK set for the issuer
	set, err := v.c.CachedSet(oidcProvider.JWKSURL)
	if err != nil {
		return nil, fmt.Errorf("failed to get JWK set: %v", err)
	}

	// validate the token with the JWK set
	token, err = jwt.Parse(
		[]byte(tokenStr),
		jwt.WithKeySet(set),
		jwt.WithValidate(true),
		jwt.WithIssuer(tokenIssuer),
		jwt.WithAudience(expectedAudience),
	)
	if err != nil {
		return nil, fmt.Errorf("token validation failed: %v", err)
	}

	oidcId := &oidcIdentity{
		provider: oidcProvider.Name,
		token:    token,
	}

	if err := oidcId.parseClaims(); err != nil {
		return nil, fmt.Errorf("failed to parse claims: %v", err)
	}

	return oidcId, nil
}

type OIDCIdentity interface {
	Provider() string
	Claims() any
	Owner() string
	Subject() string
	Issuer() string
}

type oidcIdentityKey struct{}

type oidcIdentity struct {
	token    jwt.Token
	claims   any
	provider string
}

func (oi *oidcIdentity) Provider() string {
	return oi.provider
}

func (oi *oidcIdentity) Claims() any {
	return oi.claims
}

func (oi *oidcIdentity) Subject() string {
	s, _ := oi.token.Subject()
	return s
}

func (oi *oidcIdentity) Issuer() string {
	s, _ := oi.token.Issuer()
	return s
}

func (oi *oidcIdentity) parseClaims() error {
	var claims any
	switch oi.provider {
	case GitHubActions:
		claims = &GitHubActionsClaims{}
	case Buildkite:
		claims = &BuildkiteClaims{}
	default:
		return fmt.Errorf("unsupported provider")
	}

	data, err := json.Marshal(oi.token)
	if err != nil {
		return fmt.Errorf("failed to marshal token: %w", err)
	}

	if err := json.Unmarshal(data, claims); err != nil {
		return fmt.Errorf("failed to parse claims: %w", err)
	}
	oi.claims = claims

	return nil
}

func GetOIDCIdentity(ctx context.Context) OIDCIdentity {
	oidcIdentity, ok := ctx.Value(oidcIdentityKey{}).(OIDCIdentity)
	if !ok {
		return nil
	}
	return oidcIdentity
}

func (oi *oidcIdentity) Owner() string {
	switch claims := oi.claims.(type) {
	case *GitHubActionsClaims:
		return claims.RepositoryOwner
	case *BuildkiteClaims:
		return claims.OrganizationSlug
	default:
		return ""
	}
}

type ZeroLogErrorSink struct {
}

func (ZeroLogErrorSink) Put(ctx context.Context, err error) {
	log.Error().Err(err).Msg("failed to get OIDC identity")
}
