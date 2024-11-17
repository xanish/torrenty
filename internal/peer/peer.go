package peer

import (
	"encoding/binary"
	"fmt"
	"net"
	"strconv"

	"github.com/xanish/torrenty/internal/bitfield"
	"github.com/xanish/torrenty/internal/protocol"
)

const peerNumBytes = 6 // 4 bytes for IP, 2 bytes for port

type Peer struct {
	ip       net.IP
	port     uint16
	conn     protocol.Connection
	bitfield bitfield.Bitfield
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
			ip:   peerBytes[offset : offset+4],
			port: binary.BigEndian.Uint16(peerBytes[offset+4 : offset+6]),
		})
	}

	return peers, nil
}

func (p *Peer) Connect(peerID, infoHash [20]byte) error {
	return nil
}

func (p *Peer) Close() error {
	return p.conn.Close()
}

func (p *Peer) HasPiece(index uint32) bool { return p.bitfield.HasPiece(index) }

func (p *Peer) Send(msg protocol.MessageConf) error {
	return nil
}

func (p *Peer) Read() ([]byte, error) {
	return nil, nil
}

func (p *Peer) String() string {
	return net.JoinHostPort(p.ip.String(), strconv.Itoa(int(p.port)))
}
