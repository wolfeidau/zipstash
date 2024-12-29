package client

import (
	"context"
	"fmt"
	"net/http"

	"github.com/wolfeidau/cache-service/pkg/client"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
)

type ClientCmd struct {
	Save    SaveCmd    `cmd:"" help:"save a cache entry."`
	Restore RestoreCmd `cmd:"" help:"restore a cache entry."`
}

func newClient(endpoint, token string) (*client.ClientWithResponses, error) {

	httpClient := &http.Client{Transport: otelhttp.NewTransport(http.DefaultTransport)}

	cl, err := client.NewClientWithResponses(endpoint, client.WithHTTPClient(httpClient), client.WithRequestEditorFn(func(ctx context.Context, req *http.Request) error {
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))
		return nil
	}))
	if err != nil {
		return nil, fmt.Errorf("failed to create client: %w", err)
	}

	return cl, nil
}
