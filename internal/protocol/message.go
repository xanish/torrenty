package protocol

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"github.com/xanish/torrenty/internal/bitfield"
	"io"
)

type MsgType uint8

const (
	MsgTypeChoke MsgType = iota
	MsgTypeUnChoke
	MsgTypeInterested
	MsgTypeNotInterested
	MsgTypeHave
	MsgTypeBitfield
	MsgTypeRequest
	MsgTypePiece
	MsgTypeCancel
	MsgTypePort
)

var msgTypeNames = map[MsgType]string{
	MsgTypeChoke:         "Choke",
	MsgTypeUnChoke:       "UnChoke",
	MsgTypeInterested:    "Interested",
	MsgTypeNotInterested: "NotInterested",
	MsgTypeHave:          "Have",
	MsgTypeBitfield:      "Bitfield",
	MsgTypeRequest:       "Request",
	MsgTypePiece:         "Piece",
	MsgTypeCancel:        "Cancel",
	MsgTypePort:          "Port",
}

type Message struct {
	ID      MsgType
	Payload []byte
}

func (m *Message) name() string {
	if m == nil {
		return "KeepAlive"
	}

	return msgTypeNames[m.ID]
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
	buf[4] = byte(m.ID)

	// The payload size is message dependent.
	copy(buf[5:], m.Payload)

	return buf
}

func UnmarshalMessage(r io.Reader) (*Message, error) {
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

	return &Message{
		ID:      MsgType(payloadBuf[0]),
		Payload: payloadBuf[1:],
	}, nil
}

func (m *Message) String() string {
	if m == nil {
		return m.name()
	}

	return fmt.Sprintf("message<%s>: <len=%d><id=%d>", m.name(), len(m.Payload), m.ID)
}

type MessageConf struct {
	Type       MsgType
	Bitfield   bitfield.Bitfield
	PieceIndex uint32
	Piece      []byte
	Begin      uint32
	Length     uint32
	Port       uint16
}

func NewMessage(conf MessageConf) *Message {
	switch conf.Type {
	case MsgTypeChoke:
		return &Message{ID: MsgTypeChoke}
	case MsgTypeUnChoke:
		return &Message{ID: MsgTypeUnChoke}
	case MsgTypeInterested:
		return &Message{ID: MsgTypeInterested}
	case MsgTypeNotInterested:
		return &Message{ID: MsgTypeNotInterested}
	case MsgTypeHave:
		payload := make([]byte, 4)
		binary.BigEndian.PutUint32(payload, conf.PieceIndex)

		return &Message{ID: MsgTypeHave, Payload: payload}
	case MsgTypeBitfield:
		return &Message{ID: MsgTypeBitfield, Payload: conf.Bitfield}
	case MsgTypeRequest:
		payload := make([]byte, 12)
		binary.BigEndian.PutUint32(payload[0:4], conf.PieceIndex)
		binary.BigEndian.PutUint32(payload[4:8], conf.Begin)
		binary.BigEndian.PutUint32(payload[8:12], conf.Length)

		return &Message{ID: MsgTypeRequest, Payload: payload}
	case MsgTypePiece:
		payload := &bytes.Buffer{}
		temp := make([]byte, 8)
		binary.BigEndian.PutUint32(temp[0:4], conf.PieceIndex)
		binary.BigEndian.PutUint32(temp[4:8], conf.Begin)

		payload.Write(temp)
		payload.Write(conf.Piece)

		return &Message{ID: MsgTypePiece, Payload: payload.Bytes()}
	case MsgTypeCancel:
		payload := make([]byte, 12)
		binary.BigEndian.PutUint32(payload[0:4], conf.PieceIndex)
		binary.BigEndian.PutUint32(payload[4:8], conf.Begin)
		binary.BigEndian.PutUint32(payload[8:12], conf.Length)

		return &Message{ID: MsgTypeCancel, Payload: payload}
	case MsgTypePort:
		payload := make([]byte, 2)
		binary.BigEndian.PutUint16(payload, conf.Port)

		return &Message{ID: MsgTypePort, Payload: payload}
	}

	return nil
}

func ParseHave(msg *Message) (int, error) {
	if msg.ID != MsgTypeHave {
		return 0, fmt.Errorf("expected message<have> but got %s", msg)
	}

	if len(msg.Payload) != 4 {
		return 0, fmt.Errorf("expected payload length to be 4, got %d", len(msg.Payload))
	}

	index := int(binary.BigEndian.Uint32(msg.Payload))

	return index, nil
}

func ParseRequest(msg *Message) (int, int, int, error) {
	if msg.ID != MsgTypeRequest {
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
	if msg.ID != MsgTypePiece {
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
	fmt.Println(begin, begin+len(data), len(out))
	if begin+len(data) > len(out) {
		return 0, fmt.Errorf("expected data size to be %d bytes, got %d bytes", len(out), len(data))
	}

	copy(out[begin:], data)

	return len(data), nil
}

func ParseCancel(msg *Message) (int, int, int, error) {
	if msg.ID != MsgTypeCancel {
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
