# Quick Start

This page walks through common first uses.

If the profiling concepts are new, start with [Learning p99](Learning-p99). It explains why each command matters before moving into command details.

## Probe One URL

```sh
p99 http https://api.example.com/users/123
```

This sends requests for the default duration and prints a summary.

Use it when:

- you want a quick latency snapshot
- you are checking whether an endpoint responds successfully
- you want to see p50/p95/p99 without setting up a full load test

## Probe With Load

```sh
p99 http --rps 100 --duration 2m --concurrency 20 https://api.example.com/search
```

This targets 100 requests per second for 2 minutes with at most 20 in-flight requests.

Use it when:

- you want repeatable pressure
- you want to compare before/after behavior
- you want to check the tail under a known request rate

## Save a Run

```sh
p99 http --duration 30s --rps 50 --output run.json https://api.example.com/search
```

Saved runs are useful because you can inspect them later, compare them, or export reports.

## Read a Saved Run

```sh
p99 report run.json
```

For more detail:

```sh
p99 report --details run.json
```

## Compare Two Runs

```sh
p99 compare before.json after.json
```

Fail if p99 regressed by more than 20 percent:

```sh
p99 compare before.json after.json --max-p99-regression 20%
```

## Watch an Endpoint

```sh
p99 watch --rps 50 --window 5s https://api.example.com/search
```

Use watch mode while deploying, changing configuration, or testing a service locally.

## Capture a Go CPU Profile

```sh
p99 profile --seconds 30 --output cpu.pprof http://localhost:8080/debug/pprof/profile
```

Use this when the target is a Go service with pprof enabled.

## Correlate Runtime Signals

```sh
p99 http --duration 30s --runtime http://localhost:8080 https://api.example.com/search
```

Use this to see whether Go runtime signals moved during the same window as latency.

## Analyze Spans

```sh
p99 spans traces.json
```

Use this when you have OpenTelemetry-style span JSON and want route/dependency timing.
