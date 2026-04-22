package clientx_test

import (
	"context"
	"log/slog"
	"testing"

	"github.com/DaiYuANg/arcgo/clientx"
)

func TestHookFuncsDispatch(t *testing.T) {
	var dialCalled bool
	var ioCalled bool

	h := clientx.HookFuncs{
		OnDialFunc: func(event clientx.DialEvent) {
			dialCalled = event.Protocol == clientx.ProtocolTCP
		},
		OnIOFunc: func(event clientx.IOEvent) {
			ioCalled = event.Protocol == clientx.ProtocolHTTP
		},
	}

	clientx.EmitDial([]clientx.Hook{h}, clientx.DialEvent{Protocol: clientx.ProtocolTCP})
	clientx.EmitIO([]clientx.Hook{h}, clientx.IOEvent{Protocol: clientx.ProtocolHTTP})

	if !dialCalled {
		t.Fatal("expected dial hook to be called")
	}
	if !ioCalled {
		t.Fatal("expected io hook to be called")
	}
}

func TestEmitHookPanicIsolation(t *testing.T) {
	dialCalled := false
	ioCalled := false

	hooks := []clientx.Hook{
		clientx.HookFuncs{
			OnDialFunc: func(_ clientx.DialEvent) {
				panic("dial hook panic")
			},
			OnIOFunc: func(_ clientx.IOEvent) {
				panic("io hook panic")
			},
		},
		clientx.HookFuncs{
			OnDialFunc: func(_ clientx.DialEvent) {
				dialCalled = true
			},
			OnIOFunc: func(_ clientx.IOEvent) {
				ioCalled = true
			},
		},
	}

	clientx.EmitDial(hooks, clientx.DialEvent{Protocol: clientx.ProtocolTCP})
	clientx.EmitIO(hooks, clientx.IOEvent{Protocol: clientx.ProtocolHTTP})

	if !dialCalled {
		t.Fatal("expected subsequent dial hook to be called after panic")
	}
	if !ioCalled {
		t.Fatal("expected subsequent io hook to be called after panic")
	}
}

type memoryLogHandler struct {
	records []memoryLogRecord
}

type memoryLogRecord struct {
	level   slog.Level
	message string
	attrs   map[string]any
}

func (h *memoryLogHandler) Enabled(context.Context, slog.Level) bool { return true }

func (h *memoryLogHandler) Handle(_ context.Context, record slog.Record) error {
	entry := memoryLogRecord{
		level:   record.Level,
		message: record.Message,
		attrs:   map[string]any{},
	}
	record.Attrs(func(attr slog.Attr) bool {
		entry.attrs[attr.Key] = attr.Value.Any()
		return true
	})
	h.records = append(h.records, entry)
	return nil
}

func (h *memoryLogHandler) WithAttrs([]slog.Attr) slog.Handler { return h }
func (h *memoryLogHandler) WithGroup(string) slog.Handler      { return h }

func TestLoggingHookEmitsDialAndIORecords(t *testing.T) {
	handler := &memoryLogHandler{}
	logger := slog.New(handler)
	hook := clientx.NewLoggingHook(logger)

	clientx.EmitDial([]clientx.Hook{hook}, clientx.DialEvent{Protocol: clientx.ProtocolTCP, Op: "dial", Network: "tcp", Addr: "127.0.0.1:9000"})
	clientx.EmitIO([]clientx.Hook{hook}, clientx.IOEvent{Protocol: clientx.ProtocolHTTP, Op: "get", Addr: "http://example.com", Bytes: 32})

	if len(handler.records) != 2 {
		t.Fatalf("expected 2 log records, got %d", len(handler.records))
	}
	if handler.records[0].message != "clientx dial" {
		t.Fatalf("unexpected dial log message: %s", handler.records[0].message)
	}
	if handler.records[1].message != "clientx io" {
		t.Fatalf("unexpected io log message: %s", handler.records[1].message)
	}
}
