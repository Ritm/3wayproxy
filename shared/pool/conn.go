package pool

import "context"

// RelayConn is a single relay WebSocket session (native or browser-backed).
type RelayConn interface {
	Send(frame []byte) error
	Recv() ([]byte, error)
	Close() error
}

// Dialer opens relay connections for the pool.
type Dialer interface {
	DialPlayer(ctx context.Context, idx int, ep Endpoint, sessionID uint64, resume bool) (RelayConn, error)
	DialSpectator(ctx context.Context, idx int, ep Endpoint, sessionID uint64, resume bool) (RelayConn, error)
}
