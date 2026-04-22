package tcp

import (
	"context"

	"github.com/arcgolabs/clientx"
	clientcodec "github.com/arcgolabs/clientx/codec"
)

// Client is the TCP client abstraction exposed by this package.
type Client interface {
	clientx.Closer
	clientx.Dialer
	DialCodec(ctx context.Context, codec clientcodec.Codec, framer clientcodec.Framer) (*CodecConn, error)
}

var _ Client = (*DefaultClient)(nil)
var _ clientx.Dialer = (*DefaultClient)(nil)
