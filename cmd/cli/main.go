package main

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/xanish/torrenty/internal/torrent"
)

func main() {
	config := parseFlags()
	if err := validateFlags(config); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	if err := processDownload(config); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

type DownloadFlags struct {
	TorrentFile string
	MagnetLink  string
	Destination string
}

func parseFlags() *DownloadFlags {
	torrentFile := flag.String("torrent", "", "Path to the torrent file (e.g., /path/to/file.torrent)")
	magnetLink := flag.String("magnet", "", "Magnet link URL (e.g., magnet:?xt=urn:btih:...)")
	destination := flag.String("destination", "", "Directory where the downloaded files will be saved")
	flag.Parse()

	config := &DownloadFlags{
		TorrentFile: *torrentFile,
		MagnetLink:  *magnetLink,
		Destination: *destination,
	}

	return config
}

func validateFlags(config *DownloadFlags) error {
	if config.TorrentFile == "" && config.MagnetLink == "" {
		return errors.New("you must provide either a torrent file or a magnet link URL")
	}

	if config.TorrentFile != "" {
		if _, err := os.Stat(config.TorrentFile); os.IsNotExist(err) {
			return fmt.Errorf("torrent file %s does not exist", config.TorrentFile)
		}
	}

	if config.Destination == "" {
		return errors.New("you must specify a destination folder to save the downloaded files")
	}

	if _, err := os.Stat(config.Destination); os.IsNotExist(err) {
		return fmt.Errorf("destination folder %s does not exist", config.Destination)
	}

	return nil
}

func processDownload(config *DownloadFlags) error {
	t, err := createTorrent(config)
	if err != nil {
		return err
	}

	dest := filepath.Clean(config.Destination)
	if !strings.HasSuffix(config.Destination, string(filepath.Separator)) {
		dest += string(filepath.Separator)
	}

	out, err := os.Create(dest + t.Name())
	if err != nil {
		return fmt.Errorf("could not create output file %s: %w", dest, err)
	}
	defer out.Close()

	if err = out.Truncate(int64(t.Size())); err != nil {
		return fmt.Errorf("could not allocate %d bytes for file %s: %w", t.Size(), out.Name(), err)
	}

	return t.Download(out)
}

func createTorrent(config *DownloadFlags) (*torrent.Torrent, error) {
	if config.TorrentFile != "" {
		return torrentFromFile(config)
	} else if config.MagnetLink != "" {
		return torrentFromMagnet(config)
	}

	return nil, errors.New("no torrent file or magnet link provided")
}

func torrentFromFile(config *DownloadFlags) (*torrent.Torrent, error) {
	file, err := os.Open(config.TorrentFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read torrent file %s: %s", config.TorrentFile, err.Error())
	}
	defer file.Close()

	t, err := torrent.FromFile(file)
	if err != nil {
		return nil, fmt.Errorf("failed to read torrent file %s: %s", config.TorrentFile, err.Error())
	}

	return t, nil
}

func torrentFromMagnet(config *DownloadFlags) (*torrent.Torrent, error) {
	t, err := torrent.FromMagnet(config.MagnetLink)
	if err != nil {
		return nil, fmt.Errorf("failed to create torrent from magnet %s: %s", config.MagnetLink, err.Error())
	}

	return t, nil
}
