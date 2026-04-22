package udp

import (
	"fmt"
	"net"

	"github.com/DaiYuANg/arcgo/clientx"
	clientcodec "github.com/DaiYuANg/arcgo/clientx/codec"
	"github.com/samber/oops"
)

const maxUDPPacketSize = 64 * 1024

// CodecConn wraps a connected UDP socket with codec helpers.
type CodecConn struct {
	conn  net.Conn
	codec clientcodec.Codec
	addr  string
}

// NewCodecConn wraps conn with codec helpers.
func NewCodecConn(conn net.Conn, codec clientcodec.Codec, addr string) *CodecConn {
	return &CodecConn{
		conn:  conn,
		codec: codec,
		addr:  addr,
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
		return oops.In("clientx/udp").
			With("op", "close_codec_conn", "addr", c.addr, "protocol", clientx.ProtocolUDP).
			Wrapf(err, "close udp codec conn")
	}
	return nil
}

// WriteValue encodes v and writes one UDP payload.
func (c *CodecConn) WriteValue(v any) error {
	if c == nil {
		return oops.In("clientx/udp").
			With("op", "encode", "protocol", clientx.ProtocolUDP).
			New("codec conn is nil")
	}
	if c.conn == nil {
		return oops.In("clientx/udp").
			With("op", "encode", "addr", c.addr, "protocol", clientx.ProtocolUDP, "stage", "validate_conn").
			New("conn is nil")
	}
	if c.codec == nil {
		return oops.In("clientx/udp").
			With("op", "encode", "addr", c.addr, "protocol", clientx.ProtocolUDP, "stage", "validate_codec").
			New("codec is nil")
	}
	payload, err := c.codec.Marshal(v)
	if err != nil {
		return oops.In("clientx/udp").
			With("op", "encode", "addr", c.addr, "protocol", clientx.ProtocolUDP, "stage", "marshal", "value_type", fmt.Sprintf("%T", v)).
			Wrapf(wrapCodecError("encode", c.addr, err), "encode udp value")
	}
	if _, err := c.conn.Write(payload); err != nil {
		return oops.In("clientx/udp").
			With("op", "write", "addr", c.addr, "protocol", clientx.ProtocolUDP, "payload_size", len(payload)).
			Wrapf(wrapClientError("write", c.addr, err), "write udp payload")
	}
	return nil
}

// ReadValue reads one UDP payload and decodes it into v.
func (c *CodecConn) ReadValue(v any) error {
	if c == nil {
		return oops.In("clientx/udp").
			With("op", "decode", "protocol", clientx.ProtocolUDP).
			New("codec conn is nil")
	}
	if c.conn == nil {
		return oops.In("clientx/udp").
			With("op", "decode", "addr", c.addr, "protocol", clientx.ProtocolUDP, "stage", "validate_conn").
			New("conn is nil")
	}
	if c.codec == nil {
		return oops.In("clientx/udp").
			With("op", "decode", "addr", c.addr, "protocol", clientx.ProtocolUDP, "stage", "validate_codec").
			New("codec is nil")
	}

	buf := make([]byte, maxUDPPacketSize)
	n, err := c.conn.Read(buf)
	if err != nil {
		return oops.In("clientx/udp").
			With("op", "read", "addr", c.addr, "protocol", clientx.ProtocolUDP).
			Wrapf(wrapClientError("read", c.addr, err), "read udp payload")
	}
	if err := c.codec.Unmarshal(buf[:n], v); err != nil {
		return oops.In("clientx/udp").
			With("op", "decode", "addr", c.addr, "protocol", clientx.ProtocolUDP, "stage", "unmarshal", "target_type", fmt.Sprintf("%T", v), "payload_size", n).
			Wrapf(wrapCodecError("decode", c.addr, err), "decode udp value")
	}
	return nil
}

// CodecPacketConn wraps a packet listener with codec helpers.
type CodecPacketConn struct {
	conn  net.PacketConn
	codec clientcodec.Codec
	addr  string
}

