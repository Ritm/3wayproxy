package pool

import (
	"context"

	"github.com/3wayproxy/shared/wsclient"
)

// NativeDialer uses gorilla/websocket from Go (phase 1–2).
type NativeDialer struct {
	Role Role
}

func NewNativeDialer(role Role) *NativeDialer {
	return &NativeDialer{Role: role}
}

func (d *NativeDialer) DialPlayer(ctx context.Context, _ int, ep Endpoint, sessionID uint64, resume bool) (RelayConn, error) {
	c, err := wsclient.DialPlayer(ctx, ep.URL, ep.ShardID, sessionID)
	if err != nil {
		return nil, err
	}
	if resume {
		_ = c.Resume(sessionID, 0, 0)
	}
	return c, nil
}

func (d *NativeDialer) DialSpectator(ctx context.Context, _ int, ep Endpoint, sessionID uint64, _ bool) (RelayConn, error) {
	return wsclient.DialSpectator(ctx, ep.URL, ep.RelayID, sessionID)
}
