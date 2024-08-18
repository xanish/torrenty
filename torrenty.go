package torrenty

import (
	"fmt"
	"io"
	"log"
	"os"

	"github.com/xanish/torrenty/internal/downloader"
	"github.com/xanish/torrenty/internal/metadata"
	"github.com/xanish/torrenty/internal/tracker"
	"github.com/xanish/torrenty/internal/utility"
)

const DEFAULT_PORT = 6881

func Download(r io.Reader, path string) error {
	peerID, err := utility.PeerID()
	if err != nil {
		return err
	}
	log.Printf("generated peer id %x", peerID)

	log.Print("parsing torrent file metadata")
	torrent, err := metadata.New(r)
	if err != nil {
		return err
	}

	trackerURL, err := torrent.TrackerURL(peerID, DEFAULT_PORT)
	if err != nil {
		return err
	}
	log.Printf("generated tracker request url %s", trackerURL)

	peers, err := tracker.Peers(trackerURL)
	if err != nil {
		panic(err)
	}

	if len(peers) == 0 {
		return fmt.Errorf("no peers found")
	}

	log.Printf("successfully fetched %d peers from tracker", len(peers))

	torrent.SetPeers(peers)

	file := path + torrent.Name
	out, err := os.Create(file)
	if err != nil {
		return fmt.Errorf("could not create output file %s: %w", file, err)
	}
	err = out.Truncate(int64(torrent.Size))
	if err != nil {
		return fmt.Errorf("could not allocate %d bytes for file %s: %w", torrent.Size, file, err)
	}

	log.Print("initiating download")
	err = downloader.Download(peerID, torrent, out)
	if err != nil {
		return err
	}

	return nil
}
