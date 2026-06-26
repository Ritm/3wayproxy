// Package rotate implements synchronized 2+1 relay scheduling.
package rotate

import "time"

// Schedule rotates which relay is idle among N relays (default 3).
// Client and aggregator must use the same Interval and N.
type Schedule struct {
	N        int
	Interval time.Duration
}

func (s Schedule) norm() Schedule {
	out := s
	if out.N < 2 {
		out.N = 3
	}
	if out.Interval <= 0 {
		out.Interval = 5 * time.Second
	}
	return out
}

// Epoch returns monotonic rotation epoch for time t.
func (s Schedule) Epoch(t time.Time) int64 {
	s = s.norm()
	return t.UnixNano() / int64(s.Interval)
}

// IdleIndex is the relay resting this epoch.
func (s Schedule) IdleIndex(epoch int64) int {
	s = s.norm()
	if epoch < 0 {
		epoch = 0
	}
	return int(epoch % int64(s.N))
}

// ActiveIndices returns two relay indices carrying traffic.
func (s Schedule) ActiveIndices(epoch int64) [2]int {
	s = s.norm()
	idle := s.IdleIndex(epoch)
	return [2]int{(idle + 1) % s.N, (idle + 2) % s.N}
}

// IsActive reports whether relay index is active at epoch.
func (s Schedule) IsActive(epoch int64, relayIdx int) bool {
	idle := s.IdleIndex(epoch)
	return relayIdx != idle
}
