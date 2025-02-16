package server

import (
	"crypto/sha256"
	"encoding/hex"

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

func hashValue(v string) string {
	hash := sha256.Sum256([]byte(v))
	return hex.EncodeToString(hash[:])
}
