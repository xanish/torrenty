package protocol

import (
	"bytes"
	"encoding/binary"
	"github.com/xanish/torrenty/internal/bitfield"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMessage_Name(t *testing.T) {
	t.Run("valid message type", func(t *testing.T) {
		tests := []struct {
			msgID MsgType
			name  string
		}{
			{MsgTypeChoke, "Choke"},
			{MsgTypeUnChoke, "UnChoke"},
			{MsgTypeInterested, "Interested"},
			{MsgTypeNotInterested, "NotInterested"},
			{MsgTypeHave, "Have"},
			{MsgTypeBitfield, "Bitfield"},
			{MsgTypeRequest, "Request"},
			{MsgTypePiece, "Piece"},
			{MsgTypeCancel, "Cancel"},
			{MsgTypePort, "Port"},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				msg := &Message{ID: tt.msgID}
				assert.Equal(t, tt.name, msg.name())
			})
		}
	})

	t.Run("nil message", func(t *testing.T) {
		var msg *Message
		assert.Equal(t, "KeepAlive", msg.name())
	})
}

func TestMessage_Marshal(t *testing.T) {
	t.Run("valid message with payload", func(t *testing.T) {
		msg := &Message{
			ID:      MsgTypeHave,
			Payload: []byte{0, 0, 0, 1},
		}
		expected := []byte{0, 0, 0, 5, 4, 0, 0, 0, 1}
		assert.Equal(t, expected, msg.Marshal())
	})

	t.Run("nil message (KeepAlive)", func(t *testing.T) {
		var msg *Message
		expected := []byte{0, 0, 0, 0}
		assert.Equal(t, expected, msg.Marshal())
	})
}

func TestUnmarshalMessage(t *testing.T) {
	t.Run("valid message", func(t *testing.T) {
		data := []byte{0, 0, 0, 5, 4, 0, 0, 0, 1}
		buf := bytes.NewReader(data)
		msg, err := UnmarshalMessage(buf)
		require.NoError(t, err)
		assert.NotNil(t, msg)
		assert.Equal(t, MsgTypeHave, msg.ID)
		assert.Equal(t, []byte{0, 0, 0, 1}, msg.Payload)
	})

	t.Run("invalid length", func(t *testing.T) {
		data := []byte{0, 0, 0, 5} // Missing payload
		buf := bytes.NewReader(data)
		msg, err := UnmarshalMessage(buf)
		require.Error(t, err)
		assert.Nil(t, msg)
	})

	t.Run("keep-alive message", func(t *testing.T) {
		data := []byte{0, 0, 0, 0}
		buf := bytes.NewReader(data)
		msg, err := UnmarshalMessage(buf)
		require.NoError(t, err)
		assert.Nil(t, msg)
	})
}

func TestParseHave(t *testing.T) {
	t.Run("valid have message", func(t *testing.T) {
		msg := &Message{
			ID:      MsgTypeHave,
			Payload: []byte{0, 0, 0, 5},
		}
		index, err := ParseHave(msg)
		require.NoError(t, err)
		assert.Equal(t, 5, index)
	})

	t.Run("invalid have message type", func(t *testing.T) {
		msg := &Message{
			ID:      MsgTypeChoke,
			Payload: []byte{0, 0, 0, 5},
		}
		index, err := ParseHave(msg)
		assert.Error(t, err)
		assert.Equal(t, 0, index)
	})

	t.Run("invalid payload length", func(t *testing.T) {
		msg := &Message{
			ID:      MsgTypeHave,
			Payload: []byte{0, 0},
		}
		index, err := ParseHave(msg)
		assert.Error(t, err)
		assert.Equal(t, 0, index)
	})
}

func TestParseRequest(t *testing.T) {
	t.Run("valid request message", func(t *testing.T) {
		msg := &Message{
			ID:      MsgTypeRequest,
			Payload: []byte{0, 0, 0, 5, 0, 0, 0, 10, 0, 0, 0, 15},
		}
		index, begin, length, err := ParseRequest(msg)
		require.NoError(t, err)
		assert.Equal(t, 5, index)
		assert.Equal(t, 10, begin)
		assert.Equal(t, 15, length)
	})

	t.Run("invalid request message type", func(t *testing.T) {
		msg := &Message{
			ID:      MsgTypeChoke,
			Payload: []byte{0, 0, 0, 5, 0, 0, 0, 10, 0, 0, 0, 15},
		}
		index, begin, length, err := ParseRequest(msg)
		assert.Error(t, err)
		assert.Equal(t, 0, index)
		assert.Equal(t, 0, begin)
		assert.Equal(t, 0, length)
	})

	t.Run("invalid payload length", func(t *testing.T) {
		msg := &Message{
			ID:      MsgTypeRequest,
			Payload: []byte{0, 0, 0, 5},
		}
		index, begin, length, err := ParseRequest(msg)
		assert.Error(t, err)
		assert.Equal(t, 0, index)
		assert.Equal(t, 0, begin)
		assert.Equal(t, 0, length)
	})
}

