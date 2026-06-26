package browser

import (
	"strings"
	"testing"
)

func TestPlayPageURL(t *testing.T) {
	u, err := PlayPageURL("ws://127.0.0.1:8001/ws/play", 1, 9000012345678901)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(u, "http://127.0.0.1:8001/play.html?") {
		t.Fatalf("unexpected url: %s", u)
	}
	if !strings.Contains(u, "session=9000012345678901") {
		t.Fatalf("missing session: %s", u)
	}
}
