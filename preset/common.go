package preset

import "github.com/DaiYuANg/arcgo/clientx"

func isZeroRetryConfig(cfg clientx.RetryConfig) bool {
	return !cfg.Enabled && cfg.MaxRetries == 0 && cfg.WaitMin == 0 && cfg.WaitMax == 0
}

func hasRetryHint(cfg clientx.RetryConfig) bool {
	return cfg.MaxRetries > 0 || cfg.WaitMin > 0 || cfg.WaitMax > 0
}
