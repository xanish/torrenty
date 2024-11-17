package bitfield

import "testing"

func TestHasPiece(t *testing.T) {
	tests := []struct {
		name     string
		bf       Bitfield
		index    int
		expected bool
	}{
		{
			name:     "has piece set at index 0",
			bf:       Bitfield{0b10000000},
			index:    0,
			expected: true,
		},
		{
			name:     "does not have piece set at index 1",
			bf:       Bitfield{0b10000000},
			index:    1,
			expected: false,
		},
		{
			name:     "has piece set at index 7",
			bf:       Bitfield{0b00000001},
			index:    7,
			expected: true,
		},
		{
			name:     "out of bounds index",
			bf:       Bitfield{0b10000000},
			index:    8,
			expected: false,
		},
		{
			name:     "empty bitfield",
			bf:       Bitfield{},
			index:    0,
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.bf.HasPiece(uint32(tt.index))
			if result != tt.expected {
				t.Errorf("HasPiece(%d) = %v, want %v", tt.index, result, tt.expected)
			}
		})
	}
}

func TestSetPiece(t *testing.T) {
	tests := []struct {
		name     string
		bf       Bitfield
		index    int
		expected Bitfield
	}{
		{
			name:     "set piece at index 0",
			bf:       Bitfield{0b00000000},
			index:    0,
			expected: Bitfield{0b10000000},
		},
		{
			name:     "set piece at index 7",
			bf:       Bitfield{0b00000000},
			index:    7,
			expected: Bitfield{0b00000001},
		},
		{
			name:     "set piece in middle of byte",
			bf:       Bitfield{0b01000000},
			index:    2,
			expected: Bitfield{0b01100000},
		},
		{
			name:     "set piece at out of bounds index",
			bf:       Bitfield{0b10000000},
			index:    8,
			expected: Bitfield{0b10000000},
		},
		{
			name:     "empty bitfield",
			bf:       Bitfield{},
			index:    0,
			expected: Bitfield{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.bf.SetPiece(uint32(tt.index))
			for i := range tt.bf {
				if tt.bf[i] != tt.expected[i] {
					t.Errorf("SetPiece(%d) = %v, want %v", tt.index, tt.bf, tt.expected)
				}
			}
		})
	}
}
