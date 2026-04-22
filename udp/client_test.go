package udp_test

import (
	"context"
	"errors"
	"net"
	"testing"
	"time"

	"github.com/DaiYuANg/arcgo/clientx"
	clientcodec "github.com/DaiYuANg/arcgo/clientx/codec"
	clientudp "github.com/DaiYuANg/arcgo/clientx/udp"
)

func TestDialRoundTrip(t *testing.T) {
	server, err := (&net.ListenConfig{}).ListenPacket(context.Background(), "udp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen udp server failed: %v", err)
	}
	defer closePacketConn(t, server)

	serverErr := make(chan error, 1)
	go func() {
		buf := make([]byte, 128)
		n, addr, readErr := server.ReadFrom(buf)
		if readErr != nil {
			serverErr <- readErr
			return
		}
		_, writeErr := server.WriteTo(append([]byte("ack:"), buf[:n]...), addr)
		serverErr <- writeErr
	}()

	client, err := clientudp.New(clientudp.Config{
		Address:      server.LocalAddr().String(),
		DialTimeout:  time.Second,
		ReadTimeout:  time.Second,
		WriteTimeout: time.Second,
	})
	if err != nil {
		t.Fatalf("new client failed: %v", err)
	}
	defer closeClient(t, client)

	conn, err := client.Dial(context.Background())
	if err != nil {
		t.Fatalf("dial udp failed: %v", err)
	}
	defer closeConn(t, conn)

	_, writeErr := conn.Write([]byte("ping"))
	if writeErr != nil {
		t.Fatalf("write udp failed: %v", writeErr)
	}

	buf := make([]byte, 128)
	n, err := conn.Read(buf)
	if err != nil {
		t.Fatalf("read udp failed: %v", err)
	}
	if got := string(buf[:n]); got != "ack:ping" {
		t.Fatalf("unexpected response: %q", got)
	}

	if err := <-serverErr; err != nil {
		t.Fatalf("server round trip failed: %v", err)
	}
}

func TestListenPacketReadTimeout(t *testing.T) {
	client, err := clientudp.New(clientudp.Config{
		Address:     "127.0.0.1:0",
		ReadTimeout: 40 * time.Millisecond,
	})
	if err != nil {
		t.Fatalf("new client failed: %v", err)
	}
	defer closeClient(t, client)

	conn, err := client.ListenPacket(context.Background())
	if err != nil {
		t.Fatalf("listen packet failed: %v", err)
	}
	defer closePacketConn(t, conn)

	buf := make([]byte, 16)
	start := time.Now()
	_, _, err = conn.ReadFrom(buf)
	if err == nil {
		t.Fatal("expected timeout error, got nil")
	}
	typedErr, ok := errors.AsType[*clientx.Error](err)
	if !ok {
		t.Fatalf("expected *clientx.Error, got %T", err)
	}
	if typedErr.Protocol != clientx.ProtocolUDP {
		t.Fatalf("expected protocol %q, got %q", clientx.ProtocolUDP, typedErr.Protocol)
	}
	if !clientx.IsKind(err, clientx.ErrorKindTimeout) {
		t.Fatalf("expected kind %q, got %q", clientx.ErrorKindTimeout, clientx.KindOf(err))
	}

	netErr, ok := errors.AsType[net.Error](err)
	if !ok || !netErr.Timeout() {
		t.Fatalf("expected net timeout error, got: %v", err)
	}

	if elapsed := time.Since(start); elapsed < 30*time.Millisecond {
		t.Fatalf("timeout returned too quickly: %v", elapsed)
	}
}

