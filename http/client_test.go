package http_test

import (
	"context"
	"errors"
	stdhttp "net/http"
	"net/http/httptest"
	"reflect"
	"sync/atomic"
	"testing"
	"time"

	"github.com/arcgolabs/clientx"
	clienthttp "github.com/arcgolabs/clientx/http"
	"github.com/samber/lo"
)

func TestExecuteWithNilRequest(t *testing.T) {
	srv := newHTTPTestServer(stdhttp.HandlerFunc(func(w stdhttp.ResponseWriter, _ *stdhttp.Request) {
		w.WriteHeader(stdhttp.StatusNoContent)
	}))
	defer srv.Close()

	client, err := clienthttp.New(clienthttp.Config{
		BaseURL: srv.URL,
		Timeout: time.Second,
	})
	if err != nil {
		t.Fatalf("new client failed: %v", err)
	}
	defer mustCloseClient(t, client)

	resp, err := client.Execute(context.Background(), nil, stdhttp.MethodGet, "/health")
	if err != nil {
		t.Fatalf("execute failed: %v", err)
	}
	if resp.StatusCode() != stdhttp.StatusNoContent {
		t.Fatalf("expected status %d, got %d", stdhttp.StatusNoContent, resp.StatusCode())
	}
}

func TestExecuteWrapsTransportError(t *testing.T) {
	client, err := clienthttp.New(clienthttp.Config{
		BaseURL: "http://127.0.0.1:1",
		Timeout: 150 * time.Millisecond,
	})
	if err != nil {
		t.Fatalf("new client failed: %v", err)
	}
	defer mustCloseClient(t, client)

	_, err = client.Execute(context.Background(), client.R(), stdhttp.MethodGet, "")
	if err == nil {
		t.Fatal("expected transport error, got nil")
	}
	typedErr, ok := errors.AsType[*clientx.Error](err)
	if !ok {
		t.Fatalf("expected *clientx.Error, got %T", err)
	}
	if typedErr.Protocol != clientx.ProtocolHTTP {
		t.Fatalf("expected protocol %q, got %q", clientx.ProtocolHTTP, typedErr.Protocol)
	}
	if typedErr.Op != "get" {
		t.Fatalf("expected op get, got %q", typedErr.Op)
	}
	if !lo.Contains([]clientx.ErrorKind{
		clientx.ErrorKindConnRefused,
		clientx.ErrorKindTimeout,
		clientx.ErrorKindNetwork,
	}, clientx.KindOf(err)) {
		t.Fatalf("unexpected error kind: %q", clientx.KindOf(err))
	}
}

func TestExecuteEmitsHook(t *testing.T) {
	srv := newHTTPTestServer(stdhttp.HandlerFunc(func(w stdhttp.ResponseWriter, _ *stdhttp.Request) {
		mustWriteBody(t, w, "ok")
	}))
	defer srv.Close()

	var got clientx.IOEvent
	client, err := clienthttp.New(
		clienthttp.Config{
			BaseURL: srv.URL,
			Timeout: time.Second,
		},
		clienthttp.WithHooks(clientx.HookFuncs{
			OnIOFunc: func(event clientx.IOEvent) {
				got = event
			},
		}),
	)
	if err != nil {
		t.Fatalf("new client failed: %v", err)
	}
	defer mustCloseClient(t, client)

	_, err = client.Execute(context.Background(), nil, stdhttp.MethodGet, "/health")
	if err != nil {
		t.Fatalf("execute failed: %v", err)
	}
	if got.Protocol != clientx.ProtocolHTTP {
		t.Fatalf("expected protocol %q, got %q", clientx.ProtocolHTTP, got.Protocol)
	}
	if got.Op != "get" {
		t.Fatalf("expected op get, got %q", got.Op)
	}
	if got.Bytes == 0 {
		t.Fatalf("expected response bytes > 0, got %d", got.Bytes)
	}
}