// NewCodecPacketConn wraps conn with packet codec helpers.
func NewCodecPacketConn(conn net.PacketConn, codec clientcodec.Codec, addr string) *CodecPacketConn {
	return &CodecPacketConn{
		conn:  conn,
		codec: codec,
		addr:  addr,
	}
}

// Raw returns the underlying net.PacketConn.
func (c *CodecPacketConn) Raw() net.PacketConn {
	if c == nil {
		return nil
	}
	return c.conn
}

// Close closes the underlying packet connection.
func (c *CodecPacketConn) Close() error {
	if c == nil {
		return nil
	}
	if c.conn == nil {
		return nil
	}
	if err := c.conn.Close(); err != nil {
		return oops.In("clientx/udp").
			With("op", "close_packet_codec_conn", "addr", c.addr, "protocol", clientx.ProtocolUDP).
			Wrapf(err, "close udp packet codec conn")
	}
	return nil
}

// ReadValueFrom reads one packet and decodes it into v.
func (c *CodecPacketConn) ReadValueFrom(v any) (net.Addr, error) {
	if c == nil {
		return nil, oops.In("clientx/udp").
			With("op", "decode", "protocol", clientx.ProtocolUDP).
			New("packet codec conn is nil")
	}
	if c.conn == nil {
		return nil, oops.In("clientx/udp").
			With("op", "decode", "addr", c.addr, "protocol", clientx.ProtocolUDP, "stage", "validate_conn").
			New("packet conn is nil")
	}
	if c.codec == nil {
		return nil, oops.In("clientx/udp").
			With("op", "decode", "addr", c.addr, "protocol", clientx.ProtocolUDP, "stage", "validate_codec").
			New("codec is nil")
	}

	buf := make([]byte, maxUDPPacketSize)
	n, addr, err := c.conn.ReadFrom(buf)
	if err != nil {
		return nil, oops.In("clientx/udp").
			With("op", "read_from", "addr", c.addr, "protocol", clientx.ProtocolUDP).
			Wrapf(wrapClientError("read_from", c.addr, err), "read udp packet")
	}
	if err := c.codec.Unmarshal(buf[:n], v); err != nil {
		return nil, oops.In("clientx/udp").
			With("op", "decode", "addr", c.addr, "protocol", clientx.ProtocolUDP, "stage", "unmarshal", "target_type", fmt.Sprintf("%T", v), "payload_size", n).
			Wrapf(wrapCodecError("decode", c.addr, err), "decode udp packet")
	}
	return addr, nil
}

// WriteValueTo encodes v and writes it to addr.
func (c *CodecPacketConn) WriteValueTo(v any, addr net.Addr) error {
	if c == nil {
		return oops.In("clientx/udp").
			With("op", "encode", "protocol", clientx.ProtocolUDP).
			New("packet codec conn is nil")
	}
	if c.conn == nil {
		return oops.In("clientx/udp").
			With("op", "encode", "addr", c.addr, "protocol", clientx.ProtocolUDP, "stage", "validate_conn").
			New("packet conn is nil")
	}
	if c.codec == nil {
		return oops.In("clientx/udp").
			With("op", "encode", "addr", c.addr, "protocol", clientx.ProtocolUDP, "stage", "validate_codec").
			New("codec is nil")
	}
	payload, err := c.codec.Marshal(v)
	if err != nil {
		return oops.In("clientx/udp").
			With("op", "encode", "addr", c.addr, "protocol", clientx.ProtocolUDP, "stage", "marshal", "value_type", fmt.Sprintf("%T", v)).
			Wrapf(wrapCodecError("encode", c.addr, err), "encode udp packet")
	}

	if _, err := c.conn.WriteTo(payload, addr); err != nil {
		target := c.addr
		if addr != nil {
			target = addr.String()
		}
		return oops.In("clientx/udp").
			With("op", "write_to", "addr", target, "protocol", clientx.ProtocolUDP, "payload_size", len(payload)).
			Wrapf(wrapClientError("write_to", target, err), "write udp packet")
	}
	return nil
}
