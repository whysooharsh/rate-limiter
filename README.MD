# Rate Limiter

A lightweight Go service for enforcing token-bucket rate limits in front of an API. It supports an in-memory backend, a Redis-backed backend, and a small HTTP interface for checking and configuring limits.

## Quick start

```bash
git clone https://github.com/whysooharsh/rate-limiter
cd rate-limiter
go run main.go
```

The service starts on port 8080. Two default clients are seeded: `client1` with 10 tokens and `client2` with 5 tokens.

## API

### POST /check

Check whether a request should be allowed for a client.

```bash
curl -X POST http://localhost:8080/check \
  -H "Content-Type: application/json" \
  -d '{"client_id": "client1"}'
```

### GET /status/:clientID

Fetch the current token count for a client.

```bash
curl http://localhost:8080/status/client1
```

### POST /config

Create or update a client configuration.

```bash
curl -X POST http://localhost:8080/config \
  -H "Content-Type: application/json" \
  -d '{"client_id": "client1", "max_tokens": 20, "refill_rate": 2}'
```

## Benchmarks

Use the same honest test context:

> Benchmarked on an AMD Ryzen 5 7520U system with 4 cores/8 threads and 8 GB RAM. k6, the Go service, and Redis ran on the same machine over loopback. HTTP results use 200 concurrent virtual users for 30 seconds.

### HTTP results

| Backend              |  Throughput | Average latency |      p99 latency | Failed requests |
| -------------------- | ----------: | --------------: | ---------------: | --------------: |
| In-memory            | 19.8K req/s |         2.31 ms |          8.89 ms |              0% |
| Redis + Lua          | 14.3K req/s |         3.26 ms | Not measured yet |              0% |
| Unprotected baseline | 20.5K req/s |         2.22 ms |          8.85 ms |              0% |

The reported in-memory and baseline rows use the middle-throughput run from three repetitions. The Redis throughput and average latency are averages of three runs; Redis p99 will be added after repeating the test with p99 output enabled.

All HTTP tests used one client ID per run, making them hot-key workloads. These results describe this machine and topology and should not be treated as universal capacity figures.

### Core benchmarks

| Scenario                             |                    Result | Allocations |
| ------------------------------------ | ------------------------: | ----------: |
| Token decision with refill disabled  |  approximately 14.2 ns/op | 0 allocs/op |
| Parallel independent buckets, 8 CPUs |   approximately 171 ns/op | 0 allocs/op |
| Parallel shared hot key, 8 CPUs      | approximately 1,299 ns/op | 0 allocs/op |

Independent buckets demonstrated approximately 7.15x aggregate scaling from 1 to 8 CPUs. A shared hot key intentionally serializes updates to preserve correct token accounting.

### Reproduce

```bash
go test -race ./...

go test ./limiter ./store \
  -run='^$' \
  -bench=. \
  -benchmem \
  -cpu=1,2,4,8 \
  -count=3

CLIENT_ID="memory-$(date +%s)" \
VUS=200 \
DURATION=30s \
k6 run \
  --summary-trend-stats="avg,med,p(90),p(95),p(99),max" \
  tests/load_test.js

```
