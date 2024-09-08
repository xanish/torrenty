package main

import (
	"github.com/xanish/torrenty"
	"os"
	"path/filepath"
)

func main() {
	torrentPath := os.Args[1]
	downloadPath, err := filepath.Abs(".")

	file, err := os.Open(torrentPath)
	if err != nil {
		panic(err)
	}

	err = torrenty.Download(file, downloadPath+"/")
	if err != nil {
		panic(err)
	}
}
