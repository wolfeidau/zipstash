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
	assert.Equal("aaac9235bbbb7ef591fa8c11829ddf12d5a56ff30eeee303434af961ab569788", archiveInfo.Sha256sum)
	assert.Equal(int64(1234), archiveInfo.Size)

	homeDir, err := os.UserHomeDir()
	assert.NoError(err)
	assert.Equal(home, homeDir)
}
