package reasm

import (
	"sync"
	"time"

	"github.com/3wayproxy/shared/proto"
)

const defaultTTL = 5 * time.Second

type pending struct {
	total   uint16
	parts   map[uint16][]byte
	created time.Time
}

// Assembler collects FRAGMENT frames into full IP packets.
type Assembler struct {
	mu    sync.Mutex
	ttl   time.Duration
	pending map[uint32]*pending
}

func NewAssembler() *Assembler {
	return &Assembler{
		ttl:     defaultTTL,
		pending: make(map[uint32]*pending),
	}
}

// Feed ingests one fragment. Returns assembled IP packet when complete.
func (a *Assembler) Feed(h proto.FragmentHeader, payload []byte) ([]byte, bool) {
	a.mu.Lock()
	defer a.mu.Unlock()

	a.evictLocked()

	p, ok := a.pending[h.PacketID]
	if !ok {
		p = &pending{
			total:   h.FragTotal,
			parts:   make(map[uint16][]byte),
			created: time.Now(),
		}
		a.pending[h.PacketID] = p
	}
	p.parts[h.FragIdx] = append([]byte(nil), payload...)

	if uint16(len(p.parts)) < p.total {
		return nil, false
	}
	for i := uint16(0); i < p.total; i++ {
		if _, ok := p.parts[i]; !ok {
			return nil, false
		}
	}

	var out []byte
	for i := uint16(0); i < p.total; i++ {
		out = append(out, p.parts[i]...)
	}
	delete(a.pending, h.PacketID)
	return out, true
}

func (a *Assembler) evictLocked() {
	now := time.Now()
	for id, p := range a.pending {
		if now.Sub(p.created) > a.ttl {
			delete(a.pending, id)
		}
	}
}
