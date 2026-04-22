---
title: 'clientx TCP and UDP'
linkTitle: 'tcp-and-udp'
description: 'Dial TCP and UDP clients with shared timeouts and error kinds'
weight: 3
---

## TCP and UDP

`clientx/tcp` and `clientx/udp` expose `New(cfg, opts...) (Client, error)`. Both honor the shared `*clientx.Error` model and `clientx.IsKind` checks.

These samples dial **local addresses** — start a matching listener first, or expect dial errors at runtime.

## 1) Install

```bash
go get github.com/DaiYuANg/arcgo/clientx@latest
go get github.com/DaiYuANg/arcgo/clientx/tcp@latest
go get github.com/DaiYuANg/arcgo/clientx/udp@latest
```

## 2) TCP client

```go
package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/DaiYuANg/arcgo/clientx"
	"github.com/DaiYuANg/arcgo/clientx/tcp"
)

func main() {
	ctx := context.Background()

	c, err := tcp.New(tcp.Config{
		Address:      "127.0.0.1:9000",
		DialTimeout:  time.Second,
		ReadTimeout:  2 * time.Second,
		WriteTimeout: 2 * time.Second,
	})
	if err != nil {
		log.Fatal(err)
	}
	defer func() { _ = c.Close() }()

	conn, err := c.Dial(ctx)
	if err != nil {
		if clientx.IsKind(err, clientx.ErrorKindConnRefused) {
			fmt.Println("tcp conn refused")
		}
		log.Fatal(err)
	}
	defer func() { _ = conn.Close() }()
}
```

## 3) UDP client

```go
package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/DaiYuANg/arcgo/clientx"
	"github.com/DaiYuANg/arcgo/clientx/udp"
)

func main() {
	ctx := context.Background()

	c, err := udp.New(udp.Config{
		Address:      "127.0.0.1:9001",
		DialTimeout:  time.Second,
		ReadTimeout:  500 * time.Millisecond,
		WriteTimeout: 500 * time.Millisecond,
	})
	if err != nil {
		log.Fatal(err)
	}
	defer func() { _ = c.Close() }()

	conn, err := c.Dial(ctx)
	if err != nil {
		log.Fatal(err)
	}
	defer func() { _ = conn.Close() }()

	_, err = conn.Write([]byte("ping"))
	if err != nil && clientx.IsKind(err, clientx.ErrorKindTimeout) {
		fmt.Println("udp write timeout")
	}
}
```

## Next

- Framed codecs on top of TCP/UDP: [Codec and hooks](./codec-and-hooks)
- HTTP client: [Getting Started](./getting-started)
