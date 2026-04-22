package clientx_test

import (
	"context"
	"errors"
	"reflect"
	"testing"
	"time"

	"github.com/arcgolabs/clientx"
)

// testCtxKey is a private type for context keys in tests,
// avoiding the SA1029 lint warning about using built-in string as key.
type testCtxKey string

const ctxKeyV testCtxKey = "v"

type panicRetryPolicy struct{}

func (p panicRetryPolicy) Before(ctx context.Context, _ clientx.Operation) (context.Context, error) {
	return ctx, nil
}

func (p panicRetryPolicy) After(_ context.Context, _ clientx.Operation, _ error) error {
	return nil
}

func (p panicRetryPolicy) ShouldRetry(_ context.Context, _ clientx.Operation, _ int, _ error) (bool, time.Duration) {
	panic("retry decider panic")
}

func TestInvokeWithPoliciesOrder(t *testing.T) {
	calls := make([]string, 0, 8)
	p1 := clientx.PolicyFuncs{
		BeforeFunc: func(ctx context.Context, _ clientx.Operation) (context.Context, error) {
			calls = append(calls, "before1")
			return context.WithValue(ctx, ctxKeyV, "ok"), nil
		},
		AfterFunc: func(_ context.Context, _ clientx.Operation, _ error) error {
			calls = append(calls, "after1")
			return nil
		},
	}
	p2 := clientx.PolicyFuncs{
		BeforeFunc: func(ctx context.Context, _ clientx.Operation) (context.Context, error) {
			calls = append(calls, "before2")
			got, ok := ctx.Value(ctxKeyV).(string)
			if !ok || got != "ok" {
				t.Fatalf("expected ctx value propagated, got %q", got)
			}
			return ctx, nil
		},
		AfterFunc: func(_ context.Context, _ clientx.Operation, _ error) error {
			calls = append(calls, "after2")
			return nil
		},
	}

	out, err := clientx.InvokeWithPolicies(
		context.Background(),
		clientx.Operation{Protocol: clientx.ProtocolHTTP, Kind: clientx.OperationKindRequest, Op: "get"},
		func(_ context.Context) (string, error) {
			calls = append(calls, "execute")
			return "done", nil
		},
		p1,
		p2,
	)
	if err != nil {
		t.Fatalf("invoke failed: %v", err)
	}
	if out != "done" {
		t.Fatalf("unexpected output: %q", out)
	}

	expected := []string{"before1", "before2", "execute", "after2", "after1"}
	if !reflect.DeepEqual(calls, expected) {
		t.Fatalf("unexpected call order: got=%v want=%v", calls, expected)
	}
}

func TestInvokeWithPoliciesBeforeError(t *testing.T) {
	boom := errors.New("boom")
	called := false

	_, err := clientx.InvokeWithPolicies(
		context.Background(),
		clientx.Operation{Protocol: clientx.ProtocolTCP, Kind: clientx.OperationKindDial, Op: "dial"},
		func(_ context.Context) (int, error) {
			called = true
			return 1, nil
		},
		clientx.PolicyFuncs{BeforeFunc: func(ctx context.Context, _ clientx.Operation) (context.Context, error) {
			return ctx, boom
		}},
	)
	if !errors.Is(err, boom) {
		t.Fatalf("expected before error, got %v", err)
	}
	if called {
		t.Fatal("expected invoke function not to be called")
	}
}

func TestInvokeWithPoliciesAfterErrorJoin(t *testing.T) {
	baseErr := errors.New("base")
	afterErr := errors.New("after")

	_, err := clientx.InvokeWithPolicies(
		context.Background(),
		clientx.Operation{Protocol: clientx.ProtocolUDP, Kind: clientx.OperationKindListen, Op: "listen"},
		func(_ context.Context) (int, error) {
			return 0, baseErr
		},
		clientx.PolicyFuncs{AfterFunc: func(_ context.Context, _ clientx.Operation, _ error) error {
			return afterErr
		}},
	)
	if !errors.Is(err, baseErr) || !errors.Is(err, afterErr) {
		t.Fatalf("expected joined errors, got %v", err)
	}
}

func TestInvokeWithPoliciesBeforePanicIsolation(t *testing.T) {
	called := false

	out, err := clientx.InvokeWithPolicies(
		context.Background(),
		clientx.Operation{Protocol: clientx.ProtocolHTTP, Kind: clientx.OperationKindRequest, Op: "get"},
		func(_ context.Context) (string, error) {
			called = true
			return "ok", nil
		},
		clientx.PolicyFuncs{
			BeforeFunc: func(_ context.Context, _ clientx.Operation) (context.Context, error) {
				panic("before panic")
			},
		},
	)
	if err != nil {
		t.Fatalf("invoke failed: %v", err)
	}
	if out != "ok" {
		t.Fatalf("unexpected output: %q", out)
	}
	if !called {
		t.Fatal("expected invoke function to be called")
	}
}

func TestInvokeWithPoliciesAfterPanicIsolation(t *testing.T) {
	_, err := clientx.InvokeWithPolicies(
		context.Background(),
		clientx.Operation{Protocol: clientx.ProtocolTCP, Kind: clientx.OperationKindDial, Op: "dial"},
		func(_ context.Context) (int, error) {
			return 1, nil
		},
		clientx.PolicyFuncs{
			AfterFunc: func(_ context.Context, _ clientx.Operation, _ error) error {
				panic("after panic")
			},
		},
	)
	if err != nil {
		t.Fatalf("invoke failed: %v", err)
	}
}

func TestInvokeWithPoliciesRetryDeciderPanicIsolation(t *testing.T) {
	boom := errors.New("boom")
	attempts := 0

	_, err := clientx.InvokeWithPolicies(
		context.Background(),
		clientx.Operation{Protocol: clientx.ProtocolUDP, Kind: clientx.OperationKindDial, Op: "dial"},
		func(_ context.Context) (int, error) {
			attempts++
			return 0, boom
		},
		panicRetryPolicy{},
	)
	if !errors.Is(err, boom) {
		t.Fatalf("expected boom error, got %v", err)
	}
	if attempts != 1 {
		t.Fatalf("expected 1 attempt, got %d", attempts)
	}
}
