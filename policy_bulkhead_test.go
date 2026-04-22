package clientx_test

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/arcgolabs/clientx"
)

func TestConcurrencyLimitPolicySerialize(t *testing.T) {
	policy := clientx.NewConcurrencyLimitPolicy(1)
	var active atomic.Int32
	var maxActive atomic.Int32
	fn := trackedPolicyFunc(&active, &maxActive)

	var wg sync.WaitGroup
	wg.Add(2)
	for range 2 {
		go func() {
			defer wg.Done()
			_, err := clientx.InvokeWithPolicies(
				context.Background(),
				clientx.Operation{Protocol: clientx.ProtocolHTTP, Kind: clientx.OperationKindRequest, Op: "get"},
				fn,
				policy,
			)
			if err != nil {
				t.Errorf("invoke failed: %v", err)
			}
		}()
	}
	wg.Wait()

	if got := maxActive.Load(); got != 1 {
		t.Fatalf("expected max concurrency 1, got %d", got)
	}
}

func TestConcurrencyLimitPolicyRespectsContextCancel(t *testing.T) {
	policy := clientx.NewConcurrencyLimitPolicy(1)
	op := clientx.Operation{Protocol: clientx.ProtocolTCP, Kind: clientx.OperationKindDial, Op: "dial"}

	ctx := context.Background()
	_, err := policy.Before(ctx, op)
	if err != nil {
		t.Fatalf("first acquire failed: %v", err)
	}

	timeoutCtx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()
	_, err = policy.Before(timeoutCtx, op)
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("expected deadline exceeded, got %v", err)
	}

	if err := policy.After(context.Background(), op, nil); err != nil {
		t.Fatalf("release failed: %v", err)
	}
}

func TestConcurrencyLimitPolicyReleaseAfterError(t *testing.T) {
	policy := clientx.NewConcurrencyLimitPolicy(1)
	boom := errors.New("boom")

	for i := range 2 {
		_, err := clientx.InvokeWithPolicies(
			context.Background(),
			clientx.Operation{Protocol: clientx.ProtocolUDP, Kind: clientx.OperationKindDial, Op: "dial"},
			func(_ context.Context) (int, error) {
				return 0, boom
			},
			policy,
		)
		if !errors.Is(err, boom) {
			t.Fatalf("round %d expected boom error, got %v", i, err)
		}
	}
}

func trackedPolicyFunc(active, maxActive *atomic.Int32) func(context.Context) (int, error) {
	return func(_ context.Context) (int, error) {
		current := active.Add(1)
		defer active.Add(-1)
		updateMaxActive(maxActive, current)
		time.Sleep(30 * time.Millisecond)
		return 1, nil
	}
}

func updateMaxActive(maxActive *atomic.Int32, current int32) {
	for {
		seen := maxActive.Load()
		if current <= seen || maxActive.CompareAndSwap(seen, current) {
			return
		}
	}
}
