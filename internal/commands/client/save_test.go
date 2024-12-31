package client

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCheckPath(t *testing.T) {
	tests := []struct {
		name     string
		path     string
		expected []string
		wantErr  bool
	}{
		{
			name:     "single path",
			path:     "/tmp/test",
			expected: []string{"/tmp/test"},
			wantErr:  false,
		},
		{
			name:     "multiple paths",
			path:     "/tmp/test1\n/tmp/test2\n/tmp/test3",
			expected: []string{"/tmp/test1", "/tmp/test2", "/tmp/test3"},
			wantErr:  false,
		},
		{
			name:     "empty path",
			path:     "",
			expected: nil,
			wantErr:  true,
		},
		{
			name:     "multiple paths with tilde",
			path:     "~/documents\n~/.npm",
			expected: []string{"~/documents", "~/.npm"},
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := checkPath(tt.path)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			require.Equal(t, tt.expected, result)
		})
	}
}
