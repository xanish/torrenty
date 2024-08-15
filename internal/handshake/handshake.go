package handshake

import (
	"bytes"
	"fmt"
	"io"
)

const bufLength = 49

// Handshake is a required message and must be the first message transmitted
// by the client. It is (49 + len(Pstr)) bytes long.
// Handshake: <PstrLen><Pstr><Reserved><InfoHash><PeerID>
//
// PstrLen: Length of Pstr, as a single raw byte.
//
// Pstr: String identifier of the protocol (eg. BitTorrent protocol).
//
// Reserved: eight (8) reserved bytes. All current implementations use all
// zeroes. Each bit in these bytes can be used to change the behavior of the
// protocol.
//
// InfoHash: 20-byte SHA1 hash of the info key in the metainfo file. This is
// the same hash that is transmitted in tracker requests.
//
// PeerID: 20-byte string used as a unique ID for the client. This is usually
// the same ID that is transmitted in tracker requests (but not always).
type Handshake struct {
	Pstr     string
	Reserved [8]byte
	InfoHash [20]byte
	PeerID   [20]byte
}

// New constructs and returns a Handshake object with the passed infoHash and
// peerID.
func New(infoHash, peerID [20]byte) Handshake {
	return Handshake{
		Pstr:     "BitTorrent protocol",
		Reserved: [8]byte{0x00, 0x00, 0x00, 0x00},
		InfoHash: infoHash,
		PeerID:   peerID,
	}
}

// Marshal converts the handshake metadata into a serialized byte form that can
// be transmitted via the connection.
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

// Unmarshal converts the bytes metadata received from a remote peer into a
// Handshake object.
func Unmarshal(r io.Reader) (*Handshake, error) {
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

	return &Handshake{
		Pstr:     string(payloadBuf[:pstrLen]),
		Reserved: [8]byte(payloadBuf[pstrLen : pstrLen+8]),
		InfoHash: [20]byte(payloadBuf[pstrLen+8 : pstrLen+28]),
		PeerID:   [20]byte(payloadBuf[pstrLen+28:]),
	}, nil
}
