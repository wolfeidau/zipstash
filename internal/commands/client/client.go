package client

import (
	"bufio"
	"strings"

	"github.com/wolfeidau/zipstash/api/gen/proto/go/cache/v1/cachev1connect"
	providerv1 "github.com/wolfeidau/zipstash/api/gen/proto/go/provider/v1"
)

const audience = "zipstash.wolfe.id.au"

type Globals struct {
	Debug   bool
	Version string
	Client  cachev1connect.CacheServiceClient
}

func SplitLines(s string) []string {
	var lines []string
	sc := bufio.NewScanner(strings.NewReader(s))
	for sc.Scan() {
		lines = append(lines, sc.Text())
	}
	return lines
}

func convertProviderTypeV1(tokenSource string) providerv1.Provider {
	switch tokenSource {
	case "github_actions":
		return providerv1.Provider_PROVIDER_GITHUB_ACTIONS
	case "buildkite":
		return providerv1.Provider_PROVIDER_BUILDKITE
	case "gitlab":
		return providerv1.Provider_PROVIDER_GITLAB
	default:
		return providerv1.Provider_PROVIDER_UNSPECIFIED
	}
}
