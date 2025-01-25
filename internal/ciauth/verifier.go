package ciauth

import (
	"context"
	"fmt"
	"sync"

	"github.com/coreos/go-oidc/v3/oidc"
	"golang.org/x/sync/singleflight"
)

type OIDCProviders struct {
	providerEndpoints map[string]string // readonly and set at creation time
	requestGroup      singleflight.Group
	cache             sync.Map
}

func NewOIDCProviders(providerEndpoints map[string]string) *OIDCProviders {
	return &OIDCProviders{
		providerEndpoints: providerEndpoints,
	}
}

func (o *OIDCProviders) VerifyToken(ctx context.Context, provider, token string) (*oidc.IDToken, error) {
	// Validate provider exists in provider endpoints
	if _, ok := o.providerEndpoints[provider]; !ok {
		return nil, fmt.Errorf("invalid provider")
	}

	// Try from cache first
	if p, ok := o.cache.Load(provider); ok {
		prov, ok := p.(*oidc.Provider)
		if !ok {
			return nil, fmt.Errorf("invalid provider")
		}
		return verifyWithProvider(ctx, token, prov)
	}

	// Create new provider if not in cache
	p, err, _ := o.requestGroup.Do(string(provider), func() (any, error) {
		prov, err := oidc.NewProvider(ctx, o.providerEndpoints[provider])
		if err != nil {
			return nil, fmt.Errorf("failed to create provider: %w", err)
		}
		o.cache.Store(provider, prov)
		return prov, nil
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get provider: %w", err)
	}

	prov, ok := p.(*oidc.Provider)
	if !ok {
		return nil, fmt.Errorf("invalid provider")
	}

	return verifyWithProvider(ctx, token, prov)
}

func verifyWithProvider(ctx context.Context, token string, prov *oidc.Provider) (*oidc.IDToken, error) {
	return prov.Verifier(&oidc.Config{
		SkipClientIDCheck: true,
	}).Verify(ctx, token)
}

type CIAuthIdentity struct {
	IDToken  *oidc.IDToken
	Provider string
	Subject  string
	claims   any
}

func (cia *CIAuthIdentity) ParseClaims() error {
	var claims any
	switch cia.Provider {
	case GitHubActions:
		claims = &GitHubActionsClaims{}
	case Buildkite:
		claims = &BuildkiteClaims{}
	default:
		return fmt.Errorf("unsupported provider")
	}

	if err := cia.IDToken.Claims(claims); err != nil {
		return fmt.Errorf("failed to parse claims: %w", err)
	}
	cia.claims = claims

	return nil
}

func (cia *CIAuthIdentity) GetOwner() string {
	switch claims := cia.claims.(type) {
	case *GitHubActionsClaims:
		return claims.RepositoryOwner
	case *BuildkiteClaims:
		return claims.OrganizationSlug
	default:
		return ""
	}
}
