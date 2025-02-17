package peer

import (
	"encoding/binary"
	"fmt"
	"log/slog"
	"net"
	"strconv"

	"github.com/xanish/torrenty/internal/bitfield"
	"github.com/xanish/torrenty/internal/protocol"
)

const peerNumBytes = 6 // 4 bytes for IP, 2 bytes for port

type Peer struct {
	ip             net.IP
	port           uint16
	conn           protocol.Connection
	bitfield       bitfield.Bitfield
	amChoked       bool // we are choked by the remote peer
	amInterested   bool // we are interested in the remote peer
	peerChoked     bool // the remote peer is choked by us
	peerInterested bool // the remote peer is interested in us
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
			ip:             peerBytes[offset : offset+4],
			port:           binary.BigEndian.Uint16(peerBytes[offset+4 : offset+6]),
			amChoked:       true,
			amInterested:   false,
			peerChoked:     true,
			peerInterested: false,
		})
	}

	return peers, nil
}

func (p *Peer) Connect(peerID, infoHash [20]byte) error {
	slog.Debug("Connecting to peer", slog.String("peer", p.String()))

	conn, err := protocol.NewConnection(p.String())
	if err != nil {
		return fmt.Errorf("could not create connection to peer %s:%d: %s", p.ip, p.port, err)
	}

	p.conn = conn
	err = p.conn.Handshake(peerID, infoHash)
	if err != nil {
		return fmt.Errorf("handshake failed: %s", err)
	}

	slog.Info("Handshake completed with peer", slog.String("peer", p.String()))

	msg, err := p.conn.ReadMessage()
	if err != nil {
		return fmt.Errorf("could not read message after handshake: %s", err)
	}

	if msg.Type != protocol.MsgTypeBitfield {
		return fmt.Errorf("expected bitfield message after handshake, got %v", msg.Name())
	}

	_, err = p.parseMessage(msg)

	return err
}

func (p *Peer) parseMessage(msg *protocol.Message) (*protocol.Piece, error) {
	switch msg.Type {
	case protocol.MsgTypeChoke:
		p.amChoked = true
		slog.Debug("Received CHOKE message from peer", slog.String("peer", p.String()))
	case protocol.MsgTypeUnChoke:
		p.amChoked = false
		slog.Debug("Received UNCHOKE message from peer", slog.String("peer", p.String()))
	case protocol.MsgTypeInterested:
		p.peerInterested = true
		slog.Debug("Received INTERESTED message from peer", slog.String("peer", p.String()))
	case protocol.MsgTypeNotInterested:
		p.peerInterested = false
		slog.Debug("Received NOT INTERESTED message from peer", slog.String("peer", p.String()))
	case protocol.MsgTypeHave:
		index, err := protocol.ParseHave(msg)
		if err != nil {
			return nil, err
		}
		p.bitfield.SetPiece(index)
		slog.Debug(
			"Received HAVE message from peer",
			slog.String("peer", p.String()),
			slog.Int("piece_index", int(index)),
		)
	case protocol.MsgTypeBitfield:
		p.bitfield = msg.Payload
		slog.Debug(
			"Received BITFIELD message from peer",
			slog.String("peer", p.String()),
			slog.Int("bitfield_length", len(p.bitfield)),
		)
	case protocol.MsgTypeRequest:
		_, _, _, err := protocol.ParseRequest(msg)
		if err != nil {
			return nil, err
		}
		slog.Debug("Received REQUEST message from peer", slog.String("peer", p.String()))
	case protocol.MsgTypePiece:
		piece, err := protocol.ParsePiece(msg)
		if err != nil {
			return nil, err
		}

		slog.Debug(
			"Received PIECE message from peer",
			slog.String("peer", p.String()),
			slog.Int("piece_index", int(piece.Index)),
			slog.Int("block_offset", int(piece.Start)),
			slog.Int("block_length", len(piece.Data)),
		)
		return piece, nil
	case protocol.MsgTypeCancel:
		// todo: add logic
		slog.Debug("Received CANCEL message from peer", slog.String("peer", p.String()))
	case protocol.MsgTypePort:
		// todo: add logic
		slog.Debug("Received PORT message from peer", slog.String("peer", p.String()))
	}

	return nil, nil
}

func (p *Peer) Close() error {
	if !p.conn.IsClosed() {
		return p.conn.Close()
	}

	return nil
}

func (p *Peer) HasPiece(index uint32) bool { return p.bitfield.HasPiece(index) }

func (p *Peer) Send(msg *protocol.Message) error {
	err := p.conn.SendMessage(msg)
	if err != nil {
		return err
	}

	if msg.Type == protocol.MsgTypeChoke {
		p.peerChoked = true
	} else if msg.Type == protocol.MsgTypeUnChoke {
		p.peerChoked = false
	} else if msg.Type == protocol.MsgTypeInterested {
		p.amInterested = true
	} else if msg.Type == protocol.MsgTypeNotInterested {
		p.amInterested = false
	}

	return nil
}

func (p *Peer) Receive() (*protocol.Message, error) {
	return p.conn.ReadMessage()
}

func (p *Peer) String() string {
	return net.JoinHostPort(p.ip.String(), strconv.Itoa(int(p.port)))
}
