package main

import (
	"context"
	"flag"
	"io"
	"log"
	"os"
	"os/signal"
	"sync/atomic"
	"syscall"

	"github.com/3wayproxy/client/internal/config"
	"github.com/3wayproxy/shared/fragment"
	"github.com/3wayproxy/shared/pool"
	"github.com/3wayproxy/shared/proto"
	"github.com/3wayproxy/shared/reasm"
	"github.com/3wayproxy/shared/tun"
)

func main() {
	cfgPath := flag.String("config", "config/client.dev.yaml", "path to config yaml")
	flag.Parse()

	cfg, err := config.Load(*cfgPath)
	if err != nil {
		log.Fatalf("config: %v", err)
	}

	sessionID := cfg.SessionID
	if sessionID == 0 {
		sessionID = fragment.NewSessionID()
		log.Printf("generated session_id=%d", sessionID)
	}

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	iface, err := tun.Open(tun.Config{
		Name:    cfg.TUN.Name,
		LocalIP: cfg.TUN.LocalIP,
		PeerIP:  cfg.TUN.PeerIP,
		MTU:     cfg.TUN.MTU,
	})
	if err != nil {
		log.Fatalf("tun: %v", err)
	}
	defer iface.Close()

	if err := tun.AddRoutes(iface.Name(), cfg.TUN.Routes); err != nil {
		log.Fatalf("routes: %v", err)
	}
	if err := tun.AddBypassRoutes(cfg.TUN.BypassRoutes); err != nil {
		log.Fatalf("bypass routes: %v", err)
	}

	poolCfg, err := cfg.PoolConfig(sessionID)
	if err != nil {
		log.Fatalf("pool config: %v", err)
	}
	relayPool, err := pool.New(poolCfg)
	if err != nil {
		log.Fatalf("pool: %v", err)
	}
	defer relayPool.Close()
	if err := relayPool.Start(ctx); err != nil {
		log.Fatalf("pool start: %v", err)
	}

	mode := "single-relay"
	if cfg.MultiRelay() {
		mode = "3-relay 2+1"
	}
	log.Printf("tun %s up, session_id=%d, mode=%s", iface.Name(), sessionID, mode)

	var packetID atomic.Uint32
	assembler := reasm.NewAssembler()
	errCh := make(chan error, 2)

	go func() {
		buf := make([]byte, cfg.TUN.MTU)
		for {
			select {
			case <-ctx.Done():
				errCh <- nil
				return
			default:
			}
			n, err := iface.Read(buf)
			if err != nil {
				errCh <- err
				return
			}
			if n == 0 {
				continue
			}
			pkt := append([]byte(nil), buf[:n]...)
			id := packetID.Add(1)
			log.Printf("tun read %d bytes → relay", len(pkt))
			if err := relayPool.SendPacket(ctx, id, pkt); err != nil {
				log.Printf("send packet: %v", err)
				continue
			}
		}
	}()

	go func() {
		for {
			select {
			case <-ctx.Done():
				errCh <- nil
				return
			case frame, ok := <-relayPool.Recv():
				if !ok {
					errCh <- io.EOF
					return
				}
				if proto.FrameType(frame) != proto.TypeFragment {
					continue
				}
				h, payload, _, err := proto.DecodeFragment(frame)
				if err != nil {
					log.Printf("bad fragment: %v", err)
					continue
				}
				if h.SessionID != sessionID {
					continue
				}
				pkt, ok := assembler.Feed(h, payload)
				if !ok {
					continue
				}
				log.Printf("relay → tun write %d bytes", len(pkt))
				if _, err := iface.Write(pkt); err != nil {
					errCh <- err
					return
				}
			}
		}
	}()

	if err := <-errCh; err != nil && err != io.EOF {
		log.Fatalf("run: %v", err)
	}
}
