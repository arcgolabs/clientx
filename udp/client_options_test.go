package udp_test

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/arcgolabs/clientx"
	clientudp "github.com/arcgolabs/clientx/udp"
)

func TestNewWithInvalidConfig(t *testing.T) {
	_, err := clientudp.New(clientudp.Config{})
	if err == nil {
		t.Fatal("expected config validation error, got nil")
	}
	if !errors.Is(err, clientudp.ErrInvalidConfig) {
		t.Fatalf("expected ErrInvalidConfig, got %v", err)
	}
}

func TestListenPolicyBeforeError(t *testing.T) {
	denyErr := errors.New("deny listen")
	client, err := clientudp.New(
		clientudp.Config{Address: "127.0.0.1:0"},
		clientudp.WithPolicies(clientx.PolicyFuncs{
			BeforeFunc: func(ctx context.Context, operation clientx.Operation) (context.Context, error) {
				if operation.Protocol != clientx.ProtocolUDP || operation.Kind != clientx.OperationKindListen {
					t.Fatalf("unexpected operation: %+v", operation)
				}
				return ctx, denyErr
			},
		}),
	)
	if err != nil {
		t.Fatalf("new client failed: %v", err)
	}
	defer closeClient(t, client)

	_, err = client.ListenPacket(context.Background())
	if !errors.Is(err, denyErr) {
		t.Fatalf("expected policy error, got %v", err)
	}
}

func TestDialWithConcurrencyLimitOption(t *testing.T) {
	var active atomic.Int32
	var maxActive atomic.Int32
	trackingPolicy := clientx.PolicyFuncs{
		BeforeFunc: func(ctx context.Context, _ clientx.Operation) (context.Context, error) {
			current := active.Add(1)
			updateMaxActive(&maxActive, current)
			time.Sleep(30 * time.Millisecond)
			return ctx, nil
		},
		AfterFunc: func(context.Context, clientx.Operation, error) error {
			active.Add(-1)
			return nil
		},
	}

	client, err := clientudp.New(
		clientudp.Config{Address: "127.0.0.1:1", DialTimeout: 150 * time.Millisecond},
		clientudp.WithConcurrencyLimit(1),
		clientudp.WithPolicies(trackingPolicy),
	)
	if err != nil {
		t.Fatalf("new client failed: %v", err)
	}
	defer closeClient(t, client)

	var wg sync.WaitGroup
	wg.Add(2)
	for range 2 {
		go func() {
			defer wg.Done()
			maybeDialAndClose(t, client)
		}()
	}
	wg.Wait()

	if got := maxActive.Load(); got != 1 {
		t.Fatalf("expected max in-flight 1, got %d", got)
	}
}

func TestDialWithTimeoutGuardOption(t *testing.T) {
	blockingPolicy := clientx.PolicyFuncs{
		BeforeFunc: func(ctx context.Context, _ clientx.Operation) (context.Context, error) {
			<-ctx.Done()
			return ctx, ctx.Err()
		},
	}

	client, err := clientudp.New(
		clientudp.Config{Address: "127.0.0.1:1", DialTimeout: time.Second},
		clientudp.WithTimeoutGuard(25*time.Millisecond),
		clientudp.WithPolicies(blockingPolicy),
	)
	if err != nil {
		t.Fatalf("new client failed: %v", err)
	}
	defer closeClient(t, client)

	_, err = client.Dial(context.Background())
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("expected deadline exceeded, got %v", err)
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

func maybeDialAndClose(t *testing.T, client clientudp.Client) {
	t.Helper()
	conn, err := client.Dial(context.Background())
	if err != nil {
		return
	}
	closeConn(t, conn)
}
