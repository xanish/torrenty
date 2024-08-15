// Package metadata exports utilities that allow parsing metadata from .torrent
// files.
package metadata

import (
	"bytes"
	"crypto/sha1"
	"encoding/json"
	"fmt"
	"io"
	"net/url"
	"strconv"

	"github.com/jackpal/bencode-go"
	"github.com/xanish/torrenty/internal/peer"
)

const sha1HashLen = 20

type pieceInfo struct {
	Pieces      string `bencode:"pieces"`
	PieceLength int    `bencode:"piece length"`
	Length      int    `bencode:"length"`
	Name        string `bencode:"name"`
}

// hash generates the sha1 hash of the piece info which forms the info_hash
// passed to trackers to uniquely identify the torrent being downloaded.
func (pi pieceInfo) hash() ([20]byte, error) {
	var encoded bytes.Buffer
	err := bencode.Marshal(&encoded, pi)
	if err != nil {
		return [20]byte{}, fmt.Errorf("failed to encode piece info: %w", err)
	}

	return sha1.Sum(encoded.Bytes()), nil
}

type baseInfo struct {
	Announce string    `bencode:"announce"`
	Info     pieceInfo `bencode:"info"`
}

type Metadata struct {
	Name        string      `json:"name"`
	Size        int         `json:"size"`
	Announce    string      `json:"announce"`
	InfoHash    [20]byte    `json:"infoHash"`
	Pieces      []string    `json:"pieces"`
	PieceLength int         `json:"pieceLength"`
	Peers       []peer.Peer `json:"peers"`
}

func (m *Metadata) SetPeers(peers []peer.Peer) {
	m.Peers = peers
}

func (m *Metadata) TrackerURL(peerID [20]byte, port uint16) (string, error) {
	baseUrl, err := url.Parse(m.Announce)
	if err != nil {
		return "", fmt.Errorf("failed to parse announce url: %w", err)
	}

	params := url.Values{
		"info_hash":  []string{string(m.InfoHash[:])},
		"peer_id":    []string{string(peerID[:])},
		"port":       []string{strconv.Itoa(int(port))},
		"uploaded":   []string{"0"},
		"downloaded": []string{"0"},
		"compact":    []string{"1"},
		"left":       []string{strconv.Itoa(m.Size)},
	}

	baseUrl.RawQuery = params.Encode()

	return baseUrl.String(), nil
}

func (m *Metadata) String() (string, error) {
	bytes, err := json.Marshal(m)
	if err != nil {
		return "", fmt.Errorf("failed to encode metadata to string: %w", err)
	}

	return string(bytes), nil
}

func New(r io.Reader) (Metadata, error) {
	bi := baseInfo{}

	err := bencode.Unmarshal(r, &bi)
	if err != nil {
		return Metadata{}, fmt.Errorf("failed to decode torrent metadata: %w", err)
	}

	infoHash, err := bi.Info.hash()
	if err != nil {
		return Metadata{}, err
	}

	pieces, err := split(bi.Info.Pieces)
	if err != nil {
		return Metadata{}, fmt.Errorf("failed to parse pieces: %w", err)
	}

	return Metadata{
		Name:        bi.Info.Name,
		Size:        bi.Info.Length,
		Announce:    bi.Announce,
		InfoHash:    infoHash,
		Pieces:      pieces,
		PieceLength: bi.Info.PieceLength,
		Peers:       make([]peer.Peer, 0),
	}, nil
}

func split(pieces string) ([]string, error) {
	buf := []byte(pieces)

	if len(buf)%sha1HashLen != 0 {
		return nil, fmt.Errorf("piece length is malformed: received %d", len(buf))
	}

	numHashes := len(buf) / sha1HashLen
	hashes := make([]string, numHashes)

	for i := 0; i < numHashes; i++ {
		hashes[i] = string(buf[i*sha1HashLen : (i+1)*sha1HashLen])
	}

	return hashes, nil
}
