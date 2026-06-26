package proto_test

import (
	"testing"

	"github.com/3wayproxy/shared/proto"
)

func TestHandshakeRoundTrip(t *testing.T) {
	enc := proto.EncodeHandshake(2, 0xABCDEF0123456789, []byte("1234567890123456"))
	sid, sess, nonce, err := proto.DecodeHandshake(enc)
	if err != nil {
		t.Fatal(err)
	}
	if sid != 2 || sess != 0xABCDEF0123456789 || string(nonce) != "1234567890123456" {
		t.Fatalf("got shard=%d sess=%x nonce=%q", sid, sess, nonce)
	}
}
