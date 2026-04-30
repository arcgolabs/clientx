package preset

import (
	"strings"
	"time"

	"github.com/arcgolabs/clientx"
	clienthttp "github.com/arcgolabs/clientx/http"
	collectionlist "github.com/arcgolabs/collectionx/list"
	"github.com/samber/oops"
)

type edgeHTTPPreset struct {
	timeout          time.Duration
	timeoutGuard     time.Duration
	concurrencyLimit int
	retry            clientx.RetryConfig
	userAgent        string
	disableRetry     bool
	options          []clienthttp.Option
}

// EdgeHTTPOption configures the NewEdgeHTTP preset.
type EdgeHTTPOption func(*edgeHTTPPreset)

// WithEdgeHTTPTimeout overrides the default client timeout.
func WithEdgeHTTPTimeout(timeout time.Duration) EdgeHTTPOption {
	return func(p *edgeHTTPPreset) {
		p.timeout = timeout
	}
}

// WithEdgeHTTPTimeoutGuard adds a timeout guard policy to the preset client.
func WithEdgeHTTPTimeoutGuard(timeout time.Duration) EdgeHTTPOption {
	return func(p *edgeHTTPPreset) {
		p.timeoutGuard = timeout
	}
}

// WithEdgeHTTPConcurrencyLimit adds a concurrency limit policy to the preset client.
func WithEdgeHTTPConcurrencyLimit(maxInFlight int) EdgeHTTPOption {
	return func(p *edgeHTTPPreset) {
		p.concurrencyLimit = maxInFlight
	}
}

// WithEdgeHTTPRetry overrides the preset retry configuration.
func WithEdgeHTTPRetry(cfg clientx.RetryConfig) EdgeHTTPOption {
	return func(p *edgeHTTPPreset) {
		p.retry = cfg
		p.disableRetry = false
	}
}

// WithEdgeHTTPDisableRetry disables preset-managed retries.
func WithEdgeHTTPDisableRetry() EdgeHTTPOption {
	return func(p *edgeHTTPPreset) {
		p.disableRetry = true
	}
}

// WithEdgeHTTPUserAgent overrides the preset user agent.
func WithEdgeHTTPUserAgent(userAgent string) EdgeHTTPOption {
	return func(p *edgeHTTPPreset) {
		p.userAgent = strings.TrimSpace(userAgent)
	}
}

// WithEdgeHTTPOption appends a raw HTTP client option to the preset.
func WithEdgeHTTPOption(opt clienthttp.Option) EdgeHTTPOption {
	return func(p *edgeHTTPPreset) {
		p.options = appendPresetOption(p.options, opt)
	}
}

// NewEdgeHTTP creates an HTTP client tuned for edge-facing traffic.
func NewEdgeHTTP(cfg clienthttp.Config, opts ...EdgeHTTPOption) (clienthttp.Client, error) {
	preset := defaultEdgeHTTPPreset()
	clientx.Apply(&preset, opts...)

	tuned := tuneEdgeHTTPConfig(cfg, preset)
	clientOpts := buildEdgeHTTPOptions(preset)
	client, err := clienthttp.New(tuned, clientOpts...)
	if err != nil {
		return nil, oops.In("clientx/preset").
			With(
				"op", "new_edge_http",
				"protocol", "http",
				"base_url", tuned.BaseURL,
				"timeout", tuned.Timeout,
				"user_agent", tuned.UserAgent,
				"option_count", len(clientOpts),
			).
			Wrapf(err, "build edge http client")
	}
	return client, nil
}

func defaultEdgeHTTPPreset() edgeHTTPPreset {
	return edgeHTTPPreset{
		timeout:          5 * time.Second,
		timeoutGuard:     4 * time.Second,
		concurrencyLimit: 256,
		retry: clientx.RetryConfig{
			Enabled:    true,
			MaxRetries: 2,
			WaitMin:    50 * time.Millisecond,
			WaitMax:    250 * time.Millisecond,
		},
		userAgent: "arcgolabs-clientx/edge-http",
	}
}

func tuneEdgeHTTPConfig(cfg clienthttp.Config, preset edgeHTTPPreset) clienthttp.Config {
	tuned := cfg
	if tuned.Timeout == 0 {
		tuned.Timeout = preset.timeout
	}
	if strings.TrimSpace(tuned.UserAgent) == "" && preset.userAgent != "" {
		tuned.UserAgent = preset.userAgent
	}
	if !preset.disableRetry {
		if isZeroRetryConfig(tuned.Retry) {
			tuned.Retry = preset.retry
		}
		if !tuned.Retry.Enabled && hasRetryHint(tuned.Retry) {
			tuned.Retry.Enabled = true
		}
	}
	return tuned
}

func buildEdgeHTTPOptions(preset edgeHTTPPreset) []clienthttp.Option {
	clientOpts := collectionlist.NewListWithCapacity[clienthttp.Option](2 + len(preset.options))
	if preset.timeoutGuard > 0 {
		clientOpts.Add(clienthttp.WithTimeoutGuard(preset.timeoutGuard))
	}
	if preset.concurrencyLimit > 0 {
		clientOpts.Add(clienthttp.WithConcurrencyLimit(preset.concurrencyLimit))
	}
	clientOpts.Add(preset.options...)
	return clientOpts.Values()
}
