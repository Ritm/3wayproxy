package browser

import (
	"fmt"
	"net/url"
	"strings"
)

// Config controls headless Chromium for the WS carrier.
type Config struct {
	Headless   bool   `yaml:"headless"`
	ProfileDir string `yaml:"profile_dir"`
}

// PlayPageURL builds the relay carrier page URL from a WebSocket endpoint.
func PlayPageURL(wsURL string, shardID uint16, sessionID uint64) (string, error) {
	u, err := url.Parse(wsURL)
	if err != nil {
		return "", err
	}
	scheme := "http"
	if u.Scheme == "wss" {
		scheme = "https"
	}
	if u.Scheme != "ws" && u.Scheme != "wss" {
		return "", fmt.Errorf("browser: not a ws url: %s", wsURL)
	}
	base := fmt.Sprintf("%s://%s", scheme, u.Host)
	q := url.Values{}
	q.Set("ws", wsURL)
	q.Set("shard", fmt.Sprintf("%d", shardID))
	q.Set("session", fmt.Sprintf("%d", sessionID))
	return base + "/play.html?" + q.Encode(), nil
}

// wsHost returns host part for logging.
func wsHost(wsURL string) string {
	u, err := url.Parse(wsURL)
	if err != nil {
		return wsURL
	}
	return strings.TrimPrefix(u.Host, "")
}