func TestParsePiece(t *testing.T) {
	t.Run("valid piece message", func(t *testing.T) {
		msg := &Message{
			ID:      MsgTypePiece,
			Payload: append([]byte{0, 0, 0, 5, 0, 0, 0, 10}, []byte("piece data")...),
		}
		out := make([]byte, 20)
		n, err := ParsePiece(5, out, msg)
		require.NoError(t, err)
		assert.Equal(t, 10, n)
		assert.Equal(t, "piece data", string(out[10:]))
	})

	t.Run("invalid piece message type", func(t *testing.T) {
		msg := &Message{
			ID:      MsgTypeChoke,
			Payload: []byte{0, 0, 0, 5, 0, 0, 0, 10},
		}
		out := make([]byte, 20)
		n, err := ParsePiece(5, out, msg)
		assert.Error(t, err)
		assert.Equal(t, 0, n)
	})

	t.Run("invalid payload length", func(t *testing.T) {
		msg := &Message{
			ID:      MsgTypePiece,
			Payload: []byte{0, 0, 0, 5, 0, 0, 0},
		}
		out := make([]byte, 20)
		n, err := ParsePiece(5, out, msg)
		assert.Error(t, err)
		assert.Equal(t, 0, n)
	})

	t.Run("begin offset exceeds buffer size", func(t *testing.T) {
		msg := &Message{
			ID:      MsgTypePiece,
			Payload: append([]byte{0, 0, 0, 5, 0, 0, 0, 10}, []byte("piece data")...),
		}
		out := make([]byte, 10) // Smaller buffer
		n, err := ParsePiece(5, out, msg)
		assert.Error(t, err)
		assert.Equal(t, 0, n)
	})
}

func TestParseCancel(t *testing.T) {
	t.Run("valid cancel message", func(t *testing.T) {
		msg := &Message{
			ID:      MsgTypeCancel,
			Payload: []byte{0, 0, 0, 5, 0, 0, 0, 10, 0, 0, 0, 15},
		}
		index, begin, length, err := ParseCancel(msg)
		require.NoError(t, err)
		assert.Equal(t, 5, index)
		assert.Equal(t, 10, begin)
		assert.Equal(t, 15, length)
	})

	t.Run("invalid cancel message type", func(t *testing.T) {
		msg := &Message{
			ID:      MsgTypeChoke,
			Payload: []byte{0, 0, 0, 5, 0, 0, 0, 10, 0, 0, 0, 15},
		}
		index, begin, length, err := ParseCancel(msg)
		assert.Error(t, err)
		assert.Equal(t, 0, index)
		assert.Equal(t, 0, begin)
		assert.Equal(t, 0, length)
	})

	t.Run("invalid payload length", func(t *testing.T) {
		msg := &Message{
			ID:      MsgTypeCancel,
			Payload: []byte{0, 0},
		}
		index, begin, length, err := ParseCancel(msg)
		assert.Error(t, err)
		assert.Equal(t, 0, index)
		assert.Equal(t, 0, begin)
		assert.Equal(t, 0, length)
	})
}

