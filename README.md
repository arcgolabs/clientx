## clientx

`clientx` is a protocol-oriented client package set for common network protocols.

- First-wave protocols: `http`, `tcp`, `udp`
- Shared primitives: `RetryConfig`, `TLSConfig`, typed errors (`*clientx.Error`), and optional **hooks** for dial / I/O lifecycle
- Constructors return interfaces (`http.Client`, `tcp.Client`, `udp.Client`) so implementations stay replaceable

## Current capabilities

- **`clientx/http`** — resty-based HTTP client wrapper with retry, TLS, and header options; `Execute` runs through shared policies.
- **`clientx/tcp`** — dial with timeout-wrapped `net.Conn`, optional TLS, and optional **codec + framer** (`DialCodec`).
- **`clientx/udp`** — UDP dial/listen baseline with timeout-wrapped conns and optional `DialCodec`.
- **`clientx/codec`** — pluggable codecs (`json`, `text`, `bytes`) plus registration for custom codecs; TCP pairs with length-prefixed framers.

## Package layout

- Shared errors, hooks, policies: `github.com/arcgolabs/clientx`
- HTTP client: `github.com/arcgolabs/clientx/http`
- TCP client: `github.com/arcgolabs/clientx/tcp`
- UDP client: `github.com/arcgolabs/clientx/udp`
- Codecs and framers: `github.com/arcgolabs/clientx/codec`

## Documentation map

- Release notes: [clientx v0.3.0](./release-v0.3.0)
- HTTP-only quick path: [Getting Started](./getting-started)
- TCP and UDP dial: [TCP and UDP](./tcp-and-udp)
- Codecs (TCP/UDP) and hooks: [Codec and hooks](./codec-and-hooks)

## Install / Import

```bash
go get github.com/arcgolabs/clientx@latest
go get github.com/arcgolabs/clientx/http@latest
go get github.com/arcgolabs/clientx/tcp@latest
go get github.com/arcgolabs/clientx/udp@latest
```

## Error model

- Transport errors are wrapped as `*clientx.Error`.
- Use `clientx.KindOf(err)` or `clientx.IsKind(err, kind)` for category checks.
- Wrapped errors preserve `Unwrap()` (`errors.Is` / `errors.As`).
- Timeout-shaped errors remain compatible with `net.Error` timeout checks where applicable.

## Integration guide

- **configx** — centralize retry/TLS/timeout presets, then inject into `Config` structs.
- **dix** — register `http.Client` / `tcp.Client` / `udp.Client` as interfaces from modules.
- **observabilityx** — use `clientx.NewObservabilityHook` (see package tests) to attach metrics/tracing to dial and I/O hooks.
- **logx** — avoid high-cardinality remote addresses in default structured fields unless intentional.

## Runnable examples (repository)

- [examples/clientx/edge_http](https://github.com/DaiYuANg/arcgo/tree/main/examples/clientx/edge_http)
- [examples/clientx/internal_rpc_tcp](https://github.com/DaiYuANg/arcgo/tree/main/examples/clientx/internal_rpc_tcp)
- [examples/clientx/low_latency_udp](https://github.com/DaiYuANg/arcgo/tree/main/examples/clientx/low_latency_udp)

```bash
go run ./examples/clientx/edge_http
go run ./examples/clientx/internal_rpc_tcp
go run ./examples/clientx/low_latency_udp
```

## Testing and production notes

- Prefer interface-returning constructors in tests; swap fakes/mocks at boundaries.
- Set timeouts at client construction; avoid ad-hoc per-call timeout sprawl.
- Prefer `IsKind` over string matching for retry and alerting policies.

## Notes

- `clientx` is still evolving; program against exported interfaces, not concrete types.
- Internal packages may share helpers (including `collectionx`); treat that as implementation detail unless documented.
