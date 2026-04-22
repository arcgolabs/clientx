---
title: 'clientx Codec and Hooks'
linkTitle: 'codec-and-hooks'
description: 'TCP/UDP codecs, framers, and shared dial/IO hooks'
weight: 4
---

## Codec and hooks

- **HTTP** follows normal request/response semantics; TCP/UDP can stack **`clientx/codec`** with an optional **framer** (length-prefixed frames on TCP).
- **Hooks** (`clientx.Hook`, `clientx.HookFuncs`) are shared across `clientx/http`, `clientx/tcp`, and `clientx/udp`.

For wiring **`observabilityx`** into hooks, see the tests in [`clientx/hook_observability_test.go`](https://github.com/DaiYuANg/arcgo/blob/main/clientx/hook_observability_test.go).

## 1) Install

```bash
go get github.com/DaiYuANg/arcgo/clientx@latest
go get github.com/DaiYuANg/arcgo/clientx/tcp@latest
go get github.com/DaiYuANg/arcgo/clientx/udp@latest
go get github.com/DaiYuANg/arcgo/clientx/codec@latest
go get github.com/DaiYuANg/arcgo/clientx/http@latest
```

## 2) Register a custom codec

Built-ins include `json`, `text`, and `bytes`. Register your own `codec.Codec` by name:

```go
package main

import (
	"fmt"
	"log"

	"github.com/DaiYuANg/arcgo/clientx/codec"
)

type reverseCodec struct{}

func (reverseCodec) Name() string { return "reverse" }

func (reverseCodec) Marshal(v any) ([]byte, error) {
	s, ok := v.(string)
	if !ok {
		return nil, fmt.Errorf("reverse: want string")
	}
	runes := []rune(s)
	for i, j := 0, len(runes)-1; i < j; i, j = i+1, j-1 {
		runes[i], runes[j] = runes[j], runes[i]
	}
	return []byte(string(runes)), nil
}

func (reverseCodec) Unmarshal(data []byte, v any) error {
	p, ok := v.(*string)
	if !ok || p == nil {
		return fmt.Errorf("reverse: want *string")
	}
	runes := []rune(string(data))
	for i, j := 0, len(runes)-1; i < j; i, j = i+1, j-1 {
		runes[i], runes[j] = runes[j], runes[i]
	}
	*p = string(runes)
	return nil
}

func main() {
	if err := codec.Register(reverseCodec{}); err != nil {
		log.Fatal(err)
	}
	c := codec.Must("reverse")
	out, err := c.Marshal("abc")
	if err != nil {
		log.Fatal(err)
	}
	var decoded string
	if err := c.Unmarshal(out, &decoded); err != nil {
		log.Fatal(err)
	}
	fmt.Println(decoded)
}
```

## 3) TCP: codec + length-prefixed framer

`DialCodec` needs a `codec.Codec` and a `codec.Framer`. This example assumes a compatible server is listening.

```go
package main

import (
	"context"
	"log"
	"time"

	"github.com/DaiYuANg/arcgo/clientx/codec"
	"github.com/DaiYuANg/arcgo/clientx/tcp"
)

func main() {
	ctx := context.Background()

	c, err := tcp.New(tcp.Config{
		Address:     "127.0.0.1:9000",
		DialTimeout: time.Second,
	})
	if err != nil {
		log.Fatal(err)
	}
	defer func() { _ = c.Close() }()

	cc, err := c.DialCodec(ctx, codec.JSON, codec.NewLengthPrefixed(1024*1024))
	if err != nil {
		log.Fatal(err)
	}
	defer func() { _ = cc.Close() }()

	_ = cc.WriteValue(map[string]string{"message": "ping"})
	var out map[string]string
	_ = cc.ReadValue(&out)
}
```

## 4) UDP: codec without framer

```go
package main

import (
	"context"
	"log"

	"github.com/DaiYuANg/arcgo/clientx/codec"
	"github.com/DaiYuANg/arcgo/clientx/udp"
)

func main() {
	ctx := context.Background()

	c, err := udp.New(udp.Config{Address: "127.0.0.1:9001"})
	if err != nil {
		log.Fatal(err)
	}
	defer func() { _ = c.Close() }()

	uc, err := c.DialCodec(ctx, codec.JSON)
	if err != nil {
		log.Fatal(err)
	}
	defer func() { _ = uc.Close() }()

	_ = uc.WriteValue(map[string]string{"message": "ping"})
	var out map[string]string
	_ = uc.ReadValue(&out)
}
```

## 5) Hooks (dial / IO lifecycle)

```go
package main

import (
	"log"
	"time"

	"github.com/DaiYuANg/arcgo/clientx"
	clienthttp "github.com/DaiYuANg/arcgo/clientx/http"
	"github.com/DaiYuANg/arcgo/clientx/tcp"
	"github.com/DaiYuANg/arcgo/clientx/udp"
)

func main() {
	h := clientx.HookFuncs{
		OnDialFunc: func(e clientx.DialEvent) {
			_ = e
		},
		OnIOFunc: func(e clientx.IOEvent) {
			_ = e
		},
	}

	httpC, err := clienthttp.New(clienthttp.Config{
		BaseURL: "https://example.com",
		Timeout: 5 * time.Second,
	}, clienthttp.WithHooks(h))
	if err != nil {
		log.Fatal(err)
	}
	defer func() { _ = httpC.Close() }()

	tcpC, err := tcp.New(tcp.Config{Address: "127.0.0.1:9000"}, tcp.WithHooks(h))
	if err != nil {
		log.Fatal(err)
	}
	defer func() { _ = tcpC.Close() }()

	udpC, err := udp.New(udp.Config{Address: "127.0.0.1:9001"}, udp.WithHooks(h))
	if err != nil {
		log.Fatal(err)
	}
	defer func() { _ = udpC.Close() }()
}
```

## Related

- [Getting Started](./getting-started)
- [TCP and UDP](./tcp-and-udp)
