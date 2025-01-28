package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestRealIP(t *testing.T) {
	tests := []struct {
		name           string
		headers        map[string]string
		expectedIP     string
		originalRemote string
	}{
		{
			name: "with x-forwarded-for header",
			headers: map[string]string{
				"X-Forwarded-For": "192.168.1.1",
			},
			expectedIP:     "192.168.1.1",
			originalRemote: "10.0.0.1:1234",
		},
		{
			name: "with x-real-ip header",
			headers: map[string]string{
				"X-Real-IP": "192.168.1.2",
			},
			expectedIP:     "192.168.1.2",
			originalRemote: "10.0.0.1:1234",
		},
		{
			name: "with both headers - should prefer x-forwarded-for",
			headers: map[string]string{
				"X-Forwarded-For": "192.168.1.3",
				"X-Real-IP":       "192.168.1.4",
			},
			expectedIP:     "192.168.1.3",
			originalRemote: "10.0.0.1:1234",
		},
		{
			name:           "with no headers",
			headers:        map[string]string{},
			expectedIP:     "10.0.0.1:1234",
			originalRemote: "10.0.0.1:1234",
		},
		{
			name: "with empty header values",
			headers: map[string]string{
				"X-Forwarded-For": "",
				"X-Real-IP":       "",
			},
			expectedIP:     "10.0.0.1:1234",
			originalRemote: "10.0.0.1:1234",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var capturedRemoteAddr string
			nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				capturedRemoteAddr = r.RemoteAddr
			})

			handler := RealIP(nextHandler)
			req := httptest.NewRequest("GET", "http://example.com", nil)
			req.RemoteAddr = tt.originalRemote

			for key, value := range tt.headers {
				req.Header.Set(key, value)
			}

			rr := httptest.NewRecorder()
			handler.ServeHTTP(rr, req)

			require.Equal(t, tt.expectedIP, capturedRemoteAddr)
		})
	}
}
