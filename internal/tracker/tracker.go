package tracker

import (
	"encoding/binary"
	"fmt"
	"net/http"
	"time"

	"github.com/jackpal/bencode-go"
	"github.com/xanish/torrenty/internal/peer"
)

const timeout = 5 * time.Second

type trackerResponse struct {
	Interval      int    `bencode:"interval"`
	Peers         string `bencode:"peers"`
	FailureReason string `bencode:"failure reason,omitempty"`
}

func Peers(trackerURL string) ([]peer.Peer, error) {
	c := &http.Client{Timeout: timeout}

	resp, err := c.Get(trackerURL)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	tr := trackerResponse{}
	err = bencode.Unmarshal(resp.Body, &tr)
	if err != nil {
		return nil, fmt.Errorf("failed to decode tracker response: %w", err)
	}

	if tr.FailureReason != "" {
		return nil, fmt.Errorf("failed to fetch peers from tracker due to: %s", tr.FailureReason)
	}

	return extractPeers([]byte(tr.Peers))
}

func extractPeers(bytes []byte) ([]peer.Peer, error) {
	const peerBytes = 6 // 4 for IP, 2 for port
	numPeers := len(bytes) / peerBytes

	if len(bytes)%peerBytes != 0 {
		return nil, fmt.Errorf("received malformed peers list")
	}

	peers := make([]peer.Peer, numPeers)
	for i := 0; i < numPeers; i++ {
		offset := i * peerBytes
		peers[i].IP = bytes[offset : offset+4]
		peers[i].Port = binary.BigEndian.Uint16(bytes[offset+4 : offset+6])
	}

	return peers, nil
}
