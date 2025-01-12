package ciauth

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestExtractBearerToken(t *testing.T) {
	tests := []struct {
		name        string
		authHeader  string
		expected    string
		expectError bool
	}{
		{
			name:        "valid bearer token",
			authHeader:  "Bearer abc123xyz",
			expected:    "abc123xyz",
			expectError: false,
		},
		{
			name:        "missing bearer prefix",
			authHeader:  "abc123xyz",
			expected:    "",
			expectError: true,
		},
		{
			name:        "empty authorization header",
			authHeader:  "",
			expected:    "",
			expectError: true,
		},
		{
			name:        "malformed with multiple Bearer words",
			authHeader:  "Bearer Bearer token123",
			expected:    "",
			expectError: true,
		},
		{
			name:        "wrong auth type",
			authHeader:  "Basic abc123xyz",
			expected:    "",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req, _ := http.NewRequest("GET", "/", nil)
			if tt.authHeader != "" {
				req.Header.Set("Authorization", tt.authHeader)
			}

			token, err := extractBearerToken(req.Header)
			if tt.expectError {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
			require.Equal(t, tt.expected, token)
		})
	}
}
