package archive

import (
	"context"
	"os"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/wolfeidau/zipstash/pkg/trace"
)

func TestBuildArchive(t *testing.T) {
	assert := require.New(t)

	_, err := trace.NewProvider(context.Background(), "test", "0.0.1")
	assert.NoError(err)

	home, err := os.Getwd()
	assert.NoError(err)

	os.Setenv("HOME", home)

	archiveInfo, err := BuildArchive(context.Background(), []string{"testdata"}, "test")
	assert.NoError(err)
	assert.Equal("a61f0392bd89da7c", archiveInfo.Checksum)
	assert.Equal(int64(1228), archiveInfo.Size)

	homeDir, err := os.UserHomeDir()
	assert.NoError(err)
	assert.Equal(home, homeDir)
}
