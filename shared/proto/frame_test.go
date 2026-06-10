package proto_test

import (
	"testing"

	"github.com/3wayproxy/shared/proto"
)

func TestFragmentRoundTrip(t *testing.T) {
	h := proto.FragmentHeader{
		SessionID:  0xDEADBEEFCAFEBABE,
		PacketID:   42,
		FragIdx:    0,
		FragTotal:  2,
		PayloadLen: 4,
	}
	payload := []byte{10, 20, 30, 40}
	padding := []byte{0xAA, 0xBB}

	enc := proto.EncodeFragment(h, payload, padding)
	got, gotPayload, gotPad, err := proto.DecodeFragment(enc)
	if err != nil {
		t.Fatal(err)
	}
	if got.SessionID != h.SessionID || got.PacketID != h.PacketID {
		t.Fatalf("header mismatch: %+v vs %+v", got, h)
	}
	if string(gotPayload) != string(payload) || string(gotPad) != string(padding) {
		t.Fatalf("payload/pad mismatch")
	}
}
