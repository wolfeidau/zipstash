package tokens

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
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

type gitHubActions struct {
	httpClient *http.Client
}

func newGitHubActions(httpClient *http.Client) *gitHubActions {
	if httpClient == nil {
		httpClient = http.DefaultClient
	}
	return &gitHubActions{
		httpClient: httpClient,
	}
}

func (gh *gitHubActions) getIDToken(ctx context.Context, audience string) (string, error) {
	requestURL := os.Getenv("ACTIONS_ID_TOKEN_REQUEST_URL")
	if requestURL == "" {
		return "", fmt.Errorf("missing ACTIONS_ID_TOKEN_REQUEST_URL in environment")
	}

	requestToken := os.Getenv("ACTIONS_ID_TOKEN_REQUEST_TOKEN")
	if requestToken == "" {
		return "", fmt.Errorf("missing ACTIONS_ID_TOKEN_REQUEST_TOKEN in environment")
	}

	u, err := url.Parse(requestURL)
	if err != nil {
		return "", fmt.Errorf("failed to parse request URL: %w", err)
	}
	if audience != "" {
		q := u.Query()
		q.Set("audience", audience)
		u.RawQuery = q.Encode()
	}

	req, err := http.NewRequestWithContext(ctx, "GET", u.String(), nil)
	if err != nil {
		return "", fmt.Errorf("failed to create HTTP request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+requestToken)

	resp, err := gh.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to make HTTP request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, maxBodySize))
	if err != nil {
		return "", fmt.Errorf("failed to read response body: %w", err)
	}
	body = bytes.TrimSpace(body)

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("non-successful response from minting OIDC token: %s", body)
	}

	tokenResp := new(idTokenResponse)
	if err := json.Unmarshal(body, tokenResp); err != nil {
		return "", fmt.Errorf("failed to process response as JSON: %w", err)
	}
	return tokenResp.Value, nil
}
