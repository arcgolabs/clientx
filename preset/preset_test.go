package preset_test

import (
	"context"
	"errors"
	stdhttp "net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/arcgolabs/clientx"
	clienthttp "github.com/arcgolabs/clientx/http"
	"github.com/arcgolabs/clientx/preset"
	clienttcp "github.com/arcgolabs/clientx/tcp"
	clientudp "github.com/arcgolabs/clientx/udp"
	"resty.dev/v3"
)

func TestNewEdgeHTTPRetryPreset(t *testing.T) {
	srv := httptest.NewServer(stdhttp.HandlerFunc(func(w stdhttp.ResponseWriter, _ *stdhttp.Request) {
		w.WriteHeader(stdhttp.StatusNoContent)
	}))
	defer srv.Close()

	var attempts atomic.Int32
	client, err := preset.NewEdgeHTTP(
		clienthttp.Config{BaseURL: srv.URL},
		preset.WithEdgeHTTPRetry(clientx.RetryConfig{
			Enabled:    true,
			MaxRetries: 2,
			WaitMin:    time.Millisecond,
			WaitMax:    2 * time.Millisecond,
		}),
		preset.WithEdgeHTTPOption(clienthttp.WithRequestMiddleware(func(_ *resty.Client, _ *resty.Request) error {
			if attempts.Add(1) < 3 {
				return context.DeadlineExceeded
			}
			return nil
		})),
	)
	if err != nil {
		t.Fatalf("new edge http client failed: %v", err)
	}
	defer closeHTTPClient(t, client)

	resp, err := client.Execute(context.Background(), nil, stdhttp.MethodGet, "/health")
	if err != nil {
		t.Fatalf("execute failed: %v (attempts=%d)", err, attempts.Load())
	}
	if resp.StatusCode() != stdhttp.StatusNoContent {
		t.Fatalf("expected status %d, got %d", stdhttp.StatusNoContent, resp.StatusCode())
	}
	if got := attempts.Load(); got != 3 {
		t.Fatalf("expected 3 attempts, got %d", got)
	}
}

func TestNewEdgeHTTPTimeoutGuard(t *testing.T) {
	srv := httptest.NewServer(stdhttp.HandlerFunc(func(w stdhttp.ResponseWriter, _ *stdhttp.Request) {
		time.Sleep(120 * time.Millisecond)
		w.WriteHeader(stdhttp.StatusNoContent)
	}))
	defer srv.Close()

	client, err := preset.NewEdgeHTTP(
		clienthttp.Config{BaseURL: srv.URL},
		preset.WithEdgeHTTPTimeout(2*time.Second),
		preset.WithEdgeHTTPTimeoutGuard(25*time.Millisecond),
		preset.WithEdgeHTTPDisableRetry(),
	)
	if err != nil {
		t.Fatalf("new edge http client failed: %v", err)
	}
	defer closeHTTPClient(t, client)

	_, err = client.Execute(context.Background(), nil, stdhttp.MethodGet, "/slow")
	if err == nil {
		t.Fatal("expected timeout guard error, got nil")
	}
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("expected deadline exceeded, got %v", err)
	}
}

func TestNewInternalRPCTimeoutGuard(t *testing.T) {
	blockingPolicy := clientx.PolicyFuncs{
		BeforeFunc: func(ctx context.Context, _ clientx.Operation) (context.Context, error) {
			<-ctx.Done()
			return ctx, ctx.Err()
		},
	}

	client, err := preset.NewInternalRPC(
		clienttcp.Config{Address: "127.0.0.1:1"},
		preset.WithInternalRPCDisableRetry(),
		preset.WithInternalRPCTimeoutGuard(25*time.Millisecond),
		preset.WithInternalRPCOption(clienttcp.WithPolicies(blockingPolicy)),
	)
	if err != nil {
		t.Fatalf("new internal rpc client failed: %v", err)
	}
	defer closeTCPClient(t, client)

	_, err = client.Dial(context.Background())
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("expected deadline exceeded, got %v", err)
	}
}

func TestNewLowLatencyUDPConcurrencyLimit(t *testing.T) {
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

	client, err := preset.NewLowLatencyUDP(
		clientudp.Config{Address: "127.0.0.1:1"},
		preset.WithLowLatencyUDPConcurrencyLimit(1),
		preset.WithLowLatencyUDPOption(clientudp.WithPolicies(trackingPolicy)),
	)
	if err != nil {
		t.Fatalf("new low latency udp client failed: %v", err)
	}
	defer closeUDPClient(t, client)

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

func closeHTTPClient(t *testing.T, client clienthttp.Client) {
	t.Helper()
	if err := client.Close(); err != nil {
		t.Fatalf("close http client: %v", err)
	}
}

func closeTCPClient(t *testing.T, client clienttcp.Client) {
	t.Helper()
	if err := client.Close(); err != nil {
		t.Fatalf("close tcp client: %v", err)
	}
}

func closeUDPClient(t *testing.T, client clientudp.Client) {
	t.Helper()
	if err := client.Close(); err != nil {
		t.Fatalf("close udp client: %v", err)
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
	if err := conn.Close(); err != nil {
		t.Fatalf("close udp connection: %v", err)
	}
}
