package torrent

import (
	"fmt"
	"io"
	"math"
	"time"

	"github.com/xanish/torrenty/internal/metadata"
	"github.com/xanish/torrenty/internal/peer"
	"github.com/xanish/torrenty/internal/tracker"
	workerpool "github.com/xanish/torrenty/internal/worker_pool"
)

const port uint16 = 6881

type Torrent struct {
	clientID        [20]byte
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

	return &Torrent{metadata: meta, tracker: track, clientID: clientID}, nil
}

func FromMagnet(url string) (*Torrent, error) {
	return nil, nil
}

func (t *Torrent) Download(destination io.WriterAt) error {
	peers, refreshInterval, err := t.tracker.Refresh()
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

	for index, hash := range t.metadata.Info.PieceList {
		pieceSize := t.metadata.Info.PieceLength
		numPieces := int(math.Ceil(float64(t.metadata.Info.Length) / float64(pieceSize)))
		if index == numPieces-1 {
			pieceSize = uint32(t.metadata.Info.Length % uint64(t.metadata.Info.PieceLength))
		}
		jobs <- PieceWork{uint32(index), hash, pieceSize, make([]byte, pieceSize)}
	}

	pool := workerpool.New(workers, jobs, done)
	pool.Start()

	donePieces := 0
	for donePieces < len(t.metadata.Info.PieceList) {
		res := <-done
		offset := int64(res.index * t.metadata.Info.PieceLength)
		_, err := destination.WriteAt(res.result, offset)
		if err != nil {
			// for now keeping things simple, just fail if we were unable to write any downloaded piece
			return fmt.Errorf("failed writing response for piece %d at offset  %d: %w", res.index, offset, err)
		}

		donePieces++
	}

	// todo: spawn some goroutine to regularly refresh tracker

	return nil
}
