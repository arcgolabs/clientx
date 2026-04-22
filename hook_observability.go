package clientx

import (
	"context"
	"strings"

	"github.com/DaiYuANg/arcgo/collectionx"
	"github.com/DaiYuANg/arcgo/observabilityx"
	"github.com/samber/lo"
)

// ObservabilityHookOption configures NewObservabilityHook behavior.
type ObservabilityHookOption func(*observabilityHookConfig)

type observabilityHookConfig struct {
	metricPrefix        string
	includeAddressAttrs bool
}

// WithHookMetricPrefix overrides the metric prefix used by the hook.
func WithHookMetricPrefix(prefix string) ObservabilityHookOption {
	return func(cfg *observabilityHookConfig) {
		clean := strings.TrimSpace(prefix)
		if clean != "" {
			cfg.metricPrefix = clean
		}
	}
}

// WithHookAddressAttribute controls whether address attributes are attached to emitted metrics.
func WithHookAddressAttribute(enabled bool) ObservabilityHookOption {
	return func(cfg *observabilityHookConfig) {
		cfg.includeAddressAttrs = enabled
	}
}

// NewObservabilityHook creates a Hook that emits dial and I/O metrics.
func NewObservabilityHook(obs observabilityx.Observability, opts ...ObservabilityHookOption) Hook {
	cfg := observabilityHookConfig{
		metricPrefix: "clientx",
	}
	Apply(&cfg, opts...)
	normalized := observabilityx.Normalize(obs, nil)

	return &observabilityHook{
		cfg:            cfg,
		dialTotal:      normalized.Counter(dialTotalSpec(cfg.metricPrefix)),
		dialDurationMS: normalized.Histogram(dialDurationSpec(cfg.metricPrefix)),
		ioTotal:        normalized.Counter(ioTotalSpec(cfg.metricPrefix)),
		ioDurationMS:   normalized.Histogram(ioDurationSpec(cfg.metricPrefix)),
		ioBytesTotal:   normalized.Counter(ioBytesTotalSpec(cfg.metricPrefix)),
	}
}

type observabilityHook struct {
	cfg            observabilityHookConfig
	dialTotal      observabilityx.Counter
	dialDurationMS observabilityx.Histogram
	ioTotal        observabilityx.Counter
	ioDurationMS   observabilityx.Histogram
	ioBytesTotal   observabilityx.Counter
}

func (h *observabilityHook) OnDial(event DialEvent) {
	attrs := collectionx.NewListWithCapacity[observabilityx.Attribute](6,
		observabilityx.String("protocol", string(event.Protocol)),
		observabilityx.String("op", event.Op),
		observabilityx.String("network", event.Network),
		observabilityx.String("result", resultOf(event.Err)),
	)
	if h.cfg.includeAddressAttrs && event.Addr != "" {
		attrs.Add(observabilityx.String("addr", event.Addr))
	}
	if event.Err != nil {
		attrs.Add(observabilityx.String("error_kind", string(KindOf(event.Err))))
	}

	ctx := context.Background()
	h.dialTotal.Add(ctx, 1, attrs.Values()...)
	h.dialDurationMS.Record(ctx, float64(event.Duration.Milliseconds()), attrs.Values()...)
}

func (h *observabilityHook) OnIO(event IOEvent) {
	attrs := collectionx.NewListWithCapacity[observabilityx.Attribute](6,
		observabilityx.String("protocol", string(event.Protocol)),
		observabilityx.String("op", event.Op),
		observabilityx.String("result", resultOf(event.Err)),
	)
	if h.cfg.includeAddressAttrs && event.Addr != "" {
		attrs.Add(observabilityx.String("addr", event.Addr))
	}
	if event.Err != nil {
		attrs.Add(observabilityx.String("error_kind", string(KindOf(event.Err))))
	}

	ctx := context.Background()
	h.ioTotal.Add(ctx, 1, attrs.Values()...)
	h.ioDurationMS.Record(ctx, float64(event.Duration.Milliseconds()), attrs.Values()...)
	if event.Bytes > 0 {
		h.ioBytesTotal.Add(ctx, int64(event.Bytes), attrs.Values()...)
	}
}

func dialTotalSpec(prefix string) observabilityx.CounterSpec {
	return observabilityx.NewCounterSpec(
		prefix+"_dial_total",
		observabilityx.WithDescription("Total number of dial operations."),
		observabilityx.WithLabelKeys("protocol", "op", "network", "result", "addr", "error_kind"),
	)
}

func dialDurationSpec(prefix string) observabilityx.HistogramSpec {
	return observabilityx.NewHistogramSpec(
		prefix+"_dial_duration_ms",
		observabilityx.WithDescription("Duration of dial operations in milliseconds."),
		observabilityx.WithUnit("ms"),
		observabilityx.WithLabelKeys("protocol", "op", "network", "result", "addr", "error_kind"),
	)
}

func ioTotalSpec(prefix string) observabilityx.CounterSpec {
	return observabilityx.NewCounterSpec(
		prefix+"_io_total",
		observabilityx.WithDescription("Total number of client I/O operations."),
		observabilityx.WithLabelKeys("protocol", "op", "result", "addr", "error_kind"),
	)
}

func ioDurationSpec(prefix string) observabilityx.HistogramSpec {
	return observabilityx.NewHistogramSpec(
		prefix+"_io_duration_ms",
		observabilityx.WithDescription("Duration of client I/O operations in milliseconds."),
		observabilityx.WithUnit("ms"),
		observabilityx.WithLabelKeys("protocol", "op", "result", "addr", "error_kind"),
	)
}

func ioBytesTotalSpec(prefix string) observabilityx.CounterSpec {
	return observabilityx.NewCounterSpec(
		prefix+"_io_bytes_total",
		observabilityx.WithDescription("Total number of bytes transferred through client I/O."),
		observabilityx.WithUnit("By"),
		observabilityx.WithLabelKeys("protocol", "op", "result", "addr", "error_kind"),
	)
}

func resultOf(err error) string {
	return lo.Ternary(err == nil, "ok", "error")
}
