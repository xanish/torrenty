package torrent

import (
	"bytes"
	"crypto/sha1"
	"fmt"
	"math"

	"github.com/xanish/torrenty/internal/peer"
	"github.com/xanish/torrenty/internal/protocol"
)

type PieceWork struct {
	index  uint32
	hash   [20]byte
	size   uint32
	result []byte
}

type PieceWorker struct {
	id       int
	peer     peer.Peer
	clientID [20]byte
	infoHash [20]byte
}

func NewPieceWorker(id int, clientID, infoHash [20]byte, peer peer.Peer) PieceWorker {
	return PieceWorker{
		id:       id,
		peer:     peer,
		clientID: clientID,
		infoHash: infoHash,
	}
}

func (pw PieceWorker) DoWork(jobs chan PieceWork, results chan<- PieceWork) error {
	const downloadBlockSize uint32 = 16 * 1024

	err := pw.peer.Connect(pw.clientID, pw.infoHash)
	if err != nil {
		return fmt.Errorf("failed to connect to peer %s: %v", pw.peer.String(), err)
	}
	defer pw.peer.Close()

	for piece := range jobs {
		// Peer does not have this piece so move on to find the next piece from
		// this peer which we can download
		if !pw.peer.HasPiece(piece.index) {
			isChanClosed := requeueWork(piece, jobs)
			if isChanClosed {
				return nil
			}

			continue
		}

		// download piece block-by-block
		numBlocks := uint32(math.Ceil(float64(piece.size) / float64(downloadBlockSize)))

		var i uint32
		for i = 0; i < numBlocks; i++ {
			adjustedBlockSize := downloadBlockSize
			if i == numBlocks-1 {
				adjustedBlockSize = piece.size - ((numBlocks - 1) * downloadBlockSize)
			}

			err := pw.peer.Send(protocol.MessageConf{
				Type:       protocol.MsgTypeRequest,
				PieceIndex: piece.index,
				Begin:      i * downloadBlockSize,
				Length:     adjustedBlockSize,
			})
			if err != nil {
				// Something went wrong while requesting for current piece, just
				// add it to backlog and try later
				isChanClosed := requeueWork(piece, jobs)
				if isChanClosed {
					return nil
				}

				break
			}

			msg, err := pw.peer.Receive()
			if err != nil {
				isChanClosed := requeueWork(piece, jobs)
				if isChanClosed {
					return nil
				}

				// Break out of the loop here to allow worker to download the
				// next available piece from this peer
				break
			}

			if msg.Type != protocol.MsgTypePiece {
				isChanClosed := requeueWork(piece, jobs)
				if isChanClosed {
					return nil
				}

				// Break out of the loop here to allow worker to download the
				// next available piece from this peer
				break
			}

			copy(piece.result[i*downloadBlockSize:], msg.Payload)
		}

		// check piece integrity
		hash := sha1.Sum(piece.result)
		if !bytes.Equal(hash[:], piece.hash[:]) {
			// Hash for the piece did not match so we requeue it and download
			// it later
			isChanClosed := requeueWork(piece, jobs)
			if isChanClosed {
				return nil
			}

			continue
		} else {
			results <- piece
		}

		// Inform the peer that we received the message
		err = pw.peer.Send(protocol.MessageConf{Type: protocol.MsgTypeHave, PieceIndex: piece.index})
		if err != nil {
			return fmt.Errorf("failed to send message have to peer %s: %v", pw.peer.String(), err)
		}
	}

	return nil
}

func requeueWork(work PieceWork, jobs chan<- PieceWork) bool {
	isChanClosed := false
	defer func() {
		if recover() != nil {
			isChanClosed = true
		}
	}()

	// Reset the result payload
	work.result = make([]byte, work.size)
	jobs <- work

	return isChanClosed
}
