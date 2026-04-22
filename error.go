package clientx

import (
	"context"
	"errors"
	"fmt"
	"net"
	"os"
	"strings"
	"syscall"

	"github.com/samber/lo"
)

// Protocol identifies the transport used by a client operation.
type Protocol string

const (
	// ProtocolUnknown indicates that the transport is not known.
	ProtocolUnknown Protocol = "unknown"
	// ProtocolHTTP identifies HTTP client operations.
	ProtocolHTTP Protocol = "http"
	// ProtocolTCP identifies TCP client operations.
	ProtocolTCP Protocol = "tcp"
	// ProtocolUDP identifies UDP client operations.
	ProtocolUDP Protocol = "udp"
)

// ErrorKind classifies client errors into portable categories.
type ErrorKind string

const (
	// ErrorKindUnknown indicates that the error could not be classified.
	ErrorKindUnknown ErrorKind = "unknown"
	// ErrorKindCanceled indicates that the operation was canceled.
	ErrorKindCanceled ErrorKind = "canceled"
	// ErrorKindTimeout indicates that the operation exceeded a deadline.
	ErrorKindTimeout ErrorKind = "timeout"
	// ErrorKindTemporary indicates that the error may be transient.
	ErrorKindTemporary ErrorKind = "temporary"
	// ErrorKindConnRefused indicates that the remote endpoint refused the connection.
	ErrorKindConnRefused ErrorKind = "conn_refused"
	// ErrorKindDNS indicates that DNS resolution failed.
	ErrorKindDNS ErrorKind = "dns"
	// ErrorKindTLS indicates that TLS negotiation or certificate validation failed.
	ErrorKindTLS ErrorKind = "tls"
	// ErrorKindClosed indicates that the connection or resource is already closed.
	ErrorKindClosed ErrorKind = "closed"
	// ErrorKindNetwork indicates a generic network transport failure.
	ErrorKindNetwork ErrorKind = "network"
	// ErrorKindCodec indicates encoding or decoding failures.
	ErrorKindCodec ErrorKind = "codec"
)

// Error enriches transport errors with protocol, operation, and classification metadata.
type Error struct {
	Protocol Protocol
	Op       string
	Addr     string
	Kind     ErrorKind
	Err      error
}

// Error renders the enriched client error message.
func (e *Error) Error() string {
	if e == nil {
		return ""
	}
	if e.Err == nil {
		return fmt.Sprintf("%s %s %s (%s)", e.Protocol, e.Op, e.Addr, e.Kind)
	}
	if e.Addr != "" {
		return fmt.Sprintf("%s %s %s (%s): %v", e.Protocol, e.Op, e.Addr, e.Kind, e.Err)
	}
	return fmt.Sprintf("%s %s (%s): %v", e.Protocol, e.Op, e.Kind, e.Err)
}

// Unwrap returns the underlying error.
func (e *Error) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.Err
}

// Timeout reports whether the error should be treated as a timeout.
func (e *Error) Timeout() bool {
	if e == nil {
		return false
	}
	if e.Kind == ErrorKindTimeout {
		return true
	}
	netErr, ok := errors.AsType[net.Error](e.Err)
	return ok && netErr.Timeout()
}

// Temporary reports whether the error is marked as temporary.
func (e *Error) Temporary() bool {
	if e == nil {
		return false
	}
	if e.Kind == ErrorKindTemporary {
		return true
	}
	// net.Error.Temporary() 已弃用，这里仅检查 Kind 标记
	return false
}

// WrapError wraps err using the inferred ErrorKind.
func WrapError(protocol Protocol, op, addr string, err error) error {
	return WrapErrorWithKind(protocol, op, addr, "", err)
}

// WrapErrorWithKind wraps err with the supplied or inferred ErrorKind.
func WrapErrorWithKind(protocol Protocol, op, addr string, kind ErrorKind, err error) error {
	if err == nil {
		return nil
	}
	if existing, ok := errors.AsType[*Error](err); ok && existing != nil {
		return err
	}
	if protocol == "" {
		protocol = ProtocolUnknown
	}
	return &Error{
		Protocol: protocol,
		Op:       op,
		Addr:     addr,
		Kind:     lo.Ternary(kind != "", kind, classifyErrorKind(err)),
		Err:      err,
	}
}

