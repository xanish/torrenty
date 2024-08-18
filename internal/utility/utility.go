package utility

import (
	"crypto/rand"
	"fmt"
)

func PeerID() ([20]byte, error) {
	var id [20]byte

	_, err := rand.Read(id[:])
	if err != nil {
		return [20]byte{}, fmt.Errorf("failed to generate random peer ID: %w", err)
	}

	return id, nil
}

func PieceExists(index int, bitfield []byte) bool {
	byteIndex := index / 8
	offset := index % 8

	if byteIndex < 0 || byteIndex >= len(bitfield) {
		return false
	}

	return bitfield[byteIndex]>>uint(7-offset)&1 != 0
}

func SetPiece(index int, bitfield []byte) {
	byteIndex := index / 8
	offset := index % 8

	if byteIndex < 0 || byteIndex >= len(bitfield) {
		return
	}

	bitfield[byteIndex] |= 1 << uint(7-offset)
}
