package tcp

import (
	"context"
	"crypto/tls"
	"errors"
	"net"
	"time"

	"github.com/arcgolabs/clientx"
	clientcodec "github.com/arcgolabs/clientx/codec"
	"github.com/samber/oops"
)

// DefaultClient is the default TCP client implementation.
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

// Dial establishes a TCP connection using the configured policy chain.
func (c *DefaultClient) Dial(ctx context.Context) (net.Conn, error) {
	network := c.cfg.Network
	operation := clientx.Operation{
		Protocol: clientx.ProtocolTCP,
		Kind:     clientx.OperationKindDial,
		Op:       "dial",
		Network:  network,
		Addr:     c.cfg.Address,
	}

	conn, err := invokeWithPolicies(ctx, operation, func(execCtx context.Context) (net.Conn, error) {
		start := time.Now()
		dialer := &net.Dialer{
			Timeout:   c.cfg.DialTimeout,
			KeepAlive: c.cfg.KeepAlive,
		}

		if c.cfg.TLS.Enabled {
			tlsDialer := &tls.Dialer{
				NetDialer: dialer,
				Config: &tls.Config{
					//nolint:gosec // This client must support explicitly configured insecure TLS for development and controlled environments.
					InsecureSkipVerify: c.cfg.TLS.InsecureSkipVerify,
					ServerName:         c.cfg.TLS.ServerName,
				},
			}
			conn, err := tlsDialer.DialContext(execCtx, network, c.cfg.Address)
			if err != nil {
				wrappedErr := wrapClientError("dial", c.cfg.Address, err)
				clientx.EmitDial(c.hooks, clientx.DialEvent{
					Protocol: clientx.ProtocolTCP,
					Op:       "dial",
					Network:  network,
					Addr:     c.cfg.Address,
					Duration: time.Since(start),
					Err:      wrappedErr,
				})
				return nil, wrappedErr
			}
			clientx.EmitDial(c.hooks, clientx.DialEvent{
				Protocol: clientx.ProtocolTCP,
				Op:       "dial",
				Network:  network,
				Addr:     c.cfg.Address,
				Duration: time.Since(start),
			})
			return wrapConn(conn, c.cfg, c.hooks), nil
		}

		conn, err := dialer.DialContext(execCtx, network, c.cfg.Address)
		if err != nil {
			wrappedErr := wrapClientError("dial", c.cfg.Address, err)
			clientx.EmitDial(c.hooks, clientx.DialEvent{
				Protocol: clientx.ProtocolTCP,
				Op:       "dial",
				Network:  network,
				Addr:     c.cfg.Address,
				Duration: time.Since(start),
				Err:      wrappedErr,
			})
			return nil, wrappedErr
		}
		clientx.EmitDial(c.hooks, clientx.DialEvent{
			Protocol: clientx.ProtocolTCP,
			Op:       "dial",
			Network:  network,
			Addr:     c.cfg.Address,
			Duration: time.Since(start),
		})
		return wrapConn(conn, c.cfg, c.hooks), nil
	}, c.policies...)
	if err != nil {
		return nil, err
	}
	return conn, nil
}

// DialCodec establishes a TCP connection wrapped with codec/framer helpers.
func (c *DefaultClient) DialCodec(ctx context.Context, codec clientcodec.Codec, framer clientcodec.Framer) (*CodecConn, error) {
	if codec == nil {
		return nil, wrapCodecError("dial_codec", c.cfg.Address, errors.New("codec is nil"))
	}
	if framer == nil {
		return nil, wrapCodecError("dial_codec", c.cfg.Address, errors.New("framer is nil"))
	}

	conn, err := c.Dial(ctx)
	if err != nil {
		return nil, err
	}
	return NewCodecConn(conn, codec, framer, c.cfg.Address), nil
}

func wrapConn(conn net.Conn, cfg Config, hooks []clientx.Hook) net.Conn {
	return &timeoutConn{
		Conn:         conn,
		readTimeout:  cfg.ReadTimeout,
		writeTimeout: cfg.WriteTimeout,
		addr:         cfg.Address,
		hooks:        hooks,
	}
}

func invokeWithPolicies(
	ctx context.Context,
	operation clientx.Operation,
	fn func(context.Context) (net.Conn, error),
	policies ...clientx.Policy,
) (net.Conn, error) {
	conn, err := clientx.InvokeWithPolicies(ctx, operation, fn, policies...)
	if err != nil {
		return nil, oops.In("clientx/tcp").
			With("op", operation.Op, "addr", operation.Addr, "network", operation.Network, "protocol", operation.Protocol, "operation_kind", operation.Kind).
			Wrapf(err, "execute tcp operation")
	}
	return conn, nil
}

func wrapClientError(op, addr string, err error) error {
	return oops.In("clientx/tcp").
		With("op", op, "addr", addr, "protocol", clientx.ProtocolTCP).
		Wrapf(clientx.WrapError(clientx.ProtocolTCP, op, addr, err), "tcp %s %s", op, addr)
}

func wrapCodecError(op, addr string, err error) error {
	return oops.In("clientx/tcp").
		With("op", op, "addr", addr, "protocol", clientx.ProtocolTCP, "error_kind", clientx.ErrorKindCodec).
		Wrapf(clientx.WrapErrorWithKind(clientx.ProtocolTCP, op, addr, clientx.ErrorKindCodec, err), "tcp %s %s", op, addr)
}
