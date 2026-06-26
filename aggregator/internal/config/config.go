package config

import (
	"fmt"
	"os"
	"time"

	"gopkg.in/yaml.v3"

	"github.com/3wayproxy/shared/pool"
)

type Config struct {
	SessionID uint64        `yaml:"session_id"`
	Session   SessionConfig `yaml:"session"`
	Relays    []RelayConfig `yaml:"relays"`
	// Legacy single-relay
	RelayID   uint8  `yaml:"relay_id"`
	RelayWS   string `yaml:"relay_ws"`
	Fragments int    `yaml:"fragments_per_packet"`
	TUN       TUN    `yaml:"tun"`
	NAT       NAT    `yaml:"nat"`
}

type SessionConfig struct {
	RotateIntervalSec int  `yaml:"rotate_interval_sec"`
	ChurnIntervalSec  int  `yaml:"churn_interval_sec"`
	DisconnectIdle    bool `yaml:"disconnect_idle"`
}

type RelayConfig struct {
	ID uint8  `yaml:"id"`
	WS string `yaml:"ws"`
}

type TUN struct {
	Name    string `yaml:"name"`
	LocalIP string `yaml:"local_ip"`
	PeerIP  string `yaml:"peer_ip"`
	MTU     int    `yaml:"mtu"`
}

type NAT struct {
	Enabled           bool   `yaml:"enabled"`
	EgressIF          string `yaml:"egress_interface"`
	PermissiveForward bool   `yaml:"permissive_forward"`
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
		c.TUN.LocalIP = "10.0.0.1"
	}
	if c.TUN.PeerIP == "" {
		c.TUN.PeerIP = "10.0.0.2"
	}
	return &c, nil
}

func (c *Config) PoolConfig() (pool.Config, error) {
	eps := c.endpoints()
	if len(eps) == 0 {
		return pool.Config{}, fmt.Errorf("no relays configured")
	}
	return pool.Config{
		SessionID:      c.SessionID,
		Endpoints:      eps,
		Role:           pool.RoleSpectator,
		RotateEvery:    time.Duration(c.Session.RotateIntervalSec) * time.Second,
		ChurnEvery:     time.Duration(c.Session.ChurnIntervalSec) * time.Second,
		Fragments:      c.Fragments,
		DisconnectIdle: c.Session.DisconnectIdle,
	}, nil
}

func (c *Config) endpoints() []pool.Endpoint {
	if len(c.Relays) > 0 {
		out := make([]pool.Endpoint, len(c.Relays))
		for i, r := range c.Relays {
			id := r.ID
			if id == 0 && i > 0 {
				id = uint8(i)
			}
			out[i] = pool.Endpoint{
				RelayID: id,
				URL:     r.WS,
			}
		}
		return out
	}
	if c.RelayWS != "" {
		return []pool.Endpoint{{
			RelayID: c.RelayID,
			URL:     c.RelayWS,
		}}
	}
	return nil
}

func (c *Config) MultiRelay() bool {
	return len(c.Relays) >= 3
}
