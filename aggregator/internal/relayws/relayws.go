package relayws

import (
	"context"
	"fmt"
	"net/http"
	"sync"

	"github.com/gorilla/websocket"
	"github.com/3wayproxy/shared/proto"
)

const subprotocol = "snake-game-v1"

type Client struct {
	conn *websocket.Conn
	mu   sync.Mutex
}

func DialSpectator(ctx context.Context, url string, relayID uint8, sessionID uint64) (*Client, error) {
	dialer := websocket.Dialer{Subprotocols: []string{subprotocol}}
	conn, resp, err := dialer.DialContext(ctx, url, http.Header{})
	if err != nil {
		if resp != nil {
			return nil, fmt.Errorf("ws dial: %w (http %d)", err, resp.StatusCode)
		}
		return nil, fmt.Errorf("ws dial: %w", err)
	}
	c := &Client{conn: conn}
	if err := c.write(proto.EncodeHandshakeAgg(relayID, sessionID)); err != nil {
		_ = conn.Close()
		return nil, err
	}
	return c, nil
}

func (c *Client) Send(frame []byte) error {
	return c.write(frame)
}

func (c *Client) Recv() ([]byte, error) {
	return c.read()
}

func (c *Client) Close() error {
	return c.conn.Close()
}

func (c *Client) write(b []byte) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.conn.WriteMessage(websocket.BinaryMessage, b)
}

func (c *Client) read() ([]byte, error) {
	_, data, err := c.conn.ReadMessage()
	return data, err
}
