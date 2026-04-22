package udp

import (
	"context"
	"errors"
	"net"
	"time"

	"github.com/arcgolabs/clientx"
	clientcodec "github.com/arcgolabs/clientx/codec"
	"github.com/samber/oops"
)

// DefaultClient is the default UDP client implementation.
type DefaultClient struct {
	cfg      Config
	hooks    []clientx.Hook
	policies []clientx.Policy
}

// New creates a Client from cfg and applies opts.
func New(cfg Config, opts ...Option) (Client, error) {
	normalized, err := cfg.NormalizeAndValidate()
	if err != nil {
		return nil, err
	}

	c := &DefaultClient{cfg: normalized}
	clientx.Apply(c, opts...)
	return c, nil
}

// Close releases resources held by the client.
func (c *DefaultClient) Close() error {
	return nil
}

// Dial establishes a UDP connection using the configured policy chain.
func (c *DefaultClient) Dial(ctx context.Context) (net.Conn, error) {
	network := c.cfg.Network
	operation := clientx.Operation{
		Protocol: clientx.ProtocolUDP,
		Kind:     clientx.OperationKindDial,
		Op:       "dial",
		Network:  network,
		Addr:     c.cfg.Address,
	}

	conn, err := invokeWithDialPolicies(ctx, operation, func(execCtx context.Context) (net.Conn, error) {
		start := time.Now()
		dialer := &net.Dialer{Timeout: c.cfg.DialTimeout}

		conn, err := dialer.DialContext(execCtx, network, c.cfg.Address)
		if err != nil {
			wrappedErr := wrapClientError("dial", c.cfg.Address, err)
			clientx.EmitDial(c.hooks, clientx.DialEvent{
				Protocol: clientx.ProtocolUDP,
				Op:       "dial",
				Network:  network,
				Addr:     c.cfg.Address,
				Duration: time.Since(start),
				Err:      wrappedErr,
			})
			return nil, wrappedErr
		}
		clientx.EmitDial(c.hooks, clientx.DialEvent{
			Protocol: clientx.ProtocolUDP,
			Op:       "dial",
			Network:  network,
			Addr:     c.cfg.Address,
			Duration: time.Since(start),
		})

		return &timeoutConn{
			Conn:         conn,
			readTimeout:  c.cfg.ReadTimeout,
			writeTimeout: c.cfg.WriteTimeout,
			addr:         c.cfg.Address,
			hooks:        c.hooks,
		}, nil
	}, c.policies...)
	if err != nil {
		return nil, err
	}
	return conn, nil
}

// ListenPacket opens a UDP packet listener using the configured policy chain.
func (c *DefaultClient) ListenPacket(ctx context.Context) (net.PacketConn, error) {
	network := c.cfg.Network
	operation := clientx.Operation{
		Protocol: clientx.ProtocolUDP,
		Kind:     clientx.OperationKindListen,
		Op:       "listen",
		Network:  network,
		Addr:     c.cfg.Address,
	}

	packetConn, err := invokeWithListenPolicies(ctx, operation, func(execCtx context.Context) (net.PacketConn, error) {
		start := time.Now()
		lc := &net.ListenConfig{}
		conn, err := lc.ListenPacket(execCtx, network, c.cfg.Address)
		if err != nil {
			wrappedErr := wrapClientError("listen", c.cfg.Address, err)
			clientx.EmitDial(c.hooks, clientx.DialEvent{
				Protocol: clientx.ProtocolUDP,
				Op:       "listen",
				Network:  network,
				Addr:     c.cfg.Address,
				Duration: time.Since(start),
				Err:      wrappedErr,
			})
			return nil, wrappedErr
		}
		clientx.EmitDial(c.hooks, clientx.DialEvent{
			Protocol: clientx.ProtocolUDP,
			Op:       "listen",
			Network:  network,
			Addr:     c.cfg.Address,
			Duration: time.Since(start),
		})

		return &timeoutPacketConn{
			PacketConn:   conn,
			readTimeout:  c.cfg.ReadTimeout,
			writeTimeout: c.cfg.WriteTimeout,
			addr:         c.cfg.Address,
			hooks:        c.hooks,
		}, nil
	}, c.policies...)
	if err != nil {
		return nil, err
	}
	return packetConn, nil
}

// DialCodec establishes a UDP connection wrapped with codec helpers.
func (c *DefaultClient) DialCodec(ctx context.Context, codec clientcodec.Codec) (*CodecConn, error) {
	if codec == nil {
		return nil, wrapCodecError("dial_codec", c.cfg.Address, errors.New("codec is nil"))
	}

	conn, err := c.Dial(ctx)
	if err != nil {
		return nil, err
	}
	return NewCodecConn(conn, codec, c.cfg.Address), nil
}

// ListenPacketCodec opens a packet listener wrapped with codec helpers.
func (c *DefaultClient) ListenPacketCodec(ctx context.Context, codec clientcodec.Codec) (*CodecPacketConn, error) {
	if codec == nil {
		return nil, wrapCodecError("listen_codec", c.cfg.Address, errors.New("codec is nil"))
	}

	packetConn, err := c.ListenPacket(ctx)
	if err != nil {
		return nil, err
	}
	return NewCodecPacketConn(packetConn, codec, c.cfg.Address), nil
}

func invokeWithDialPolicies(
	ctx context.Context,
	operation clientx.Operation,
	fn func(context.Context) (net.Conn, error),
	policies ...clientx.Policy,
) (net.Conn, error) {
	conn, err := clientx.InvokeWithPolicies(ctx, operation, fn, policies...)
	if err != nil {
		return nil, oops.In("clientx/udp").
			With("op", operation.Op, "addr", operation.Addr, "network", operation.Network, "protocol", operation.Protocol, "operation_kind", operation.Kind).
			Wrapf(err, "execute udp dial operation")
	}
	return conn, nil
}

func invokeWithListenPolicies(
	ctx context.Context,
	operation clientx.Operation,
	fn func(context.Context) (net.PacketConn, error),
	policies ...clientx.Policy,
) (net.PacketConn, error) {
	conn, err := clientx.InvokeWithPolicies(ctx, operation, fn, policies...)
	if err != nil {
		return nil, oops.In("clientx/udp").
			With("op", operation.Op, "addr", operation.Addr, "network", operation.Network, "protocol", operation.Protocol, "operation_kind", operation.Kind).
			Wrapf(err, "execute udp listen operation")
	}
	return conn, nil
}

func wrapClientError(op, addr string, err error) error {
	return oops.In("clientx/udp").
		With("op", op, "addr", addr, "protocol", clientx.ProtocolUDP).
		Wrapf(clientx.WrapError(clientx.ProtocolUDP, op, addr, err), "udp %s %s", op, addr)
}

func wrapCodecError(op, addr string, err error) error {
	return oops.In("clientx/udp").
		With("op", op, "addr", addr, "protocol", clientx.ProtocolUDP, "error_kind", clientx.ErrorKindCodec).
		Wrapf(clientx.WrapErrorWithKind(clientx.ProtocolUDP, op, addr, clientx.ErrorKindCodec, err), "udp %s %s", op, addr)
}
