package tracker

import (
	"encoding/binary"
	"fmt"
	"net"
	"net/http"
	"strconv"
	"time"

	"github.com/jackpal/bencode-go"
)

const timeout = 5 * time.Second

type Peer struct {
	IP   net.IP
	Port uint16
}

func (p Peer) String() string {
	return net.JoinHostPort(p.IP.String(), strconv.Itoa(int(p.Port)))
}

type trackerResponse struct {
	Interval int    `bencode:"interval"`
	Peers    string `bencode:"peers"`
}

func Peers(trackerURL string) ([]Peer, error) {
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

	return extractPeers([]byte(tr.Peers))
}

func extractPeers(bytes []byte) ([]Peer, error) {
	const peerBytes = 6 // 4 for IP, 2 for port
	numPeers := len(bytes) / peerBytes

	if len(bytes)%peerBytes != 0 {
		return nil, fmt.Errorf("received malformed peers list")
	}

	peers := make([]Peer, numPeers)
	for i := 0; i < numPeers; i++ {
		offset := i * peerBytes
		peers[i].IP = bytes[offset : offset+4]
		peers[i].Port = binary.BigEndian.Uint16(bytes[offset+4 : offset+6])
	}

	return peers, nil
}
