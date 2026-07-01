package config

import (
	"fmt"
	"os"
	"strings"
	"time"

	"gopkg.in/yaml.v3"

	"github.com/3wayproxy/shared/pool"
)

type Config struct {
	SessionID uint64        `yaml:"session_id"`
	Session   SessionConfig `yaml:"session"`
	Relays    []RelayConfig `yaml:"relays"`
	Carrier   string        `yaml:"carrier"` // native (default) | browser
	Browser   BrowserConfig `yaml:"browser"`
	// Legacy single-relay (phase 1)
	RelayWS   string `yaml:"relay_ws"`
	ShardID   uint16 `yaml:"shard_id"`
	Fragments int    `yaml:"fragments_per_packet"`
	TUN       TUN    `yaml:"tun"`
}

type BrowserConfig struct {
	Headless   bool   `yaml:"headless"`
	ProfileDir string `yaml:"profile_dir"`
}

type SessionConfig struct {
	RotateIntervalSec int  `yaml:"rotate_interval_sec"`
	ChurnIntervalSec  int  `yaml:"churn_interval_sec"`
	DisconnectIdle    bool `yaml:"disconnect_idle"` // true = рвать WS idle-ноги (DPI); false = стабильнее
}

type RelayConfig struct {
	WS      string `yaml:"ws"`
	ShardID uint16 `yaml:"shard_id"`
}

type TUN struct {
	Name          string   `yaml:"name"`
	LocalIP       string   `yaml:"local_ip"`
	PeerIP        string   `yaml:"peer_ip"`
	MTU           int      `yaml:"mtu"`
	Routes        []string `yaml:"routes"`
	BypassRoutes  []string `yaml:"bypass_routes"`
}

func Load(path string) (*Config, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var c Config
	if err := yaml.Unmarshal(b, &c); err != nil {
		return nil, err
	}
	if c.Fragments < 1 {
		c.Fragments = 1
	}
	if len(c.Relays) >= 3 {
		c.Fragments = 2
	}
	if c.Session.RotateIntervalSec <= 0 {
		c.Session.RotateIntervalSec = 5
	}
	if c.Session.ChurnIntervalSec <= 0 {
		c.Session.ChurnIntervalSec = 3
	}
	if c.TUN.MTU == 0 {
		c.TUN.MTU = 1200
	}
	if c.TUN.LocalIP == "" {
		c.TUN.LocalIP = "10.0.0.2"
	}
	if c.TUN.PeerIP == "" {
		c.TUN.PeerIP = "10.0.0.1"
	}
	if err := c.validate(); err != nil {
		return nil, err
	}
	return &c, nil
}

func (c *Config) validate() error {
	for i, r := range c.Relays {
		if strings.Contains(r.WS, "/ws/spectator") {
			return fmt.Errorf("relays[%d]: client must use /ws/play, not /ws/spectator (spectator is for aggregator only)", i)
		}
	}
	if strings.Contains(c.RelayWS, "/ws/spectator") {
		return fmt.Errorf("relay_ws: client must use /ws/play, not /ws/spectator")
	}
	return nil
}

func (c *Config) PoolConfig(sessionID uint64) (pool.Config, error) {
	eps := c.endpoints()
	if len(eps) == 0 {
		return pool.Config{}, fmt.Errorf("no relays configured")
	}
	churnEvery := time.Duration(c.Session.ChurnIntervalSec) * time.Second
	if c.UseBrowser() {
		// Browser reconnect = new Playwright page (~1s); churn breaks TCP through tunnel.
		churnEvery = 0
	}
	return pool.Config{
		SessionID:      sessionID,
		Endpoints:      eps,
		Dialer:         pool.NewNativeDialer(pool.RolePlayer),
		RotateEvery:    time.Duration(c.Session.RotateIntervalSec) * time.Second,
		ChurnEvery:     churnEvery,
		Fragments:      c.Fragments,
		DisconnectIdle: c.Session.DisconnectIdle,
	}, nil
}

func (c *Config) endpoints() []pool.Endpoint {
	if len(c.Relays) > 0 {
		out := make([]pool.Endpoint, len(c.Relays))
		for i, r := range c.Relays {
			out[i] = pool.Endpoint{
				RelayID: uint8(i),
				ShardID: r.ShardID,
				URL:     r.WS,
			}
			if out[i].ShardID == 0 && r.ShardID == 0 {
				out[i].ShardID = uint16(i)
			}
		}
		return out
	}
	if c.RelayWS != "" {
		return []pool.Endpoint{{
			RelayID: 0,
			ShardID: c.ShardID,
			URL:     c.RelayWS,
		}}
	}
	return nil
}

func (c *Config) MultiRelay() bool {
	return len(c.Relays) >= 3
}

func (c *Config) UseBrowser() bool {
	return c.Carrier == "browser"
}
