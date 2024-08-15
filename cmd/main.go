package main

import (
	"os"
	"path/filepath"

	"github.com/xanish/torrenty"
)

func main() {
	path, err := filepath.Abs("debian.torrent")
	if err != nil {
		panic(err)
	}

	f, err := os.Open(path)
	if err != nil {
		panic(err)
	}

	err = torrenty.Download(f)
	if err != nil {
		panic(err)
	}
}
