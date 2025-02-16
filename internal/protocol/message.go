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

func (m *Message) Name() string {
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

func (m *Message) String() string {
	if m == nil {
		return m.Name()
	}

	return fmt.Sprintf("message<%s>: <len=%d><id=%d>", m.Name(), len(m.Payload), m.Type)
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

func NewChokeMessage() *Message {
	return &Message{Type: MsgTypeChoke}
}

func NewUnChokeMessage() *Message {
	return &Message{Type: MsgTypeUnChoke}
}

func NewInterestedMessage() *Message {
	return &Message{Type: MsgTypeInterested}
}

func NewNotInterestedMessage() *Message {
	return &Message{Type: MsgTypeNotInterested}
}

func NewHaveMessage(pieceIndex uint32) *Message {
	payload := make([]byte, 4)
	binary.BigEndian.PutUint32(payload, pieceIndex)

	return &Message{Type: MsgTypeHave, Payload: payload}
}

func NewBitfieldMessage(bf bitfield.Bitfield) *Message {
	return &Message{Type: MsgTypeBitfield, Payload: bf}
}

func NewRequestMessage(pieceIndex, begin, length uint32) *Message {
	payload := make([]byte, 12)
	binary.BigEndian.PutUint32(payload[0:4], pieceIndex)
	binary.BigEndian.PutUint32(payload[4:8], begin)
	binary.BigEndian.PutUint32(payload[8:12], length)

	return &Message{Type: MsgTypeRequest, Payload: payload}
}

func NewPieceMessage(pieceIndex, begin uint32, piece []byte) *Message {
	buf := new(bytes.Buffer)
	binary.Write(buf, binary.BigEndian, pieceIndex)
	binary.Write(buf, binary.BigEndian, begin)
	buf.Write(piece)

	return &Message{Type: MsgTypePiece, Payload: buf.Bytes()}
}

func NewCancelMessage(pieceIndex, begin, length uint32) *Message {
	payload := make([]byte, 12)
	binary.BigEndian.PutUint32(payload[0:4], pieceIndex)
	binary.BigEndian.PutUint32(payload[4:8], begin)
	binary.BigEndian.PutUint32(payload[8:12], length)

	return &Message{Type: MsgTypeCancel, Payload: payload}
}

func NewPortMessage(port uint16) *Message {
	payload := make([]byte, 2)
	binary.BigEndian.PutUint16(payload, port)

	return &Message{Type: MsgTypePort, Payload: payload}
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
