package tcp_test

import (
	"context"
	"errors"
	"net"
	"sync"
	"testing"
	"time"

	"github.com/DaiYuANg/arcgo/clientx"
	clientcodec "github.com/DaiYuANg/arcgo/clientx/codec"
	clienttcp "github.com/DaiYuANg/arcgo/clientx/tcp"
	"github.com/samber/lo"
)

func TestDialErrorIsTyped(t *testing.T) {
	client, err := clienttcp.New(clienttcp.Config{
		Address:     "127.0.0.1:1",
		DialTimeout: 150 * time.Millisecond,
	})
	if err != nil {
		t.Fatalf("new client failed: %v", err)
	}
	defer closeClient(t, client)

	_, err = client.Dial(context.Background())
	if err == nil {
		t.Fatal("expected dial error, got nil")
	}

	typedErr, ok := errors.AsType[*clientx.Error](err)
	if !ok {
		t.Fatalf("expected *clientx.Error, got %T", err)
	}
	if typedErr.Protocol != clientx.ProtocolTCP {
		t.Fatalf("expected protocol %q, got %q", clientx.ProtocolTCP, typedErr.Protocol)
	}
	if !lo.Contains([]clientx.ErrorKind{
		clientx.ErrorKindConnRefused,
		clientx.ErrorKindTimeout,
		clientx.ErrorKindNetwork,
	}, typedErr.Kind) {
		t.Fatalf("unexpected error kind: %q", typedErr.Kind)
	}
}

func TestReadTimeoutIsTypedAndStillNetError(t *testing.T) {
	ln, err := (&net.ListenConfig{}).Listen(context.Background(), "tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen tcp failed: %v", err)
	}
	defer closeListener(t, ln)

	done := make(chan struct{})
	var doneOnce sync.Once
	closeDone := func() {
		doneOnce.Do(func() { close(done) })
	}
	go func() {
		serverConn, acceptErr := ln.Accept()
		if acceptErr != nil {
			closeDone()
			return
		}
		defer closeConn(t, serverConn)
		<-done
	}()

	client, err := clienttcp.New(clienttcp.Config{
		Address:     ln.Addr().String(),
		DialTimeout: time.Second,
		ReadTimeout: 40 * time.Millisecond,
	})
	if err != nil {
		t.Fatalf("new client failed: %v", err)
	}
	defer closeClient(t, client)

	conn, err := client.Dial(context.Background())
	if err != nil {
		closeDone()
		t.Fatalf("dial tcp failed: %v", err)
	}
	defer closeConn(t, conn)
	defer closeDone()

	buf := make([]byte, 8)
	_, err = conn.Read(buf)
	if err == nil {
		t.Fatal("expected read timeout error, got nil")
	}
	if !clientx.IsKind(err, clientx.ErrorKindTimeout) {
		t.Fatalf("expected kind %q, got %q", clientx.ErrorKindTimeout, clientx.KindOf(err))
	}

	netErr, ok := errors.AsType[net.Error](err)
	if !ok || !netErr.Timeout() {
		t.Fatalf("expected timeout net.Error, got %v", err)
	}
}

func TestDialCodecRoundTrip(t *testing.T) {
	type payload struct {
		Message string `json:"message"`
	}

	ln, err := (&net.ListenConfig{}).Listen(context.Background(), "tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen tcp failed: %v", err)
	}
	defer closeListener(t, ln)

	serverErr := make(chan error, 1)
	go serveCodecRoundTrip(ln, serverErr)

	client, err := clienttcp.New(clienttcp.Config{
		Address:      ln.Addr().String(),
		DialTimeout:  time.Second,
		ReadTimeout:  time.Second,
		WriteTimeout: time.Second,
	})
	if err != nil {
		t.Fatalf("new client failed: %v", err)
	}
	defer closeClient(t, client)

	cc, err := client.DialCodec(context.Background(), clientcodec.JSON, clientcodec.NewLengthPrefixed(1024))
	if err != nil {
		t.Fatalf("dial codec failed: %v", err)
	}
	defer closeCodecConn(t, cc)

	if err := cc.WriteValue(payload{Message: "ping"}); err != nil {
		t.Fatalf("client write value failed: %v", err)
	}

	var resp payload
	if err := cc.ReadValue(&resp); err != nil {
		t.Fatalf("client read value failed: %v", err)
	}
	if resp.Message != "ack:ping" {
		t.Fatalf("unexpected response: %+v", resp)
	}

	if err := <-serverErr; err != nil {
		t.Fatalf("server failed: %v", err)
	}
}

