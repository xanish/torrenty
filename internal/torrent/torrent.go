package torrent

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"math"
	"time"

	"github.com/xanish/torrenty/internal/metadata"
	"github.com/xanish/torrenty/internal/peer"
	"github.com/xanish/torrenty/internal/tracker"
	workerpool "github.com/xanish/torrenty/internal/worker_pool"
)

const port uint16 = 6881

type Torrent struct {
	ctx             context.Context
	clientID        [20]byte
	metadata        *metadata.Metadata
	tracker         *tracker.Tracker
	peers           []peer.Peer
	refreshInterval time.Duration
}

func FromFile(ctx context.Context, r io.Reader) (*Torrent, error) {
	meta, err := metadata.FromFile(r)
	if err != nil {
		return nil, fmt.Errorf("failed to parse metadata: %w", err)
	}

	clientID, err := peerID()
	if err != nil {
		return nil, fmt.Errorf("failed to generate peer ID: %w", err)
	}

	track := tracker.New(clientID, port, *meta)

	return &Torrent{metadata: meta, tracker: track, clientID: clientID, ctx: ctx}, nil
}

func FromMagnet(ctx context.Context, url string) (*Torrent, error) {
	return nil, nil
}

func (t *Torrent) Name() string {
	return t.metadata.Info.Name
}

func (t *Torrent) Size() uint64 {
	return t.metadata.Info.Length
}

func (t *Torrent) Download(destination io.WriterAt) error {
	peers, refreshInterval, err := t.tracker.Refresh("started")
	if err != nil {
		return fmt.Errorf("failed to refresh tracker metadata: %w", err)
	}

	t.peers = peers
	t.refreshInterval = refreshInterval

	workers := make([]workerpool.Worker[PieceWork, PieceWork], 0, len(peers))
	for id, remotePeer := range t.peers {
		workers = append(workers, NewPieceWorker(id+1, t.clientID, t.metadata.InfoHash, remotePeer))
	}

	// todo: maybe segregate logic for single and multi file download
	jobs, done := make(chan PieceWork), make(chan PieceWork)
	defer func() {
		close(jobs)
		close(done)
	}()

	pieces, err := t.metadata.Info.PieceList()
	if err != nil {
		return fmt.Errorf("failed to split and parse pieces: %w", err)
	}

	for index, hash := range pieces {
		pieceSize := t.metadata.Info.PieceLength
		numPieces := int(math.Ceil(float64(t.metadata.Info.Length) / float64(pieceSize)))
		if index == numPieces-1 {
			pieceSize = uint32(t.metadata.Info.Length % uint64(t.metadata.Info.PieceLength))
		}
		jobs <- PieceWork{uint32(index), hash, pieceSize, make([]byte, pieceSize)}
	}

	pool := workerpool.New(workers, jobs, done)

	go func() {
		pool.DoWork(t.ctx)
	}()

	donePieces := 0
	for donePieces < len(pieces) {
		res := <-done
		offset := int64(res.index * t.metadata.Info.PieceLength)
		_, err := destination.WriteAt(res.result, offset)
		if err != nil {
			// for now keeping things simple, just fail if we were unable to write any downloaded piece
			return fmt.Errorf("failed writing response for piece %d at offset  %d: %w", res.index, offset, err)
		}

		donePieces++
		t.tracker.UpdateProgress(uint64(len(res.result)), 0)
	}

	go func() {
		// todo: now that we're refreshing tracker maybe i should start connecting to new peers if we get any
		t.refreshTrackerLoop()
	}()

	return nil
}

func (t *Torrent) refreshTrackerLoop() {
	ticker := time.NewTicker(t.refreshInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			peers, interval, err := t.tracker.Refresh("started")
			if err != nil {
				slog.Error("Failed to refresh tracker", slog.Any("error", err))
				continue
			}

			t.peers = peers
			t.refreshInterval = interval
			slog.Info(
				"Tracker refreshed",
				slog.Int("num_peers", len(t.peers)),
				slog.Duration("interval", t.refreshInterval),
			)

		case <-t.ctx.Done():
			_, _, err := t.tracker.Refresh("stopped")
			if err != nil {
				slog.Error("Failed to send stop event to tracker", slog.Any("error", err))
				continue
			}

			slog.Info("Tracker refresh loop stopped")
			return
		}
	}
}
