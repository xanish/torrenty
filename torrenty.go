package torrenty

import (
	"fmt"
	"io"

	"github.com/xanish/torrenty/internal/metadata"
	"github.com/xanish/torrenty/internal/tracker"
	"github.com/xanish/torrenty/internal/utility"
)

const DEFAULT_PORT = 6881

func Download(r io.Reader) error {
	peerID, err := utility.PeerID()
	if err != nil {
		return err
	}

	torrent, err := metadata.New(r)
	if err != nil {
		return err
	}

	trackerURL, err := torrent.TrackerURL(peerID, DEFAULT_PORT)
	if err != nil {
		return err
	}

	peers, err := tracker.Peers(trackerURL)
	if err != nil {
		panic(err)
	}

	torrent.SetPeers(peers)

	fmt.Println(torrent.Peers)

	return nil
}
