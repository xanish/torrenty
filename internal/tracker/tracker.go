package tracker

import (
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/jackpal/bencode-go"
	"github.com/xanish/torrenty/internal/metadata"
	"github.com/xanish/torrenty/internal/peer"
)

type Tracker struct {
	peerID   [20]byte
	port     uint16
	torrent  metadata.Metadata
	progress Progress
}

type Progress struct {
	downloaded uint64
	uploaded   uint64
	remaining  uint64
}

type Response struct {
	FailureReason  string `bencode:"failure reason"`
	WarningMessage string `bencode:"warning message"`
	Interval       int64  `bencode:"interval"`
	MinInterval    int64  `bencode:"min interval,omitempty"`
	TrackerID      string `bencode:"tracker id"`
	Complete       int64  `bencode:"complete"`
	Incomplete     int64  `bencode:"incomplete"`
	Peers          string `bencode:"peers"`
}

func New(peerID [20]byte, port uint16, torrent metadata.Metadata) *Tracker {
	return &Tracker{
		peerID:  peerID,
		port:    port,
		torrent: torrent,
		progress: Progress{
			remaining: torrent.Info.Length,
		},
	}
}

func (t *Tracker) URL(event string) string {
	baseUrl, _ := url.Parse(t.torrent.Announce)

	params := url.Values{
		"info_hash":  []string{string(t.torrent.InfoHash[:])},
		"peer_id":    []string{string(t.peerID[:])},
		"port":       []string{strconv.Itoa(int(t.port))},
		"uploaded":   []string{strconv.FormatUint(t.progress.uploaded, 10)},
		"downloaded": []string{strconv.FormatUint(t.progress.downloaded, 10)},
		"left":       []string{strconv.FormatUint(t.progress.remaining, 10)},
		"compact":    []string{"1"},
		"event":      []string{event},
	}

	baseUrl.RawQuery = params.Encode()

	return baseUrl.String()
}

func (t *Tracker) Refresh(event string) ([]peer.Peer, time.Duration, error) {
	c := &http.Client{Timeout: 10 * time.Second}
	trackerURL := t.URL(event)

	resp, err := c.Get(trackerURL)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to fetch tracker metadata: %w", err)
	}
	defer resp.Body.Close()

	trackerResp := Response{}
	err = bencode.Unmarshal(resp.Body, &trackerResp)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to decode tracker response: %w", err)
	}

	if trackerResp.FailureReason != "" {
		return nil, 0, fmt.Errorf("tracker returned an error: %s", trackerResp.FailureReason)
	}

	if trackerResp.WarningMessage != "" {
		// todo: log msg
	}

	peers, err := peer.New(trackerResp.Peers)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to parse peers from tracker response: %w", err)
	}

	interval := time.Duration(trackerResp.Interval) * time.Second
	if interval == 0 && trackerResp.MinInterval > 0 {
		interval = time.Duration(trackerResp.MinInterval) * time.Second
	}

	return peers, interval, nil
}
