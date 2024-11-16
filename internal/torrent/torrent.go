package torrent

import (
	"fmt"
	"io"
	"time"

	"github.com/xanish/torrenty/internal/metadata"
	"github.com/xanish/torrenty/internal/peer"
	"github.com/xanish/torrenty/internal/tracker"
)

const port uint16 = 6881

type Torrent struct {
	metadata        *metadata.Metadata
	tracker         *tracker.Tracker
	peers           []peer.Peer
	refreshInterval time.Duration
}

func FromFile(r io.Reader) (*Torrent, error) {
	meta, err := metadata.FromFile(r)
	if err != nil {
		return nil, fmt.Errorf("failed to parse metadata: %w", err)
	}

	clientID, err := peerID()
	if err != nil {
		return nil, fmt.Errorf("failed to generate peer ID: %w", err)
	}

	track := tracker.New(clientID, port, *meta)

	return &Torrent{metadata: meta, tracker: track}, nil
}

func FromMagnet(url string) (*Torrent, error) {
	return nil, nil
}

func (t *Torrent) Download(destination string) error {
	peers, refreshInterval, err := t.tracker.Refresh()
	if err != nil {
		return fmt.Errorf("failed to refresh tracker metadata: %w", err)
	}

	t.peers = peers
	t.refreshInterval = refreshInterval

	// todo: add logic to begin download using workers

	return nil
}
