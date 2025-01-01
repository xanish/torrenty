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
	ip           net.IP
	port         uint16
	conn         protocol.Connection
	bitfield     bitfield.Bitfield
	amChoked     bool
	amInterested bool
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
	conn, err := protocol.NewConnection(p.String(), peerID, infoHash)
	if err != nil {
		return err
	}
	p.conn = *conn

	msg, err := p.conn.ReadMessage()
	if err != nil {
		return err
	}

	_, err = p.parseMessage(msg)

	return err
}

func (p *Peer) parseMessage(msg *protocol.Message) (*protocol.Piece, error) {
	switch msg.Type {
	case protocol.MsgTypeChoke:
		p.amChoked = true
	case protocol.MsgTypeUnChoke:
		p.amChoked = false
	case protocol.MsgTypeInterested:
		p.amInterested = true
	case protocol.MsgTypeNotInterested:
		p.amInterested = false
	case protocol.MsgTypeHave:
		index, err := protocol.ParseHave(msg)
		if err != nil {
			return nil, err
		}
		p.bitfield.SetPiece(index)
	case protocol.MsgTypeBitfield:
		p.bitfield = msg.Payload
	case protocol.MsgTypeRequest:
		_, _, _, err := protocol.ParseRequest(msg)
		if err != nil {
			return nil, err
		}
	case protocol.MsgTypePiece:
		piece, err := protocol.ParsePiece(msg)
		if err != nil {
			return nil, err
		}

		return piece, nil
	case protocol.MsgTypeCancel:
		// todo: add logic
	case protocol.MsgTypePort:
		// todo: add logic
	}

	return nil, nil
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
