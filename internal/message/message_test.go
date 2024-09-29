package message

import (
	"fmt"
	"testing"
)

func TestMessageType(t *testing.T) {
	tests := []struct {
		name string
		got  string
		want string
	}{
		{
			"should be a valid Choke msg",
			NewChoke().name(),
			"Choke",
		},
		{
			"should be a valid UnChoke msg",
			NewUnChoke().name(),
			"UnChoke",
		},
		{
			"should be a valid Interested msg",
			NewInterested().name(),
			"Interested",
		},
		{
			"should be a valid NotInterested msg",
			NewNotInterested().name(),
			"NotInterested",
		},
		{
			"should be a valid Have msg",
			NewHave(0).name(),
			"Have",
		},
		{
			"should be a valid Bitfield msg",
			NewBitfield([]byte{}).name(),
			"Bitfield",
		},
		{
			"should be a valid Request msg",
			NewRequest(0, 0, 128).name(),
			"Request",
		},
		{
			"should be a valid Piece msg",
			NewPiece(0, 0, []byte{}).name(),
			"Piece",
		},
		{
			"should be a valid Cancel msg",
			NewCancel(0, 0, 128).name(),
			"Cancel",
		},
		{
			"should be a valid Port msg",
			NewPort(8080).name(),
			"Port",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.got != tt.want {
				t.Errorf("expected message id to be %v, got %v", tt.want, tt.got)
			}
		})
	}
}

func TestMessage_String(t *testing.T) {
	expectedFormat := "message<%s>: <len=%d><id=%d>"

	tests := []struct {
		name string
		got  string
		want string
	}{
		{
			"should be a valid string Choke msg",
			NewChoke().String(),
			fmt.Sprintf(expectedFormat, "Choke", 0, Choke),
		},
		{
			"should be a valid string UnChoke msg",
			NewUnChoke().String(),
			fmt.Sprintf(expectedFormat, "UnChoke", 0, UnChoke),
		},
		{
			"should be a valid string Interested msg",
			NewInterested().String(),
			fmt.Sprintf(expectedFormat, "Interested", 0, Interested),
		},
		{
			"should be a valid string NotInterested msg",
			NewNotInterested().String(),
			fmt.Sprintf(expectedFormat, "NotInterested", 0, NotInterested),
		},
		{
			"should be a valid string Have msg",
			NewHave(0).String(),
			fmt.Sprintf(expectedFormat, "Have", 4, Have),
		},
		{
			"should be a valid string Bitfield msg",
			NewBitfield([]byte{1, 2, 3, 4}).String(),
			fmt.Sprintf(expectedFormat, "Bitfield", 4, Bitfield),
		},
		{
			"should be a valid string Request msg",
			NewRequest(0, 0, 128).String(),
			fmt.Sprintf(expectedFormat, "Request", 12, Request),
		},
		{
			"should be a valid string Piece msg",
			NewPiece(1, 2, []byte{1, 2, 3, 4}).String(),
			fmt.Sprintf(expectedFormat, "Piece", 12, Piece),
		},
		{
			"should be a valid string Cancel msg",
			NewCancel(1, 2, 128).String(),
			fmt.Sprintf(expectedFormat, "Cancel", 12, Cancel),
		},
		{
			"should be a valid string Port msg",
			NewPort(8080).String(),
			fmt.Sprintf(expectedFormat, "Port", 2, Port),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.got != tt.want {
				t.Errorf("expected message string to be %v, got %v", tt.want, tt.got)
			}
		})
	}
}
