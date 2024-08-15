package peer

type Connection struct{}

func New(peer Peer, infoHash, peerID [20]byte) *Connection {
	return &Connection{}
}
