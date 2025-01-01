package protocol

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"

	"github.com/xanish/torrenty/internal/bitfield"
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
	Type    MsgType
	Payload []byte
}

func (m *Message) name() string {
	if m == nil {
		return "KeepAlive"
	}

	return msgTypeNames[m.Type]
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

	// The message Type is a single decimal byte.
	buf[4] = byte(m.Type)

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
		Type:    MsgType(payloadBuf[0]),
		Payload: payloadBuf[1:],
	}, nil
}

func (m *Message) String() string {
	if m == nil {
		return m.name()
	}

	return fmt.Sprintf("message<%s>: <len=%d><id=%d>", m.name(), len(m.Payload), m.Type)
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
		return &Message{Type: MsgTypeChoke}
	case MsgTypeUnChoke:
		return &Message{Type: MsgTypeUnChoke}
	case MsgTypeInterested:
		return &Message{Type: MsgTypeInterested}
	case MsgTypeNotInterested:
		return &Message{Type: MsgTypeNotInterested}
	case MsgTypeHave:
		payload := make([]byte, 4)
		binary.BigEndian.PutUint32(payload, conf.PieceIndex)

		return &Message{Type: MsgTypeHave, Payload: payload}
	case MsgTypeBitfield:
		return &Message{Type: MsgTypeBitfield, Payload: conf.Bitfield}
	case MsgTypeRequest:
		payload := make([]byte, 12)
		binary.BigEndian.PutUint32(payload[0:4], conf.PieceIndex)
		binary.BigEndian.PutUint32(payload[4:8], conf.Begin)
		binary.BigEndian.PutUint32(payload[8:12], conf.Length)

		return &Message{Type: MsgTypeRequest, Payload: payload}
	case MsgTypePiece:
		payload := &bytes.Buffer{}
		temp := make([]byte, 8)
		binary.BigEndian.PutUint32(temp[0:4], conf.PieceIndex)
		binary.BigEndian.PutUint32(temp[4:8], conf.Begin)

		payload.Write(temp)
		payload.Write(conf.Piece)

		return &Message{Type: MsgTypePiece, Payload: payload.Bytes()}
	case MsgTypeCancel:
		payload := make([]byte, 12)
		binary.BigEndian.PutUint32(payload[0:4], conf.PieceIndex)
		binary.BigEndian.PutUint32(payload[4:8], conf.Begin)
		binary.BigEndian.PutUint32(payload[8:12], conf.Length)

		return &Message{Type: MsgTypeCancel, Payload: payload}
	case MsgTypePort:
		payload := make([]byte, 2)
		binary.BigEndian.PutUint16(payload, conf.Port)

		return &Message{Type: MsgTypePort, Payload: payload}
	}

	return nil
}

func ParseHave(msg *Message) (uint32, error) {
	if msg.Type != MsgTypeHave {
		return 0, fmt.Errorf("expected message<have> but got %s", msg)
	}

	if len(msg.Payload) != 4 {
		return 0, fmt.Errorf("expected payload length to be 4, got %d", len(msg.Payload))
	}

	index := binary.BigEndian.Uint32(msg.Payload)

	return index, nil
}

func ParseRequest(msg *Message) (int, int, int, error) {
	if msg.Type != MsgTypeRequest {
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

type Piece struct {
	Index uint32
	Start uint32
	Data  []byte
}

func ParsePiece(msg *Message) (*Piece, error) {
	if msg.Type != MsgTypePiece {
		return nil, fmt.Errorf("expected message<piece> but got %s", msg)
	}

	if len(msg.Payload) < 8 {
		return nil, fmt.Errorf("expected payload to have at-least 8 bytes, got %d", len(msg.Payload))
	}

	return &Piece{
		Index: binary.BigEndian.Uint32(msg.Payload[0:4]),
		Start: binary.BigEndian.Uint32(msg.Payload[4:8]),
		Data:  msg.Payload[8:],
	}, nil
}

func ParseCancel(msg *Message) (int, int, int, error) {
	if msg.Type != MsgTypeCancel {
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
