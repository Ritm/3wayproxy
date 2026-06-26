package proto

import "encoding/binary"

const (
	TypeHandshakeAgg uint8 = 0xA1
)

// EncodeHandshake builds a client → relay HANDSHAKE frame.
func EncodeHandshake(shardID uint16, sessionID uint64, nonce []byte) []byte {
	if len(nonce) < 16 {
		padded := make([]byte, 16)
		copy(padded, nonce)
		nonce = padded
	}
	out := make([]byte, 28)
	out[0] = TypeHandshake
	out[1] = Version
	binary.BigEndian.PutUint16(out[2:4], shardID)
	binary.BigEndian.PutUint64(out[4:12], sessionID)
	copy(out[12:28], nonce[:16])
	return out
}

// DecodeHandshake parses HANDSHAKE.
func DecodeHandshake(buf []byte) (shardID uint16, sessionID uint64, nonce []byte, err error) {
	if len(buf) < 28 {
		return 0, 0, nil, ErrShortBuffer
	}
	if buf[0] != TypeHandshake {
		return 0, 0, nil, ErrBadType
	}
	shardID = binary.BigEndian.Uint16(buf[2:4])
	sessionID = binary.BigEndian.Uint64(buf[4:12])
	nonce = append([]byte(nil), buf[12:28]...)
	return shardID, sessionID, nonce, nil
}

// EncodeHandshakeAck builds relay → client HANDSHAKE_ACK.
func EncodeHandshakeAck(sessionID uint64, mtuHint uint16, relayShardID uint8) []byte {
	out := make([]byte, 13)
	out[0] = TypeHandshakeAck
	out[1] = Version
	binary.BigEndian.PutUint64(out[2:10], sessionID)
	binary.BigEndian.PutUint16(out[10:12], mtuHint)
	out[12] = relayShardID
	return out
}

// DecodeHandshakeAck parses HANDSHAKE_ACK.
func DecodeHandshakeAck(buf []byte) (sessionID uint64, mtuHint uint16, relayShardID uint8, err error) {
	if len(buf) < 13 {
		return 0, 0, 0, ErrShortBuffer
	}
	if buf[0] != TypeHandshakeAck {
		return 0, 0, 0, ErrBadType
	}
	sessionID = binary.BigEndian.Uint64(buf[2:10])
	mtuHint = binary.BigEndian.Uint16(buf[10:12])
	relayShardID = buf[12]
	return sessionID, mtuHint, relayShardID, nil
}

// EncodeHandshakeAgg builds aggregator → relay spectator handshake.
func EncodeHandshakeAgg(relayID uint8, sessionID uint64) []byte {
	out := make([]byte, 10)
	out[0] = TypeHandshakeAgg
	out[1] = relayID
	binary.BigEndian.PutUint64(out[2:10], sessionID)
	return out
}

// DecodeHandshakeAgg parses aggregator handshake.
func DecodeHandshakeAgg(buf []byte) (relayID uint8, sessionID uint64, err error) {
	if len(buf) < 10 {
		return 0, 0, ErrShortBuffer
	}
	if buf[0] != TypeHandshakeAgg {
		return 0, 0, ErrBadType
	}
	return buf[1], binary.BigEndian.Uint64(buf[2:10]), nil
}

// FrameType returns the leading type byte or 0 if empty.
func FrameType(buf []byte) uint8 {
	if len(buf) == 0 {
		return 0
	}
	return buf[0]
}
