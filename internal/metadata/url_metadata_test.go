package metadata

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFromURL(t *testing.T) {
	t.Run("valid magnet link", func(t *testing.T) {
		magnet := "magnet:?xt=urn:btih:H6NKYFMMPXUN7SVROHVFRIL2VPPX7PET&dn=ubuntu-24.10-desktop-amd64.iso&xl=5665497088&tr=https%3A%2F%2Ftorrent.ubuntu.com%2Fannounce&tr=https%3A%2F%2Ftorrent.ubuntu.com%2Fannounce&tr=https%3A%2F%2Fipv6.torrent.ubuntu.com%2Fannounce"
		result, err := FromURL(magnet)

		assert.NoError(t, err)
		assert.Equal(t, [20]byte([]byte("H6NKYFMMPXUN7SVROHVFRIL2VPPX7PET")), result.InfoHash)
		assert.Equal(t, "ubuntu-24.10-desktop-amd64.iso", result.Info.Name)
		assert.Equal(t, int64(5665497088), result.Info.Length)
		assert.Equal(t, "https://torrent.ubuntu.com/announce", result.Announce)
		assert.Len(t, result.AnnounceList, 3)
		assert.Contains(t, result.AnnounceList, []string{"https://torrent.ubuntu.com/announce"})
		assert.Contains(t, result.AnnounceList, []string{"https://torrent.ubuntu.com/announce"})
		assert.Contains(t, result.AnnounceList, []string{"https://ipv6.torrent.ubuntu.com/announce"})
	})

	t.Run("invalid scheme", func(t *testing.T) {
		magnet := "https://example.com"
		result, err := FromURL(magnet)

		assert.Error(t, err)
		assert.Nil(t, result)
	})

	t.Run("missing info hash", func(t *testing.T) {
		magnet := "magnet:?dn=ubuntu-24.10-desktop-amd64.iso&xl=5665497088&tr=https%3A%2F%2Ftorrent.ubuntu.com%2Fannounce&tr=https%3A%2F%2Ftorrent.ubuntu.com%2Fannounce&tr=https%3A%2F%2Fipv6.torrent.ubuntu.com%2Fannounce"
		result, err := FromURL(magnet)

		assert.Error(t, err)
		assert.Equal(t, "magnet link does not contain info hash", err.Error())
		assert.Nil(t, result)
	})

	t.Run("missing display name", func(t *testing.T) {
		magnet := "magnet:?xt=urn:btih:H6NKYFMMPXUN7SVROHVFRIL2VPPX7PET&xl=5665497088&tr=https%3A%2F%2Ftorrent.ubuntu.com%2Fannounce&tr=https%3A%2F%2Ftorrent.ubuntu.com%2Fannounce&tr=https%3A%2F%2Fipv6.torrent.ubuntu.com%2Fannounce"
		result, err := FromURL(magnet)

		assert.Error(t, err)
		assert.Equal(t, "magnet link does not contain display name", err.Error())
		assert.Nil(t, result)
	})

	t.Run("missing trackers", func(t *testing.T) {
		magnet := "magnet:?xt=urn:btih:H6NKYFMMPXUN7SVROHVFRIL2VPPX7PET&dn=ubuntu-24.10-desktop-amd64.iso&xl=5665497088"
		result, err := FromURL(magnet)

		assert.Error(t, err)
		assert.Equal(t, "magnet link does not contain trackers", err.Error())
		assert.Nil(t, result)
	})

	t.Run("missing length parameter", func(t *testing.T) {
		magnet := "magnet:?xt=urn:btih:H6NKYFMMPXUN7SVROHVFRIL2VPPX7PET&dn=ubuntu-24.10-desktop-amd64.iso&tr=https%3A%2F%2Ftorrent.ubuntu.com%2Fannounce&tr=https%3A%2F%2Ftorrent.ubuntu.com%2Fannounce&tr=https%3A%2F%2Fipv6.torrent.ubuntu.com%2Fannounce"
		result, err := FromURL(magnet)

		assert.NoError(t, err)
		assert.Equal(t, int64(0), result.Info.Length)
	})

	t.Run("malformed length parameter", func(t *testing.T) {
		magnet := "magnet:?xt=urn:btih:H6NKYFMMPXUN7SVROHVFRIL2VPPX7PET&dn=ubuntu-24.10-desktop-amd64.iso&xl=asdf&tr=https%3A%2F%2Ftorrent.ubuntu.com%2Fannounce&tr=https%3A%2F%2Ftorrent.ubuntu.com%2Fannounce&tr=https%3A%2F%2Fipv6.torrent.ubuntu.com%2Fannounce"
		result, err := FromURL(magnet)

		assert.NoError(t, err)
		assert.Equal(t, int64(0), result.Info.Length)
	})

	t.Run("empty magnet link", func(t *testing.T) {
		magnet := ""
		result, err := FromURL(magnet)

		assert.Error(t, err)
		assert.Nil(t, result)
	})
}