func TestDialCodecRoundTrip(t *testing.T) {
	type payload struct {
		Message string `json:"message"`
	}

	serverClient, err := clientudp.New(clientudp.Config{
		Address:      "127.0.0.1:0",
		ReadTimeout:  time.Second,
		WriteTimeout: time.Second,
	})
	if err != nil {
		t.Fatalf("new server client failed: %v", err)
	}
	defer closeClient(t, serverClient)

	server, err := serverClient.ListenPacketCodec(context.Background(), clientcodec.JSON)
	if err != nil {
		t.Fatalf("listen packet codec failed: %v", err)
	}
	defer closeCodecPacketConn(t, server)

	serverErr := make(chan error, 1)
	go func() {
		var req payload
		addr, readErr := server.ReadValueFrom(&req)
		if readErr != nil {
			serverErr <- readErr
			return
		}
		serverErr <- server.WriteValueTo(payload{Message: "ack:" + req.Message}, addr)
	}()

	client, err := clientudp.New(clientudp.Config{
		Address:      server.Raw().LocalAddr().String(),
		DialTimeout:  time.Second,
		ReadTimeout:  time.Second,
		WriteTimeout: time.Second,
	})
	if err != nil {
		t.Fatalf("new client failed: %v", err)
	}
	defer closeClient(t, client)

	codecConn, err := client.DialCodec(context.Background(), clientcodec.JSON)
	if err != nil {
		t.Fatalf("dial codec failed: %v", err)
	}
	defer closeCodecConn(t, codecConn)

	if err := codecConn.WriteValue(payload{Message: "ping"}); err != nil {
		t.Fatalf("write value failed: %v", err)
	}

	var resp payload
	if err := codecConn.ReadValue(&resp); err != nil {
		t.Fatalf("read value failed: %v", err)
	}
	if resp.Message != "ack:ping" {
		t.Fatalf("unexpected response: %+v", resp)
	}

	if err := <-serverErr; err != nil {
		t.Fatalf("server codec failed: %v", err)
	}
}

func TestDialCodecWithNilCodec(t *testing.T) {
	client, err := clientudp.New(clientudp.Config{Address: "127.0.0.1:9000"})
	if err != nil {
		t.Fatalf("new client failed: %v", err)
	}
	defer closeClient(t, client)

	_, err = client.DialCodec(context.Background(), nil)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !clientx.IsKind(err, clientx.ErrorKindCodec) {
		t.Fatalf("expected kind %q, got %q", clientx.ErrorKindCodec, clientx.KindOf(err))
	}
}

func TestListenPacketEmitsIOHook(t *testing.T) {
	var got clientx.IOEvent
	client, err := clientudp.New(
		clientudp.Config{
			Address:     "127.0.0.1:0",
			ReadTimeout: 40 * time.Millisecond,
		},
		clientudp.WithHooks(clientx.HookFuncs{
			OnIOFunc: func(event clientx.IOEvent) {
				got = event
			},
		}),
	)
	if err != nil {
		t.Fatalf("new client failed: %v", err)
	}
	defer closeClient(t, client)

	conn, err := client.ListenPacket(context.Background())
	if err != nil {
		t.Fatalf("listen packet failed: %v", err)
	}
	defer closePacketConn(t, conn)

	buf := make([]byte, 8)
	_, _, err = conn.ReadFrom(buf)
	if err == nil {
		t.Fatal("expected timeout error, got nil")
	}
	if got.Protocol != clientx.ProtocolUDP {
		t.Fatalf("expected protocol %q, got %q", clientx.ProtocolUDP, got.Protocol)
	}
	if got.Op != "read_from" {
		t.Fatalf("expected op read_from, got %q", got.Op)
	}
	if got.Err == nil {
		t.Fatal("expected hook error to be set")
	}
}

func closeClient(t *testing.T, client clientudp.Client) {
	t.Helper()
	if err := client.Close(); err != nil {
		t.Fatalf("close udp client: %v", err)
	}
}

func closeConn(t *testing.T, conn net.Conn) {
	t.Helper()
	if err := conn.Close(); err != nil {
		t.Fatalf("close udp connection: %v", err)
	}
}

func closePacketConn(t *testing.T, conn net.PacketConn) {
	t.Helper()
	if err := conn.Close(); err != nil {
		t.Fatalf("close udp packet connection: %v", err)
	}
}

func closeCodecConn(t *testing.T, conn *clientudp.CodecConn) {
	t.Helper()
	if err := conn.Close(); err != nil {
		t.Fatalf("close udp codec connection: %v", err)
	}
}

func closeCodecPacketConn(t *testing.T, conn *clientudp.CodecPacketConn) {
	t.Helper()
	if err := conn.Close(); err != nil {
		t.Fatalf("close udp codec packet connection: %v", err)
	}
}
