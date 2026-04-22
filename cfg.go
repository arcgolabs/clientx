package clientx

import "time"

// RetryConfig configures built-in retry behavior for higher-level clients.
type RetryConfig struct {
	Enabled    bool
	MaxRetries int
	WaitMin    time.Duration
	WaitMax    time.Duration
}

// TLSConfig configures optional TLS transport behavior.
type TLSConfig struct {
	Enabled            bool
	InsecureSkipVerify bool
	ServerName         string
}
