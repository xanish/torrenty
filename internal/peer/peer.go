package peer

import (
	"encoding/binary"
	"fmt"
	"net"
)

const peerNumBytes = 6 // 4 bytes for IP, 2 bytes for port

type Peer struct {
	IP   net.IP
	Port uint16
}

func New(encodedPeers string) ([]Peer, error) {
	peerBytes := []byte(encodedPeers)
	numPeers := len(peerBytes) / peerNumBytes

	if len(peerBytes)%peerNumBytes != 0 {
		return nil, fmt.Errorf("received malformed peers list")
	}

	peers := make([]Peer, 0, numPeers)
	for i := 0; i < numPeers; i++ {
		offset := i * peerNumBytes
		peers = append(peers, Peer{
			IP:   peerBytes[offset : offset+4],
			Port: binary.BigEndian.Uint16(peerBytes[offset+4 : offset+6]),
		})
	}

	return peers, nil
}
