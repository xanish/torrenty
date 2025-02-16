package protocol

import (
	"bytes"
	"fmt"
	"io"
)

type Handshake struct {
	Pstr     string
	Reserved [8]byte
	InfoHash [20]byte
	PeerID   [20]byte
}

func NewHandshake(infoHash, peerID [20]byte) Handshake {
	return Handshake{
		Pstr:     "BitTorrent protocol",
		Reserved: [8]byte{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00},
		InfoHash: infoHash,
		PeerID:   peerID,
	}
}

func (h *Handshake) Marshal() ([]byte, error) {
	buf := new(bytes.Buffer)

	buf.WriteByte(byte(len(h.Pstr)))
	buf.WriteString(h.Pstr)
	buf.Write(h.Reserved[:])
	buf.Write(h.InfoHash[:])
	buf.Write(h.PeerID[:])

	return buf.Bytes(), nil
}

func UnmarshalHandshake(r io.Reader) (*Handshake, error) {
	lengthBuf := make([]byte, 1)
	_, err := io.ReadFull(r, lengthBuf)
	if err != nil {
		return nil, fmt.Errorf("failed to read handshake payload length: %w", err)
	}

	pstrLen := int(lengthBuf[0])
	if pstrLen == 0 {
		return nil, fmt.Errorf("handshake payload length cannot be 0")
	}

	payloadBuf := make([]byte, 48+pstrLen)
	_, err = io.ReadFull(r, payloadBuf)
	if err != nil {
		return nil, fmt.Errorf("failed to read handshake payload: %w", err)
	}

	h := &Handshake{
		Pstr:     string(payloadBuf[:pstrLen]),
		Reserved: [8]byte(payloadBuf[pstrLen : pstrLen+8]),
		InfoHash: [20]byte(payloadBuf[pstrLen+8 : pstrLen+28]),
		PeerID:   [20]byte(payloadBuf[pstrLen+28:]),
	}
	return h, nil
}
