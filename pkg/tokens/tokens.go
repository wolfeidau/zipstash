package tokens

import (
	"context"
	"fmt"
	"net/http"
)

const (
	maxBodySize = 64 * 1024 // 64KB
)

// GetToken retrieves an ID token from the specified source. It supports the "github_actions" and "local" sources.
// If the source is "github_actions", it will use the ACTIONS_ID_TOKEN_REQUEST_URL and ACTIONS_ID_TOKEN_REQUEST_TOKEN
// environment variables to request an ID token from the GitHub Actions service. If the source is "local", it will
// return a hardcoded ID token.
func GetToken(ctx context.Context, source, audience string, httpClient *http.Client) (string, error) {
	switch source {
	case "github_actions":
		return newGitHubActions(httpClient).getIDToken(ctx, audience)
	case "buildkite":
		return newBuildkite().getIDToken(ctx, audience)
	case "local":
		return getLocalIDToken(ctx, audience)
	default:
		return "", fmt.Errorf("unknown token source: %s", source)
	}
}

func getLocalIDToken(_ context.Context, _ string) (string, error) {
	return "abc123cde456", nil
}

// idTokenResponse is the response from minting an ID token.
type idTokenResponse struct {
	Value string `json:"value,omitempty"`
}
