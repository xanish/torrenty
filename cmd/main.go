package main

import (
	"github.com/xanish/torrenty"
	"os"
	"path/filepath"
)

func main() {
	downloadPath, err := filepath.Abs(".")
	torrentPath, err := filepath.Abs("debian.torrent")
	if err != nil {
		panic(err)
	}

	file, err := os.Open(torrentPath)
	if err != nil {
		panic(err)
	}

	err = torrenty.Download(file, downloadPath)
	if err != nil {
		panic(err)
	}
}
