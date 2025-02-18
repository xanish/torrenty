package tracker

import (
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"strconv"
	"sync"
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
	mu       sync.RWMutex
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

	slog.Debug("Generated tracker URL", slog.String("tracker_url", trackerURL))

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
		slog.Warn(
			"Tracker returned a warning",
			slog.String("message", trackerResp.WarningMessage),
		)
	}

	peers, err := peer.New(trackerResp.Peers)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to parse peers from tracker response: %w", err)
	}

	interval := time.Duration(trackerResp.Interval) * time.Second
	if interval == 0 && trackerResp.MinInterval > 0 {
		interval = time.Duration(trackerResp.MinInterval) * time.Second
	}

	slog.Info("Tracker response received",
		slog.String("tracker_url", trackerURL),
		slog.Int64("interval", int64(interval.Seconds())),
		slog.Int("num_peers", len(peers)),
	)

	return peers, interval, nil
}

func (t *Tracker) UpdateProgress(downloaded, uploaded uint64) {
	t.mu.Lock()
	defer t.mu.Unlock()

	t.progress.downloaded = downloaded
	t.progress.uploaded = uploaded
	t.progress.remaining = t.torrent.Info.Length - downloaded

	slog.Debug(
		"Tracker progress updated",
		slog.Uint64("downloaded", downloaded),
		slog.Uint64("uploaded", uploaded),
		slog.Uint64("remaining", t.progress.remaining),
	)
}
