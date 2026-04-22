package clientx

import (
	"context"

	"github.com/samber/oops"
)

type concurrencyLimitPolicy struct {
	sem chan struct{}
}

// NewConcurrencyLimitPolicy limits concurrent executions to maxInFlight.
func NewConcurrencyLimitPolicy(maxInFlight int) Policy {
	if maxInFlight <= 0 {
		maxInFlight = 1
	}
	return &concurrencyLimitPolicy{sem: make(chan struct{}, maxInFlight)}
}

func (p *concurrencyLimitPolicy) Before(ctx context.Context, operation Operation) (context.Context, error) {
	ctx = normalizeContext(ctx)
	if p == nil || p.sem == nil {
		return ctx, oops.In("clientx").
			With("op", "policy_before", "policy", "concurrency_limit", "protocol", operation.Protocol, "operation_kind", operation.Kind, "network", operation.Network, "addr", operation.Addr).
			New("concurrency limit policy is nil")
	}
	select {
	case p.sem <- struct{}{}:
		return ctx, nil
	case <-ctx.Done():
		return ctx, oops.In("clientx").
			With("op", "policy_before", "policy", "concurrency_limit", "protocol", operation.Protocol, "operation_kind", operation.Kind, "network", operation.Network, "addr", operation.Addr, "max_in_flight", cap(p.sem), "in_flight", len(p.sem)).
			Wrapf(ctx.Err(), "acquire concurrency slot")
	}
}

func (p *concurrencyLimitPolicy) After(_ context.Context, _ Operation, _ error) error {
	select {
	case <-p.sem:
	default:
	}
	return nil
}
