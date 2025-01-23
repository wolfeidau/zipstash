package server

import (
	providerv1 "github.com/wolfeidau/zipstash/api/gen/proto/go/provider/v1"
	"github.com/wolfeidau/zipstash/internal/provider"
)

func fromProviderV1(prov providerv1.Provider) string {
	switch prov {
	case providerv1.Provider_PROVIDER_GITHUB_ACTIONS:
		return provider.GitHubActions
	case providerv1.Provider_PROVIDER_GITLAB:
		return provider.GitLab
	case providerv1.Provider_PROVIDER_BUILDKITE:
		return provider.Buildkite
	default:
		return provider.Unspecified
	}
}
