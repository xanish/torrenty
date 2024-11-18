package torrent

import (
	"github.com/xanish/torrenty/internal/peer"
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
	return nil
}
