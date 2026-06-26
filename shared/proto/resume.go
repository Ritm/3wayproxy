package proto

import "encoding/binary"

// EncodeResume builds RESUME after WSS churn.
func EncodeResume(sessionID uint64, lastSeqSent, lastSeqRecv uint32) []byte {
	out := make([]byte, 17)
	out[0] = TypeResume
	binary.BigEndian.PutUint64(out[1:9], sessionID)
	binary.BigEndian.PutUint32(out[9:13], lastSeqSent)
	binary.BigEndian.PutUint32(out[13:17], lastSeqRecv)
	return out
}

// DecodeResume parses RESUME.
func DecodeResume(buf []byte) (sessionID uint64, lastSeqSent, lastSeqRecv uint32, err error) {
	if len(buf) < 17 {
		return 0, 0, 0, ErrShortBuffer
	}
	if buf[0] != TypeResume {
		return 0, 0, 0, ErrBadType
	}
	sessionID = binary.BigEndian.Uint64(buf[1:9])
	lastSeqSent = binary.BigEndian.Uint32(buf[9:13])
	lastSeqRecv = binary.BigEndian.Uint32(buf[13:17])
	return sessionID, lastSeqSent, lastSeqRecv, nil
}