func TestDialCodecWithNilCodec(t *testing.T) {
	client, err := clienttcp.New(clienttcp.Config{Address: "127.0.0.1:9000"})
	if err != nil {
		t.Fatalf("new client failed: %v", err)
	}
	defer closeClient(t, client)

	_, err = client.DialCodec(context.Background(), nil, clientcodec.NewLengthPrefixed(1024))
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !clientx.IsKind(err, clientx.ErrorKindCodec) {
		t.Fatalf("expected kind %q, got %q", clientx.ErrorKindCodec, clientx.KindOf(err))
	}
}

func TestDialEmitsHookOnError(t *testing.T) {
	var got clientx.DialEvent
	client, err := clienttcp.New(
		clienttcp.Config{
			Address:     "127.0.0.1:1",
			DialTimeout: 100 * time.Millisecond,
		},
		clienttcp.WithHooks(clientx.HookFuncs{
			OnDialFunc: func(event clientx.DialEvent) {
				got = event
			},
		}),
	)
	if err != nil {
		t.Fatalf("new client failed: %v", err)
	}
	defer closeClient(t, client)

	_, err = client.Dial(context.Background())
	if err == nil {
		t.Fatal("expected dial error, got nil")
	}
	if got.Protocol != clientx.ProtocolTCP {
		t.Fatalf("expected protocol %q, got %q", clientx.ProtocolTCP, got.Protocol)
	}
	if got.Op != "dial" {
		t.Fatalf("expected op dial, got %q", got.Op)
	}
	if got.Err == nil {
		t.Fatal("expected hook error to be set")
	}
}

func TestNewWithInvalidConfig(t *testing.T) {
	_, err := clienttcp.New(clienttcp.Config{})
	if err == nil {
		t.Fatal("expected config validation error, got nil")
	}
	if !errors.Is(err, clienttcp.ErrInvalidConfig) {
		t.Fatalf("expected ErrInvalidConfig, got %v", err)
	}
}

func TestDialPolicyBeforeError(t *testing.T) {
	denyErr := errors.New("deny dial")
	client, err := clienttcp.New(
		clienttcp.Config{Address: "127.0.0.1:1"},
		clienttcp.WithPolicies(clientx.PolicyFuncs{
			BeforeFunc: func(ctx context.Context, operation clientx.Operation) (context.Context, error) {
				if operation.Protocol != clientx.ProtocolTCP || operation.Kind != clientx.OperationKindDial {
					t.Fatalf("unexpected operation: %+v", operation)
				}
				return ctx, denyErr
			},
		}),
	)
	if err != nil {
		t.Fatalf("new client failed: %v", err)
	}
	defer closeClient(t, client)

	_, err = client.Dial(context.Background())
	if !errors.Is(err, denyErr) {
		t.Fatalf("expected policy error, got %v", err)
	}
}

func closeClient(t *testing.T, client clienttcp.Client) {
	t.Helper()
	if err := client.Close(); err != nil {
		t.Fatalf("close tcp client: %v", err)
	}
}

func closeListener(t *testing.T, listener net.Listener) {
	t.Helper()
	if err := listener.Close(); err != nil {
		t.Fatalf("close tcp listener: %v", err)
	}
}

func closeConn(t *testing.T, conn net.Conn) {
	t.Helper()
	if err := conn.Close(); err != nil {
		t.Fatalf("close tcp connection: %v", err)
	}
}

func closeCodecConn(t *testing.T, conn *clienttcp.CodecConn) {
	t.Helper()
	if err := conn.Close(); err != nil {
		t.Fatalf("close tcp codec connection: %v", err)
	}
}

func serveCodecRoundTrip(listener net.Listener, serverErr chan<- error) {
	type payload struct {
		Message string `json:"message"`
	}

	conn, acceptErr := listener.Accept()
	if acceptErr != nil {
		serverErr <- acceptErr
		return
	}
	defer func() {
		if closeErr := conn.Close(); closeErr != nil {
			select {
			case serverErr <- closeErr:
			default:
			}
		}
	}()

	codecConn := clienttcp.NewCodecConn(conn, clientcodec.JSON, clientcodec.NewLengthPrefixed(1024), listener.Addr().String())
	var req payload
	readErr := codecConn.ReadValue(&req)
	if readErr != nil {
		serverErr <- readErr
		return
	}
	serverErr <- codecConn.WriteValue(payload{Message: "ack:" + req.Message})
}
