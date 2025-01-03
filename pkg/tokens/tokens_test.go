package tokens

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestGetToken(t *testing.T) {
	tests := []struct {
		name      string
		source    string
		audience  string
		wantErr   bool
		errString string
	}{
		{
			name:      "unknown source",
			source:    "invalid_source",
			audience:  "test-audience",
			wantErr:   true,
			errString: "unknown token source: invalid_source",
		},
		{
			name:     "github actions source",
			source:   "github_actions",
			audience: "test-audience",
			wantErr:  false,
		},
		{
			name:     "local source",
			source:   "local",
			audience: "test-audience",
			wantErr:  false,
		},
		{
			name:      "empty source",
			source:    "",
			audience:  "test-audience",
			wantErr:   true,
			errString: "unknown token source: ",
		},
		{
			name:     "empty audience",
			source:   "github_actions",
			audience: "",
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// create a test http server
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write([]byte(`{"value": "test-token"}`))
			}))
			defer server.Close()

			t.Setenv("ACTIONS_ID_TOKEN_REQUEST_URL", server.URL)
			t.Setenv("ACTIONS_ID_TOKEN_REQUEST_TOKEN", "test-token")

			token, err := GetToken(context.Background(), tt.source, tt.audience, server.Client())
			if tt.wantErr {
				require.Error(t, err)
				require.Equal(t, tt.errString, err.Error())
				require.Empty(t, token)
				return
			}
			require.NoError(t, err)
			require.NotEmpty(t, token)
		})
	}
}