func TestNewMessage(t *testing.T) {
	t.Run("valid choke message", func(t *testing.T) {
		msg := NewMessage(MessageConf{Type: MsgTypeChoke})
		assert.NotNil(t, msg)
		assert.Equal(t, MsgTypeChoke, msg.ID)
		assert.Empty(t, msg.Payload)
	})

	t.Run("valid un-choke message", func(t *testing.T) {
		msg := NewMessage(MessageConf{Type: MsgTypeUnChoke})
		assert.NotNil(t, msg)
		assert.Equal(t, MsgTypeUnChoke, msg.ID)
		assert.Empty(t, msg.Payload)
	})

	t.Run("valid interested message", func(t *testing.T) {
		msg := NewMessage(MessageConf{Type: MsgTypeInterested})
		assert.NotNil(t, msg)
		assert.Equal(t, MsgTypeInterested, msg.ID)
		assert.Empty(t, msg.Payload)
	})

	t.Run("valid not-interested message", func(t *testing.T) {
		msg := NewMessage(MessageConf{Type: MsgTypeNotInterested})
		assert.NotNil(t, msg)
		assert.Equal(t, MsgTypeNotInterested, msg.ID)
		assert.Empty(t, msg.Payload)
	})

	t.Run("valid have message", func(t *testing.T) {
		conf := MessageConf{
			Type:       MsgTypeHave,
			PieceIndex: 10,
		}
		msg := NewMessage(conf)
		assert.NotNil(t, msg)
		assert.Equal(t, MsgTypeHave, msg.ID)
		assert.Equal(t, []byte{0, 0, 0, 10}, msg.Payload)
	})

	t.Run("create bitfield message", func(t *testing.T) {
		bitfieldData := bitfield.Bitfield{0x01, 0x02, 0x03}
		msg := NewMessage(MessageConf{
			Type:     MsgTypeBitfield,
			Bitfield: bitfieldData,
		})

		assert.NotNil(t, msg)
		assert.Equal(t, MsgTypeBitfield, msg.ID)
		assert.Equal(t, []byte(bitfieldData), msg.Payload)
	})

	t.Run("valid request message", func(t *testing.T) {
		conf := MessageConf{
			Type:       MsgTypeRequest,
			PieceIndex: 5,
			Begin:      10,
			Length:     15,
		}
		msg := NewMessage(conf)
		assert.NotNil(t, msg)
		assert.Equal(t, MsgTypeRequest, msg.ID)
		expectedPayload := []byte{0, 0, 0, 5, 0, 0, 0, 10, 0, 0, 0, 15}
		assert.Equal(t, expectedPayload, msg.Payload)
	})

	t.Run("create piece message", func(t *testing.T) {
		pieceIndex := uint32(5)
		begin := uint32(10)
		pieceData := []byte{0x01, 0x02, 0x03, 0x04}

		msg := NewMessage(MessageConf{
			Type:       MsgTypePiece,
			PieceIndex: pieceIndex,
			Begin:      begin,
			Piece:      pieceData,
		})

		assert.NotNil(t, msg)
		assert.Equal(t, MsgTypePiece, MsgType(msg.ID))

		// Validate the structure of the payload: first 8 bytes are index and begin, then the piece data
		expectedPayload := []byte{0x0, 0x0, 0x0, 0x5, 0x0, 0x0, 0x0, 0xa, 0x1, 0x2, 0x3, 0x4}
		assert.Equal(t, expectedPayload, msg.Payload)
	})

	t.Run("create cancel message", func(t *testing.T) {
		pieceIndex := uint32(5)
		begin := uint32(10)
		length := uint32(15)

		msg := NewMessage(MessageConf{
			Type:       MsgTypeCancel,
			PieceIndex: pieceIndex,
			Begin:      begin,
			Length:     length,
		})

		assert.NotNil(t, msg)
		assert.Equal(t, MsgTypeCancel, MsgType(msg.ID))

		// Validate the payload
		expectedPayload := []byte{0x0, 0x0, 0x0, 0x5, 0x0, 0x0, 0x0, 0xa, 0x0, 0x0, 0x0, 0xf}
		assert.Equal(t, expectedPayload, msg.Payload)
	})

	t.Run("create port message", func(t *testing.T) {
		port := uint16(6881) // Example port number

		msg := NewMessage(MessageConf{
			Type: MsgTypePort,
			Port: port,
		})

		assert.NotNil(t, msg)
		assert.Equal(t, MsgTypePort, MsgType(msg.ID))

		// Validate the port message payload (should be a 2-byte payload for the port)
		expectedPayload := make([]byte, 2)
		binary.BigEndian.PutUint16(expectedPayload, port)
		assert.Equal(t, expectedPayload, msg.Payload)
	})

	t.Run("invalid message type", func(t *testing.T) {
		conf := MessageConf{
			Type: MsgType(100), // Invalid type
		}
		msg := NewMessage(conf)
		assert.Nil(t, msg)
	})
}

func TestMessage_String(t *testing.T) {
	t.Run("valid message string", func(t *testing.T) {
		msg := &Message{
			ID:      MsgTypeHave,
			Payload: []byte{0, 0, 0, 1},
		}
		expected := "message<Have>: <len=4><id=4>"
		assert.Equal(t, expected, msg.String())
	})

	t.Run("nil message string", func(t *testing.T) {
		var msg *Message
		assert.Equal(t, "KeepAlive", msg.String())
	})
}
