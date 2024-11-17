package metadata

import (
	"errors"
	"net/url"
	"strconv"
	"strings"
)

type MagnetLink struct {
	InfoHash         string
	DisplayName      string
	Length           uint64
	Trackers         []string
	WebSeeds         []string
	AcceptableSource string
	Keyword          string
}

func (ml *MagnetLink) asMetadata() *Metadata {
	announceList := make([][]string, 0, len(ml.Trackers))
	for _, tracker := range ml.Trackers {
		announceList = append(announceList, []string{tracker})
	}

	return &Metadata{
		Info: PieceInfo{
			Name:   ml.DisplayName,
			Length: ml.Length,
		},
		InfoHash:     [20]byte([]byte(ml.InfoHash)),
		Announce:     ml.Trackers[0],
		AnnounceList: announceList,
	}
}

func FromURL(magnet string) (*Metadata, error) {
	parsed, err := url.Parse(magnet)
	if err != nil {
		return nil, err
	}

	if parsed.Scheme != "magnet" {
		return nil, errors.New("invalid magnet URL")
	}

	ml := MagnetLink{}
	for key, values := range parsed.Query() {
		switch key {
		case "xt":
			ml.InfoHash = strings.TrimPrefix(values[0], "urn:btih:")
		case "dn":
			ml.DisplayName = values[0]
		case "xl":
			ml.Length, _ = strconv.ParseUint(values[0], 10, 64)
		case "tr":
			ml.Trackers = append(ml.Trackers, values...)
		case "ws":
			ml.WebSeeds = append(ml.WebSeeds, values...)
		case "as":
			ml.AcceptableSource = values[0]
		case "kt":
			ml.Keyword = values[0]
		}
	}

	if ml.InfoHash == "" {
		return nil, errors.New("magnet link does not contain info hash")
	}

	if ml.DisplayName == "" {
		return nil, errors.New("magnet link does not contain display name")
	}

	if len(ml.Trackers) == 0 {
		return nil, errors.New("magnet link does not contain trackers")
	}

	return ml.asMetadata(), nil
}
