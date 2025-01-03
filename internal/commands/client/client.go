package client

import (
	"bufio"
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/wolfeidau/zipstash/pkg/client"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
)

func newClient(endpoint, token, version string) (*client.ClientWithResponses, error) {

	httpClient := &http.Client{Transport: otelhttp.NewTransport(http.DefaultTransport)}

	cl, err := client.NewClientWithResponses(endpoint, client.WithHTTPClient(httpClient), client.WithRequestEditorFn(func(ctx context.Context, req *http.Request) error {
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))
		req.Header.Set("User-Agent", fmt.Sprintf("zipstash/%s", version))
		return nil
	}))
	if err != nil {
		return nil, fmt.Errorf("failed to create client: %w", err)
	}

	return cl, nil
}

func SplitLines(s string) []string {
	var lines []string
	sc := bufio.NewScanner(strings.NewReader(s))
	for sc.Scan() {
		lines = append(lines, sc.Text())
	}
	return lines
}
