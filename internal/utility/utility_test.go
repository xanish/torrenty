package utility

import (
	"bytes"
	"testing"
)

func TestPieceExists(t *testing.T) {
	bitfield := []byte{0b10110101, 0b01010000}
	expected := []bool{true, false, true, true, false, true, false, true, false, true, false, true, false, false, false, false, false}
	for i, want := range expected {
		got := PieceExists(i, bitfield)
		if got != want {
			if want {
				t.Errorf("expected piece %d to be set", i)
			} else {
				t.Errorf("expected piece %d to not be set", i)
			}
		}
	}
}

func TestSetPiece(t *testing.T) {
	tests := []struct {
		bitfield []byte
		piece    int
		expected []byte
	}{
		{
			[]byte{0b10110101, 0b01010000},
			15,
			[]byte{0b10110101, 0b01010001},
		},
		{
			[]byte{0b10110101, 0b01010000},
			20,
			[]byte{0b10110101, 0b01010000},
		},
		{
			[]byte{0b10110101, 0b01010000},
			0,
			[]byte{0b10110101, 0b01010000},
		},
	}

	for _, test := range tests {
		got := test.bitfield
		SetPiece(test.piece, got)
		want := test.expected

		if !bytes.Equal(got, want) {
			t.Errorf("expected bitfield to be %08b, got %08b", want, got)
		}
	}
}
