package message

import (
	"bytes"
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

	return fmt.Sprintf("message<%s>: <len=%d><id=%d>", m.name(), len(m.Payload), m.ID)
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

func ParseHave(msg *Message) (int, error) {
	if msg.ID != Have {
		return 0, fmt.Errorf("expected message<have> but got %s", msg)
	}

	if len(msg.Payload) != 4 {
		return 0, fmt.Errorf("expected payload length to be 4, got %d", len(msg.Payload))
	}

	index := int(binary.BigEndian.Uint32(msg.Payload))

	return index, nil
}

func ParseRequest(msg *Message) (int, int, int, error) {
	if msg.ID != Request {
		return 0, 0, 0, fmt.Errorf("expected message<request> but got %s", msg)
	}

	if len(msg.Payload) < 12 {
		return 0, 0, 0, fmt.Errorf("expected payload to have exactly 12 bytes, got %d", len(msg.Payload))
	}

	parsedIndex := int(binary.BigEndian.Uint32(msg.Payload[0:4]))
	parsedBegin := int(binary.BigEndian.Uint32(msg.Payload[4:8]))
	parsedLength := int(binary.BigEndian.Uint32(msg.Payload[8:]))

	return parsedIndex, parsedBegin, parsedLength, nil
}

func ParsePiece(index int, out []byte, msg *Message) (int, error) {
	if msg.ID != Piece {
		return 0, fmt.Errorf("expected message<piece> but got %s", msg)
	}

	if len(msg.Payload) < 8 {
		return 0, fmt.Errorf("expected payload to have at-least 8 bytes, got %d", len(msg.Payload))
	}

	parsedIndex := int(binary.BigEndian.Uint32(msg.Payload[0:4]))
	if parsedIndex != index {
		return 0, fmt.Errorf("expected piece index %d, got %d", index, parsedIndex)
	}

	begin := int(binary.BigEndian.Uint32(msg.Payload[4:8]))
	if begin >= len(out) {
		return 0, fmt.Errorf("expected begin offset %d, got %d", begin, len(out))
	}

	data := msg.Payload[8:]
	if begin+len(data) > len(out) {
		return 0, fmt.Errorf("expected data size to be %d bytes, got %d bytes", len(out), len(data))
	}

	copy(out[begin:], data)

	return len(data), nil
}

func ParseCancel(msg *Message) (int, int, int, error) {
	if msg.ID != Cancel {
		return 0, 0, 0, fmt.Errorf("expected message<cancel> but got %s", msg)
	}

	if len(msg.Payload) < 12 {
		return 0, 0, 0, fmt.Errorf("expected payload to have exactly 12 bytes, got %d", len(msg.Payload))
	}

	parsedIndex := int(binary.BigEndian.Uint32(msg.Payload[0:4]))
	parsedBegin := int(binary.BigEndian.Uint32(msg.Payload[4:8]))
	parsedLength := int(binary.BigEndian.Uint32(msg.Payload[8:]))

	return parsedIndex, parsedBegin, parsedLength, nil
}

func NewChoke() *Message {
	return &Message{ID: Choke}
}

func NewUnChoke() *Message {
	return &Message{ID: UnChoke}
}

func NewInterested() *Message {
	return &Message{ID: Interested}
}

func NewNotInterested() *Message {
	return &Message{ID: NotInterested}
}

func NewHave(index int) *Message {
	payload := make([]byte, 4)
	binary.BigEndian.PutUint32(payload, uint32(index))

	return &Message{ID: Have, Payload: payload}
}

func NewBitfield(bitfield []byte) *Message {
	return &Message{ID: Bitfield, Payload: bitfield}
}

func NewRequest(index, begin, length int) *Message {
	payload := make([]byte, 12)
	binary.BigEndian.PutUint32(payload[0:4], uint32(index))
	binary.BigEndian.PutUint32(payload[4:8], uint32(begin))
	binary.BigEndian.PutUint32(payload[8:12], uint32(length))

	return &Message{ID: Request, Payload: payload}
}

func NewPiece(index, begin int, block []byte) *Message {
	payload := &bytes.Buffer{}

	temp := make([]byte, 8)
	binary.BigEndian.PutUint32(temp[0:4], uint32(index))
	binary.BigEndian.PutUint32(temp[4:8], uint32(begin))

	payload.Write(temp)
	payload.Write(block)

	return &Message{ID: Piece, Payload: payload.Bytes()}
}

func NewCancel(index, begin, length int) *Message {
	payload := make([]byte, 12)
	binary.BigEndian.PutUint32(payload[0:4], uint32(index))
	binary.BigEndian.PutUint32(payload[4:8], uint32(begin))
	binary.BigEndian.PutUint32(payload[8:12], uint32(length))

	return &Message{ID: Cancel, Payload: payload}
}

func NewPort(port int) *Message {
	payload := make([]byte, 2)
	binary.BigEndian.PutUint16(payload, uint16(port))

	return &Message{ID: Port, Payload: payload}
}
