package peer

import (
	"bytes"
	"fmt"
	"net"
	"time"

	"github.com/xanish/torrenty/internal/handshake"
	"github.com/xanish/torrenty/internal/logger"
	"github.com/xanish/torrenty/internal/message"
	"github.com/xanish/torrenty/internal/utility"
)

type Connection struct {
	Conn         net.Conn
	Peer         Peer
	Bitfield     []byte
	AmChoked     bool
	AmInterested bool
}

// newConnection tries to set up a connection to the remote peer via handshake.
func newConnection(peer Peer, infoHash, peerID [20]byte) (*Connection, error) {
	conn, err := net.DialTimeout("tcp", peer.String(), 15*time.Second)
	if err != nil {
		return nil, fmt.Errorf("error connecting to %s: %w", peer.String(), err)
	}

	_, err = exchangeHandshake(conn, infoHash, peerID)
	if err != nil {
		// We won't want to defer the connection close since this connection
		// object will be used for fetching pieces. So only close on errors.
		_ = conn.Close()
		return nil, err
	}

	// Once we successfully establish a new connection and exchange a handshake,
	// we immediately receive a Bitfield message from the remote peer. It is
	// optional, and may not be received if the peer has no pieces.
	bitfield, err := readBitfield(conn)
	if err != nil {
		// Only close on errors.
		_ = conn.Close()
		return nil, err
	}

	return &Connection{
		conn,
		peer,
		bitfield,
		true,
		false,
	}, nil
}

// exchangeHandshake initiates handshake to identify itself to the peer and
// inform them about the protocol this client follows and the file it is
// interested in.
func exchangeHandshake(conn net.Conn, infoHash, peerID [20]byte) (*handshake.Handshake, error) {
	_ = conn.SetDeadline(time.Now().Add(5 * time.Second))
	defer func(conn net.Conn, t time.Time) {
		_ = conn.SetDeadline(t)
	}(conn, time.Time{}) // Disable the deadline

	req := handshake.New(infoHash, peerID)
	marshaled, err := req.Marshal()
	if err != nil {
		return nil, fmt.Errorf("failed to marshal handshake request: %w", err)
	}

	_, err = conn.Write(marshaled)
	if err != nil {
		return nil, fmt.Errorf("failed to send handshake request: %w", err)
	}

	res, err := handshake.Unmarshal(conn)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal handshake response: %w", err)
	}

	if !bytes.Equal(res.InfoHash[:], infoHash[:]) {
		return nil, fmt.Errorf("expected infohash to be %x, but got %x", infoHash, res.InfoHash)
	}

	// Ideally we should verify the peerID received in response with the peerID
	// present in non-compacted response of tracker and drop the connection if
	// they do not match

	return res, nil
}

// readBitfield extracts Bitfield from the message payload.
func readBitfield(conn net.Conn) ([]byte, error) {
	_ = conn.SetDeadline(time.Now().Add(5 * time.Second))
	defer func(conn net.Conn, t time.Time) {
		_ = conn.SetDeadline(t)
	}(conn, time.Time{}) // Disable the deadline

	msg, err := message.Unmarshal(conn)
	if err != nil {
		return nil, err
	}

	if msg == nil {
		return nil, fmt.Errorf("expected message<bitfield> but got %s", msg)
	}

	if msg.ID != message.Bitfield {
		return nil, fmt.Errorf("expected message<bitfield> but got %s", msg.ID)
	}

	return msg.Payload, nil
}

