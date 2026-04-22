package clientx

import (
	"context"
	"errors"
	"math"
	"math/rand/v2"
	"net"
	"time"
)

// RetryPolicyConfig configures retry behavior for NewRetryPolicy.
type RetryPolicyConfig struct {
	MaxAttempts int
	BaseDelay   time.Duration
	MaxDelay    time.Duration
	Multiplier  float64
	JitterRatio float64
	Retryable   func(error) bool
}

type retryPolicy struct {
	maxAttempts int
	baseDelay   time.Duration
	maxDelay    time.Duration
	multiplier  float64
	jitterRatio float64
	retryable   func(error) bool
}

// NewRetryPolicy creates a retry policy with sane defaults for transient failures.
func NewRetryPolicy(cfg RetryPolicyConfig) Policy {
	maxAttempts := cfg.MaxAttempts
	if maxAttempts <= 0 {
		maxAttempts = 3
	}

	baseDelay := cfg.BaseDelay
	if baseDelay <= 0 {
		baseDelay = 100 * time.Millisecond
	}

	maxDelay := cfg.MaxDelay
	if maxDelay <= 0 {
		maxDelay = 2 * time.Second
	}
	if maxDelay < baseDelay {
		maxDelay = baseDelay
	}

	multiplier := cfg.Multiplier
	if multiplier < 1 {
		multiplier = 2
	}

	jitterRatio := cfg.JitterRatio
	if jitterRatio < 0 {
		jitterRatio = 0
	}
	if jitterRatio > 1 {
		jitterRatio = 1
	}

	retryable := cfg.Retryable
	if retryable == nil {
		retryable = defaultRetryable
	}

	return &retryPolicy{
		maxAttempts: maxAttempts,
		baseDelay:   baseDelay,
		maxDelay:    maxDelay,
		multiplier:  multiplier,
		jitterRatio: jitterRatio,
		retryable:   retryable,
	}
}

func (p *retryPolicy) Before(ctx context.Context, _ Operation) (context.Context, error) {
	return ctx, nil
}

func (p *retryPolicy) After(_ context.Context, _ Operation, _ error) error {
	return nil
}

func (p *retryPolicy) ShouldRetry(ctx context.Context, _ Operation, attempt int, err error) (bool, time.Duration) {
	if err == nil {
		return false, 0
	}
	if p.maxAttempts <= 1 || attempt >= p.maxAttempts {
		return false, 0
	}
	if ctx != nil && ctx.Err() != nil {
		return false, 0
	}
	if !p.retryable(err) {
		return false, 0
	}
	return true, p.backoffDelay(attempt)
}

func (p *retryPolicy) backoffDelay(attempt int) time.Duration {
	if attempt < 1 {
		attempt = 1
	}
	delayFloat := float64(p.baseDelay) * math.Pow(p.multiplier, float64(attempt-1))
	delay := time.Duration(delayFloat)
	if delay < 0 || delay > p.maxDelay {
		delay = p.maxDelay
	}
	if p.jitterRatio <= 0 {
		return delay
	}

	delta := float64(delay) * p.jitterRatio
	minDelay := float64(delay) - delta
	maxDelay := float64(delay) + delta
	if maxDelay <= minDelay {
		return delay
	}
	//nolint:gosec // Retry jitter only needs non-cryptographic randomness.
	jittered := minDelay + rand.Float64()*(maxDelay-minDelay)
	if jittered < 0 {
		jittered = 0
	}
	jitteredDelay := time.Duration(jittered)
	if jitteredDelay > p.maxDelay {
		return p.maxDelay
	}
	return jitteredDelay
}

func defaultRetryable(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, context.Canceled) {
		return false
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return true
	}

	kind := KindOf(err)
	switch kind {
	case ErrorKindTimeout, ErrorKindTemporary, ErrorKindConnRefused, ErrorKindDNS, ErrorKindNetwork:
		return true
	case ErrorKindUnknown, ErrorKindCanceled, ErrorKindClosed, ErrorKindTLS, ErrorKindCodec:
		return false
	}

	if netErr, ok := errors.AsType[net.Error](err); ok {
		return netErr.Timeout()
	}
	return false
}
