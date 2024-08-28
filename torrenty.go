package torrenty

import (
	"fmt"
	"io"
	"log"
	"os"

	"github.com/xanish/torrenty/internal/downloader"
	"github.com/xanish/torrenty/internal/logger"
	"github.com/xanish/torrenty/internal/metadata"
	"github.com/xanish/torrenty/internal/utility"
)

const defaultPort = 6881

func Download(r io.Reader, path string) error {
	f, err := os.OpenFile(path+"process.log", os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
	if err != nil {
		logger.Log(logger.Fatal, "failed to open log file: %s", err)
	}
	defer f.Close()

	log.SetOutput(f)

	peerID, err := utility.PeerID()
	if err != nil {
		return err
	}
	logger.Log(logger.Info, "generated peer id %x", peerID)

	logger.Log(logger.Info, "parsing torrent file metadata")
	torrent, err := metadata.New(r)
	if err != nil {
		return err
	}

	tr, err := torrent.SyncWithTracker(peerID, defaultPort)
	if err != nil {
		panic(err)
	}

	if len(tr.Peers) == 0 {
		return fmt.Errorf("no peers found")
	}

	logger.Log(logger.Info, "successfully fetched %d peers from tracker", len(tr.Peers))

	torrent.SetPeers(tr.Peers)
	torrent.SetRefreshInterval(tr.RefreshInterval)

	file := path + torrent.Name
	out, err := os.Create(file)
	if err != nil {
		return fmt.Errorf("could not create output file %s: %w", file, err)
	}
	defer func(out *os.File) {
		_ = out.Close()
	}(out)

	err = out.Truncate(int64(torrent.Size))
	if err != nil {
		return fmt.Errorf("could not allocate %d bytes for file %s: %w", torrent.Size, file, err)
	}

	logger.Log(logger.Info, "initiating download")
	err = downloader.Download(peerID, torrent, out)
	if err != nil {
		return err
	}

	return nil
}
