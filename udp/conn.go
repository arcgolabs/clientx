package udp

import (
	"net"
	"time"

	"github.com/DaiYuANg/arcgo/clientx"
	"github.com/samber/oops"
)

type timeoutConn struct {
	net.Conn
	readTimeout  time.Duration
	writeTimeout time.Duration
	addr         string
	hooks        []clientx.Hook
}

func (c *timeoutConn) Read(b []byte) (int, error) {
	return c.runIO("read", b, c.readTimeout, c.SetReadDeadline, c.Conn.Read)
}

func (c *timeoutConn) Write(b []byte) (int, error) {
	return c.runIO("write", b, c.writeTimeout, c.SetWriteDeadline, c.Conn.Write)
}

type timeoutPacketConn struct {
	net.PacketConn
	readTimeout  time.Duration
	writeTimeout time.Duration
	addr         string
	hooks        []clientx.Hook
}

func (c *timeoutPacketConn) ReadFrom(b []byte) (int, net.Addr, error) {
	start := time.Now()
	if err := applyDeadline(c.SetReadDeadline, c.readTimeout, "read_from", c.addr); err != nil {
		emitIO("read_from", c.addr, 0, time.Since(start), err, c.hooks)
		return 0, nil, err
	}

	n, addr, err := c.PacketConn.ReadFrom(b)
	if err != nil {
		wrappedErr := wrapClientError("read_from", c.addr, err)
		emitIO("read_from", c.addr, n, time.Since(start), wrappedErr, c.hooks)
		return n, addr, wrappedErr
	}

	emitIO("read_from", c.addr, n, time.Since(start), nil, c.hooks)
	return n, addr, nil
}

func (c *timeoutPacketConn) WriteTo(b []byte, addr net.Addr) (int, error) {
	target := targetAddr(c.addr, addr)
	start := time.Now()
	if err := applyDeadline(c.SetWriteDeadline, c.writeTimeout, "write_to", target); err != nil {
		emitIO("write_to", target, 0, time.Since(start), err, c.hooks)
		return 0, err
	}

	n, err := c.PacketConn.WriteTo(b, addr)
	if err != nil {
		wrappedErr := wrapClientError("write_to", target, err)
		emitIO("write_to", target, n, time.Since(start), wrappedErr, c.hooks)
		return n, wrappedErr
	}
	emitIO("write_to", target, n, time.Since(start), nil, c.hooks)
	return n, nil
}

func (c *timeoutConn) runIO(
	op string,
	data []byte,
	timeout time.Duration,
	setDeadline func(time.Time) error,
	run func([]byte) (int, error),
) (int, error) {
	start := time.Now()
	if err := applyDeadline(setDeadline, timeout, op, c.addr); err != nil {
		emitIO(op, c.addr, 0, time.Since(start), err, c.hooks)
		return 0, err
	}

	n, err := run(data)
	if err != nil {
		wrappedErr := wrapClientError(op, c.addr, err)
		emitIO(op, c.addr, n, time.Since(start), wrappedErr, c.hooks)
		return n, wrappedErr
	}

	emitIO(op, c.addr, n, time.Since(start), nil, c.hooks)
	return n, nil
}

func applyDeadline(setDeadline func(time.Time) error, timeout time.Duration, op, addr string) error {
	if timeout <= 0 {
		return nil
	}
	if err := setDeadline(time.Now().Add(timeout)); err != nil {
		return oops.In("clientx/udp").
			With("op", op, "addr", addr, "protocol", clientx.ProtocolUDP, "stage", "set_deadline", "timeout", timeout).
			Wrapf(err, "set udp deadline")
	}
	return nil
}

func emitIO(op, addr string, bytes int, duration time.Duration, err error, hooks []clientx.Hook) {
	clientx.EmitIO(hooks, clientx.IOEvent{
		Protocol: clientx.ProtocolUDP,
		Op:       op,
		Addr:     addr,
		Bytes:    bytes,
		Duration: duration,
		Err:      err,
	})
}

func targetAddr(fallback string, addr net.Addr) string {
	if addr == nil {
		return fallback
	}
	return addr.String()
}
