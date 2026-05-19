# Run Files and Reports

`p99 http --output run.json` saves a full run file.

Run files contain:

- probe configuration
- start and end timestamps
- summary statistics
- histogram buckets
- slow request samples
- error breakdown
- shape analysis
- runtime correlation when requested

Durations are stored as nanoseconds.

## Why Save Runs

Saved runs let you:

- compare before and after
- generate reports later
- export metrics
- keep CI artifacts
- inspect slow samples after a test finishes

## Read a Run

```sh
p99 report run.json
```

## Detailed Report

```sh
p99 report --details run.json
```

This is the best starting point when a saved run looks bad.

## Markdown Report

```sh
p99 report --format markdown --output run.md run.json
```

Use Markdown for human review.

## Prometheus Metrics

```sh
p99 report --format prometheus run.json
```

Use Prometheus output for systems that understand text exposition.

## OpenTelemetry Metrics

```sh
p99 report --format otel run.json
```

Use OpenTelemetry-compatible JSON when an OTLP-style pipeline expects metrics.
