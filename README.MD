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

> Benchmarked on an AMD Ryzen 5 7520U system with 4 cores/8 threads and 8 GB RAM. `hey`, the Go service, and Redis all ran on the same machine over loopback. HTTP results use 200 concurrent connections for 30 seconds. (Earlier runs used k6 as the load generator; k6 itself was consuming up to 600% CPU on this machine, which was competing with the server under test — these results were re-measured with `hey`, a lighter-weight client, to remove that confound.)

### HTTP results

| Backend              |  Throughput | Average latency | p99 latency | Failed requests |
| -------------------- | ----------: | --------------: | ----------: | --------------: |
| In-memory            | 30.1K req/s |          6.6 ms |     17.2 ms |              0% |
| Redis + Lua          | 19.4K req/s |         10.3 ms |     14.7 ms |              0% |
| Unprotected baseline | 32.7K req/s |          6.1 ms |     16.5 ms |              0% |

Client was reconfigured to an effectively unlimited bucket (`max_tokens`/`refill_rate` set very high) before each run, so these numbers reflect allowed-traffic throughput rather than mostly-rejected traffic. All HTTP tests used one client ID per run, making them hot-key workloads. These results describe this machine and topology and should not be treated as universal capacity figures.

**Note:** profiling under load (`pprof`) showed the token bucket logic itself accounts for a small fraction of total CPU time — the majority goes to the Go HTTP stack itself (socket syscalls, request/response buffering, timestamping), which is expected for any HTTP service at this scale. Profiling also surfaced a real bug: `/check` was making two separate calls into the store per request instead of one, adding latency and a small race window between them — both backends now return the decision and updated token count from a single atomic call.

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

curl -X POST http://localhost:8080/config \
  -H "Content-Type: application/json" \
  -d '{"client_id":"hotclient","max_tokens":1000000000,"refill_rate":1000000000}'

hey -z 30s -c 200 -m POST -H 'Content-Type: application/json' \
  -d '{"client_id":"hotclient"}' http://localhost:8080/check
```
