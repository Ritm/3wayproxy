package reasm_test

import (
	"testing"

	"github.com/3wayproxy/shared/proto"
	"github.com/3wayproxy/shared/reasm"
)

func TestAssemblerTwoFragments(t *testing.T) {
	a := reasm.NewAssembler()
	h1 := proto.FragmentHeader{SessionID: 1, PacketID: 7, FragIdx: 0, FragTotal: 2, PayloadLen: 3}
	h2 := proto.FragmentHeader{SessionID: 1, PacketID: 7, FragIdx: 1, FragTotal: 2, PayloadLen: 3}

	_, ok := a.Feed(h1, []byte{1, 2, 3})
	if ok {
		t.Fatal("expected incomplete")
	}
	out, ok := a.Feed(h2, []byte{4, 5, 6})
	if !ok || string(out) != string([]byte{1, 2, 3, 4, 5, 6}) {
		t.Fatalf("got %v ok=%v", out, ok)
	}
}
