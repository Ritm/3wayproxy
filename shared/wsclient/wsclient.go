package wsclient

import (
	"context"
	"fmt"
	"net/http"
	"sync"

	"github.com/gorilla/websocket"
	"github.com/3wayproxy/shared/proto"
)

const subprotocol = "snake-game-v1"

type Conn struct {
	conn *websocket.Conn
	mu   sync.Mutex
}

func DialPlayer(ctx context.Context, url string, shardID uint16, sessionID uint64) (*Conn, error) {
	c, err := dial(ctx, url)
	if err != nil {
		return nil, err
	}
	if err := c.write(proto.EncodeHandshake(shardID, sessionID, nil)); err != nil {
		_ = c.Close()
		return nil, err
	}
	ack, err := c.read()
	if err != nil {
		_ = c.Close()
		return nil, err
	}
	if proto.FrameType(ack) != proto.TypeHandshakeAck {
		_ = c.Close()
		return nil, fmt.Errorf("expected handshake ack, got %02x", proto.FrameType(ack))
	}
	return c, nil
}

func DialSpectator(ctx context.Context, url string, relayID uint8, sessionID uint64) (*Conn, error) {
	c, err := dial(ctx, url)
	if err != nil {
		return nil, err
	}
	if err := c.write(proto.EncodeHandshakeAgg(relayID, sessionID)); err != nil {
		_ = c.Close()
		return nil, err
	}
	return c, nil
}

func (c *Conn) Resume(sessionID uint64, lastSeqSent, lastSeqRecv uint32) error {
	return c.write(proto.EncodeResume(sessionID, lastSeqSent, lastSeqRecv))
}

func (c *Conn) Send(frame []byte) error {
	return c.write(frame)
}

func (c *Conn) Recv() ([]byte, error) {
	return c.read()
}

func (c *Conn) Close() error {
	if c.conn == nil {
		return nil
	}
	return c.conn.Close()
}

func dial(ctx context.Context, url string) (*Conn, error) {
	dialer := websocket.Dialer{Subprotocols: []string{subprotocol}}
	conn, resp, err := dialer.DialContext(ctx, url, http.Header{})
	if err != nil {
		if resp != nil {
			return nil, fmt.Errorf("ws dial: %w (http %d)", err, resp.StatusCode)
		}
		return nil, fmt.Errorf("ws dial: %w", err)
	}
	return &Conn{conn: conn}, nil
}

func (c *Conn) write(b []byte) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.conn.WriteMessage(websocket.BinaryMessage, b)
}

func (c *Conn) read() ([]byte, error) {
	_, data, err := c.conn.ReadMessage()
	return data, err
}
