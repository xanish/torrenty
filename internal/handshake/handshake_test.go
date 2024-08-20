package handshake

import (
	"bytes"
	"errors"
	"reflect"
	"testing"
)

func TestNew(t *testing.T) {
	infoHash := [20]byte{6, 113, 44, 71, 91, 121, 93, 30, 30, 115, 54, 33, 113, 104, 85, 108, 101, 76, 27, 11}
	peerID := [20]byte{2, 69, 110, 76, 7, 82, 70, 59, 76, 87, 10, 20, 89, 109, 16, 62, 90, 11, 9, 64}
	got := New(infoHash, peerID)
	want := Handshake{
		Pstr:     "BitTorrent protocol",
		Reserved: [8]byte{0x00, 0x00, 0x00, 0x00},
		InfoHash: [20]byte{6, 113, 44, 71, 91, 121, 93, 30, 30, 115, 54, 33, 113, 104, 85, 108, 101, 76, 27, 11},
		PeerID:   [20]byte{2, 69, 110, 76, 7, 82, 70, 59, 76, 87, 10, 20, 89, 109, 16, 62, 90, 11, 9, 64},
	}

	if !reflect.DeepEqual(got, want) {
		t.Errorf("New(infoHash, peerID) = %#v; want %#v", got, want)
	}
}

func TestHandshake_Marshal(t *testing.T) {
	h := New(
		[20]byte{6, 113, 44, 71, 91, 121, 93, 30, 30, 115, 54, 33, 113, 104, 85, 108, 101, 76, 27, 11},
		[20]byte{2, 69, 110, 76, 7, 82, 70, 59, 76, 87, 10, 20, 89, 109, 16, 62, 90, 11, 9, 64},
	)

	got, err := h.Marshal()
	want := []byte{0x13, 0x42, 0x69, 0x74, 0x54, 0x6f, 0x72, 0x72, 0x65, 0x6e, 0x74, 0x20, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x63, 0x6f, 0x6c, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x6, 0x71, 0x2c, 0x47, 0x5b, 0x79, 0x5d, 0x1e, 0x1e, 0x73, 0x36, 0x21, 0x71, 0x68, 0x55, 0x6c, 0x65, 0x4c, 0x1b, 0xb, 0x2, 0x45, 0x6e, 0x4c, 0x7, 0x52, 0x46, 0x3b, 0x4c, 0x57, 0xa, 0x14, 0x59, 0x6d, 0x10, 0x3e, 0x5a, 0xb, 0x9, 0x40}

	if err != nil {
		t.Fatalf("expected error to be nil, got \"%v\"", err)
	}

	if !reflect.DeepEqual(got, want) {
		t.Errorf("Handshake.Marshal = %#v; want %#v", got, want)
	}
}

func TestUnmarshal(t *testing.T) {
	tests := map[string]struct {
		input  []byte
		output *Handshake
		err    error
	}{
		"should unmarshal successfully": {
			input: []byte{0x13, 0x42, 0x69, 0x74, 0x54, 0x6f, 0x72, 0x72, 0x65, 0x6e, 0x74, 0x20, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x63, 0x6f, 0x6c, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x6, 0x71, 0x2c, 0x47, 0x5b, 0x79, 0x5d, 0x1e, 0x1e, 0x73, 0x36, 0x21, 0x71, 0x68, 0x55, 0x6c, 0x65, 0x4c, 0x1b, 0xb, 0x2, 0x45, 0x6e, 0x4c, 0x7, 0x52, 0x46, 0x3b, 0x4c, 0x57, 0xa, 0x14, 0x59, 0x6d, 0x10, 0x3e, 0x5a, 0xb, 0x9, 0x40},
			output: &Handshake{
				Pstr:     "BitTorrent protocol",
				Reserved: [8]byte{0x00, 0x00, 0x00, 0x00},
				InfoHash: [20]byte{6, 113, 44, 71, 91, 121, 93, 30, 30, 115, 54, 33, 113, 104, 85, 108, 101, 76, 27, 11},
				PeerID:   [20]byte{2, 69, 110, 76, 7, 82, 70, 59, 76, 87, 10, 20, 89, 109, 16, 62, 90, 11, 9, 64},
			},
			err: nil,
		},
		"should fail if no payload or payload length detected": {
			input:  []byte{},
			output: nil,
			err:    errors.New("failed to read handshake payload length: EOF"),
		},
		"should fail when PstrLen is 0": {
			input:  []byte{0},
			output: nil,
			err:    errors.New("handshake payload length cannot be 0"),
		},
		"should fail on partial payload": {
			input:  []byte{0x13, 0x42, 0x69, 0x74, 0x54, 0x6f, 0x72, 0x72, 0x65},
			output: nil,
			err:    errors.New("failed to read handshake payload: unexpected EOF"),
		},
	}

	for _, test := range tests {
		h, err := Unmarshal(bytes.NewReader(test.input))
		if test.err == nil {
			if err != nil {
				t.Errorf("expected error to be nil, got %#v", err)
			}

			if !reflect.DeepEqual(h, test.output) {
				t.Errorf("Unmarshal = %#v; want %#v", h, test.output)
			}
		} else {
			if err == nil {
				t.Errorf("expected error to be %#v, got %#v", test.err, err)
			}

			if err.Error() != test.err.Error() {
				t.Errorf("Unmarshal = %#v; want %#v", err, test.err)
			}
		}
	}
}
