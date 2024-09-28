package message

import (
	"log"
	"testing"
)

func TestMessageType(t *testing.T) {
	tests := []struct {
		name string
		got  uint8
		want uint8
	}{
		{"should be a valid msg Choke id", NewChoke().ID, Choke},
		{"should be a valid msg UnChoke id", NewUnChoke().ID, UnChoke},
		{"should be a valid msg Interested id", NewInterested().ID, Interested},
		{"should be a valid msg NotInterested id", NewNotInterested().ID, NotInterested},
		{"should be a valid msg Have id", NewHave(0).ID, Have},
		{"should be a valid msg Bitfield id", NewBitfield([]byte{}).ID, Bitfield},
		{"should be a valid msg Request id", NewRequest(0, 0, 128).ID, Request},
		{"should be a valid msg Piece id", NewPiece(0, 0, []byte{}).ID, Piece},
		{"should be a valid msg Cancel id", NewCancel(0, 0, 128).ID, Cancel},
		{"should be a valid msg Port id", NewPort(8080).ID, Port},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.got != tt.want {
				t.Errorf("expected message id to be %v, got %v", tt.want, tt.got)
			}
		})
	}
}

func TestMessagePayload(t *testing.T) {
	tests := []struct {
		name string
		got  int
		want int
	}{
		{"should be a valid msg Choke payload", len(NewChoke().Payload), 0},
		{"should be a valid msg UnChoke payload", len(NewUnChoke().Payload), 0},
		{"should be a valid msg Interested payload", len(NewInterested().Payload), 0},
		{"should be a valid msg NotInterested payload", len(NewNotInterested().Payload), 0},
		{"should be a valid msg Have payload", len(NewHave(0).Payload), 4},
		{"should be a valid msg Bitfield payload", len(NewBitfield([]byte{1, 2, 3, 4}).Payload), 4},
		{"should be a valid msg Request payload", len(NewRequest(0, 0, 128).Payload), 12},
		{"should be a valid msg Piece payload", len(NewPiece(1, 2, []byte{1, 2, 3, 4}).Payload), 12},
		{"should be a valid msg Cancel payload", len(NewCancel(1, 2, 128).Payload), 12},
		{"should be a valid msg Port payload", len(NewPort(8080).Payload), 2},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.got != tt.want {
				log.Println("ok", tt.want, tt.got)
				t.Errorf("expected payload size to be %v, got %v", tt.want, tt.got)
			}
		})
	}
}
