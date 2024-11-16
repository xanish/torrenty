package metadata

import (
	"errors"
	"io"

	"github.com/jackpal/bencode-go"
)

type Metadata struct {
	Info         PieceInfo  `bencode:"info"`
	Announce     string     `bencode:"announce"`
	AnnounceList [][]string `bencode:"announce-list,omitempty"`
	CreationDate int64      `bencode:"creation date,omitempty"`
	Comment      string     `bencode:"comment,omitempty"`
	CreatedBy    string     `bencode:"created by,omitempty"`
	Encoding     string     `bencode:"encoding,omitempty"`
	URLList      []string   `bencode:"url-list,omitempty"`
}

type PieceInfo struct {
	Name        string `bencode:"name"`
	Length      int64  `bencode:"length,omitempty"`
	MD5Sum      string `bencode:"md5sum,omitempty"`
	Files       []File `bencode:"files,omitempty"`
	PieceLength int64  `bencode:"piece length"`
	Pieces      string `bencode:"pieces"`
	Private     int    `bencode:"private,omitempty"`
}

type File struct {
	Length int64    `bencode:"length"`
	MD5Sum string   `bencode:"md5sum,omitempty"`
	Path   []string `bencode:"path"`
}

func FromFile(r io.Reader) (*Metadata, error) {
	m := Metadata{}
	err := bencode.Unmarshal(r, &m)
	if err != nil {
		return nil, err
	}

	if m.Announce == "" && len(m.AnnounceList) == 0 {
		return nil, errors.New("file does not contain any announces")
	}

	if m.Announce == "" && len(m.AnnounceList) > 0 {
		if len(m.AnnounceList[0]) != 1 {
			return nil, errors.New("file does not contain any announces")
		}
		m.Announce = m.AnnounceList[0][0]
	}

	if m.Info.Name == "" {
		return nil, errors.New("file does not contain any files to download")
	}

	if m.Info.PieceLength == 0 {
		return nil, errors.New("file does not contain any piece length")
	}

	if m.Info.Pieces == "" {
		return nil, errors.New("file does not contain any pieces")
	}

	return &m, nil
}
