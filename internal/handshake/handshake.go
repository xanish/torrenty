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
func (h Handshake) Marshal() ([]byte, error) {
	errs := make([]error, 0, 5)
	buf := bytes.NewBuffer(make([]byte, 0, bufLength+len(h.Pstr)))
	errs[0] = buf.WriteByte(byte(len(h.Pstr)))
	_, errs[1] = buf.WriteString(h.Pstr)
	_, errs[2] = buf.Write(h.Reserved[:])
	_, errs[3] = buf.Write(h.InfoHash[:])
	_, errs[4] = buf.Write(h.PeerID[:])

	for _, err := range errs {
		if err != nil {
			return nil, fmt.Errorf("failed to marshal handshake payload: %w", err)
		}
	}

	return buf.Bytes(), nil
}

// Unmarshal converts the bytes metadata received from a remote peer into a
// Handshake object.
func Unmarshal(r io.Reader) (Handshake, error) {
	buf, err := io.ReadAll(r)
	if err != nil {
		return Handshake{}, fmt.Errorf("failed to read handshake payload: %w", err)
	}

	pstrLen := int(buf[0])
	payload := buf[1+pstrLen:]

	return Handshake{
		Pstr:     string(buf[1:pstrLen]),
		Reserved: [8]byte(payload[:8]),
		InfoHash: [20]byte(payload[8:28]),
		PeerID:   [20]byte(payload[28:]),
	}, nil
}
