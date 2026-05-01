# p99

`p99` is a latency profiler for tail latency.

It probes an HTTP endpoint, records request latencies, reports percentiles, keeps bounded samples of the slowest requests, and writes run data that can be reported on or compared later.

The goal is to answer a practical question:

> Where is time actually going, and how bad is the tail?

Average latency is not a primary signal in `p99`. The first-class numbers are the distribution and the tail: p50, p90, p95, p99, p999, max, error rate, and error classes.

## Install

```sh
go install github.com/justin/p99/cmd/p99@latest
```

From a local checkout:

```sh
go build ./cmd/p99
```

## HTTP probing

```sh
p99 http https://api.example.com/users/123
```

Run at a target request rate with a concurrency limit:

```sh
p99 http --rps 100 --duration 2m --concurrency 20 https://api.example.com/search
```

Write a JSON run file:

```sh
p99 http --duration 30s --rps 50 --output run.json https://api.example.com/search
```

Use a method, headers, a request body, and an expected status:

```sh
p99 http \
  --method POST \
  --header 'Authorization: Bearer token' \
  --header 'Content-Type: application/json' \
  --body-file request.json \
  --status 201 \
  https://api.example.com/items
```

By default, any 2xx or 3xx response is treated as success. If `--status` is supplied, only those status codes are treated as success.

## Thresholds

Thresholds make `p99` useful in CI. A violated threshold exits non-zero.

```sh
p99 http https://api.example.com/search --p99-under 250ms --error-rate-under 0.5
```

`--error-rate-under` is expressed as a percent, so `0.5` means 0.5%.

Compare two saved runs and fail on a p99 regression:

```sh
p99 compare before.json after.json --max-p99-regression 20%
```

## Reports

Read a saved run file and print the terminal summary again:

```sh
p99 report run.json
```

Compare two runs:

```sh
p99 compare before.json after.json
```

The compare output includes changes in p50, p95, p99, p999, max, error rate, and request count.

## Error classes

HTTP probing classifies failures into a small set:

- `timeout`
- `DNS`
- `connection_refused`
- `TLS`
- `HTTP_4xx`
- `HTTP_5xx`
- `body/read_error`
- `unknown`

The classification is intentionally small. It is meant to make the terminal summary and JSON output useful without pretending to diagnose the whole system from one failed request.

## Shape analysis

`p99` includes conservative latency distribution heuristics:

- `stable`
- `spiky`
- `bimodal`
- `degrading_over_time`
- `insufficient_signal`

The shape analysis provides hints, not conclusions. For example, a bimodal distribution can happen with cache misses, retries, pool waits, or slow dependencies. It is a prompt for investigation, not proof.

## JSON output

A run file contains:

- probe config
- start and end timestamps
- summary statistics
- histogram buckets
- bounded slow request samples
- error breakdown
- shape analysis

Durations in JSON are stored as nanoseconds.

## Development

Run the test suite:

```sh
go test ./...
```

The tests use local `httptest.Server` instances and avoid real network calls.

The package layout is intentionally small:

```text
cmd/p99/
internal/cli/
internal/probe/
internal/latency/
internal/output/
internal/compare/
internal/errors/
internal/timeutil/
```

Future work is expected to build on these boundaries: live watch mode, pprof correlation, Go runtime signal correlation, span/dependency input, and additional export formats.
