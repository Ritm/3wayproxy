package pool

import (
	"context"
	"fmt"
	"log"
	"sync"
	"sync/atomic"
	"time"

	"github.com/3wayproxy/shared/fragment"
	"github.com/3wayproxy/shared/proto"
	"github.com/3wayproxy/shared/rotate"
	"github.com/3wayproxy/shared/wsclient"
)

type Role int

const (
	RolePlayer Role = iota
	RoleSpectator
)

type Endpoint struct {
	RelayID uint8
	ShardID uint16
	URL     string
}

type Config struct {
	SessionID      uint64
	Endpoints      []Endpoint
	Role           Role
	RotateEvery    time.Duration
	ChurnEvery     time.Duration
	Fragments      int
	DisconnectIdle bool // false = idle relay stays connected (stable); true = drop WS on idle leg
}

type Pool struct {
	cfg    Config
	sched  rotate.Schedule
	conns  []*wsclient.Conn
	mu     sync.Mutex
	sendMu sync.Mutex
	recvCh chan []byte
	closed atomic.Bool
}

func New(cfg Config) (*Pool, error) {
	if len(cfg.Endpoints) == 0 {
		return nil, fmt.Errorf("pool: no endpoints")
	}
	if cfg.Fragments < 1 {
		cfg.Fragments = 1
	}
	if len(cfg.Endpoints) >= 3 && cfg.Fragments < 2 {
		cfg.Fragments = 2
	}
	if cfg.RotateEvery <= 0 {
		cfg.RotateEvery = 5 * time.Second
	}
	if cfg.ChurnEvery <= 0 {
		cfg.ChurnEvery = 3 * time.Second
	}
	return &Pool{
		cfg:    cfg,
		sched:  rotate.Schedule{N: len(cfg.Endpoints), Interval: cfg.RotateEvery},
		conns:  make([]*wsclient.Conn, len(cfg.Endpoints)),
		recvCh: make(chan []byte, 256),
	}, nil
}

func (p *Pool) Start(ctx context.Context) error {
	for i := range p.cfg.Endpoints {
		if err := p.connect(ctx, i, false); err != nil {
			p.Close()
			return fmt.Errorf("connect relay %d: %w", i, err)
		}
		go p.recvLoop(ctx, i)
	}
	if len(p.cfg.Endpoints) >= 3 {
		go p.rotateLoop(ctx)
	}
	go p.churnLoop(ctx)
	return nil
}

func (p *Pool) Recv() <-chan []byte {
	return p.recvCh
}

func (p *Pool) Close() {
	if p.closed.Swap(true) {
		return
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	for i, c := range p.conns {
		if c != nil {
			_ = c.Close()
			p.conns[i] = nil
		}
	}
	close(p.recvCh)
}

// SendPacket fragments and sends via active relays (2+1 when 3 endpoints).
func (p *Pool) SendPacket(ctx context.Context, packetID uint32, pkt []byte) error {
	p.sendMu.Lock()
	defer p.sendMu.Unlock()

	n := p.cfg.Fragments
	if len(p.cfg.Endpoints) < 3 {
		n = 1
	}
	frames := fragment.Split(p.cfg.SessionID, packetID, pkt, n)
	if len(p.cfg.Endpoints) < 3 {
		return p.sendTo(ctx, 0, frames[0])
	}
	if len(frames) < 2 {
		return fmt.Errorf("pool: expected 2 fragments, got %d", len(frames))
	}
	epoch := p.sched.Epoch(time.Now())
	active := p.sched.ActiveIndices(epoch)
	if err := p.ensureConnected(ctx, active[0], active[1]); err != nil {
		return err
	}
	if err := p.sendTo(ctx, active[0], frames[0]); err != nil {
		return err
	}
	return p.sendTo(ctx, active[1], frames[1])
}

func (p *Pool) ensureConnected(ctx context.Context, indices ...int) error {
	for _, idx := range indices {
		if !p.isConnected(idx) {
			if err := p.connect(ctx, idx, true); err != nil {
				return fmt.Errorf("pool: relay %d: %w", idx, err)
			}
		}
	}
	return nil
}

func (p *Pool) sendTo(ctx context.Context, idx int, frame []byte) error {
	if !p.isConnected(idx) {
		if err := p.connect(ctx, idx, true); err != nil {
			return fmt.Errorf("pool: relay %d not connected: %w", idx, err)
		}
	}
	p.mu.Lock()
	c := p.conns[idx]
	p.mu.Unlock()
	if c == nil {
		return fmt.Errorf("pool: relay %d not connected", idx)
	}
	return c.Send(frame)
}

func (p *Pool) connect(ctx context.Context, idx int, resume bool) error {
	ep := p.cfg.Endpoints[idx]
	p.mu.Lock()
	if p.conns[idx] != nil {
		_ = p.conns[idx].Close()
		p.conns[idx] = nil
	}
	p.mu.Unlock()

	var (
		c   *wsclient.Conn
		err error
	)
	switch p.cfg.Role {
	case RolePlayer:
		c, err = wsclient.DialPlayer(ctx, ep.URL, ep.ShardID, p.cfg.SessionID)
		if err == nil && resume {
			_ = c.Resume(p.cfg.SessionID, 0, 0)
		}
	case RoleSpectator:
		c, err = wsclient.DialSpectator(ctx, ep.URL, ep.RelayID, p.cfg.SessionID)
	default:
		return fmt.Errorf("pool: unknown role")
	}
	if err != nil {
		return err
	}
	p.mu.Lock()
	p.conns[idx] = c
	p.mu.Unlock()
	log.Printf("pool: connected relay %d %s", idx, ep.URL)
	return nil
}

func (p *Pool) disconnect(idx int) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.conns[idx] != nil {
		_ = p.conns[idx].Close()
		p.conns[idx] = nil
		log.Printf("pool: disconnected relay %d (idle)", idx)
	}
}

