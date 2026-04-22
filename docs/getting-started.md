---
title: 'clientx Getting Started'
linkTitle: 'getting-started'
description: 'Use clientx/http with retries, timeouts, and typed errors'
weight: 2
---

## Getting Started

This page uses **`clientx/http`** only: construct a client from `Config`, run one request through `Execute`, and classify failures with `clientx.IsKind`.

`clienthttp.New` returns `(Client, error)` — always handle the construction error.

## 1) Install

```bash
go get github.com/arcgolabs/clientx@latest
go get github.com/arcgolabs/clientx/http@latest
```

## 2) Create `main.go`

`Execute` signature is `Execute(ctx context.Context, req *resty.Request, method, endpoint string)`. Pass `nil` for `req` to let the client build a default request from `R()`.

```go
package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/arcgolabs/clientx"
	clienthttp "github.com/arcgolabs/clientx/http"
)

func main() {
	ctx := context.Background()

	c, err := clienthttp.New(clienthttp.Config{
		BaseURL: "https://example.com",
		Timeout: 10 * time.Second,
		Retry: clientx.RetryConfig{
			Enabled:    true,
			MaxRetries: 2,
			WaitMin:    100 * time.Millisecond,
			WaitMax:    500 * time.Millisecond,
		},
	})
	if err != nil {
		log.Fatal(err)
	}
	defer func() { _ = c.Close() }()

	resp, err := c.Execute(ctx, nil, http.MethodGet, "/")
	if err != nil {
		if clientx.IsKind(err, clientx.ErrorKindTimeout) {
			fmt.Println("http timeout")
		}
		log.Fatal(err)
	}
	fmt.Println(resp.StatusCode())
}
```

## 3) Run

```bash
go mod init example.com/clientx-http
go get github.com/arcgolabs/clientx@latest
go get github.com/arcgolabs/clientx/http@latest
go run .
```

## Next

- TCP and UDP dial clients: [TCP and UDP](./tcp-and-udp)
- Codecs (TCP/UDP) and hooks: [Codec and hooks](./codec-and-hooks)
