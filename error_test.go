package clientx_test

import (
	"context"
	"errors"
	"net"
	"os"
	"syscall"
	"testing"

	"github.com/DaiYuANg/arcgo/clientx"
)

type netErrMock struct {
	timeout   bool
	temporary bool
}

func (e netErrMock) Error() string {
	return "net error mock"
}

func (e netErrMock) Timeout() bool {
	return e.timeout
}

func (e netErrMock) Temporary() bool {
	return e.temporary
}

func TestWrapErrorNil(t *testing.T) {
	if got := clientx.WrapError(clientx.ProtocolTCP, "dial", "127.0.0.1:8080", nil); got != nil {
		t.Fatalf("expected nil, got %v", got)
	}
}

func TestWrapErrorCanceled(t *testing.T) {
	err := clientx.WrapError(clientx.ProtocolTCP, "dial", "127.0.0.1:8080", context.Canceled)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !clientx.IsKind(err, clientx.ErrorKindCanceled) {
		t.Fatalf("expected kind %q, got %q", clientx.ErrorKindCanceled, clientx.KindOf(err))
	}
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected wrapped canceled error, got %v", err)
	}
}

func TestWrapErrorConnRefused(t *testing.T) {
	baseErr := &net.OpError{
		Op:  "dial",
		Net: "tcp",
		Err: &os.SyscallError{Err: syscall.ECONNREFUSED},
	}
	err := clientx.WrapError(clientx.ProtocolTCP, "dial", "127.0.0.1:1", baseErr)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !clientx.IsKind(err, clientx.ErrorKindConnRefused) {
		t.Fatalf("expected kind %q, got %q", clientx.ErrorKindConnRefused, clientx.KindOf(err))
	}
}

func TestWrapErrorStillImplementsNetError(t *testing.T) {
	err := clientx.WrapError(clientx.ProtocolUDP, "read", "127.0.0.1:0", netErrMock{timeout: true})
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	netErr, ok := errors.AsType[net.Error](err)
	if !ok {
		t.Fatalf("expected net.Error, got %T", err)
	}
	if !netErr.Timeout() {
		t.Fatal("expected timeout=true")
	}
	if !clientx.IsKind(err, clientx.ErrorKindTimeout) {
		t.Fatalf("expected kind %q, got %q", clientx.ErrorKindTimeout, clientx.KindOf(err))
	}
}

func TestWrapErrorIdempotent(t *testing.T) {
	once := clientx.WrapError(clientx.ProtocolHTTP, "get", "http://127.0.0.1:1", context.DeadlineExceeded)
	twice := clientx.WrapError(clientx.ProtocolHTTP, "get", "http://127.0.0.1:1", once)
	if !errors.Is(twice, once) {
		t.Fatal("expected wrapped error to stay unchanged on second wrap")
	}
}
