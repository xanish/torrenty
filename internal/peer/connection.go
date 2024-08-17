package peer

import (
	"bytes"
	"fmt"
	"net"
	"time"

	"github.com/xanish/torrenty/internal/handshake"
	"github.com/xanish/torrenty/internal/message"
)

type Connection struct {
	Conn     net.Conn
	Bitfield []byte
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
		bitfield,
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

// readBitfield tries to read the Bitfield message if it was sent by the remote
// peer on a successful handshake.
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
		return nil, fmt.Errorf("expected bitfield message but got %s", msg)
	}

	if msg.ID != message.Bitfield {
		return nil, fmt.Errorf("expected bitfield message but got %s", msg.ID)
	}

	return msg.Payload, nil
}

func (c *Connection) SendChoke() error {
	_, err := c.Conn.Write(message.NewChoke().Marshal())
	if err != nil {
		return fmt.Errorf("failed to send choke: %w", err)
	}

	return nil
}

func (c *Connection) SendUnChoke() error {
	_, err := c.Conn.Write(message.NewUnChoke().Marshal())
	if err != nil {
		return fmt.Errorf("failed to send unchoke: %w", err)
	}

	return nil
}

func (c *Connection) SendInterested() error {
	_, err := c.Conn.Write(message.NewInterested().Marshal())
	if err != nil {
		return fmt.Errorf("failed to send interested: %w", err)
	}

	return nil
}

func (c *Connection) SendNotInterested() error {
	_, err := c.Conn.Write(message.NewNotInterested().Marshal())
	if err != nil {
		return fmt.Errorf("failed to send not interested: %w", err)
	}

	return nil
}

func (c *Connection) SendHave(index int) error {
	_, err := c.Conn.Write(message.NewHave(index).Marshal())
	if err != nil {
		return fmt.Errorf("failed to send have for index %d: %w", index, err)
	}

	return nil
}

func (c *Connection) SendBitField(bitfield []byte) error {
	_, err := c.Conn.Write(message.NewBitfield(bitfield).Marshal())
	if err != nil {
		return fmt.Errorf("failed to send bitfield: %w", err)
	}

	return nil
}

func (c *Connection) SendRequest(index, begin, length int) error {
	_, err := c.Conn.Write(message.NewRequest(index, begin, length).Marshal())
	if err != nil {
		return fmt.Errorf("failed to send request for index %d, begin %d, length %d: %w", index, begin, length, err)
	}

	return nil
}

func (c *Connection) SendPiece(index, begin int, block []byte) error {
	_, err := c.Conn.Write(message.NewPiece(index, begin, block).Marshal())
	if err != nil {
		return fmt.Errorf("failed to send piece for index %d, begin %d: %w", index, begin, err)
	}

	return nil
}

func (c *Connection) SendCancel(index, begin, length int) error {
	_, err := c.Conn.Write(message.NewCancel(index, begin, length).Marshal())
	if err != nil {
		return fmt.Errorf("failed to send cancel for index %d, begin %d, length %d: %w", index, begin, length, err)
	}

	return nil
}

func (c *Connection) SendPort(port int) error {
	_, err := c.Conn.Write(message.NewPort(port).Marshal())
	if err != nil {
		return fmt.Errorf("failed to send port for index %d: %w", port, err)
	}

	return nil
}
