package downloader

import (
	"bytes"
	"crypto/sha1"
	"fmt"
	"log"
	"math"

	"github.com/xanish/torrenty/internal/message"
	"github.com/xanish/torrenty/internal/metadata"
	"github.com/xanish/torrenty/internal/peer"
	"github.com/xanish/torrenty/internal/utility"
)

func Download(peerID [20]byte, torrent metadata.Metadata) {
	for _, peer := range torrent.Peers {
		log.Printf("connecting to peer %s", peer.String())
		conn, err := peer.Connect(torrent.InfoHash, peerID)
		if err != nil {
			fmt.Println(err)
		}

		if conn != nil {
			fmt.Println(conn.Bitfield)

			conn.SendInterested()
			readMessage(conn, 0, nil)

			for idx, piece := range torrent.Pieces {
				// if bitfield has the piece
				exists := utility.PieceExists(idx, conn.Bitfield)
				if !exists {
					break
				}

				pieceSize := torrent.PieceLength
				numPieces := int(math.Ceil(float64(torrent.Size) / float64(pieceSize)))
				if idx == numPieces-1 {
					pieceSize = torrent.Size % torrent.PieceLength
				}

				blockSize := 16 * 1024
				numBlocks := int(math.Ceil(float64(pieceSize) / float64(blockSize)))

				downloadedPiece := make([]byte, pieceSize)

				for i := 0; i < numBlocks; i++ {
					adjustedBlockSize := blockSize
					if i == numBlocks-1 {
						adjustedBlockSize = pieceSize - ((numBlocks - 1) * blockSize)
					}

					log.Printf("downloading piece %d with begin offset %d and block size %d", idx, i*blockSize, adjustedBlockSize)
					err = conn.SendRequest(idx, i*blockSize, adjustedBlockSize)
					if err != nil {
						fmt.Println(err)
					}

					err = readMessage(conn, idx, downloadedPiece)
					if err != nil {
						fmt.Println(err)
					}
				}

				// check piece integrity
				hash := sha1.Sum(downloadedPiece)
				if !bytes.Equal(hash[:], piece[:]) {
					fmt.Printf("failed integrity check for piece %d\n", idx)
				} else {
					fmt.Printf("successfully integrity check for piece %d\n", idx)
				}

				// inform peer you got the piece
				err = conn.SendHave(idx)
				if err != nil {
					fmt.Println(err)
				}
				break
			}
		}
		fmt.Println()
	}
}

func readMessage(peer *peer.Connection, index int, buf []byte) error {
	msg, err := message.Unmarshal(peer.Conn)
	if err != nil {
		return err
	}

	fmt.Println(msg)

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
