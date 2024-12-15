package server

import "testing"

func TestOffsetsForDownload(t *testing.T) {
	tests := []struct {
		name     string
		total    int64
		partSize int64
		expected []offset
	}{
		{
			name:     "10MB",
			total:    10 * 1024 * 1024, // 10 MB
			partSize: 5 * 1024 * 1024,  // 5 MB
			expected: []offset{
				{
					Part:  1,
					Start: 0,
					End:   5 * 1024 * 1024,
				}, {
					Part:  2,
					Start: 5*1024*1024 + 1,
					End:   10 * 1024 * 1024,
				},
			},
		},
		{
			name:     "14MB",
			total:    14 * 1024 * 1024, // 10 MB
			partSize: 5 * 1024 * 1024,  // 5 MB
			expected: []offset{
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
					End:   14 * 1024 * 1024,
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			offsets := calculateOffsets(tt.total, tt.partSize)
			if len(offsets) != len(tt.expected) {
				t.Errorf("calculateOffsetsForDownload() = %v, want %v", offsets, tt.expected)
			}
			for i, offset := range offsets {
				if offset.Start != tt.expected[i].Start || offset.End != tt.expected[i].End {
					t.Errorf("calculateOffsetsForDownload() = %v, want %v", offsets, tt.expected)
				}
			}
		})
	}
}
