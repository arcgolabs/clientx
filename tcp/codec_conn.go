package tcp

import (
	"fmt"
	"net"

	"github.com/arcgolabs/clientx"
	clientcodec "github.com/arcgolabs/clientx/codec"
	"github.com/samber/oops"
)

// CodecConn wraps a TCP connection with codec and framer helpers.
type CodecConn struct {
	conn   net.Conn
	codec  clientcodec.Codec
	framer clientcodec.Framer
	addr   string
}

// NewCodecConn wraps conn with codec/framer helpers.
func NewCodecConn(conn net.Conn, codec clientcodec.Codec, framer clientcodec.Framer, addr string) *CodecConn {
	return &CodecConn{
		conn:   conn,
		codec:  codec,
		framer: framer,
		addr:   addr,
	}
}

// Raw returns the underlying net.Conn.
func (c *CodecConn) Raw() net.Conn {
	if c == nil {
		return nil
	}
	return c.conn
}

// Close closes the underlying connection.
func (c *CodecConn) Close() error {
	if c == nil {
		return nil
	}
	if c.conn == nil {
		return nil
	}
	if err := c.conn.Close(); err != nil {
		return oops.In("clientx/tcp").
			With("op", "close_codec_conn", "addr", c.addr, "protocol", clientx.ProtocolTCP).
			Wrapf(err, "close tcp codec conn")
	}
	return nil
}

// WriteValue encodes v and writes it as one framed payload.
func (c *CodecConn) WriteValue(v any) error {
	if c == nil {
		return oops.In("clientx/tcp").
			With("op", "encode", "protocol", clientx.ProtocolTCP).
			New("codec conn is nil")
	}
	if c.conn == nil {
		return oops.In("clientx/tcp").
			With("op", "encode", "addr", c.addr, "protocol", clientx.ProtocolTCP, "stage", "validate_conn").
			New("conn is nil")
	}
	if c.codec == nil {
		return oops.In("clientx/tcp").
			With("op", "encode", "addr", c.addr, "protocol", clientx.ProtocolTCP, "stage", "validate_codec").
			New("codec is nil")
	}
	if c.framer == nil {
		return oops.In("clientx/tcp").
			With("op", "encode", "addr", c.addr, "protocol", clientx.ProtocolTCP, "stage", "validate_framer").
			New("framer is nil")
	}

	payload, err := c.codec.Marshal(v)
	if err != nil {
		return oops.In("clientx/tcp").
			With("op", "encode", "addr", c.addr, "protocol", clientx.ProtocolTCP, "stage", "marshal", "value_type", fmt.Sprintf("%T", v)).
			Wrapf(wrapCodecError("encode", c.addr, err), "encode tcp value")
	}
	if err := c.framer.WriteFrame(c.conn, payload); err != nil {
		return oops.In("clientx/tcp").
			With("op", "write_frame", "addr", c.addr, "protocol", clientx.ProtocolTCP, "payload_size", len(payload)).
			Wrapf(wrapClientError("write_frame", c.addr, err), "write tcp frame")
	}
	return nil
}

// ReadValue reads one framed payload and decodes it into v.
func (c *CodecConn) ReadValue(v any) error {
	if c == nil {
		return oops.In("clientx/tcp").
			With("op", "decode", "protocol", clientx.ProtocolTCP).
			New("codec conn is nil")
	}
	if c.conn == nil {
		return oops.In("clientx/tcp").
			With("op", "decode", "addr", c.addr, "protocol", clientx.ProtocolTCP, "stage", "validate_conn").
			New("conn is nil")
	}
	if c.codec == nil {
		return oops.In("clientx/tcp").
			With("op", "decode", "addr", c.addr, "protocol", clientx.ProtocolTCP, "stage", "validate_codec").
			New("codec is nil")
	}
	if c.framer == nil {
		return oops.In("clientx/tcp").
			With("op", "decode", "addr", c.addr, "protocol", clientx.ProtocolTCP, "stage", "validate_framer").
			New("framer is nil")
	}

	frame, err := c.framer.ReadFrame(c.conn)
	if err != nil {
		return oops.In("clientx/tcp").
			With("op", "read_frame", "addr", c.addr, "protocol", clientx.ProtocolTCP).
			Wrapf(wrapClientError("read_frame", c.addr, err), "read tcp frame")
	}
	if err := c.codec.Unmarshal(frame, v); err != nil {
		return oops.In("clientx/tcp").
			With("op", "decode", "addr", c.addr, "protocol", clientx.ProtocolTCP, "stage", "unmarshal", "target_type", fmt.Sprintf("%T", v), "payload_size", len(frame)).
			Wrapf(wrapCodecError("decode", c.addr, err), "decode tcp value")
	}
	return nil
}
