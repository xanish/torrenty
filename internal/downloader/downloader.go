package downloader

import (
	"bytes"
	"crypto/sha1"
	"fmt"
	"math"
	"net"
	"os"

	"github.com/schollz/progressbar/v3"
	"github.com/xanish/torrenty/internal/logger"
	"github.com/xanish/torrenty/internal/metadata"
	"github.com/xanish/torrenty/internal/peer"
	"github.com/xanish/torrenty/internal/utility"
)

const maxDownloadBlockSize = 16 * 1024

type work struct {
	id     int
	hash   [20]byte
	size   int
	result []byte
}

func retry(w *work, jobs chan<- *work) bool {
	channelClosed := false

	defer func() {
		if recover() != nil {
			channelClosed = true
		}
	}()

	w.result = make([]byte, w.size)
	jobs <- w

	return channelClosed
}

func executeWorker(id int, torrent metadata.Metadata, peerID [20]byte, peer peer.Peer, jobs chan *work, results chan<- *work) error {
	logger.Log(logger.Debug, "[worker:%d] connecting to peer %s", id, peer.String())
	conn, err := peer.Connect(torrent.InfoHash, peerID)
	if err != nil {
		return fmt.Errorf("[worker:%d] connecting to peer %s failed: %w", id, peer.String(), err)
	}
	defer func(Conn net.Conn) {
		_ = Conn.Close()
	}(conn.Conn)

	// Client connections start out as "choked" and "not interested"
	err = conn.SendUnChoke()
	err = conn.SendInterested()
	if err != nil {
		return fmt.Errorf("[worker:%d] sending interested message to peer %s failed: %w", id, peer.String(), err)
	}

	err = conn.ReadMessage(0, nil)
	if err != nil {
		return fmt.Errorf("[worker:%d] reading response for message<interested> from peer %s failed: %w", id, peer.String(), err)
	}

	for job := range jobs {
		// continue to next piece of work if this peer does not have the piece
		exists := utility.PieceExists(job.id, conn.Bitfield)
		if !exists {
			logger.Log(logger.Debug, "[worker:%d] peer %s does not have piece for job %d", id, peer.String(), job.id)
			retry(job, jobs)
			continue
		}

		// download piece block-by-block
		numBlocks := int(math.Ceil(float64(job.size) / float64(maxDownloadBlockSize)))
		for i := 0; i < numBlocks; i++ {
			adjustedBlockSize := maxDownloadBlockSize
			if i == numBlocks-1 {
				adjustedBlockSize = job.size - ((numBlocks - 1) * maxDownloadBlockSize)
			}

			err = conn.SendRequest(job.id, i*maxDownloadBlockSize, adjustedBlockSize)
			if err != nil {
				retry(job, jobs)
				logger.Log(logger.Error, "[worker:%d] sending message<request> to peer %s failed: %s", id, peer.String(), err)
				return nil
			}

			err = conn.ReadMessage(job.id, job.result)
			if err != nil {
				retry(job, jobs)
				logger.Log(logger.Error, "[worker:%d] reading response for message<request> from peer %s failed: %s", id, peer.String(), err)
				// just break here so that worker can continue fetching more jobs instead of exiting
				break
			}
		}

		// check piece integrity
		hash := sha1.Sum(job.result)
		if !bytes.Equal(hash[:], job.hash[:]) {
			retry(job, jobs)
			logger.Log(logger.Info, "[worker:%d] integrity check for piece %d downloaded from %s failed", id, job.id, peer.String())
			logger.Log(logger.Info, "[worker:%d] expected piece hash to be %x got %x", id, job.hash[:], hash[:])

			// TODO: temp ack-ing to see if some peers stop sending the same piece over and over
			err = conn.SendHave(job.id)
			continue
		} else {
			logger.Log(logger.Info, "[worker:%d] piece %d verified successfully", id, job.id)
			results <- job
		}

		// inform peer you got the piece
		err = conn.SendHave(job.id)
		if err != nil {
			return fmt.Errorf("[worker:%d] sending message<have> to peer %s failed: %w", id, peer.String(), err)
		}
	}

	return nil
}

func Download(peerID [20]byte, torrent metadata.Metadata, w *os.File) error {
	jobs := make(chan *work, len(torrent.Pieces))
	done := make(chan *work, len(torrent.Peers))
	for index, hash := range torrent.Pieces {
		pieceSize := torrent.PieceLength
		numPieces := int(math.Ceil(float64(torrent.Size) / float64(pieceSize)))
		if index == numPieces-1 {
			pieceSize = torrent.Size % torrent.PieceLength
		}
		jobs <- &work{index, hash, pieceSize, make([]byte, pieceSize)}
	}

	for id, remotePeer := range torrent.Peers {
		logger.Log(logger.Info, "starting worker %d with peer %s", id, remotePeer.String())
		go func() {
			// TODO: try to use some pattern here to restart broken workers
			err := executeWorker(id, torrent, peerID, remotePeer, jobs, done)
			if err != nil {
				logger.Log(logger.Error, "[worker:%d] failed with error: %s", id, err)
			}
		}()
	}

	donePieces := 0
	bar := progressbar.DefaultBytes(
		int64(torrent.Size),
		"Downloading "+torrent.Name,
	)
	for donePieces < len(torrent.Pieces) {
		res := <-done
		offset := int64(res.id * torrent.PieceLength)
		_, err := w.WriteAt(res.result, offset)
		if err != nil {
			return fmt.Errorf("failed writing response for piece %d at offset  %d: %w", res.id, offset, err)
		}

		_ = bar.Add(torrent.PieceLength)
		donePieces++

		percent := float64(donePieces) / float64(len(torrent.Pieces)) * 100
		logger.Log(logger.Info, "downloaded piece %d", res.id)
		logger.Log(logger.Info, "progress: (%0.2f%%)", percent)
	}

	close(jobs)
	close(done)
	_ = bar.Close()

	return nil
}
