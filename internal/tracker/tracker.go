package tracker

import (
	"encoding/binary"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/jackpal/bencode-go"
	"github.com/xanish/torrenty/internal/peer"
)

const timeout = 5 * time.Second

type rawResponse struct {
	Interval      int    `bencode:"interval"`
	Peers         string `bencode:"peers"`
	FailureReason string `bencode:"failure reason,omitempty"`
}

type Response struct {
	Peers           []peer.Peer
	RefreshInterval int
}

func Sync(trackerURL string) (*Response, error) {
	c := &http.Client{Timeout: timeout}

	resp, err := c.Get(trackerURL)
	if err != nil {
		return nil, err
	}
	defer func(Body io.ReadCloser) {
		_ = Body.Close()
	}(resp.Body)

	rr := rawResponse{}
	err = bencode.Unmarshal(resp.Body, &rr)
	if err != nil {
		return nil, fmt.Errorf("failed to decode tracker response: %w", err)
	}

	if rr.FailureReason != "" {
		return nil, fmt.Errorf("failed to fetch peers from tracker due to: %s", rr.FailureReason)
	}

	peers, err := extractPeers([]byte(rr.Peers))
	if err != nil {
		return nil, err
	}

	return &Response{
		Peers:           peers,
		RefreshInterval: rr.Interval,
	}, nil
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
