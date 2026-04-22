package clientx

import (
	"context"
	"time"
)

type timeoutPolicy struct {
	timeout time.Duration
}

type timeoutCancelKey struct{}

// NewTimeoutPolicy applies a per-operation timeout when the parent context is looser.
func NewTimeoutPolicy(timeout time.Duration) Policy {
	return &timeoutPolicy{timeout: timeout}
}

func (p *timeoutPolicy) Before(ctx context.Context, _ Operation) (context.Context, error) {
	ctx = normalizeContext(ctx)
	if p.timeout <= 0 {
		return ctx, nil
	}

	if deadline, ok := ctx.Deadline(); ok && time.Until(deadline) <= p.timeout {
		return ctx, nil
	}

	nextCtx, cancel := context.WithTimeout(ctx, p.timeout)
	return context.WithValue(nextCtx, timeoutCancelKey{}, cancel), nil
}

func (p *timeoutPolicy) After(ctx context.Context, _ Operation, _ error) error {
	if ctx == nil {
		return nil
	}
	cancel, ok := ctx.Value(timeoutCancelKey{}).(context.CancelFunc)
	if ok && cancel != nil {
		cancel()
	}
	return nil
}
