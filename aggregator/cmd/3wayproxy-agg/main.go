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

	"github.com/3wayproxy/aggregator/internal/config"
	"github.com/3wayproxy/aggregator/internal/ippkt"
	"github.com/3wayproxy/aggregator/internal/nat"
	"github.com/3wayproxy/shared/pool"
	"github.com/3wayproxy/shared/proto"
	"github.com/3wayproxy/shared/reasm"
	"github.com/3wayproxy/shared/tun"
)

func main() {
	cfgPath := flag.String("config", "config/aggregator.dev.yaml", "path to config yaml")
	flag.Parse()

	cfg, err := config.Load(*cfgPath)
	if err != nil {
		log.Fatalf("config: %v", err)
	}
	if cfg.SessionID == 0 {
		log.Fatal("session_id must be set in aggregator config (same as client)")
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

	if err := tun.EnsurePeerRoute(iface.Name(), cfg.TUN.PeerIP); err != nil {
		log.Fatalf("peer route: %v", err)
	}
	if dev, err := tun.PeerRouteDev(cfg.TUN.PeerIP); err == nil {
		if dev != iface.Name() {
			log.Fatalf(
				"route to peer %s goes via %s, not %s — "+
					"if client runs on the same host, use ./scripts/run-client-netns.sh",
				cfg.TUN.PeerIP, dev, iface.Name(),
			)
		}
		log.Printf("route to peer %s → %s", cfg.TUN.PeerIP, dev)
	}

	if cfg.NAT.Enabled {
		egress, err := nat.Setup(iface.Name(), cfg.NAT.EgressIF, cfg.TUN.PeerIP, cfg.NAT.PermissiveForward)
		if err != nil {
			log.Fatalf("nat: %v", err)
		}
		log.Printf("nat enabled, tun=%s egress=%s masq=%s", iface.Name(), egress, cfg.TUN.PeerIP)
	}

	poolCfg, err := cfg.PoolConfig()
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
	log.Printf("tun %s up, session_id=%d, mode=%s", iface.Name(), cfg.SessionID, mode)

	var packetID atomic.Uint32
	assembler := reasm.NewAssembler()
	errCh := make(chan error, 2)

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
				if h.SessionID != cfg.SessionID {
					continue
				}
				pkt, ok := assembler.Feed(h, payload)
				if !ok {
					continue
				}
				if _, err := iface.Write(pkt); err != nil {
					errCh <- err
					return
				}
				log.Printf("tun write (client→internet) %s", ippkt.LogSummary(pkt))
			}
		}
	}()

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
			log.Printf("tun read (internet→client) %s", ippkt.LogSummary(pkt))
			if err := relayPool.SendPacket(ctx, id, pkt); err != nil {
				log.Printf("send packet: %v", err)
				continue
			}
		}
	}()

	if err := <-errCh; err != nil && err != io.EOF {
		log.Fatalf("run: %v", err)
	}
}
