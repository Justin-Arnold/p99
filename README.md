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

## Live watch

Watch mode runs repeated probe windows and redraws a compact terminal summary after each window:

```sh
p99 watch --rps 50 --window 5s --concurrency 10 https://api.example.com/search
```

For scripting or tests, run a fixed number of windows and disable terminal clearing:

```sh
p99 watch --window 5s --iterations 3 --clear=false https://api.example.com/search
```

Watch mode reports the same core latency and error signals as `p99 http`, but it is optimized for observing the tail while a service is changing.

## Go pprof capture

Capture a Go CPU profile from a pprof endpoint:

```sh
p99 profile --seconds 30 --output cpu.pprof http://localhost:8080/debug/pprof/profile
```

By default, `p99 profile` also tries to run:

```sh
go tool pprof -top
```

against the saved profile. If the Go toolchain is not available, the profile is still saved and the command reports that the top view was unavailable.

Disable the top view:

```sh
p99 profile --top=false --output cpu.pprof http://localhost:8080/debug/pprof/profile
```

Run an HTTP probe while the profile capture is active:

```sh
p99 profile \
  --seconds 30 \
  --output cpu.pprof \
  --probe https://api.example.com/search \
  --probe-rps 100 \
  --probe-concurrency 20 \
  http://localhost:8080/debug/pprof/profile
```

This prints the profile capture details and the correlated HTTP latency summary. It does not claim that a CPU hot spot caused a latency spike; it puts the signals next to each other so the next investigation step is grounded.

## Go runtime signal correlation

`p99` can collect Go runtime signals before and after a measured window and print the movement beside latency or profile results.

Run a standalone runtime correlation window:

```sh
p99 runtime --duration 10s http://localhost:8080
```

Correlate runtime movement with an HTTP probe:

```sh
p99 http \
  --duration 30s \
  --rps 100 \
  --concurrency 20 \
  --runtime http://localhost:8080 \
  https://api.example.com/search
```

Use the same runtime correlation in watch mode:

```sh
p99 watch --window 5s --runtime http://localhost:8080 https://api.example.com/search
```

Or while capturing a CPU profile:

```sh
p99 profile \
  --seconds 30 \
  --runtime http://localhost:8080 \
  --probe https://api.example.com/search \
  http://localhost:8080/debug/pprof/profile
```

Runtime collection reads common Go debug endpoints from the base URL:

- `/debug/vars` for expvar `memstats`
- `/debug/pprof/goroutine?debug=1` for goroutine count
- `/debug/pprof/mutex?debug=1` for mutex profile sample presence
- `/debug/pprof/block?debug=1` for block profile sample presence

It reports:

- goroutine count
- heap allocation, heap in-use, heap system bytes
- GC cycles
- GC pause total and last GC pause
- mutex profile samples
- block profile samples
- scheduler pause distribution when exported through expvar-compatible runtime metrics
- DB pool wait counters when the service exports compatible wait count and duration fields

Some signals require the target service to opt in. `net/http/pprof` exposes pprof handlers. `expvar` exposes `memstats`. Mutex and block profiles are only useful when the service enables runtime profiling with `runtime.SetMutexProfileFraction` or `runtime.SetBlockProfileRate`. DB pool waits are not a standard pprof signal; `p99` reads common expvar-style fields when an application exports them.

Runtime hints are deliberately conservative. They describe signals that moved during the same window as tail latency, not root cause.

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
- runtime correlation data when `--runtime` is used

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
internal/profile/
internal/timeutil/
```

Future work is expected to build on these boundaries: Go runtime signal correlation, span/dependency input, richer profile analysis, and additional export formats.
