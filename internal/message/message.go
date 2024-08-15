package message

import (
	"encoding/binary"
	"fmt"
	"io"
)

const (
	Choke = iota
	UnChoke
	Interested
	NotInterested
	Have
	Bitfield
	Request
	Piece
	Cancel
	Port
)

type Message struct {
	ID      uint8
	Payload []byte
}

func (m *Message) name() string {
	if m == nil {
		return "KeepAlive"
	}

	messageType := fmt.Sprintf("unknown message id: %d", m.ID)
	switch m.ID {
	case Choke:
		messageType = "Choke"
	case UnChoke:
		messageType = "UnChoke"
	case Interested:
		messageType = "Interested"
	case NotInterested:
		messageType = "NotInterested"
	case Have:
		messageType = "Have"
	case Bitfield:
		messageType = "Bitfield"
	case Request:
		messageType = "Request"
	case Piece:
		messageType = "Piece"
	case Cancel:
		messageType = "Cancel"
	case Port:
		messageType = "Port"
	}

	return messageType
}

func (m *Message) String() string {
	if m == nil {
		return m.name()
	}

	return fmt.Sprintf("%s: <len=%d><id=%d>", m.name(), len(m.Payload), m.ID)
}

func (m *Message) Marshal() []byte {
	// Default keep-alive message with 0 length of four byte big-endian value.
	// A keep-alive message must be sent to maintain the connection alive if no
	// command have been sent for a given amount of time (Generally 2 minutes).
	if m == nil {
		return make([]byte, 4)
	}

	length := uint32(len(m.Payload) + 1)

	// The length prefix is a four byte big-endian value.
	buf := make([]byte, 4+length)
	binary.BigEndian.PutUint32(buf[0:4], length)

	// The message ID is a single decimal byte.
	buf[4] = m.ID

	// The payload size is message dependent.
	copy(buf[5:], m.Payload)

	return buf
}

func Unmarshal(r io.Reader) (*Message, error) {
	lengthBuf := make([]byte, 4)
	_, err := io.ReadFull(r, lengthBuf)
	if err != nil {
		return nil, fmt.Errorf("failed to read message length: %w", err)
	}
	length := binary.BigEndian.Uint32(lengthBuf)

	// Default keep-alive message with 0 length of four byte big-endian value.
	if length == 0 {
		return nil, nil
	}

	payloadBuf := make([]byte, length)
	_, err = io.ReadFull(r, payloadBuf)
	if err != nil {
		return nil, fmt.Errorf("failed to read message payload: %w", err)
	}

	m := Message{
		ID:      payloadBuf[0],
		Payload: payloadBuf[1:],
	}

	return &m, nil
}
