package ciauth

import "errors"

var (
	ErrInvalidProvider = errors.New("invalid provider")

	DefaultOIDCProviders = map[string]OIDCProvider{
		"https://token.actions.githubusercontent.com": {
			Name:    GitHubActions,
			JWKSURL: "https://token.actions.githubusercontent.com/.well-known/jwks",
		},
		"https://gitlab.com": {
			Name:    GitLab,
			JWKSURL: "https://gitlab.com/oauth/discovery/keys",
		},
		"https://agent.buildkite.com": {
			Name:    Buildkite,
			JWKSURL: "https://agent.buildkite.com/.well-known/jwks",
		},
	}

	DefaultProviderNames = []string{
		GitHubActions,
		GitLab,
		Buildkite,
	}
)

type OIDCProvider struct {
	Name    string
	JWKSURL string
}
