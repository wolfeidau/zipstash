package server

import (
	"net/url"
	"path"

	providerv1 "github.com/wolfeidau/zipstash/api/gen/proto/go/provider/v1"
	"github.com/wolfeidau/zipstash/internal/ciauth"
)

func fromProviderV1(prov providerv1.Provider) string {
	switch prov {
	case providerv1.Provider_PROVIDER_GITHUB_ACTIONS:
		return ciauth.GitHubActions
	case providerv1.Provider_PROVIDER_GITLAB:
		return ciauth.GitLab
	case providerv1.Provider_PROVIDER_BUILDKITE:
		return ciauth.Buildkite
	default:
		return ciauth.Unspecified
	}
}

// escapeValue escapes a string for use in a delimited index value.
// Using escape as it means the value is still readable but the value is safe
// to use in a delimited index.
func escapeValue(v string) string {
	return url.QueryEscape(v)
}

func buildCacheKey(owner, provider, os, arch, key string) string {
	return path.Join(owner, provider, os, arch, key)
}
