package preset

import (
	"time"

	"github.com/DaiYuANg/arcgo/clientx"
	clientudp "github.com/DaiYuANg/arcgo/clientx/udp"
	"github.com/DaiYuANg/arcgo/collectionx"
	"github.com/samber/oops"
)

type lowLatencyUDPPreset struct {
	dialTimeout      time.Duration
	readTimeout      time.Duration
	writeTimeout     time.Duration
	timeoutGuard     time.Duration
	concurrencyLimit int
	options          []clientudp.Option
}

// LowLatencyUDPOption configures the NewLowLatencyUDP preset.
type LowLatencyUDPOption func(*lowLatencyUDPPreset)

// WithLowLatencyUDPDialTimeout overrides the preset dial timeout.
func WithLowLatencyUDPDialTimeout(timeout time.Duration) LowLatencyUDPOption {
	return func(p *lowLatencyUDPPreset) {
		p.dialTimeout = timeout
	}
}

// WithLowLatencyUDPReadTimeout overrides the preset read timeout.
func WithLowLatencyUDPReadTimeout(timeout time.Duration) LowLatencyUDPOption {
	return func(p *lowLatencyUDPPreset) {
		p.readTimeout = timeout
	}
}

// WithLowLatencyUDPWriteTimeout overrides the preset write timeout.
func WithLowLatencyUDPWriteTimeout(timeout time.Duration) LowLatencyUDPOption {
	return func(p *lowLatencyUDPPreset) {
		p.writeTimeout = timeout
	}
}

// WithLowLatencyUDPTimeoutGuard adds a timeout guard policy to the preset client.
func WithLowLatencyUDPTimeoutGuard(timeout time.Duration) LowLatencyUDPOption {
	return func(p *lowLatencyUDPPreset) {
		p.timeoutGuard = timeout
	}
}

// WithLowLatencyUDPConcurrencyLimit adds a concurrency limit policy to the preset client.
func WithLowLatencyUDPConcurrencyLimit(maxInFlight int) LowLatencyUDPOption {
	return func(p *lowLatencyUDPPreset) {
		p.concurrencyLimit = maxInFlight
	}
}

// WithLowLatencyUDPOption appends a raw UDP client option to the preset.
func WithLowLatencyUDPOption(opt clientudp.Option) LowLatencyUDPOption {
	return func(p *lowLatencyUDPPreset) {
		p.options = appendPresetOption(p.options, opt)
	}
}

// NewLowLatencyUDP creates a UDP client tuned for low-latency traffic.
func NewLowLatencyUDP(cfg clientudp.Config, opts ...LowLatencyUDPOption) (clientudp.Client, error) {
	preset := defaultLowLatencyUDPPreset()
	clientx.Apply(&preset, opts...)

	tuned := tuneLowLatencyUDPConfig(cfg, preset)
	clientOpts := buildLowLatencyUDPOptions(preset)
	client, err := clientudp.New(tuned, clientOpts...)
	if err != nil {
		return nil, oops.In("clientx/preset").
			With(
				"op", "new_low_latency_udp",
				"protocol", "udp",
				"network", tuned.Network,
				"addr", tuned.Address,
				"dial_timeout", tuned.DialTimeout,
				"read_timeout", tuned.ReadTimeout,
				"write_timeout", tuned.WriteTimeout,
				"option_count", len(clientOpts),
			).
			Wrapf(err, "build low latency udp client")
	}
	return client, nil
}

func defaultLowLatencyUDPPreset() lowLatencyUDPPreset {
	return lowLatencyUDPPreset{
		dialTimeout:      300 * time.Millisecond,
		readTimeout:      150 * time.Millisecond,
		writeTimeout:     150 * time.Millisecond,
		timeoutGuard:     200 * time.Millisecond,
		concurrencyLimit: 1024,
	}
}

func tuneLowLatencyUDPConfig(cfg clientudp.Config, preset lowLatencyUDPPreset) clientudp.Config {
	tuned := cfg
	if tuned.DialTimeout == 0 {
		tuned.DialTimeout = preset.dialTimeout
	}
	if tuned.ReadTimeout == 0 {
		tuned.ReadTimeout = preset.readTimeout
	}
	if tuned.WriteTimeout == 0 {
		tuned.WriteTimeout = preset.writeTimeout
	}
	return tuned
}

func buildLowLatencyUDPOptions(preset lowLatencyUDPPreset) []clientudp.Option {
	clientOpts := collectionx.NewListWithCapacity[clientudp.Option](2 + len(preset.options))
	if preset.timeoutGuard > 0 {
		clientOpts.Add(clientudp.WithTimeoutGuard(preset.timeoutGuard))
	}
	if preset.concurrencyLimit > 0 {
		clientOpts.Add(clientudp.WithConcurrencyLimit(preset.concurrencyLimit))
	}
	clientOpts.Add(preset.options...)
	return clientOpts.Values()
}