// ReadMessage reads the message received from remote Peer.
func (c *Connection) ReadMessage(index int, buf []byte) error {
	msg, err := message.Unmarshal(c.Conn)
	if err != nil {
		return err
	}

	// keep-alive
	if msg == nil {
		logger.Log(logger.Debug, "received msg<KeepAlive> from remote peer %s", c.Peer)
		return nil
	}

	switch msg.ID {
	case message.Choke:
		logger.Log(logger.Debug, "received msg<Choke> from remote peer %s", c.Peer)
		c.AmChoked = true
	case message.UnChoke:
		logger.Log(logger.Debug, "received msg<UnChoke> from remote peer %s", c.Peer)
		c.AmChoked = false
	case message.Interested:
		logger.Log(logger.Debug, "received msg<Interested> from remote peer %s", c.Peer)
		c.AmInterested = true
	case message.NotInterested:
		logger.Log(logger.Debug, "received msg<NotInterested> from remote peer %s", c.Peer)
		c.AmInterested = false
	case message.Have:
		logger.Log(logger.Debug, "received msg<Have, piece:%d> from remote peer %s", index, c.Peer)
		index, err := message.ParseHave(msg)
		if err != nil {
			return err
		}
		utility.SetPiece(index, c.Bitfield)
	case message.Bitfield:
		logger.Log(logger.Debug, "received msg<Bitfield> = %x from remote peer %s", msg.Payload, c.Peer)
		c.Bitfield = msg.Payload
	case message.Request:
		logger.Log(logger.Debug, "received msg<Request> from remote peer %s", c.Peer)
		_, _, _, err := message.ParseRequest(msg)
		if err != nil {
			return err
		}
	case message.Piece:
		logger.Log(logger.Debug, "received msg<Piece, %d> from remote peer %s", index, c.Peer)
		_, err := message.ParsePiece(index, buf, msg)
		if err != nil {
			return err
		}
	case message.Cancel:
		logger.Log(logger.Debug, "received msg<Cancel> from remote peer %s", c.Peer)
	case message.Port:
		logger.Log(logger.Debug, "received msg<Port> from remote peer %s", c.Peer)
	}

	return nil
}

// SendChoke sends a message to Choke the remote Peer.
func (c *Connection) SendChoke() error {
	_, err := c.Conn.Write(message.NewChoke().Marshal())
	if err != nil {
		return fmt.Errorf("failed to send choke: %w", err)
	}

	return nil
}

// SendUnChoke sends a message to UnChoke the remote Peer.
func (c *Connection) SendUnChoke() error {
	_, err := c.Conn.Write(message.NewUnChoke().Marshal())
	if err != nil {
		return fmt.Errorf("failed to send unchoke: %w", err)
	}

	return nil
}

// SendInterested sends a message to denote client interest in the pieces
// provided by remote Peer.
func (c *Connection) SendInterested() error {
	_, err := c.Conn.Write(message.NewInterested().Marshal())
	if err != nil {
		return fmt.Errorf("failed to send interested: %w", err)
	}

	return nil
}

// SendNotInterested sends a message to denote client is not interested in the
// pieces provided by remote Peer.
func (c *Connection) SendNotInterested() error {
	_, err := c.Conn.Write(message.NewNotInterested().Marshal())
	if err != nil {
		return fmt.Errorf("failed to send not interested: %w", err)
	}

	return nil
}

// SendHave sends a message informing the remote Peer that it has received
// the piece present at index.
func (c *Connection) SendHave(index int) error {
	_, err := c.Conn.Write(message.NewHave(index).Marshal())
	if err != nil {
		return fmt.Errorf("failed to send have for index %d: %w", index, err)
	}

	return nil
}

// SendBitField sends a message to the remote Peer containing the Bitfield
// denoting all pieces that are available with the client for sharing.
func (c *Connection) SendBitField(bitfield []byte) error {
	_, err := c.Conn.Write(message.NewBitfield(bitfield).Marshal())
	if err != nil {
		return fmt.Errorf("failed to send bitfield: %w", err)
	}

	return nil
}

// SendRequest sends a message requesting remote Peer to share a block of
// length belonging to piece "index".
func (c *Connection) SendRequest(index, begin, length int) error {
	_, err := c.Conn.Write(message.NewRequest(index, begin, length).Marshal())
	if err != nil {
		return fmt.Errorf("failed to send request for index %d, begin %d, length %d: %w", index, begin, length, err)
	}

	return nil
}

// SendPiece sends a message containing the block starting at begin and
// belonging to piece "index" as requested by remote Peer.
func (c *Connection) SendPiece(index, begin int, block []byte) error {
	_, err := c.Conn.Write(message.NewPiece(index, begin, block).Marshal())
	if err != nil {
		return fmt.Errorf("failed to send piece for index %d, begin %d: %w", index, begin, err)
	}

	return nil
}

// SendCancel sends a message to cancel an earlier request for block starting
// at begin and belonging to piece "index".
func (c *Connection) SendCancel(index, begin, length int) error {
	_, err := c.Conn.Write(message.NewCancel(index, begin, length).Marshal())
	if err != nil {
		return fmt.Errorf("failed to send cancel for index %d, begin %d, length %d: %w", index, begin, length, err)
	}

	return nil
}

// SendPort sends a message used by newer versions of clients that support
// connection via the decentralized DHT tracker network.
func (c *Connection) SendPort(port int) error {
	_, err := c.Conn.Write(message.NewPort(port).Marshal())
	if err != nil {
		return fmt.Errorf("failed to send port for index %d: %w", port, err)
	}

	return nil
}
