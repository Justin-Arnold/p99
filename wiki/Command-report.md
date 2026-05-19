# Command: p99 report

`p99 report` reads a saved HTTP run JSON file and prints or exports a report.

## Problem It Solves

You often need to inspect a run after it was captured. `report` lets you keep probe collection separate from analysis and export.

## Basic Use

```sh
p99 report run.json
```

## Detailed Report

```sh
p99 report --details run.json
```

This includes histogram buckets, slow samples, shape details, and runtime signals when present.

## Flags

### `--format`

Output format.

Supported values:

- `text`
- `json`
- `markdown`
- `prometheus`
- `otel`

Use `text` for terminal reading, `markdown` for human reports, `json` for saved data, and metrics formats for integrations.

### `--output`

Write report output to a file.

Example:

```sh
p99 report --format markdown --output run.md run.json
```

Use it when generating artifacts in CI or attaching reports to an incident.

### `--details`

Include all detail sections in text output.

Use it when you want a full saved-run inspection.

### `--histogram`

Include histogram buckets.

Use it when you want to understand the distribution shape beyond percentiles.

### `--slow-samples`

Include retained slow request samples.

Use it when you need concrete examples behind the tail.

### `--shape`

Include detailed shape analysis.

Use it when you want p99's conservative distribution interpretation and extra explanation.

### `--runtime`

Include runtime correlation if the run contains it.

Use it when the original run used `p99 http --runtime`.
