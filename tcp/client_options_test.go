package tcp_test

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/arcgolabs/clientx"
	clienttcp "github.com/arcgolabs/clientx/tcp"
)

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

	client, err := clienttcp.New(
		clienttcp.Config{Address: "127.0.0.1:1", DialTimeout: 150 * time.Millisecond},
		clienttcp.WithConcurrencyLimit(1),
		clienttcp.WithPolicies(trackingPolicy),
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

	client, err := clienttcp.New(
		clienttcp.Config{Address: "127.0.0.1:1", DialTimeout: time.Second},
		clienttcp.WithTimeoutGuard(25*time.Millisecond),
		clienttcp.WithPolicies(blockingPolicy),
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

func maybeDialAndClose(t *testing.T, client clienttcp.Client) {
	t.Helper()
	conn, err := client.Dial(context.Background())
	if err != nil {
		return
	}
	closeConn(t, conn)
}