// IsKind reports whether err is a *Error with the given kind.
func IsKind(err error, kind ErrorKind) bool {
	e, ok := errors.AsType[*Error](err)
	if !ok {
		return false
	}
	return e.Kind == kind
}

// KindOf returns the ErrorKind carried by err when available.
func KindOf(err error) ErrorKind {
	e, ok := errors.AsType[*Error](err)
	if !ok {
		return ErrorKindUnknown
	}
	return e.Kind
}

func classifyErrorKind(err error) ErrorKind {
	if err == nil {
		return ErrorKindUnknown
	}
	if kind, ok := classifyContextError(err); ok {
		return kind
	}
	if kind, ok := classifyClosedError(err); ok {
		return kind
	}
	if kind, ok := classifyTypedNetworkError(err); ok {
		return kind
	}
	return classifyMessageError(err)
}

func isConnRefused(err error) bool {
	if errors.Is(err, syscall.ECONNREFUSED) {
		return true
	}
	if errno, ok := errors.AsType[syscall.Errno](err); ok {
		return lo.Contains([]syscall.Errno{syscall.ECONNREFUSED, syscall.Errno(10061)}, errno)
	}
	if sysErr, ok := errors.AsType[*os.SyscallError](err); ok {
		return isConnRefused(sysErr.Err)
	}
	return false
}

func classifyContextError(err error) (ErrorKind, bool) {
	switch {
	case errors.Is(err, context.Canceled):
		return ErrorKindCanceled, true
	case errors.Is(err, context.DeadlineExceeded):
		return ErrorKindTimeout, true
	default:
		return "", false
	}
}

func classifyClosedError(err error) (ErrorKind, bool) {
	if errors.Is(err, net.ErrClosed) || errors.Is(err, os.ErrClosed) {
		return ErrorKindClosed, true
	}
	return "", false
}

func classifyTypedNetworkError(err error) (ErrorKind, bool) {
	var kind ErrorKind
	_, ok := lo.Find([]func(error) (ErrorKind, bool){
		classifyDNSError,
		classifyTimeoutNetworkError,
		classifyOpNetworkError,
		classifyConnRefusedError,
		classifyGenericNetworkError,
	}, func(classify func(error) (ErrorKind, bool)) bool {
		var matched bool
		kind, matched = classify(err)
		return matched
	})
	return kind, ok
}

func classifyDNSError(err error) (ErrorKind, bool) {
	if dnsErr, ok := errors.AsType[*net.DNSError](err); ok && dnsErr != nil {
		return ErrorKindDNS, true
	}
	return "", false
}

func classifyTimeoutNetworkError(err error) (ErrorKind, bool) {
	if netErr, ok := errors.AsType[net.Error](err); ok && netErr.Timeout() {
		return ErrorKindTimeout, true
	}
	return "", false
}

func classifyOpNetworkError(err error) (ErrorKind, bool) {
	opErr, ok := errors.AsType[*net.OpError](err)
	if !ok || opErr == nil {
		return "", false
	}
	if opErr.Err != nil && isConnRefused(opErr.Err) {
		return ErrorKindConnRefused, true
	}
	return ErrorKindNetwork, true
}

func classifyConnRefusedError(err error) (ErrorKind, bool) {
	if isConnRefused(err) {
		return ErrorKindConnRefused, true
	}
	return "", false
}

func classifyGenericNetworkError(err error) (ErrorKind, bool) {
	if netErr, ok := errors.AsType[net.Error](err); ok && netErr != nil {
		return ErrorKindNetwork, true
	}
	return "", false
}

func classifyMessageError(err error) ErrorKind {
	msg := strings.ToLower(err.Error())
	switch {
	case strings.Contains(msg, "tls"), strings.Contains(msg, "x509"), strings.Contains(msg, "certificate"):
		return ErrorKindTLS
	case strings.Contains(msg, "use of closed network connection"), strings.Contains(msg, "file already closed"):
		return ErrorKindClosed
	case strings.Contains(msg, "network"):
		return ErrorKindNetwork
	default:
		return ErrorKindUnknown
	}
}
