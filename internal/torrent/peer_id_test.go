package torrent

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPeerID(t *testing.T) {
	id, err := peerID()

	require.NoError(t, err, "expected no error, but got %v", err)

	zeroID := [20]byte{}
	assert.NotEqual(t, zeroID, id, "expected ID to be non-zero, but got zero ID")
}
