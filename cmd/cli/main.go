package main

import (
	"errors"
	"flag"
	"fmt"
	"os"
)

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

	if _, err := os.Stat(config.TorrentFile); os.IsNotExist(err) {
		return fmt.Errorf("torrent file %s does not exist", config.TorrentFile)
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
	if config.TorrentFile != "" {
		file, err := os.Open(config.TorrentFile)
		if err != nil {
			return fmt.Errorf("failed to read torrent file %s: %s", config.TorrentFile, err.Error())
		}
		defer file.Close()

		// todo: actual download logic here
		return nil
	}

	// todo: actual download logic for magnet link here
	return nil
}

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
