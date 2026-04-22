package tcp

import (
	"errors"
	"strings"
	"time"

	"github.com/arcgolabs/clientx"
	"github.com/samber/oops"
)

// Config configures the TCP client implementation.
type Config struct {
	Network      string
	Address      string
	DialTimeout  time.Duration
	ReadTimeout  time.Duration
	WriteTimeout time.Duration
	KeepAlive    time.Duration
	TLS          clientx.TLSConfig
}

const defaultDialTimeout = 5 * time.Second

// ErrInvalidConfig indicates that the TCP client configuration is invalid.
var ErrInvalidConfig = errors.New("invalid tcp client config")

// NormalizeAndValidate normalizes cfg and validates all supported options.
func (cfg Config) NormalizeAndValidate() (Config, error) {
	out := cfg
	out.Network = strings.TrimSpace(out.Network)
	out.Address = strings.TrimSpace(out.Address)

	if out.Network == "" {
		out.Network = "tcp"
	}
	if out.Address == "" {
		return Config{}, oops.In("clientx/tcp").
			With("op", "normalize_validate", "field", "address", "network", out.Network).
			Wrapf(ErrInvalidConfig, "address is required")
	}
	if out.DialTimeout == 0 {
		out.DialTimeout = defaultDialTimeout
	}
	if out.DialTimeout < 0 || out.ReadTimeout < 0 || out.WriteTimeout < 0 || out.KeepAlive < 0 {
		return Config{}, oops.In("clientx/tcp").
			With(
				"op", "normalize_validate",
				"field", "timeout",
				"network", out.Network,
				"dial_timeout", out.DialTimeout,
				"read_timeout", out.ReadTimeout,
				"write_timeout", out.WriteTimeout,
				"keep_alive", out.KeepAlive,
			).
			Wrapf(ErrInvalidConfig, "timeout values must be >= 0")
	}
	if out.TLS.Enabled && !strings.HasPrefix(out.Network, "tcp") {
		return Config{}, oops.In("clientx/tcp").
			With("op", "normalize_validate", "field", "network", "network", out.Network, "tls_enabled", out.TLS.Enabled).
			Wrapf(ErrInvalidConfig, "tls requires tcp network")
	}

	return out, nil
}
