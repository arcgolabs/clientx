package tcp

import (
	"time"

	"github.com/DaiYuANg/arcgo/clientx"
)

// Option configures a DefaultClient.
type Option func(*DefaultClient)

// WithHooks appends client hooks.
func WithHooks(hooks ...clientx.Hook) Option {
	return func(c *DefaultClient) {
		c.hooks = clientx.AppendHooks(c.hooks, hooks...)
	}
}

// WithPolicies appends execution policies.
func WithPolicies(policies ...clientx.Policy) Option {
	return func(c *DefaultClient) {
		c.policies = clientx.AppendPolicies(c.policies, policies...)
	}
}

// WithConcurrencyLimit adds a concurrency limit policy.
func WithConcurrencyLimit(maxInFlight int) Option {
	return func(c *DefaultClient) {
		c.policies = clientx.AppendPolicies(c.policies, clientx.NewConcurrencyLimitPolicy(maxInFlight))
	}
}

// WithTimeoutGuard adds a timeout guard policy.
func WithTimeoutGuard(timeout time.Duration) Option {
	return func(c *DefaultClient) {
		c.policies = clientx.AppendPolicies(c.policies, clientx.NewTimeoutPolicy(timeout))
	}
}
