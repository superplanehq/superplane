---
title: Performance Profiling
---

# Performance Profiling

The dev server exposes Go's [`net/http/pprof`](https://pkg.go.dev/net/http/pprof) endpoints so you can capture CPU, heap, goroutine, and contention profiles of the running backend. The whole application (public API, internal gRPC API, and all workers) runs as a single process, so one pprof server covers everything.

## How it is wired

The pprof HTTP server is started in [`pkg/server/server.go`](../../pkg/server/server.go) and is gated by environment variables:

| Env var | Default (dev) | Purpose |
| --- | --- | --- |
| `PPROF_ENABLED` | `yes` | Starts the pprof server only when set to `yes`. |
| `PPROF_PORT` | `6060` | Port the pprof server listens on. |

In the dev environment these default to on, and the port is published from the `app` container in [`docker-compose.dev.yml`](../../docker-compose.dev.yml). The endpoint is **unauthenticated** and exposes internal runtime details, so it is intentionally off by default in production: the release deployment does not set `PPROF_ENABLED`, so leave it unset there.

When enabled, the server also turns on block and mutex contention sampling (`runtime.SetBlockProfileRate` and `runtime.SetMutexProfileFraction`) so the `/debug/pprof/block` and `/debug/pprof/mutex` profiles are populated.

## Capturing profiles

Make sure the dev server is running (`make dev.up` then `make dev.server`). The port is published to the host, so you can hit it from either the host or inside the container.

### Makefile helpers

```bash
# 30-second CPU profile (override with SECONDS=<n>)
make profile.cpu

# Heap (in-use memory) profile
make profile.heap

# Quick dump of all goroutines (useful for stuck/leaked goroutines)
make profile.goroutines
```

### Raw pprof / curl

```bash
# CPU profile for 30s
go tool pprof "http://localhost:6060/debug/pprof/profile?seconds=30"

# Heap snapshot
go tool pprof "http://localhost:6060/debug/pprof/heap"

# Goroutine dump
curl "http://localhost:6060/debug/pprof/goroutine?debug=2"
```

If you don't have the Go toolchain on the host, run the same commands inside the container:

```bash
docker compose -f docker-compose.dev.yml exec app \
  go tool pprof "http://localhost:6060/debug/pprof/profile?seconds=30"
```

### Interactive flame graphs

For the interactive web UI (requires `graphviz` installed locally):

```bash
go tool pprof -http=:8082 "http://localhost:6060/debug/pprof/profile?seconds=30"
```

This opens a browser with flame graphs, call graphs, and source views.

## Profile types

| Endpoint | What it shows |
| --- | --- |
| `/debug/pprof/profile` | CPU usage sampled over a time window (`?seconds=N`). |
| `/debug/pprof/heap` | In-use and allocated heap memory. |
| `/debug/pprof/goroutine` | Stack traces of all current goroutines. |
| `/debug/pprof/block` | Where goroutines block on synchronization primitives. |
| `/debug/pprof/mutex` | Lock contention. |

Browse `http://localhost:6060/debug/pprof/` for the full index.
