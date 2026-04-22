package preset

import (
	"time"

	"github.com/DaiYuANg/arcgo/collectionx"
	"github.com/arcgolabs/clientx"
	clienttcp "github.com/arcgolabs/clientx/tcp"
	"github.com/samber/oops"
)

type internalRPCPreset struct {
	dialTimeout      time.Duration
	readTimeout      time.Duration
	writeTimeout     time.Duration
	keepAlive        time.Duration
	timeoutGuard     time.Duration
	concurrencyLimit int
	retryPolicy      clientx.RetryPolicyConfig
	disableRetry     bool
	options          []clienttcp.Option
}

// InternalRPCOption configures the NewInternalRPC preset.
type InternalRPCOption func(*internalRPCPreset)

// WithInternalRPCDialTimeout overrides the preset dial timeout.
func WithInternalRPCDialTimeout(timeout time.Duration) InternalRPCOption {
	return func(p *internalRPCPreset) {
		p.dialTimeout = timeout
	}
}

// WithInternalRPCReadTimeout overrides the preset read timeout.
func WithInternalRPCReadTimeout(timeout time.Duration) InternalRPCOption {
	return func(p *internalRPCPreset) {
		p.readTimeout = timeout
	}
}

// WithInternalRPCWriteTimeout overrides the preset write timeout.
func WithInternalRPCWriteTimeout(timeout time.Duration) InternalRPCOption {
	return func(p *internalRPCPreset) {
		p.writeTimeout = timeout
	}
}

// WithInternalRPCKeepAlive overrides the preset keepalive interval.
func WithInternalRPCKeepAlive(keepAlive time.Duration) InternalRPCOption {
	return func(p *internalRPCPreset) {
		p.keepAlive = keepAlive
	}
}

// WithInternalRPCTimeoutGuard adds a timeout guard policy to the preset client.
func WithInternalRPCTimeoutGuard(timeout time.Duration) InternalRPCOption {
	return func(p *internalRPCPreset) {
		p.timeoutGuard = timeout
	}
}

// WithInternalRPCConcurrencyLimit adds a concurrency limit policy to the preset client.
func WithInternalRPCConcurrencyLimit(maxInFlight int) InternalRPCOption {
	return func(p *internalRPCPreset) {
		p.concurrencyLimit = maxInFlight
	}
}

// WithInternalRPCRetryPolicy overrides the preset retry policy.
func WithInternalRPCRetryPolicy(cfg clientx.RetryPolicyConfig) InternalRPCOption {
	return func(p *internalRPCPreset) {
		p.retryPolicy = cfg
		p.disableRetry = false
	}
}

// WithInternalRPCDisableRetry disables preset-managed retries.
func WithInternalRPCDisableRetry() InternalRPCOption {
	return func(p *internalRPCPreset) {
		p.disableRetry = true
	}
}

// WithInternalRPCOption appends a raw TCP client option to the preset.
func WithInternalRPCOption(opt clienttcp.Option) InternalRPCOption {
	return func(p *internalRPCPreset) {
		p.options = appendPresetOption(p.options, opt)
	}
}

// NewInternalRPC creates a TCP client tuned for internal RPC traffic.
func NewInternalRPC(cfg clienttcp.Config, opts ...InternalRPCOption) (clienttcp.Client, error) {
	preset := defaultInternalRPCPreset()
	clientx.Apply(&preset, opts...)

	tuned := tuneInternalRPCConfig(cfg, preset)
	clientOpts := buildInternalRPCOptions(preset)
	client, err := clienttcp.New(tuned, clientOpts...)
	if err != nil {
		return nil, oops.In("clientx/preset").
			With(
				"op", "new_internal_rpc",
				"protocol", "tcp",
				"network", tuned.Network,
				"addr", tuned.Address,
				"dial_timeout", tuned.DialTimeout,
				"read_timeout", tuned.ReadTimeout,
				"write_timeout", tuned.WriteTimeout,
				"option_count", len(clientOpts),
			).
			Wrapf(err, "build internal rpc client")
	}
	return client, nil
}

func defaultInternalRPCPreset() internalRPCPreset {
	return internalRPCPreset{
		dialTimeout:      1500 * time.Millisecond,
		readTimeout:      2 * time.Second,
		writeTimeout:     2 * time.Second,
		keepAlive:        30 * time.Second,
		timeoutGuard:     2 * time.Second,
		concurrencyLimit: 512,
		retryPolicy: clientx.RetryPolicyConfig{
			MaxAttempts: 3,
			BaseDelay:   20 * time.Millisecond,
			MaxDelay:    120 * time.Millisecond,
			Multiplier:  2,
			JitterRatio: 0.2,
		},
	}
}

func tuneInternalRPCConfig(cfg clienttcp.Config, preset internalRPCPreset) clienttcp.Config {
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
	if tuned.KeepAlive == 0 {
		tuned.KeepAlive = preset.keepAlive
	}
	return tuned
}

func buildInternalRPCOptions(preset internalRPCPreset) []clienttcp.Option {
	clientOpts := collectionx.NewListWithCapacity[clienttcp.Option](3 + len(preset.options))
	if preset.timeoutGuard > 0 {
		clientOpts.Add(clienttcp.WithTimeoutGuard(preset.timeoutGuard))
	}
	if preset.concurrencyLimit > 0 {
		clientOpts.Add(clienttcp.WithConcurrencyLimit(preset.concurrencyLimit))
	}
	if !preset.disableRetry {
		clientOpts.Add(clienttcp.WithPolicies(clientx.NewRetryPolicy(preset.retryPolicy)))
	}
	clientOpts.Add(preset.options...)
	return clientOpts.Values()
}
