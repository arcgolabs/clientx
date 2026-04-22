package clientx

import (
	"context"
	"net"
)

// Closer represents a client that can release internal resources.
type Closer interface {
	Close() error
}

// Dialer represents stream-oriented dialing capability.
type Dialer interface {
	Dial(ctx context.Context) (net.Conn, error)
}

// PacketListener represents packet-oriented listening capability.
type PacketListener interface {
	ListenPacket(ctx context.Context) (net.PacketConn, error)
}
