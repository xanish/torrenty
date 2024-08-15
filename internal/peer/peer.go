package peer

import (
	"net"
	"strconv"
)

type Peer struct {
	IP   net.IP
	Port uint16
}

func (p Peer) Connect(infoHash, peerID [20]byte) (*Connection, error) {
	return newConnection(p, infoHash, peerID)
}

func (p Peer) String() string {
	return net.JoinHostPort(p.IP.String(), strconv.Itoa(int(p.Port)))
}
