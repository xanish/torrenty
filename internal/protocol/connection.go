package protocol

import (
	"bytes"
	"fmt"
	"net"
	"time"
)

type Connection struct {
	conn net.Conn
}

func NewConnection(addr string, peerID, infoHash [20]byte) (*Connection, error) {
	conn, err := net.DialTimeout("tcp", addr, 10*time.Second)
	if err != nil {
		return nil, fmt.Errorf("error connecting to %s: %w", addr, err)
	}

	_, err = exchangeHandshake(conn, infoHash, peerID)
	if err != nil {
		return nil, err
	}

	return &Connection{conn: conn}, nil
}

func exchangeHandshake(conn net.Conn, infoHash, peerID [20]byte) (*Handshake, error) {
	_ = conn.SetDeadline(time.Now().Add(5 * time.Second))
	defer func(conn net.Conn, t time.Time) {
		_ = conn.SetDeadline(t)
	}(conn, time.Time{}) // Disable the deadline

	req := NewHandshake(infoHash, peerID)
	marshaled, err := req.Marshal()
	if err != nil {
		return nil, fmt.Errorf("failed to marshal handshake request: %w", err)
	}

	_, err = conn.Write(marshaled)
	if err != nil {
		return nil, fmt.Errorf("failed to send handshake request: %w", err)
	}

	res, err := UnmarshalHandshake(conn)
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

func (c Connection) ReadMessage() (*Message, error) {
	msg, err := UnmarshalMessage(c.conn)
	if err != nil {
		return nil, err
	}

	// keep-alive
	if msg == nil {
		return nil, nil
	}

	return msg, nil
}

func (c Connection) SendMessage(msg *Message) error {
	_, err := c.conn.Write(msg.Marshal())
	if err != nil {
		return err
	}

	return nil
}

func (c Connection) Close() error {
	return c.conn.Close()
}
