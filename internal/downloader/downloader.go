package downloader

import (
	"bytes"
	"crypto/sha1"
	"fmt"
	"math"
	"os"

	"github.com/xanish/torrenty/internal/logger"
	"github.com/xanish/torrenty/internal/message"
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

func retry(w *work, jobs chan<- *work) {
	w.result = make([]byte, w.size)
	jobs <- w
}

func executeWorker(id int, torrent metadata.Metadata, peerID [20]byte, peer peer.Peer, jobs chan *work, results chan<- *work) error {
	logger.Log(logger.Debug, "[worker:%d] connecting to peer %s", id, peer.String())
	conn, err := peer.Connect(torrent.InfoHash, peerID)
	if err != nil {
		return fmt.Errorf("[worker:%d] connecting to peer %s failed: %w", id, peer.String(), err)
	}

	err = conn.SendInterested()
	if err != nil {
		return fmt.Errorf("[worker:%d] sending interested message to peer %s failed: %w", id, peer.String(), err)
	}

	err = readMessage(conn, 0, nil)
	if err != nil {
		return fmt.Errorf("[worker:%d] reading response for message<interested> from peer %s failed: %w", id, peer.String(), err)
	}

	for job := range jobs {
		// continue to next piece of work if this peer does not have the piece
		exists := utility.PieceExists(job.id, conn.Bitfield)
		if !exists {
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
				// just break here so that worker can continue fetching more jobs instead of exiting
				break
			}

			err = readMessage(conn, job.id, job.result)
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
	todo := make(chan *work, len(torrent.Pieces))
	done := make(chan *work, 10)
	for index, hash := range torrent.Pieces {
		pieceSize := torrent.PieceLength
		numPieces := int(math.Ceil(float64(torrent.Size) / float64(pieceSize)))
		if index == numPieces-1 {
			pieceSize = torrent.Size % torrent.PieceLength
		}
		todo <- &work{index, hash, pieceSize, make([]byte, pieceSize)}
	}

	for id, remotePeer := range torrent.Peers {
		logger.Log(logger.Info, "starting worker %d with peer %s", id, remotePeer.String())
		go func() {
			err := executeWorker(id, torrent, peerID, remotePeer, todo, done)
			if err != nil {
				logger.Log(logger.Error, "[worker:%d] failed with error: %s", id, err)
			}
		}()
	}

	donePieces := 0
	for donePieces < len(torrent.Pieces) {
		res := <-done
		offset := int64(res.id * torrent.PieceLength)
		_, err := w.WriteAt(res.result, offset)
		if err != nil {
			return fmt.Errorf("failed writing response for piece %d at offset  %d: %w", res.id, offset, err)
		}
		donePieces++

		percent := float64(donePieces) / float64(len(torrent.Pieces)) * 100
		logger.Log(logger.Info, "downloaded piece %d", res.id)
		logger.Log(logger.Info, "progress: (%0.2f%%)", percent)
	}

	close(todo)

	return nil
}

func readMessage(peer *peer.Connection, index int, buf []byte) error {
	msg, err := message.Unmarshal(peer.Conn)
	if err != nil {
		return err
	}

	// keep-alive
	if msg == nil {
		return nil
	}

	switch msg.ID {
	case message.Choke:
		peer.AmChoked = true
	case message.UnChoke:
		peer.AmChoked = false
	case message.Interested:
		peer.AmInterested = true
	case message.NotInterested:
		peer.AmInterested = false
	case message.Have:
		index, err := message.ParseHave(msg)
		if err != nil {
			return err
		}
		utility.SetPiece(index, peer.Bitfield)
	case message.Bitfield:
	case message.Request:
		_, _, _, err := message.ParseRequest(msg)
		if err != nil {
			return err
		}
	case message.Piece:
		_, err := message.ParsePiece(index, buf, msg)
		if err != nil {
			return err
		}
	case message.Cancel:
	case message.Port:
	}

	return nil
}
