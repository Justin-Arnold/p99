# Command: p99 spans

`p99 spans` reads OpenTelemetry-style span JSON and reports route, service, dependency, and slow trace timing.

## Problem It Solves

Synthetic probes tell you what happened from the outside. Spans can show where request time was represented inside the system. Use `spans` when you need route or dependency timing from trace data.

## Basic Use

```sh
p99 spans traces.json
```

## Input Format

The input may be:

- OTLP JSON with `resourceSpans`, `scopeSpans`, and `spans`
- a flat JSON array of span objects

`p99` reads common OpenTelemetry fields such as `traceId`, `spanId`, `parentSpanId`, `name`, `kind`, `startTimeUnixNano`, `endTimeUnixNano`, `attributes`, `status`, and `service.name`.

## Flags

### `--json-out`

Write the analyzed span report as JSON.

Use it when you want to save the analysis for later.

### `--markdown-out`

Write a Markdown span report.

Use it for human review in incident notes or pull requests.

### `--prometheus-out`

Write Prometheus text exposition metrics.

Use it when you want span-derived metrics in a Prometheus-compatible artifact.

### `--otel-out`

Write OpenTelemetry-compatible JSON metrics.

Use it when another pipeline expects OTLP-style metric JSON.

### `--slow-samples`

Number of slow traces to retain.

Use more samples when you need more concrete examples of slow traces.

### `--format`

Terminal output format.

Supported values:

- `text`
- `json`
- `markdown`
- `prometheus`
- `otel`

Use `text` for humans, `json` for lossless report data, and metrics formats for integrations.

## How p99 Interprets Spans

For each trace, `p99` uses the server span as the request span. If there is no server span, it falls back to a root span with no parent.

Dependency spans are identified from span kind and common attributes such as `db.system`, `rpc.system`, `peer.service`, and network target attributes.

Dependency time may overlap. It is not exclusive wall-clock time.
