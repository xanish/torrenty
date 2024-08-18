package downloader

import (
	"fmt"
	"log"

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
			for idx, _ := range torrent.Pieces {
				// if bitfield has the piece
				exists := utility.PieceExists(idx, conn.Bitfield)
				if !exists {
					break
				}

				// download piece
				err = conn.SendRequest(idx, 0, 16384)
				if err != nil {
					fmt.Println(err)
				}

				err = readMessage(conn)
				if err != nil {
					fmt.Println(err)
				}

				// check piece integrity
				// inform peer you got the piece
			}
		}
		fmt.Println()
	}
}

func readMessage(peer *peer.Connection) error {
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
	case message.Cancel:
	case message.Port:
	}

	return nil
}
