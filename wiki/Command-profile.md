# Command: p99 profile

`p99 profile` captures a Go pprof profile and can run a latency probe and runtime correlation during the same window.

## Problem It Solves

When tail latency is bad, you may need to know what Go code was doing at the same time. `profile` saves the raw pprof file and optionally prints `go tool pprof -top`.

## Basic Use

```sh
p99 profile --seconds 30 --output cpu.pprof http://localhost:8080/debug/pprof/profile
```

## With Probe Correlation

```sh
p99 profile \
  --seconds 30 \
  --probe https://api.example.com/search \
  --probe-rps 100 \
  http://localhost:8080/debug/pprof/profile
```

## Flags

### `--seconds`

Profile capture duration.

Use longer captures for more representative CPU samples. Use shorter captures for quick local checks.

### `--timeout`

Timeout for the profile HTTP request.

Set this above `--seconds` so the profile endpoint has time to complete.

### `--output` and `--out`

Path for the saved pprof file.

Use it whenever you want to inspect the profile later with `go tool pprof`.

### `--top`

Whether to run `go tool pprof -top` after capture.

Example:

```sh
p99 profile --top=false --output cpu.pprof http://localhost:8080/debug/pprof/profile
```

Disable it when the Go toolchain is not available or when you only need the raw file.

### `--top-count`

Number of pprof rows to print.

Use it to keep terminal output short or expand it for deeper review.

### `--probe`

HTTP URL to probe while the profile is captured.

Use it to put CPU profile data next to user-facing latency data.

### `--probe-duration`

Duration for the correlated HTTP probe.

Defaults to the profile duration. Override it if you need a shorter or longer probe window.

### `--probe-rps`

Target requests per second for the correlated probe.

Use it to create realistic load while profiling.

### `--probe-concurrency`

Maximum in-flight requests for the correlated probe.

Use it to bound probe pressure.

### `--runtime`

Go runtime base URL for before/after runtime signal correlation.

Use it to see GC, goroutine, heap, mutex/block, scheduler, and DB-pool-wait movement during the profile window.

### `--runtime-timeout`

Timeout for runtime endpoint requests.

Use it to keep runtime collection bounded.
