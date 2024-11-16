package tracker

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/xanish/torrenty/internal/metadata"
)

func TestTracker(t *testing.T) {
	torrent := metadata.Metadata{
		Announce: "https://torrent.ubuntu.com/announce",
		InfoHash: [20]byte{0xc9, 0x6b, 0x2a, 0x9b, 0xa9, 0xed, 0x1b, 0xd2, 0x58, 0x2c, 0x78, 0x5d, 0xa8, 0xa8, 0x6f, 0x2c, 0xd5, 0x51, 0x7c, 0x7},
	}

	peerID := [20]byte{0x00, 0x01, 0x02, 0x03, 0x04}
	port := uint16(6881)

	t.Run("new tracker", func(t *testing.T) {
		tracker := New(peerID, port, torrent)

		assert.NotNil(t, tracker)
		assert.Equal(t, peerID, tracker.peerID)
		assert.Equal(t, port, tracker.port)
		assert.Equal(t, torrent, tracker.torrent)
	})

	t.Run("generate correct url", func(t *testing.T) {
		tracker := New(peerID, port, torrent)
		url := tracker.URL()

		expectedURL := "https://torrent.ubuntu.com/announce?compact=1&downloaded=0&event=started&info_hash=%C9k%2A%9B%A9%ED%1B%D2X%2Cx%5D%A8%A8o%2C%D5Q%7C%07&left=0&peer_id=%00%01%02%03%04%00%00%00%00%00%00%00%00%00%00%00%00%00%00%00&port=6881&uploaded=0"
		assert.Equal(t, expectedURL, url)
		assert.Contains(t, url, "info_hash=%C9k%2A%9B%A9%ED%1B%D2X%2Cx%5D%A8%A8o%2C%D5Q%7C%07")
		assert.Contains(t, url, "peer_id=%00%01%02%03%04%00%00%00%00%00%00%00%00%00%00%00%00%00%00%00")
		assert.Contains(t, url, "port=6881")
		assert.Contains(t, url, "event=started")
		assert.Contains(t, url, "compact=1")
		assert.Contains(t, url, "downloaded=0")
		assert.Contains(t, url, "uploaded=0")
	})

	t.Run("refresh successful response", func(t *testing.T) {
		handler := func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/x-bittorrent")
			w.WriteHeader(http.StatusOK)
			response := []byte(
				"d" +
					"8:interval" + "i1800e" +
					"8:complete" + "i10e" +
					"10:incomplete" + "i5e" +
					"5:peers" + "12:" +
					string([]byte{
						192, 0, 2, 123, 0x1A, 0xE1, // 0x1AE1 = 6881
						127, 0, 0, 1, 0x1A, 0xE9, // 0x1AE9 = 6889
					}) + "e")
			w.Write(response)
		}

		// Mock HTTP server
		server := httptest.NewServer(http.HandlerFunc(handler))
		defer server.Close()

		// Update tracker to use the test server URL
		torrent.Announce = server.URL
		tracker := New(peerID, port, torrent)
		peers, duration, err := tracker.Refresh()

		assert.NoError(t, err)
		assert.NotNil(t, peers)
		assert.Equal(t, time.Duration(1800)*time.Second, duration)
	})

	t.Run("refresh failed response", func(t *testing.T) {
		// Simulate an error response from the tracker
		handler := func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/x-bittorrent")
			w.WriteHeader(http.StatusOK)
			response := []byte("d" + "14:failure reason" + "12:Server Errore" + "e")
			w.Write(response)
		}

		// Mock HTTP server
		server := httptest.NewServer(http.HandlerFunc(handler))
		defer server.Close()

		// Update tracker to use the test server URL
		torrent.Announce = server.URL
		tracker := New(peerID, port, torrent)
		peers, duration, err := tracker.Refresh()

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "tracker returned an error")
		assert.Nil(t, peers)
		assert.Equal(t, time.Duration(0), duration)
	})

	t.Run("refresh invalid bencode", func(t *testing.T) {
		handler := func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/x-bittorrent")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("invalid data"))
		}

		// Mock HTTP server
		server := httptest.NewServer(http.HandlerFunc(handler))
		defer server.Close()

		// Update tracker to use the test server URL
		torrent.Announce = server.URL
		tracker := New(peerID, port, torrent)
		peers, duration, err := tracker.Refresh()

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to decode tracker response")
		assert.Nil(t, peers)
		assert.Equal(t, time.Duration(0), duration)
	})
}
