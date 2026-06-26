package rotate_test

import (
	"testing"
	"time"

	"github.com/3wayproxy/shared/rotate"
)

func TestRotate2Plus1(t *testing.T) {
	s := rotate.Schedule{N: 3, Interval: time.Second}
	for epoch := int64(0); epoch < 6; epoch++ {
		idle := s.IdleIndex(epoch)
		active := s.ActiveIndices(epoch)
		if active[0] == idle || active[1] == idle || active[0] == active[1] {
			t.Fatalf("epoch %d idle=%d active=%v", epoch, idle, active)
		}
	}
}
