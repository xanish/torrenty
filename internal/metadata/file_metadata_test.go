package metadata

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFromFile(t *testing.T) {
	t.Run("valid metadata", func(t *testing.T) {
		data := []byte("d8:announce35:https://torrent.ubuntu.com/announce13:announce-listll35:https://torrent.ubuntu.com/announceel40:https://ipv6.torrent.ubuntu.com/announceee7:comment29:Ubuntu CD releases.ubuntu.com10:created by13:mktorrent 1.113:creation datei1728557557e4:infod6:lengthi5665497088e4:name30:ubuntu-24.10-desktop-amd64.iso12:piece lengthi262144e6:pieces40:abcdefghijklmnopqrstuvwxyz1234567890qweree")
		r := bytes.NewReader(data)

		expectedMetadata := &Metadata{
			Announce: "https://torrent.ubuntu.com/announce",
			AnnounceList: [][]string{
				{"https://torrent.ubuntu.com/announce"},
				{"https://ipv6.torrent.ubuntu.com/announce"},
			},
			Info: PieceInfo{
				Name:        "ubuntu-24.10-desktop-amd64.iso",
				Length:      5665497088,
				MD5Sum:      "",
				Files:       nil,
				PieceLength: 262144,
				Pieces:      "abcdefghijklmnopqrstuvwxyz1234567890qwer",
				Private:     0,
			},
			CreationDate: 1728557557,
			Comment:      "Ubuntu CD releases.ubuntu.com",
			CreatedBy:    "mktorrent 1.1",
			Encoding:     "",
			URLList:      nil,
			InfoHash:     [20]byte{0x96, 0x7b, 0x5a, 0x77, 0xd, 0x1d, 0xa8, 0x1c, 0x6f, 0xf1, 0x69, 0xa3, 0xdd, 0xb2, 0x9, 0x36, 0x6c, 0xb8, 0xe8, 0x44},
		}

		metadata, err := FromFile(r)
		assert.NoError(t, err)
		assert.Equal(t, expectedMetadata, metadata)
	})

	t.Run("empty data", func(t *testing.T) {
		data := []byte("")
		r := bytes.NewReader(data)

		metadata, err := FromFile(r)
		assert.Error(t, err)
		assert.Nil(t, metadata)
	})

	t.Run("invalid bencode", func(t *testing.T) {
		data := []byte("invalid bencode data")
		r := bytes.NewReader(data)

		metadata, err := FromFile(r)
		assert.Error(t, err)
		assert.Nil(t, metadata)
	})

	t.Run("file array present", func(t *testing.T) {
		data := []byte("d8:announce35:http://tracker.example.com/announce4:infod5:filesld6:lengthi111e4:pathl7:111.txteed6:lengthi222e4:pathl7:222.txteee4:name13:directoryName12:piece lengthi262144e6:pieces40:abcdefghijklmnopqrstuvwxyz1234567890qweree")
		r := bytes.NewReader(data)

		expectedMetadata := &Metadata{
			Announce:     "http://tracker.example.com/announce",
			AnnounceList: nil,
			Info: PieceInfo{
				Name:   "directoryName",
				Length: 0,
				MD5Sum: "",
				Files: []File{
					{Length: 111, MD5Sum: "", Path: []string{"111.txt"}},
					{Length: 222, MD5Sum: "", Path: []string{"222.txt"}},
				},
				PieceLength: 262144,
				Pieces:      "abcdefghijklmnopqrstuvwxyz1234567890qwer",
				Private:     0,
			},
			CreationDate: 0,
			Comment:      "",
			CreatedBy:    "",
			Encoding:     "",
			URLList:      nil,
			InfoHash:     [20]byte{0xeb, 0x12, 0xb1, 0x3f, 0xf3, 0xc2, 0x5c, 0x57, 0x2b, 0x39, 0x7, 0xe8, 0xe5, 0x29, 0x90, 0x7, 0x72, 0x68, 0xc3, 0x89},
		}

		metadata, err := FromFile(r)
		assert.NoError(t, err)
		assert.Equal(t, expectedMetadata, metadata)
	})

	t.Run("missing required fields", func(t *testing.T) {
		data := []byte("d8:announce13:announce-urle")
		r := bytes.NewReader(data)

		metadata, err := FromFile(r)
		assert.Error(t, err)
		assert.Nil(t, metadata)
	})

	t.Run("missing announce and empty announce-list", func(t *testing.T) {
		data := []byte("d8:announce-listle4:infoed5:name4:abcd12:piece lengthi262144e6:pieces12:piecesstringe")
		r := bytes.NewReader(data)

		metadata, err := FromFile(r)
		assert.Error(t, err)
		assert.Nil(t, metadata)
	})

	t.Run("missing download file name", func(t *testing.T) {
		data := []byte("d8:announce13:announce-url4:infoed12:piece lengthi262144e6:pieces12:piecesstringe")
		r := bytes.NewReader(data)

		metadata, err := FromFile(r)
		assert.Error(t, err)
		assert.Nil(t, metadata)
	})

	t.Run("missing piece length", func(t *testing.T) {
		data := []byte("d8:announce13:announce-url4:infoed5:name4:abcd6:pieces12:piecesstringe")
		r := bytes.NewReader(data)

		metadata, err := FromFile(r)
		assert.Error(t, err)
		assert.Nil(t, metadata)
	})

	t.Run("missing pieces", func(t *testing.T) {
		data := []byte("d8:announce13:announce-url4:infoed5:name4:abcd12:piece lengthi262144e")
		r := bytes.NewReader(data)

		metadata, err := FromFile(r)
		assert.Error(t, err)
		assert.Nil(t, metadata)
	})

	t.Run("announce set from announce-list", func(t *testing.T) {
		data := []byte("d13:announce-listll35:https://torrent.ubuntu.com/announceel40:https://ipv6.torrent.ubuntu.com/announceee7:comment29:Ubuntu CD releases.ubuntu.com10:created by13:mktorrent 1.113:creation datei1728557557e4:infod6:lengthi5665497088e4:name30:ubuntu-24.10-desktop-amd64.iso12:piece lengthi262144e6:pieces40:abcdefghijklmnopqrstuvwxyz1234567890qweree")
		r := bytes.NewReader(data)

		expectedMetadata := &Metadata{
			Announce: "https://torrent.ubuntu.com/announce",
			AnnounceList: [][]string{
				{"https://torrent.ubuntu.com/announce"},
				{"https://ipv6.torrent.ubuntu.com/announce"},
			},
			Info: PieceInfo{
				Name:        "ubuntu-24.10-desktop-amd64.iso",
				Length:      5665497088,
				MD5Sum:      "",
				Files:       nil,
				PieceLength: 262144,
				Pieces:      "abcdefghijklmnopqrstuvwxyz1234567890qwer",
				Private:     0,
			},
			CreationDate: 1728557557,
			Comment:      "Ubuntu CD releases.ubuntu.com",
			CreatedBy:    "mktorrent 1.1",
			Encoding:     "",
			URLList:      nil,
			InfoHash:     [20]byte{0x96, 0x7b, 0x5a, 0x77, 0xd, 0x1d, 0xa8, 0x1c, 0x6f, 0xf1, 0x69, 0xa3, 0xdd, 0xb2, 0x9, 0x36, 0x6c, 0xb8, 0xe8, 0x44},
		}

		metadata, err := FromFile(r)
		assert.NoError(t, err)
		assert.Equal(t, expectedMetadata, metadata)
	})

	t.Run("invalid announce", func(t *testing.T) {
		data := []byte("d8:announce36:https://torrent^.ubuntu.com/announce13:announce-listll35:https://torrent.ubuntu.com/announceel40:https://ipv6.torrent.ubuntu.com/announceee7:comment29:Ubuntu CD releases.ubuntu.com10:created by13:mktorrent 1.113:creation datei1728557557e4:infod6:lengthi5665497088e4:name30:ubuntu-24.10-desktop-amd64.iso12:piece lengthi262144e6:pieces40:abcdefghijklmnopqrstuvwxyz1234567890qweree")
		r := bytes.NewReader(data)

		metadata, err := FromFile(r)
		assert.Error(t, err)
		assert.Nil(t, metadata)
	})
}
