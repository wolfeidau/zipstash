package ciauth

import (
	"context"
	"fmt"
	"sync"

	"github.com/coreos/go-oidc/v3/oidc"
	"golang.org/x/sync/singleflight"
)

const (
	GitHubActions = "GitHubActions"
	GitLab        = "GitLab"
	Buildkite     = "Buildkite"
)

var (
	DefaultProviderEndpoints = map[string]string{
		GitHubActions: "https://token.actions.githubusercontent.com",
		GitLab:        "https://gitlab.com",
		Buildkite:     "https://buildkite.com",
	}
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
	p, err, _ := o.requestGroup.Do(provider, func() (any, error) {
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