func (p *Pool) recvLoop(ctx context.Context, idx int) {
	for {
		select {
		case <-ctx.Done():
			return
		default:
		}
		p.mu.Lock()
		c := p.conns[idx]
		p.mu.Unlock()
		if c == nil {
			time.Sleep(50 * time.Millisecond)
			continue
		}
		frame, err := c.Recv()
		if err != nil {
			if p.closed.Load() || ctx.Err() != nil {
				return
			}
			time.Sleep(100 * time.Millisecond)
			continue
		}
		if proto.FrameType(frame) != proto.TypeFragment {
			continue
		}
		if p.closed.Load() {
			return
		}
		select {
		case p.recvCh <- frame:
		case <-ctx.Done():
			return
		}
	}
}

func (p *Pool) rotateLoop(ctx context.Context) {
	tick := time.NewTicker(500 * time.Millisecond)
	defer tick.Stop()
	var lastEpoch int64 = -1
	for {
		select {
		case <-ctx.Done():
			return
		case <-tick.C:
			epoch := p.sched.Epoch(time.Now())
			if epoch == lastEpoch {
				continue
			}
			lastEpoch = epoch
			idle := p.sched.IdleIndex(epoch)
			active := p.sched.ActiveIndices(epoch)
			log.Printf("pool: rotate epoch=%d idle=%d active=%d,%d", epoch, idle, active[0], active[1])

			p.sendMu.Lock()
			for _, ai := range active {
				if !p.isConnected(ai) {
					if err := p.connect(ctx, ai, true); err != nil {
						log.Printf("pool: reconnect active %d: %v", ai, err)
					}
				}
			}
			if p.cfg.DisconnectIdle {
				p.disconnect(idle)
			}
			p.sendMu.Unlock()
		}
	}
}

func (p *Pool) churnLoop(ctx context.Context) {
	if len(p.cfg.Endpoints) < 2 {
		return
	}
	tick := time.NewTicker(p.cfg.ChurnEvery)
	defer tick.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-tick.C:
			n := len(p.cfg.Endpoints)
			if n >= 3 {
				epoch := p.sched.Epoch(time.Now())
				idle := p.sched.IdleIndex(epoch)
				// Churn только idle-ногу — не рвём активные WS во время трафика
				if p.isConnected(idle) {
					log.Printf("pool: churn idle relay %d", idle)
					p.sendMu.Lock()
					_ = p.connect(ctx, idle, true)
					p.sendMu.Unlock()
				}
			} else {
				idx := 0
				log.Printf("pool: churn relay %d", idx)
				p.sendMu.Lock()
				_ = p.connect(ctx, idx, true)
				p.sendMu.Unlock()
			}
		}
	}
}

func (p *Pool) isConnected(idx int) bool {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.conns[idx] != nil
}
