package udp

import (
	"errors"
	"strings"
	"time"

	"github.com/samber/oops"
)

// Config configures the UDP client implementation.
type Config struct {
	Network      string
	Address      string
	DialTimeout  time.Duration
	ReadTimeout  time.Duration
	WriteTimeout time.Duration
}

const defaultDialTimeout = 5 * time.Second

// ErrInvalidConfig indicates that the UDP client configuration is invalid.
var ErrInvalidConfig = errors.New("invalid udp client config")

// NormalizeAndValidate normalizes cfg and validates all supported options.
func (cfg Config) NormalizeAndValidate() (Config, error) {
	out := cfg
	out.Network = strings.TrimSpace(out.Network)
	out.Address = strings.TrimSpace(out.Address)

	if out.Network == "" {
		out.Network = "udp"
	}
	if out.Address == "" {
		return Config{}, oops.In("clientx/udp").
			With("op", "normalize_validate", "field", "address", "network", out.Network).
			Wrapf(ErrInvalidConfig, "address is required")
	}
	if out.DialTimeout == 0 {
		out.DialTimeout = defaultDialTimeout
	}
	if out.DialTimeout < 0 || out.ReadTimeout < 0 || out.WriteTimeout < 0 {
		return Config{}, oops.In("clientx/udp").
			With(
				"op", "normalize_validate",
				"field", "timeout",
				"network", out.Network,
				"dial_timeout", out.DialTimeout,
				"read_timeout", out.ReadTimeout,
				"write_timeout", out.WriteTimeout,
			).
			Wrapf(ErrInvalidConfig, "timeout values must be >= 0")
	}
	if !strings.HasPrefix(out.Network, "udp") {
		return Config{}, oops.In("clientx/udp").
			With("op", "normalize_validate", "field", "network", "network", out.Network).
			Wrapf(ErrInvalidConfig, "network must be udp/udp4/udp6")
	}

	return out, nil
}
