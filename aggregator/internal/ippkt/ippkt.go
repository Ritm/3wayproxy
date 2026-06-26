package ippkt

import "fmt"

// LogSummary returns a short description of an IPv4 packet for logging.
func LogSummary(pkt []byte) string {
	if len(pkt) < 20 {
		return fmt.Sprintf("%dB (short)", len(pkt))
	}
	vhl := pkt[0]
	if vhl>>4 != 4 {
		return fmt.Sprintf("%dB (not-ipv4)", len(pkt))
	}
	ihl := int(vhl&0x0f) * 4
	if len(pkt) < ihl {
		return fmt.Sprintf("%dB (bad-ihl)", len(pkt))
	}
	src := fmt.Sprintf("%d.%d.%d.%d", pkt[12], pkt[13], pkt[14], pkt[15])
	dst := fmt.Sprintf("%d.%d.%d.%d", pkt[16], pkt[17], pkt[18], pkt[19])
	proto := pkt[9]
	return fmt.Sprintf("%dB %s→%s proto=%d", len(pkt), src, dst, proto)
}
