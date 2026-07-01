package browser

import (
	"context"
	"encoding/base64"
	"fmt"
	"log"
	"sync"
	"sync/atomic"
	"time"

	"github.com/playwright-community/playwright-go"

	"github.com/3wayproxy/shared/pool"
)

const dialTimeout = 90 * time.Second

// Dialer opens WebSocket sessions via Chromium + ws_carrier.js.
type Dialer struct {
	mgr *Manager
}

func NewDialer(mgr *Manager) *Dialer {
	return &Dialer{mgr: mgr}
}

func (d *Dialer) DialPlayer(ctx context.Context, idx int, ep pool.Endpoint, sessionID uint64, resume bool) (pool.RelayConn, error) {
	dctx, cancel := context.WithTimeout(ctx, dialTimeout)
	defer cancel()
	type res struct {
		c   pool.RelayConn
		err error
	}
	ch := make(chan res, 1)
	go func() {
		c, err := d.open(dctx, idx, ep, sessionID, resume)
		ch <- res{c, err}
	}()
	select {
	case <-dctx.Done():
		return nil, fmt.Errorf("browser: relay %d timeout after %s (check: sudo -E, system chrome, VPN)", idx, dialTimeout)
	case r := <-ch:
		return r.c, r.err
	}
}

func (d *Dialer) DialSpectator(ctx context.Context, idx int, ep pool.Endpoint, sessionID uint64, resume bool) (pool.RelayConn, error) {
	return nil, fmt.Errorf("browser carrier: spectator role not supported (use native dialer on aggregator)")
}

func (d *Dialer) open(ctx context.Context, idx int, ep pool.Endpoint, sessionID uint64, resume bool) (pool.RelayConn, error) {
	browser, err := d.mgr.browserInstance()
	if err != nil {
		return nil, err
	}
	playURL, err := PlayPageURL(ep.URL, ep.ShardID, sessionID)
	if err != nil {
		return nil, err
	}
	log.Printf("browser: relay %d opening %s", idx, playURL)

	bctx, err := browser.NewContext(playwright.BrowserNewContextOptions{
		IgnoreHttpsErrors: playwright.Bool(true),
	})
	if err != nil {
		return nil, fmt.Errorf("browser context: %w", err)
	}

	page, err := bctx.NewPage()
	if err != nil {
		_ = bctx.Close()
		return nil, fmt.Errorf("new page: %w", err)
	}

	recvCh := make(chan []byte, 2048)
	sendCh := make(chan []byte, 256)
	closed := &atomic.Bool{}
	var wg sync.WaitGroup

	deliver := func(source *playwright.BindingSource, args ...interface{}) interface{} {
		if closed.Load() || len(args) == 0 {
			return nil
		}
		b64, ok := args[0].(string)
		if !ok {
			return nil
		}
		b, err := base64.StdEncoding.DecodeString(b64)
		if err != nil {
			log.Printf("browser: relay %d tunDeliver decode: %v", idx, err)
			return nil
		}
		select {
		case recvCh <- b:
		case <-ctx.Done():
		}
		return nil
	}

	if err := page.ExposeBinding("tunDeliver", deliver, false); err != nil {
		_ = page.Close()
		_ = bctx.Close()
		return nil, fmt.Errorf("expose binding: %w", err)
	}

	if _, err := page.Goto(playURL, playwright.PageGotoOptions{
		WaitUntil: playwright.WaitUntilStateCommit,
		Timeout:   playwright.Float(45000),
	}); err != nil {
		_ = page.Close()
		_ = bctx.Close()
		return nil, fmt.Errorf("goto %s: %w", playURL, err)
	}

	if _, err := page.WaitForFunction("() => window.__carrier && window.__carrier.isReady()", nil, playwright.PageWaitForFunctionOptions{
		Timeout: playwright.Float(45000),
	}); err != nil {
		_ = page.Close()
		_ = bctx.Close()
		return nil, fmt.Errorf("carrier not ready: %w", err)
	}

	if resume {
		if _, err := page.Evaluate(`([sid, a, b]) => window.__carrier.resume(BigInt(sid), a, b)`, []interface{}{
			fmt.Sprintf("%d", sessionID), 0, 0,
		}); err != nil {
			log.Printf("browser: resume relay %d: %v", idx, err)
		}
	}

	wg.Add(1)
	go func() {
		defer wg.Done()
		for {
			select {
			case <-ctx.Done():
				return
			case frame, ok := <-sendCh:
				if !ok || closed.Load() {
					return
				}
				b64 := base64.StdEncoding.EncodeToString(frame)
				if _, err := page.Evaluate(`b64 => window.__carrier.sendBase64(b64)`, b64); err != nil {
					if !closed.Load() {
						log.Printf("browser: relay %d send: %v", idx, err)
					}
					return
				}
			}
		}
	}()

	log.Printf("browser: relay %d ready %s", idx, wsHost(ep.URL))
	return &relayConn{
		page:   page,
		bctx:   bctx,
		recvCh: recvCh,
		sendCh: sendCh,
		closed: closed,
		ctx:    ctx,
		wg:     &wg,
	}, nil
}

type relayConn struct {
	page   playwright.Page
	bctx   playwright.BrowserContext
	recvCh chan []byte
	sendCh chan []byte
	closed *atomic.Bool
	ctx    context.Context
	wg     *sync.WaitGroup
}

func (c *relayConn) Send(frame []byte) error {
	if c.closed.Load() {
		return fmt.Errorf("browser: conn closed")
	}
	dup := append([]byte(nil), frame...)
	select {
	case c.sendCh <- dup:
		return nil
	case <-c.ctx.Done():
		return c.ctx.Err()
	}
}

func (c *relayConn) Recv() ([]byte, error) {
	for {
		if c.closed.Load() {
			return nil, fmt.Errorf("browser: conn closed")
		}
		select {
		case b := <-c.recvCh:
			return b, nil
		case <-c.ctx.Done():
			return nil, c.ctx.Err()
		case <-time.After(100 * time.Millisecond):
		}
	}
}

func (c *relayConn) Close() error {
	if c.closed.Swap(true) {
		return nil
	}
	close(c.sendCh)
	c.wg.Wait()
	_ = c.page.Close()
	return c.bctx.Close()
}
