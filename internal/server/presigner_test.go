package server

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestOffsetsForDownload(t *testing.T) {
	tests := []struct {
		name     string
		total    int64
		partSize int64
		expected []Offset
	}{
		{
			name:     "10MB",
			total:    10 * 1024 * 1024, // 10 MB
			partSize: 5 * 1024 * 1024,  // 5 MB
			expected: []Offset{
				{
					Part:  1,
					Start: 0,
					End:   5 * 1024 * 1024,
				}, {
					Part:  2,
					Start: 5*1024*1024 + 1,
					End:   10*1024*1024 - 1,
				},
			},
		},
		{
			name:     "14MB",
			total:    14 * 1024 * 1024, // 14 MB
			partSize: 5 * 1024 * 1024,  // 5 MB
			expected: []Offset{
				{
					Part:  1,
					Start: 0,
					End:   5 * 1024 * 1024,
				}, {
					Part:  2,
					Start: 5*1024*1024 + 1,
					End:   10 * 1024 * 1024,
				},
				{
					Part:  3,
					Start: 10*1024*1024 + 1,
					End:   14*1024*1024 - 1,
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert := require.New(t)
			offsets := calculateOffsets(tt.total, tt.partSize)
			if len(offsets) != len(tt.expected) {
				t.Errorf("calculateOffsetsForDownload() = %v, want %v", offsets, tt.expected)
			}

			for i, offset := range offsets {
				assert.Equal(tt.expected[i].Part, offset.Part)
				assert.Equal(tt.expected[i].Start, offset.Start)
				assert.Equal(tt.expected[i].End, offset.End)
			}

		})
	}
}
