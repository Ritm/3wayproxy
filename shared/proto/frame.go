// Package proto defines the binary wire format for 3wayproxy carrier frames.
package proto

import (
	"encoding/binary"
	"errors"
	"fmt"
)

const Version uint8 = 1

const (
	TypeHandshake    uint8 = 0x01
	TypeHandshakeAck uint8 = 0x02
	TypeResume       uint8 = 0x03
	TypeFragment     uint8 = 0x04
	TypeAck          uint8 = 0x05
	TypeHeartbeat    uint8 = 0x06
)

var (
	ErrShortBuffer = errors.New("proto: buffer too short")
	ErrBadType     = errors.New("proto: unknown frame type")
)

// FragmentHeader is the fixed prefix of a FRAGMENT frame (without padding).
type FragmentHeader struct {
	SessionID  uint64
	PacketID   uint32
	FragIdx    uint16
	FragTotal  uint16
	PayloadLen uint16
}

// EncodeFragment builds a FRAGMENT frame with optional random padding.
func EncodeFragment(h FragmentHeader, payload, padding []byte) []byte {
	n := 1 + 8 + 4 + 2 + 2 + 2 + len(payload) + len(padding)
	out := make([]byte, n)
	out[0] = TypeFragment
	binary.BigEndian.PutUint64(out[1:9], h.SessionID)
	binary.BigEndian.PutUint32(out[9:13], h.PacketID)
	binary.BigEndian.PutUint16(out[13:15], h.FragIdx)
	binary.BigEndian.PutUint16(out[15:17], h.FragTotal)
	binary.BigEndian.PutUint16(out[17:19], h.PayloadLen)
	copy(out[19:19+len(payload)], payload)
	copy(out[19+len(payload):], padding)
	return out
}

// DecodeFragment parses a FRAGMENT frame and returns header, payload, padding.
func DecodeFragment(buf []byte) (FragmentHeader, []byte, []byte, error) {
	if len(buf) < 19 {
		return FragmentHeader{}, nil, nil, ErrShortBuffer
	}
	if buf[0] != TypeFragment {
		return FragmentHeader{}, nil, nil, fmt.Errorf("%w: %02x", ErrBadType, buf[0])
	}
	h := FragmentHeader{
		SessionID:  binary.BigEndian.Uint64(buf[1:9]),
		PacketID:   binary.BigEndian.Uint32(buf[9:13]),
		FragIdx:    binary.BigEndian.Uint16(buf[13:15]),
		FragTotal:  binary.BigEndian.Uint16(buf[15:17]),
		PayloadLen: binary.BigEndian.Uint16(buf[17:19]),
	}
	need := 19 + int(h.PayloadLen)
	if len(buf) < need {
		return FragmentHeader{}, nil, nil, ErrShortBuffer
	}
	payload := buf[19:need]
	padding := buf[need:]
	return h, payload, padding, nil
}
