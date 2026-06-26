package fragment

import (
	"crypto/rand"
	"encoding/binary"

	"github.com/3wayproxy/shared/proto"
)

// Split divides an IP packet into n fragments with random padding on each.
func Split(sessionID uint64, packetID uint32, packet []byte, n int) [][]byte {
	if n < 1 {
		n = 1
	}
	if len(packet) == 0 {
		return nil
	}

	chunkSize := (len(packet) + n - 1) / n
	var frames [][]byte
	for i := 0; i < n; i++ {
		start := i * chunkSize
		if start >= len(packet) {
			break
		}
		end := start + chunkSize
		if end > len(packet) {
			end = len(packet)
		}
		payload := packet[start:end]
		padding := randomPadding(0, 32)
		h := proto.FragmentHeader{
			SessionID:  sessionID,
			PacketID:   packetID,
			FragIdx:    uint16(i),
			FragTotal:  uint16(n),
			PayloadLen: uint16(len(payload)),
		}
		frames = append(frames, proto.EncodeFragment(h, payload, padding))
	}
	return frames
}

func randomPadding(min, max int) []byte {
	if max <= min {
		return nil
	}
	var b [1]byte
	if _, err := rand.Read(b[:]); err != nil {
		return nil
	}
	n := min + int(b[0])%(max-min+1)
	if n == 0 {
		return nil
	}
	p := make([]byte, n)
	_, _ = rand.Read(p)
	return p
}

// NewSessionID returns a random session id.
func NewSessionID() uint64 {
	var b [8]byte
	_, _ = rand.Read(b[:])
	return binary.BigEndian.Uint64(b[:])
}
