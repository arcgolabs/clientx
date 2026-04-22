package http

import (
	"time"

	"github.com/arcgolabs/clientx"
)

// Option configures a DefaultClient.
type Option func(*DefaultClient)

// WithRequestMiddleware adds a resty request middleware.
func WithRequestMiddleware(fn func(*resty.Client, *resty.Request) error) Option {
	return func(c *DefaultClient) {
		c.Raw().AddRequestMiddleware(fn)
	}
}

// WithResponseMiddleware adds a resty response middleware.
func WithResponseMiddleware(fn func(*resty.Client, *resty.Response) error) Option {
	return func(c *DefaultClient) {
		c.Raw().AddResponseMiddleware(fn)
	}
}

// WithHeader adds a default request header.
func WithHeader(key, value string) Option {
	return func(c *DefaultClient) {
		c.Raw().SetHeader(key, value)
	}
}

// WithHooks appends client hooks.
func WithHooks(hooks ...clientx.Hook) Option {
	return appendHooks(hooks...)
}

// WithPolicies appends execution policies.
func WithPolicies(policies ...clientx.Policy) Option {
	return appendPolicies(policies...)
}

// WithConcurrencyLimit adds a concurrency limit policy.
func WithConcurrencyLimit(maxInFlight int) Option {
	return appendPolicies(clientx.NewConcurrencyLimitPolicy(maxInFlight))
}

// WithTimeoutGuard adds a timeout guard policy.
func WithTimeoutGuard(timeout time.Duration) Option {
	return appendPolicies(clientx.NewTimeoutPolicy(timeout))
}

func appendHooks(hooks ...clientx.Hook) Option {
	return func(c *DefaultClient) {
		c.hooks = clientx.AppendHooks(c.hooks, hooks...)
	}
}

func appendPolicies(policies ...clientx.Policy) Option {
	return func(c *DefaultClient) {
		c.policies = clientx.AppendPolicies(c.policies, policies...)
	}
}
