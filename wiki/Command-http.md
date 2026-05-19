# Command: p99 http

`p99 http` probes an HTTP target and reports latency, errors, distribution shape, slow samples, and optional runtime/export data.

## Problem It Solves

You need to know what users experience at the tail under a known request pattern. `p99 http` gives you a repeatable probe that can be saved, compared, and used in CI.

## Basic Use

```sh
p99 http https://api.example.com/search
```

## Common Use

```sh
p99 http \
  --duration 2m \
  --rps 100 \
  --concurrency 20 \
  --timeout 2s \
  --output run.json \
  https://api.example.com/search
```

## Flags

### `--duration`

Measured probe duration.

Example:

```sh
p99 http --duration 2m https://api.example.com/search
```

Use it to control how much data is collected. Short runs are useful for quick checks. Longer runs are better for tail confidence.

### `--rps`

Target requests per second.

Example:

```sh
p99 http --rps 100 https://api.example.com/search
```

Use it to apply consistent pressure. If you compare runs, keep this value the same.

### `--concurrency`

Maximum number of in-flight requests.

Example:

```sh
p99 http --rps 100 --concurrency 20 https://api.example.com/search
```

This prevents the probe from creating unlimited concurrent work if the target slows down.

### `--warmup`

Warmup duration before measurements are recorded.

Example:

```sh
p99 http --warmup 10s --duration 1m https://api.example.com/search
```

Use it when the first few requests are not representative because of cold connections, caches, or initialization.

### `--timeout`

Per-request timeout.

Example:

```sh
p99 http --timeout 500ms https://api.example.com/search
```

Timeouts are counted and classified. Use a timeout that matches your user or service budget.

### `--method`

HTTP method.

Example:

```sh
p99 http --method POST https://api.example.com/items
```

Use it for non-GET probes.

### `--header` and `-H`

Add a request header. Can be repeated.

Example:

```sh
p99 http --header 'Authorization: Bearer token' --header 'Content-Type: application/json' https://api.example.com/search
```

Use headers for authentication, content type, tenant selection, or routing behavior.

### `--body-file`

Read the request body from a file.

Example:

```sh
p99 http --method POST --body-file request.json https://api.example.com/search
```

Use it when probing POST, PUT, or PATCH endpoints.

### `--status`

Expected HTTP status code. Can be repeated or comma-separated.

Example:

```sh
p99 http --status 200 --status 204 https://api.example.com/search
```

By default, 2xx and 3xx are treated as success. Use `--status` when the endpoint has a narrower expected result.

### `--output` and `--out`

Write the full JSON run file.

Example:

```sh
p99 http --output run.json https://api.example.com/search
```

Use this whenever you may want to compare, report, or inspect the run later.

### `--slow-samples`

Number of slow request samples to retain.

Example:

```sh
p99 http --slow-samples 20 https://api.example.com/search
```

Use more samples when you need concrete examples of slow requests.

### `--p99-under`

Fail if measured p99 exceeds a duration.

Example:

```sh
p99 http --p99-under 250ms https://api.example.com/search
```

Use this in CI to enforce a tail latency budget.

### `--error-rate-under`

Fail if error rate exceeds a percentage.

Example:

```sh
p99 http --error-rate-under 0.5 https://api.example.com/search
```

`0.5` means 0.5 percent.

### `--runtime`

Collect Go runtime signals from a base URL before and after the probe.

Example:

```sh
p99 http --runtime http://localhost:8080 https://api.example.com/search
```

Use it when the target is a Go service and you want GC, heap, goroutine, mutex/block, or DB-pool-wait signals beside latency.

### `--runtime-timeout`

Timeout for runtime endpoint requests.

Example:

```sh
p99 http --runtime http://localhost:8080 --runtime-timeout 2s https://api.example.com/search
```

Use it when runtime endpoints are local and should respond quickly, or remote and need more time.

### `--markdown-output`

Write a Markdown report.

Example:

```sh
p99 http --markdown-output run.md https://api.example.com/search
```

Use it for human review in pull requests, incidents, or notes.

### `--prometheus-output`

Write Prometheus text exposition metrics.

Example:

```sh
p99 http --prometheus-output run.prom https://api.example.com/search
```

Use it when another tool expects Prometheus-style metrics.

### `--otel-output`

Write OpenTelemetry-compatible JSON metrics.

Example:

```sh
p99 http --otel-output run.otlp.json https://api.example.com/search
```

Use it when you want to feed run metrics into an OTLP-style pipeline.
