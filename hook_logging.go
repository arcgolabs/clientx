package clientx

import (
	"log/slog"

	"github.com/arcgolabs/collectionx"
)

// LoggingHookOption configures NewLoggingHook behavior.
type LoggingHookOption func(*loggingHookConfig)

type loggingHookConfig struct {
	includeAddress bool
}

// WithLoggingHookAddress controls whether emitted logs include target addresses.
func WithLoggingHookAddress(enabled bool) LoggingHookOption {
	return func(cfg *loggingHookConfig) {
		cfg.includeAddress = enabled
	}
}

// NewLoggingHook creates a Hook that logs dial and I/O events with slog.
func NewLoggingHook(logger *slog.Logger, opts ...LoggingHookOption) Hook {
	cfg := loggingHookConfig{includeAddress: true}
	Apply(&cfg, opts...)
	if logger == nil {
		logger = slog.Default()
	}
	return &loggingHook{logger: logger, cfg: cfg}
}

type loggingHook struct {
	logger *slog.Logger
	cfg    loggingHookConfig
}

func (h *loggingHook) OnDial(event DialEvent) {
	if h == nil || h.logger == nil {
		return
	}
	attrs := collectionx.NewListWithCapacity[any](10,
		"protocol", event.Protocol,
		"op", event.Op,
		"network", event.Network,
		"duration", event.Duration,
	)
	if h.cfg.includeAddress && event.Addr != "" {
		attrs.Add("addr", event.Addr)
	}
	if event.Err != nil {
		attrs.Add("error", event.Err, "error_kind", KindOf(event.Err))
		h.logger.Error("clientx dial", attrs.Values()...)
		return
	}
	h.logger.Debug("clientx dial", attrs.Values()...)
}

func (h *loggingHook) OnIO(event IOEvent) {
	if h == nil || h.logger == nil {
		return
	}
	attrs := collectionx.NewListWithCapacity[any](10,
		"protocol", event.Protocol,
		"op", event.Op,
		"bytes", event.Bytes,
		"duration", event.Duration,
	)
	if h.cfg.includeAddress && event.Addr != "" {
		attrs.Add("addr", event.Addr)
	}
	if event.Err != nil {
		attrs.Add("error", event.Err, "error_kind", KindOf(event.Err))
		h.logger.Error("clientx io", attrs.Values()...)
		return
	}
	h.logger.Debug("clientx io", attrs.Values()...)
}
