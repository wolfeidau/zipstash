package client

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCheckPath(t *testing.T) {
	homeDir, err := os.UserHomeDir()
	require.NoError(t, err)

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
			path:     "/tmp/test1 /tmp/test2 /tmp/test3",
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
			name:     "path with tilde",
			path:     "~/documents",
			expected: []string{filepath.Join(homeDir, "documents")},
			wantErr:  false,
		},
		{
			name: "multiple paths with tilde",
			path: "~/docs ~/downloads ~/desktop",
			expected: []string{
				filepath.Join(homeDir, "docs"),
				filepath.Join(homeDir, "downloads"),
				filepath.Join(homeDir, "desktop"),
			},
			wantErr: false,
		},
		{
			name: "mixed paths with tilde",
			path: "~/documents /tmp/test ~/downloads",
			expected: []string{
				filepath.Join(homeDir, "documents"),
				"/tmp/test",
				filepath.Join(homeDir, "downloads"),
			},
			wantErr: false,
		},
		{
			name:     "path with spaces",
			path:     "  /tmp/test1   /tmp/test2  ",
			expected: []string{"/tmp/test1", "/tmp/test2"},
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
