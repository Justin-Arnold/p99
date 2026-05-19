# Command: p99 runtime

`p99 runtime` collects Go runtime signals before and after a time window and reports the movement.

## Problem It Solves

Tail latency can correlate with runtime behavior: GC pauses, heap growth, goroutine buildup, mutex contention, blocking, scheduler pauses, or DB pool waits. `runtime` gives a focused before/after view without running an HTTP probe.

## Basic Use

```sh
p99 runtime --duration 10s http://localhost:8080
```

The base URL should expose Go debug endpoints such as `/debug/vars` and `/debug/pprof/goroutine`.

## Flags

### `--duration`

Window length between the before and after snapshots.

Use it to match the interval you want to inspect.

### `--timeout`

Timeout for each runtime endpoint request.

Use it to keep missing or slow debug endpoints from hanging the command.

## Signals

`p99 runtime` reads:

- `/debug/vars` for expvar `memstats`
- `/debug/pprof/goroutine?debug=1`
- `/debug/pprof/mutex?debug=1`
- `/debug/pprof/block?debug=1`

Some signals require the target service to enable profiling. Mutex and block profiles are only meaningful if the service configured `runtime.SetMutexProfileFraction` or `runtime.SetBlockProfileRate`.
