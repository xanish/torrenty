package main

import (
	"errors"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/xanish/torrenty/internal/torrent"
)

func main() {
	config := parseFlags()

	logFile, err := os.OpenFile("./torrenty.log", os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0644)
	if err != nil {
		fmt.Println("failed to open log file:", err)
		os.Exit(1)
	}
	defer logFile.Close()

	logger := slog.New(slog.NewTextHandler(logFile, &slog.HandlerOptions{Level: slog.LevelInfo}))
	slog.SetDefault(logger)

	if err := validateFlags(config); err != nil {
		slog.Error("Validation failed", slog.Any("error", err))
		os.Exit(1)
	}

	if err := processDownload(config); err != nil {
		slog.Error("Download failed", slog.Any("error", err))
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

	startTime := time.Now()
	slog.Info("Starting download",
		slog.String("file", t.Name()),
		slog.String("destination", dest),
		slog.Time("start_time", startTime),
		slog.Group("params",
			slog.String("torrent_file", config.TorrentFile),
			slog.String("magnet_link", config.MagnetLink),
			slog.String("destination", config.Destination),
		),
	)

	if err = t.Download(out); err != nil {
		return err
	}
	endTime := time.Now()
	duration := endTime.Sub(startTime)

	slog.Info("Download completed",
		slog.String("file", t.Name()),
		slog.Duration("duration", duration),
		slog.Time("end_time", endTime),
		slog.Group("params",
			slog.String("torrent_file", config.TorrentFile),
			slog.String("magnet_link", config.MagnetLink),
			slog.String("destination", config.Destination),
		),
	)

	return nil
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

	slog.Info("Torrent file parsed", slog.String("file", config.TorrentFile))

	return t, nil
}

func torrentFromMagnet(config *DownloadFlags) (*torrent.Torrent, error) {
	t, err := torrent.FromMagnet(config.MagnetLink)
	if err != nil {
		return nil, fmt.Errorf("failed to create torrent from magnet %s: %s", config.MagnetLink, err.Error())
	}

	slog.Info("Torrent created from magnet link", slog.String("magnet", config.MagnetLink))

	return t, nil
}
