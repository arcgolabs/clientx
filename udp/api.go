package udp

import (
	"context"

	"github.com/arcgolabs/clientx"
	clientcodec "github.com/arcgolabs/clientx/codec"
)

// Client is the UDP client abstraction exposed by this package.
type Client interface {
	clientx.Closer
	clientx.Dialer
	clientx.PacketListener
	DialCodec(ctx context.Context, codec clientcodec.Codec) (*CodecConn, error)
	ListenPacketCodec(ctx context.Context, codec clientcodec.Codec) (*CodecPacketConn, error)
}

var _ Client = (*DefaultClient)(nil)
var _ clientx.PacketListener = (*DefaultClient)(nil)
