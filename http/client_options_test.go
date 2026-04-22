package http_test

import (
	"context"
	"errors"
	stdhttp "net/http"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/arcgolabs/clientx"
	clienthttp "github.com/arcgolabs/clientx/http"
)

func TestExecuteWithConcurrencyLimitOption(t *testing.T) {
	var active atomic.Int32
	var maxActive atomic.Int32

	srv := newHTTPTestServer(stdhttp.HandlerFunc(func(w stdhttp.ResponseWriter, _ *stdhttp.Request) {
		current := active.Add(1)
		defer active.Add(-1)
		updateMaxActive(&maxActive, current)
		time.Sleep(40 * time.Millisecond)
		w.WriteHeader(stdhttp.StatusNoContent)
	}))
	defer srv.Close()

	client, err := clienthttp.New(
		clienthttp.Config{BaseURL: srv.URL, Timeout: 2 * time.Second},
		clienthttp.WithConcurrencyLimit(1),
	)
	if err != nil {
		t.Fatalf("new client failed: %v", err)
	}
	defer mustCloseClient(t, client)

	errCh := make(chan error, 2)
	var wg sync.WaitGroup
	wg.Add(2)
	for range 2 {
		go func() {
			defer wg.Done()
			_, execErr := client.Execute(context.Background(), nil, stdhttp.MethodGet, "/health")
			errCh <- execErr
		}()
	}
	wg.Wait()
	close(errCh)

	for execErr := range errCh {
		if execErr != nil {
			t.Fatalf("execute failed: %v", execErr)
		}
	}

	if got := maxActive.Load(); got != 1 {
		t.Fatalf("expected max in-flight 1, got %d", got)
	}
}

func TestExecuteWithTimeoutGuardOption(t *testing.T) {
	srv := newHTTPTestServer(stdhttp.HandlerFunc(func(w stdhttp.ResponseWriter, _ *stdhttp.Request) {
		time.Sleep(120 * time.Millisecond)
		w.WriteHeader(stdhttp.StatusNoContent)
	}))
	defer srv.Close()

	client, err := clienthttp.New(
		clienthttp.Config{
			BaseURL: srv.URL,
			Timeout: time.Second,
		},
		clienthttp.WithTimeoutGuard(30*time.Millisecond),
	)
	if err != nil {
		t.Fatalf("new client failed: %v", err)
	}
	defer mustCloseClient(t, client)

	_, err = client.Execute(context.Background(), nil, stdhttp.MethodGet, "/slow")
	if err == nil {
		t.Fatal("expected timeout guard error, got nil")
	}
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("expected context deadline exceeded, got %v", err)
	}
	if !clientx.IsKind(err, clientx.ErrorKindTimeout) {
		t.Fatalf("expected timeout kind, got %q", clientx.KindOf(err))
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
