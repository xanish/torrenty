package torrent

import (
	"crypto/rand"
	"fmt"
)

func peerID() ([20]byte, error) {
	var id [20]byte

	_, err := rand.Read(id[:])
	if err != nil {
		return [20]byte{}, fmt.Errorf("failed to generate random peer ID: %w", err)
	}

	return id, nil
}