func TestNewWithInvalidBaseURL(t *testing.T) {
	_, err := clienthttp.New(clienthttp.Config{BaseURL: "://bad"})
	if err == nil {
		t.Fatal("expected config validation error, got nil")
	}
	if !errors.Is(err, clienthttp.ErrInvalidConfig) {
		t.Fatalf("expected ErrInvalidConfig, got %v", err)
	}
}

func TestExecuteAppliesPolicies(t *testing.T) {
	srv := newHTTPTestServer(stdhttp.HandlerFunc(func(w stdhttp.ResponseWriter, _ *stdhttp.Request) {
		w.WriteHeader(stdhttp.StatusNoContent)
	}))
	defer srv.Close()

	calls := make([]string, 0, 3)
	client, err := clienthttp.New(
		clienthttp.Config{BaseURL: srv.URL, Timeout: time.Second},
		clienthttp.WithPolicies(clientx.PolicyFuncs{
			BeforeFunc: func(ctx context.Context, operation clientx.Operation) (context.Context, error) {
				calls = append(calls, "before")
				if operation.Protocol != clientx.ProtocolHTTP || operation.Kind != clientx.OperationKindRequest {
					t.Fatalf("unexpected operation: %+v", operation)
				}
				return ctx, nil
			},
			AfterFunc: func(_ context.Context, _ clientx.Operation, _ error) error {
				calls = append(calls, "after")
				return nil
			},
		}),
	)
	if err != nil {
		t.Fatalf("new client failed: %v", err)
	}
	defer mustCloseClient(t, client)

	_, err = client.Execute(context.Background(), nil, stdhttp.MethodGet, "/health")
	if err != nil {
		t.Fatalf("execute failed: %v", err)
	}
	if !reflect.DeepEqual(calls, []string{"before", "after"}) {
		t.Fatalf("unexpected policy calls: %v", calls)
	}
}

func TestExecuteRetriesFromConfig(t *testing.T) {
	srv := newHTTPTestServer(stdhttp.HandlerFunc(func(w stdhttp.ResponseWriter, _ *stdhttp.Request) {
		w.WriteHeader(stdhttp.StatusNoContent)
	}))
	defer srv.Close()

	var attempts atomic.Int32
	client, err := clienthttp.New(
		clienthttp.Config{
			BaseURL: srv.URL,
			Timeout: time.Second,
			Retry: clientx.RetryConfig{
				Enabled:    true,
				MaxRetries: 2,
				WaitMin:    time.Millisecond,
				WaitMax:    2 * time.Millisecond,
			},
		},
		clienthttp.WithRequestMiddleware(func(_ *resty.Client, _ *resty.Request) error {
			if attempts.Add(1) < 3 {
				return context.DeadlineExceeded
			}
			return nil
		}),
	)
	if err != nil {
		t.Fatalf("new client failed: %v", err)
	}
	defer mustCloseClient(t, client)

	resp, err := client.Execute(context.Background(), nil, stdhttp.MethodGet, "/health")
	if err != nil {
		t.Fatalf("execute with retry failed: %v", err)
	}
	if resp.StatusCode() != stdhttp.StatusNoContent {
		t.Fatalf("expected status %d, got %d", stdhttp.StatusNoContent, resp.StatusCode())
	}
	if got := attempts.Load(); got != 3 {
		t.Fatalf("expected 3 attempts, got %d", got)
	}
}

func newHTTPTestServer(handler stdhttp.Handler) *httptest.Server {
	server := httptest.NewUnstartedServer(handler)
	server.Config.ReadHeaderTimeout = time.Second
	server.Start()
	return server
}

func mustCloseClient(t *testing.T, client clienthttp.Client) {
	t.Helper()
	if err := client.Close(); err != nil {
		t.Fatalf("close client failed: %v", err)
	}
}

func mustWriteBody(t *testing.T, w stdhttp.ResponseWriter, body string) {
	t.Helper()
	if _, err := w.Write([]byte(body)); err != nil {
		t.Fatalf("write response body failed: %v", err)
	}
}
