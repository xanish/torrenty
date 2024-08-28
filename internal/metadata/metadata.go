// Package metadata exports utilities that allow parsing metadata from .torrent
// files.
package metadata

import (
	"bytes"
	"crypto/sha1"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/jackpal/bencode-go"
	"github.com/xanish/torrenty/internal/peer"
)

const (
	sha1HashLen = 20
	timeout     = 5 * time.Second
)

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
	Name            string      `json:"name"`
	Size            int         `json:"size"`
	Announce        string      `json:"announce"`
	InfoHash        [20]byte    `json:"infoHash"`
	Pieces          [][20]byte  `json:"pieces"`
	PieceLength     int         `json:"pieceLength"`
	Peers           []peer.Peer `json:"peers"`
	RefreshInterval int         `json:"refreshInterval"`
}

func (m *Metadata) SetPeers(peers []peer.Peer) {
	m.Peers = peers
}

func (m *Metadata) SetRefreshInterval(duration int) {
	m.RefreshInterval = duration
}

func (m *Metadata) trackerURL(peerID [20]byte, port uint16) (string, error) {
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

type rawResponse struct {
	Interval      int    `bencode:"interval"`
	Peers         string `bencode:"peers"`
	FailureReason string `bencode:"failure reason,omitempty"`
}

type Response struct {
	Peers           []peer.Peer
	RefreshInterval int
}

func (m *Metadata) SyncWithTracker(peerID [20]byte, port uint16) (*Response, error) {
	c := &http.Client{Timeout: timeout}

	trackerURL, err := m.trackerURL(peerID, port)
	if err != nil {
		return nil, err
	}

	resp, err := c.Get(trackerURL)
	if err != nil {
		return nil, err
	}
	defer func(Body io.ReadCloser) {
		_ = Body.Close()
	}(resp.Body)

	rr := rawResponse{}
	err = bencode.Unmarshal(resp.Body, &rr)
	if err != nil {
		return nil, fmt.Errorf("failed to decode tracker response: %w", err)
	}

	if rr.FailureReason != "" {
		return nil, fmt.Errorf("failed to fetch peers from tracker due to: %s", rr.FailureReason)
	}

	peers, err := extractPeers([]byte(rr.Peers))
	if err != nil {
		return nil, err
	}

	return &Response{
		Peers:           peers,
		RefreshInterval: rr.Interval,
	}, nil
}

func (m *Metadata) String() (string, error) {
	buf, err := json.Marshal(m)
	if err != nil {
		return "", fmt.Errorf("failed to encode metadata to string: %w", err)
	}

	return string(buf), nil
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

func split(pieces string) ([][20]byte, error) {
	buf := []byte(pieces)

	if len(buf)%sha1HashLen != 0 {
		return nil, fmt.Errorf("piece length is malformed: received %d", len(buf))
	}

	numHashes := len(buf) / sha1HashLen
	hashes := make([][20]byte, numHashes)

	for i := 0; i < numHashes; i++ {
		copy(hashes[i][:], buf[i*sha1HashLen:(i+1)*sha1HashLen])
	}

	return hashes, nil
}

func extractPeers(bytes []byte) ([]peer.Peer, error) {
	const peerBytes = 6 // 4 for IP, 2 for port
	numPeers := len(bytes) / peerBytes

	if len(bytes)%peerBytes != 0 {
		return nil, fmt.Errorf("received malformed peers list")
	}

	peers := make([]peer.Peer, numPeers)
	for i := 0; i < numPeers; i++ {
		offset := i * peerBytes
		peers[i].IP = bytes[offset : offset+4]
		peers[i].Port = binary.BigEndian.Uint16(bytes[offset+4 : offset+6])
	}

	return peers, nil
}
