package http

import (
	"errors"
	"net/url"
	"strings"
	"time"

	"github.com/DaiYuANg/arcgo/clientx"
	"github.com/DaiYuANg/arcgo/collectionx"
	"github.com/samber/oops"
)

// Config configures the HTTP client implementation.
type Config struct {
	BaseURL   string
	Timeout   time.Duration
	Headers   collectionx.Map[string, string]
	UserAgent string
	Retry     clientx.RetryConfig
	TLS       clientx.TLSConfig
}

const defaultTimeout = 30 * time.Second

// ErrInvalidConfig indicates that the HTTP client configuration is invalid.
var ErrInvalidConfig = errors.New("invalid http client config")

// NormalizeAndValidate normalizes cfg and validates all supported options.
func (cfg Config) NormalizeAndValidate() (Config, error) {
	out := cfg
	out.BaseURL = strings.TrimSpace(out.BaseURL)
	out.UserAgent = strings.TrimSpace(out.UserAgent)

	if out.BaseURL != "" {
		if _, err := url.ParseRequestURI(out.BaseURL); err != nil {
			return Config{}, oops.In("clientx/http").
				With("op", "normalize_validate", "field", "base_url", "base_url", out.BaseURL).
				Wrapf(errors.Join(ErrInvalidConfig, err), "validate http client config")
		}
	}

	if out.Timeout == 0 {
		out.Timeout = defaultTimeout
	}
	if out.Timeout < 0 {
		return Config{}, oops.In("clientx/http").
			With("op", "normalize_validate", "field", "timeout", "timeout", out.Timeout).
			Wrapf(ErrInvalidConfig, "timeout must be >= 0")
	}

	if out.Retry.MaxRetries < 0 {
		return Config{}, oops.In("clientx/http").
			With("op", "normalize_validate", "field", "retry.max_retries", "max_retries", out.Retry.MaxRetries).
			Wrapf(ErrInvalidConfig, "retry.max_retries must be >= 0")
	}
	if out.Retry.WaitMin < 0 || out.Retry.WaitMax < 0 {
		return Config{}, oops.In("clientx/http").
			With("op", "normalize_validate", "field", "retry.wait", "wait_min", out.Retry.WaitMin, "wait_max", out.Retry.WaitMax).
			Wrapf(ErrInvalidConfig, "retry wait durations must be >= 0")
	}
	if out.Retry.WaitMax > 0 && out.Retry.WaitMin > out.Retry.WaitMax {
		return Config{}, oops.In("clientx/http").
			With("op", "normalize_validate", "field", "retry.wait", "wait_min", out.Retry.WaitMin, "wait_max", out.Retry.WaitMax).
			Wrapf(ErrInvalidConfig, "retry.wait_min must be <= retry.wait_max")
	}

	return out, nil
}
