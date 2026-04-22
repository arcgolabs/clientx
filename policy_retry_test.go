package clientx_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/DaiYuANg/arcgo/clientx"
)

func TestRetryPolicyRetriesUntilSuccess(t *testing.T) {
	attempts := 0
	policy := clientx.NewRetryPolicy(clientx.RetryPolicyConfig{
		MaxAttempts: 3,
		BaseDelay:   time.Millisecond,
		MaxDelay:    time.Millisecond,
		JitterRatio: 0,
	})

	result, err := clientx.InvokeWithPolicies(
		context.Background(),
		clientx.Operation{Protocol: clientx.ProtocolHTTP, Kind: clientx.OperationKindRequest, Op: "get"},
		func(_ context.Context) (string, error) {
			attempts++
			if attempts < 3 {
				return "", clientx.WrapError(clientx.ProtocolHTTP, "get", "example", context.DeadlineExceeded)
			}
			return "ok", nil
		},
		policy,
	)
	if err != nil {
		t.Fatalf("invoke failed: %v", err)
	}
	if result != "ok" {
		t.Fatalf("unexpected result: %q", result)
	}
	if attempts != 3 {
		t.Fatalf("expected 3 attempts, got %d", attempts)
	}
}

func TestRetryPolicyStopsAtMaxAttempts(t *testing.T) {
	attempts := 0
	policy := clientx.NewRetryPolicy(clientx.RetryPolicyConfig{
		MaxAttempts: 3,
		BaseDelay:   time.Millisecond,
		MaxDelay:    time.Millisecond,
		JitterRatio: 0,
	})

	_, err := clientx.InvokeWithPolicies(
		context.Background(),
		clientx.Operation{Protocol: clientx.ProtocolTCP, Kind: clientx.OperationKindDial, Op: "dial"},
		func(_ context.Context) (int, error) {
			attempts++
			return 0, clientx.WrapError(clientx.ProtocolTCP, "dial", "127.0.0.1:1", context.DeadlineExceeded)
		},
		policy,
	)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if attempts != 3 {
		t.Fatalf("expected 3 attempts, got %d", attempts)
	}
}

func TestRetryPolicySkipsNonRetryableError(t *testing.T) {
	attempts := 0
	policy := clientx.NewRetryPolicy(clientx.RetryPolicyConfig{MaxAttempts: 5})

	_, err := clientx.InvokeWithPolicies(
		context.Background(),
		clientx.Operation{Protocol: clientx.ProtocolUDP, Kind: clientx.OperationKindDial, Op: "dial"},
		func(_ context.Context) (int, error) {
			attempts++
			return 0, clientx.WrapErrorWithKind(clientx.ProtocolUDP, "dial", "127.0.0.1:1", clientx.ErrorKindCodec, errors.New("codec"))
		},
		policy,
	)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if attempts != 1 {
		t.Fatalf("expected 1 attempt, got %d", attempts)
	}
}

func TestRetryPolicyContextCancelDuringBackoff(t *testing.T) {
	attempts := 0
	policy := clientx.NewRetryPolicy(clientx.RetryPolicyConfig{
		MaxAttempts: 3,
		BaseDelay:   100 * time.Millisecond,
		MaxDelay:    100 * time.Millisecond,
		JitterRatio: 0,
	})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go func() {
		time.Sleep(20 * time.Millisecond)
		cancel()
	}()

	_, err := clientx.InvokeWithPolicies(
		ctx,
		clientx.Operation{Protocol: clientx.ProtocolHTTP, Kind: clientx.OperationKindRequest, Op: "get"},
		func(_ context.Context) (int, error) {
			attempts++
			return 0, clientx.WrapError(clientx.ProtocolHTTP, "get", "example", context.DeadlineExceeded)
		},
		policy,
	)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected canceled error, got %v", err)
	}
	if attempts != 1 {
		t.Fatalf("expected 1 attempt, got %d", attempts)
	}
}
