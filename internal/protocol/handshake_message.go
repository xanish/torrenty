package protocol

import (
	"bytes"
	"fmt"
	"io"
)

const bufLength = 49

type Handshake struct {
	Pstr     string
	Reserved [8]byte
	InfoHash [20]byte
	PeerID   [20]byte
}

func NewHandshake(infoHash, peerID [20]byte) Handshake {
	return Handshake{
		Pstr:     "BitTorrent protocol",
		Reserved: [8]byte{0x00, 0x00, 0x00, 0x00},
		InfoHash: infoHash,
		PeerID:   peerID,
	}
}

func (h *Handshake) Marshal() ([]byte, error) {
	errs := make([]error, 0, 5)
	buf := bytes.NewBuffer(make([]byte, 0, bufLength+len(h.Pstr)))

	errs = append(errs, buf.WriteByte(byte(len(h.Pstr))))

	_, err := buf.WriteString(h.Pstr)
	errs = append(errs, err)

	_, err = buf.Write(h.Reserved[:])
	errs = append(errs, err)

	_, err = buf.Write(h.InfoHash[:])
	errs = append(errs, err)

	_, err = buf.Write(h.PeerID[:])
	errs = append(errs, err)

	for _, err := range errs {
		if err != nil {
			return nil, fmt.Errorf("failed to marshal handshake payload: %w", err)
		}
	}

	return buf.Bytes(), nil
}

func (h *Handshake) Unmarshal(r io.Reader) error {
	lengthBuf := make([]byte, 1)
	_, err := io.ReadFull(r, lengthBuf)
	if err != nil {
		return fmt.Errorf("failed to read handshake payload length: %w", err)
	}

	pstrLen := int(lengthBuf[0])
	if pstrLen == 0 {
		return fmt.Errorf("handshake payload length cannot be 0")
	}

	payloadBuf := make([]byte, 48+pstrLen)
	_, err = io.ReadFull(r, payloadBuf)
	if err != nil {
		return fmt.Errorf("failed to read handshake payload: %w", err)
	}

	h.Pstr = string(payloadBuf[:pstrLen])
	h.Reserved = [8]byte(payloadBuf[pstrLen : pstrLen+8])
	h.InfoHash = [20]byte(payloadBuf[pstrLen+8 : pstrLen+28])
	h.PeerID = [20]byte(payloadBuf[pstrLen+28:])

	return nil
}
