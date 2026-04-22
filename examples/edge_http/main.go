// Package main demonstrates the edge HTTP preset client.
package main

import (
	"context"
	"fmt"
	"io"
	"log"
	stdhttp "net/http"
	"net/http/httptest"
	"time"

	clienthttp "github.com/DaiYuANg/arcgo/clientx/http"
	"github.com/DaiYuANg/arcgo/clientx/preset"
)

func main() {
	srv := httptest.NewServer(stdhttp.HandlerFunc(func(w stdhttp.ResponseWriter, r *stdhttp.Request) {
		if r.URL.Path != "/ping" {
			w.WriteHeader(stdhttp.StatusNotFound)
			return
		}
		w.WriteHeader(stdhttp.StatusOK)
		if _, err := w.Write([]byte("pong")); err != nil {
			return
		}
	}))
	defer srv.Close()

	client, err := preset.NewEdgeHTTP(
		clienthttp.Config{BaseURL: srv.URL},
		preset.WithEdgeHTTPDisableRetry(),
		preset.WithEdgeHTTPTimeout(2*time.Second),
		preset.WithEdgeHTTPTimeoutGuard(1500*time.Millisecond),
	)
	if err != nil {
		panic(err)
	}
	defer func() {
		if closeErr := client.Close(); closeErr != nil {
			log.Printf("close edge HTTP client: %v", closeErr)
		}
	}()

	resp, err := client.Execute(context.Background(), nil, stdhttp.MethodGet, "/ping")
	if err != nil {
		panic(err)
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		panic(err)
	}
	if _, err = fmt.Printf("status=%d body=%q\n", resp.StatusCode(), string(body)); err != nil {
		log.Printf("print edge HTTP result: %v", err)
	}
}
